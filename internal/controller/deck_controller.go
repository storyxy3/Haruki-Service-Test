package controller

import (
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"Haruki-Service-API/internal/builder"
	"Haruki-Service-API/internal/model"
	"Haruki-Service-API/internal/service"
	"Haruki-Service-API/pkg/asset"
	"Haruki-Service-API/pkg/masterdata"
)

// DeckController handles deck module endpoints.
type DeckController struct {
	drawing     *service.DrawingService
	cards       service.CardDataSource
	events      service.EventDataSource
	assets      *asset.AssetHelper
	assetDir    string
	userData    *service.UserDataService
	recommender service.DeckRecommender
}

func NewDeckController(
	drawing *service.DrawingService,
	cards service.CardDataSource,
	events service.EventDataSource,
	assets *asset.AssetHelper,
	userData *service.UserDataService,
	recommender service.DeckRecommender,
) *DeckController {
	assetDir := ""
	if assets != nil {
		assetDir = assets.Primary()
	}
	return &DeckController{
		drawing:     drawing,
		cards:       cards,
		events:      events,
		assets:      assets,
		assetDir:    assetDir,
		userData:    userData,
		recommender: recommender,
	}
}

func (c *DeckController) ensure() error {
	if c == nil || c.drawing == nil {
		return fmt.Errorf("deck controller is not initialized")
	}
	return nil
}

func (c *DeckController) BuildDeckRecommendRequest(req map[string]interface{}) (map[string]interface{}, error) {
	if err := c.ensure(); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, fmt.Errorf("deck request is empty")
	}

	region, _ := req["region"].(string)
	if strings.TrimSpace(region) == "" {
		return nil, fmt.Errorf("deck request missing region")
	}

	if _, ok := req["profile"]; !ok {
		return nil, fmt.Errorf("deck request missing profile")
	}
	deckData, ok := req["deck_data"]
	if !ok {
		return nil, fmt.Errorf("deck request missing deck_data")
	}
	if list, ok := deckData.([]interface{}); ok && len(list) == 0 {
		return nil, fmt.Errorf("deck request deck_data is empty")
	}

	return req, nil
}

func (c *DeckController) RenderDeckRecommend(req map[string]interface{}) ([]byte, error) {
	payload, err := c.BuildDeckRecommendRequest(req)
	if err != nil {
		return nil, err
	}
	return c.drawing.GenerateDeckRecommend(payload)
}

func (c *DeckController) BuildDeckRecommendAutoRequest(query model.DeckAutoQuery) (map[string]interface{}, error) {
	if err := c.ensure(); err != nil {
		return nil, err
	}
	if c.recommender != nil && c.recommender.Enabled() {
		payload, err := c.buildDeckRecommendAutoWithBackend(query)
		if err == nil {
			return payload, nil
		}
	}
	return c.buildDeckRecommendAutoLocal(query)
}

func (c *DeckController) buildDeckRecommendAutoWithBackend(query model.DeckAutoQuery) (map[string]interface{}, error) {
	if c.recommender == nil || !c.recommender.Enabled() {
		return nil, fmt.Errorf("deck recommender backend not enabled")
	}
	if c.cards == nil {
		return nil, fmt.Errorf("deck card source not configured")
	}
	if c.userData == nil {
		return nil, fmt.Errorf("user data is required for deck auto recommend")
	}
	userBytes, err := c.userData.RawBytes()
	if err != nil {
		return nil, err
	}
	region, recType, err := c.normalizeDeckAutoQuery(query)
	if err != nil {
		return nil, err
	}
	option, err := c.buildBackendOption(region, recType, query)
	if err != nil {
		return nil, err
	}
	expanded := c.recommender.ExpandAlgorithms(option)
	result, err := c.recommender.Recommend(service.DeckRecommendRequest{
		Region:      region,
		UserData:    userBytes,
		MusicMeta:   c.userData.MusicMetaBytes(),
		BatchOption: expanded,
	})
	if err != nil {
		return nil, err
	}
	return c.buildDrawingPayloadFromBackendResult(region, recType, query, result)
}

func (c *DeckController) buildDeckRecommendAutoLocal(query model.DeckAutoQuery) (map[string]interface{}, error) {
	if c.cards == nil {
		return nil, fmt.Errorf("deck card source not configured")
	}
	if c.userData == nil || c.userData.GetRawData() == nil {
		return nil, fmt.Errorf("user data is required for deck auto recommend")
	}

	region, recType, err := c.normalizeDeckAutoQuery(query)
	if err != nil {
		return nil, err
	}

	raw := c.userData.GetRawData()
	candidates := make([]deckCandidate, 0, len(raw.UserCards))
	for _, uc := range raw.UserCards {
		card, err := c.cards.GetCardByID(uc.CardID)
		if err != nil || card == nil {
			continue
		}
		candidates = append(candidates, deckCandidate{
			card:     card,
			userCard: uc,
			power:    calculateDeckCardPower(card),
		})
	}
	if len(candidates) == 0 {
		return nil, fmt.Errorf("no available user cards for deck recommend")
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].power == candidates[j].power {
			return candidates[i].card.ID < candidates[j].card.ID
		}
		return candidates[i].power > candidates[j].power
	})

	limit := query.Limit
	if limit <= 0 || limit > 5 {
		limit = 5
	}
	if len(candidates) < limit {
		limit = len(candidates)
	}

	deckCards := make([]map[string]interface{}, 0, limit)
	totalPower := 0
	for _, pick := range candidates[:limit] {
		totalPower += pick.power
		thumb := builder.BuildCardThumbnail(
			c.assets,
			c.assetDir,
			pick.card,
			builder.ThumbnailOptions{
				AfterTraining: service.IsAfterTraining(&pick.userCard),
				TrainRank:     intPtr(pick.userCard.MasterRank),
				Level:         intPtr(pick.userCard.Level),
				IsPcard:       false,
			},
		)

		eventBonus := 0.0
		if recType == "event" || recType == "bonus" {
			eventBonus = 20.0
		}
		deckCards = append(deckCards, map[string]interface{}{
			"card_thumbnail":    thumb,
			"chara_id":          pick.card.CharacterID,
			"skill_level":       "4",
			"is_after_training": service.IsAfterTraining(&pick.userCard),
			"skill_rate":        120.0,
			"event_bonus_rate":  eventBonus,
			"is_before_story":   true,
			"is_after_story":    true,
			"has_canvas_bonus":  false,
		})
	}

	profile := c.userData.DetailedProfile(region)
	if profile == nil {
		now := time.Now().Unix()
		profile = &model.DetailedProfileCardRequest{
			ID:              "1",
			Region:          strings.ToUpper(region),
			Nickname:        "Unknown",
			Source:          "local_fallback",
			UpdateTime:      now,
			IsHideUID:       true,
			LeaderImagePath: "",
			HasFrame:        false,
		}
	}

	score := totalPower * 3
	deckData := map[string]interface{}{
		"card_data":               deckCards,
		"score":                   score,
		"live_score":              score,
		"mysekai_event_point":     score,
		"event_bonus_rate":        20.0,
		"support_deck_bonus_rate": 0.0,
		"multi_live_score_up":     20.0,
		"total_power":             totalPower,
	}

	payload := map[string]interface{}{
		"region":         region,
		"profile":        profile,
		"deck_data":      []map[string]interface{}{deckData},
		"recommend_type": recType,
		"model_name":     []string{"dfs"},
		"cost_times":     map[string]float64{"dfs": 0.01},
		"wait_times":     map[string]float64{"dfs": 0.00},
		"target":         "score",
	}

	if recType == "challenge" {
		payload["live_type"] = "single"
	} else {
		payload["live_type"] = "multi"
	}

	if query.EventID != nil && *query.EventID > 0 {
		payload["event_id"] = *query.EventID
		payload["event_name"] = fmt.Sprintf("Event #%d", *query.EventID)
		if c.events != nil {
			if event, err := c.events.GetEventByID(*query.EventID); err == nil && event != nil {
				payload["event_name"] = event.Name
				banner := c.resolveEventBannerPath(event.AssetBundleName)
				if banner != "" {
					payload["event_banner_path"] = banner
				}
			}
		}
	}

	return payload, nil
}

func (c *DeckController) normalizeDeckAutoQuery(query model.DeckAutoQuery) (region string, recType string, err error) {
	region = strings.ToLower(strings.TrimSpace(query.Region))
	if region == "" && c.cards != nil {
		region = strings.ToLower(strings.TrimSpace(c.cards.DefaultRegion()))
	}
	if region == "" {
		region = "jp"
	}
	recType = strings.ToLower(strings.TrimSpace(query.RecommendType))
	if recType == "" {
		recType = "event"
	}
	switch recType {
	case "event", "challenge", "no_event", "bonus", "mysekai":
	default:
		return "", "", fmt.Errorf("unsupported recommend_type: %s", recType)
	}
	return region, recType, nil
}

func (c *DeckController) buildBackendOption(region, recType string, query model.DeckAutoQuery) (map[string]interface{}, error) {
	eventID := 0
	if query.EventID != nil && *query.EventID > 0 {
		eventID = *query.EventID
	}
	if eventID == 0 && recType != "no_event" && recType != "challenge" {
		if id := c.pickCurrentOrNextEventID(); id > 0 {
			eventID = id
		}
	}
	limit := query.Limit
	if limit <= 0 {
		limit = 4
	}
	option := map[string]interface{}{
		"region":                       region,
		"algorithm":                    "all",
		"timeout_ms":                   60000,
		"limit":                        limit,
		"target":                       "score",
		"live_type":                    "multi",
		"member":                       5,
		"multi_live_teammate_power":    250000,
		"multi_live_teammate_score_up": 200,
		"rarity_1_config":              defaultDeckConfig12(),
		"rarity_2_config":              defaultDeckConfig12(),
		"rarity_3_config":              defaultDeckConfig34bd(),
		"rarity_4_config":              defaultDeckConfig34bd(),
		"rarity_birthday_config":       defaultDeckConfig34bd(),
		"single_card_configs":          []interface{}{},
		"fixed_cards":                  []int{},
		"fixed_characters":             []int{},
		"best_skill_as_leader":         true,
		"keep_after_training_state":    false,
	}
	switch recType {
	case "challenge":
		option["live_type"] = "challenge"
		option["event_id"] = nil
	case "no_event":
		option["live_type"] = "multi"
		option["event_id"] = nil
	case "bonus":
		option["algorithm"] = "dfs"
		option["live_type"] = "solo"
		option["target"] = "bonus"
		option["target_bonus_list"] = pickBonusTargets(query.TargetBonuses, query.Args)
		option["rarity_1_config"] = noChangeDeckConfig()
		option["rarity_2_config"] = noChangeDeckConfig()
		option["rarity_3_config"] = noChangeDeckConfig()
		option["rarity_4_config"] = noChangeDeckConfig()
		option["rarity_birthday_config"] = noChangeDeckConfig()
		if eventID > 0 {
			option["event_id"] = eventID
		}
	case "mysekai":
		option["algorithm"] = "ga"
		option["live_type"] = "mysekai"
		option["event_id"] = nil
		option["rarity_1_config"] = noChangeDeckConfig()
		option["rarity_2_config"] = noChangeDeckConfig()
		option["rarity_3_config"] = noChangeDeckConfig()
		option["rarity_4_config"] = noChangeDeckConfig()
		option["rarity_birthday_config"] = noChangeDeckConfig()
	default:
		if eventID > 0 {
			option["event_id"] = eventID
		}
	}
	return option, nil
}

func (c *DeckController) buildDrawingPayloadFromBackendResult(region, recType string, query model.DeckAutoQuery, result *service.DeckRecommendResult) (map[string]interface{}, error) {
	if result == nil || len(result.Decks) == 0 {
		return nil, fmt.Errorf("deck recommender returned no deck results")
	}
	profile := c.userData.DetailedProfile(region)
	if profile == nil {
		now := time.Now().Unix()
		profile = &model.DetailedProfileCardRequest{
			ID:              "1",
			Region:          strings.ToUpper(region),
			Nickname:        "Unknown",
			Source:          "deck_recommend_backend",
			UpdateTime:      now,
			IsHideUID:       true,
			LeaderImagePath: "",
		}
	}
	userCardMap := map[int]service.RawUserCard{}
	if raw := c.userData.GetRawData(); raw != nil {
		for _, uc := range raw.UserCards {
			userCardMap[uc.CardID] = uc
		}
	}
	deckData := make([]map[string]interface{}, 0, len(result.Decks))
	for _, d := range result.Decks {
		cardData := make([]map[string]interface{}, 0, len(d.Cards))
		for _, dc := range d.Cards {
			card, err := c.cards.GetCardByID(dc.CardID)
			if err != nil || card == nil {
				continue
			}
			userCard := userCardMap[dc.CardID]
			afterTraining := strings.EqualFold(dc.DefaultImage, "special_training")
			if !afterTraining {
				afterTraining = service.IsAfterTraining(&userCard)
			}
			level := userCard.Level
			if level <= 0 {
				level = 60
			}
			masterRank := userCard.MasterRank
			thumb := builder.BuildCardThumbnail(c.assets, c.assetDir, card, builder.ThumbnailOptions{
				AfterTraining: afterTraining,
				TrainRank:     intPtr(masterRank),
				Level:         intPtr(level),
				IsPcard:       false,
			})
			cardData = append(cardData, map[string]interface{}{
				"card_thumbnail":    thumb,
				"chara_id":          card.CharacterID,
				"skill_level":       fmt.Sprintf("%d", dc.SkillLevel),
				"is_after_training": afterTraining,
				"skill_rate":        dc.SkillRate,
				"event_bonus_rate":  dc.EventBonusRate,
				"is_before_story":   dc.IsBeforeStory,
				"is_after_story":    dc.IsAfterStory,
				"has_canvas_bonus":  dc.HasCanvasBonus,
			})
		}
		deckData = append(deckData, map[string]interface{}{
			"card_data":               cardData,
			"score":                   d.Score,
			"live_score":              d.LiveScore,
			"mysekai_event_point":     d.MysekaiEventPoint,
			"event_bonus_rate":        d.EventBonusRate,
			"support_deck_bonus_rate": d.SupportDeckBonusRate,
			"multi_live_score_up":     d.MultiLiveScoreUp,
			"total_power":             d.TotalPower,
			"challenge_score_delta":   d.ChallengeScoreDelta,
		})
	}
	payload := map[string]interface{}{
		"region":         region,
		"profile":        profile,
		"deck_data":      deckData,
		"recommend_type": recType,
		"target":         "score",
		"model_name":     result.DeckAlgs,
		"cost_times":     result.CostTimes,
		"wait_times":     result.WaitTimes,
	}
	if recType == "challenge" {
		payload["live_type"] = "single"
	} else {
		payload["live_type"] = "multi"
	}
	if query.EventID != nil && *query.EventID > 0 {
		payload["event_id"] = *query.EventID
	}
	return payload, nil
}

func (c *DeckController) RenderDeckRecommendAuto(query model.DeckAutoQuery) ([]byte, error) {
	payload, err := c.BuildDeckRecommendAutoRequest(query)
	if err != nil {
		return nil, err
	}
	return c.drawing.GenerateDeckRecommend(payload)
}

type deckCandidate struct {
	card     *masterdata.Card
	userCard service.RawUserCard
	power    int
}

func calculateDeckCardPower(card *masterdata.Card) int {
	if card == nil {
		return 0
	}
	var p1, p2, p3 int
	for _, param := range card.CardParameters {
		switch param.CardParameterType {
		case "param1":
			if param.Power > p1 {
				p1 = param.Power
			}
		case "param2":
			if param.Power > p2 {
				p2 = param.Power
			}
		case "param3":
			if param.Power > p3 {
				p3 = param.Power
			}
		}
	}
	return p1 + p2 + p3 + card.SpecialTrainingPower1BonusFixed + card.SpecialTrainingPower2BonusFixed + card.SpecialTrainingPower3BonusFixed
}

func (c *DeckController) resolveEventBannerPath(assetBundleName string) string {
	if c.assets == nil || c.assetDir == "" || strings.TrimSpace(assetBundleName) == "" {
		return ""
	}
	return asset.ResolveAssetPath(
		c.assets,
		c.assetDir,
		filepath.Join("home", "banner", assetBundleName, assetBundleName+".png"),
		filepath.Join("event", assetBundleName, "banner.png"),
	)
}

func (c *DeckController) pickCurrentOrNextEventID() int {
	if c.events == nil {
		return 0
	}
	now := time.Now().UnixMilli()
	events := c.events.GetEvents()
	var current *masterdata.Event
	var next *masterdata.Event
	for _, ev := range events {
		if ev == nil {
			continue
		}
		if ev.StartAt <= now && now <= ev.AggregateAt {
			if current == nil || ev.StartAt > current.StartAt {
				current = ev
			}
			continue
		}
		if ev.StartAt > now {
			if next == nil || ev.StartAt < next.StartAt {
				next = ev
			}
		}
	}
	if current != nil {
		return current.ID
	}
	if next != nil {
		return next.ID
	}
	return 0
}

func defaultDeckConfig12() map[string]interface{} {
	return map[string]interface{}{
		"disable":      false,
		"level_max":    true,
		"episode_read": true,
		"master_max":   true,
		"skill_max":    true,
		"canvas":       false,
	}
}

func defaultDeckConfig34bd() map[string]interface{} {
	return map[string]interface{}{
		"disable":      false,
		"level_max":    true,
		"episode_read": false,
		"master_max":   true,
		"skill_max":    true,
		"canvas":       false,
	}
}

func noChangeDeckConfig() map[string]interface{} {
	return map[string]interface{}{
		"disable":      false,
		"level_max":    false,
		"episode_read": false,
		"master_max":   false,
		"skill_max":    false,
		"canvas":       false,
	}
}

func pickBonusTargets(list []int, args string) []int {
	if len(list) > 0 {
		return list
	}
	parts := strings.Fields(strings.TrimSpace(args))
	var values []int
	for _, p := range parts {
		v, err := strconv.Atoi(strings.TrimSpace(p))
		if err != nil || v <= 0 {
			continue
		}
		values = append(values, v)
	}
	if len(values) == 0 {
		values = []int{120}
	}
	return values
}

func intPtr(v int) *int {
	return &v
}
