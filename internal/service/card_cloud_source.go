package service

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"Haruki-Service-API/pkg/masterdata"

	"entgo.io/ent/dialect/sql"

	sekai "haruki-cloud/database/sekai"
	"haruki-cloud/database/sekai/card"
	"haruki-cloud/database/sekai/cardcostume3d"
	"haruki-cloud/database/sekai/cardsupplie"
	"haruki-cloud/database/sekai/costume3d"
	"haruki-cloud/database/sekai/gamecharacter"
	"haruki-cloud/database/sekai/gacha"
	"haruki-cloud/database/sekai/skill"
)

var skillPlaceholder = regexp.MustCompile(`\{\{(.*?)\}\}`)

// CloudCardSource 实现 CardDataSource，直接查询 Haruki-Cloud 数据库。
type CloudCardSource struct {
	client      *sekai.Client
	region      string
	queryRegion string

	cardMu    sync.RWMutex
	cardCache map[int]*masterdata.Card

	charMu       sync.RWMutex
	charCache    map[int]*masterdata.Character
	supplyMu     sync.RWMutex
	supplyByID   map[int]string
	skillMu      sync.RWMutex
	skillCache   map[int]*masterdata.Skill
	gachaMu      sync.RWMutex
	gachaByCard  map[int]*masterdata.Gacha
	gachaCache   map[int]*masterdata.Gacha
	costumeMu     sync.RWMutex
	costumeByCard map[int][]*masterdata.Costume3d
}

// NewCloudCardSource 构造 Cloud 数据源；若 client 为空则返回 nil。
func NewCloudCardSource(client *sekai.Client, defaultRegion string) *CloudCardSource {
	if client == nil {
		return nil
	}
	region := strings.TrimSpace(defaultRegion)
	if region == "" {
		region = "JP"
	}
	return &CloudCardSource{
		client:       client,
		region:       region,
		queryRegion:  strings.ToLower(region),
		cardCache:    make(map[int]*masterdata.Card),
		charCache:    make(map[int]*masterdata.Character),
		supplyByID:   make(map[int]string),
		skillCache:   make(map[int]*masterdata.Skill),
		gachaByCard:  make(map[int]*masterdata.Gacha),
		gachaCache:   make(map[int]*masterdata.Gacha),
		costumeByCard: make(map[int][]*masterdata.Costume3d),
	}
}

func (c *CloudCardSource) DefaultRegion() string {
	return c.region
}

func (c *CloudCardSource) context() context.Context {
	return context.Background()
}

func (c *CloudCardSource) GetCardByID(id int) (*masterdata.Card, error) {
	if id == 0 {
		return nil, fmt.Errorf("invalid card id")
	}
	c.cardMu.RLock()
	if cached, ok := c.cardCache[id]; ok {
		c.cardMu.RUnlock()
		return cloneCard(cached), nil
	}
	c.cardMu.RUnlock()

	ctx := c.context()
	entity, err := c.client.Card.
		Query().
		Where(
			card.ServerRegionEQ(c.queryRegion),
			card.GameIDEQ(int64(id)),
		).
		Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("query card %d failed: %w", id, err)
	}
	model, err := convertCardEntity(entity)
	if err != nil {
		return nil, err
	}

	c.cardMu.Lock()
	c.cardCache[id] = model
	c.cardMu.Unlock()

	return cloneCard(model), nil
}

func (c *CloudCardSource) GetCardByCharacterAndSeq(charID, seq int) (*masterdata.Card, error) {
	if charID == 0 {
		return nil, fmt.Errorf("character id is required")
	}
	ctx := c.context()
	items, err := c.client.Card.
		Query().
		Where(
			card.ServerRegionEQ(c.queryRegion),
			card.CharacterIDEQ(int64(charID)),
		).
		Order(card.ByReleaseAt()).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("query cards by character failed: %w", err)
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("no cards found for character %d", charID)
	}

	var target *sekai.Card
	if seq < 0 {
		index := len(items) + seq
		if index < 0 || index >= len(items) {
			return nil, fmt.Errorf("card sequence out of range: %d (total: %d)", seq, len(items))
		}
		target = items[index]
	} else {
		if seq < 1 || seq > len(items) {
			return nil, fmt.Errorf("card sequence out of range: %d (total: %d)", seq, len(items))
		}
		target = items[seq-1]
	}
	model, err := convertCardEntity(target)
	if err != nil {
		return nil, err
	}
	c.cardMu.Lock()
	c.cardCache[model.ID] = model
	c.cardMu.Unlock()
	return cloneCard(model), nil
}

func (c *CloudCardSource) FilterCards(info *CardQueryInfo) ([]*masterdata.Card, error) {
	if info == nil {
		return nil, fmt.Errorf("query info is required")
	}
	ctx := c.context()
	query := c.client.Card.Query().Where(card.ServerRegionEQ(c.queryRegion))
	if info.CharacterID != 0 {
		query = query.Where(card.CharacterIDEQ(int64(info.CharacterID)))
	}
	if info.Rarity != "" {
		query = query.Where(card.CardRarityTypeEQ(info.Rarity))
	}
	if info.Attr != "" {
		query = query.Where(card.AttrEQ(info.Attr))
	}
	if info.Year != 0 {
		start := time.Date(info.Year, time.January, 1, 0, 0, 0, 0, time.UTC).UnixMilli()
		end := time.Date(info.Year+1, time.January, 1, 0, 0, 0, 0, time.UTC).UnixMilli()
		query = query.Where(card.ReleaseAtGTE(start), card.ReleaseAtLT(end))
	}
	entities, err := query.Order(card.ByReleaseAt()).All(ctx)
	if err != nil {
		return nil, err
	}
	var results []*masterdata.Card
	for _, entity := range entities {
		model, err := convertCardEntity(entity)
		if err != nil {
			return nil, err
		}
		if info.SkillType != "" {
			skill, err := c.GetSkillByID(model.SkillID)
			if err != nil || skill == nil || skill.DescriptionSpriteName != info.SkillType {
				continue
			}
		}
		if info.SupplyType != "" && !matchesSupplyFilter(info.SupplyType, c.GetCardSupplyType(model)) {
			continue
		}
		results = append(results, cloneCard(model))
	}
	return results, nil
}

func (c *CloudCardSource) GetCharacterByID(id int) (*masterdata.Character, error) {
	if id == 0 {
		return nil, fmt.Errorf("character id is required")
	}
	c.charMu.RLock()
	if cached, ok := c.charCache[id]; ok {
		c.charMu.RUnlock()
		result := *cached
		return &result, nil
	}
	c.charMu.RUnlock()

	ctx := c.context()
	entity, err := c.client.Gamecharacter.
		Query().
		Where(
			gamecharacter.ServerRegionEQ(c.queryRegion),
			gamecharacter.GameIDEQ(int64(id)),
		).
		Only(ctx)
	if err != nil {
		return nil, err
	}
	model := &masterdata.Character{
		ID:        id,
		FirstName: entity.FirstName,
		GivenName: entity.GivenName,
		Unit:      entity.Unit,
	}
	c.charMu.Lock()
	c.charCache[id] = model
	c.charMu.Unlock()
	result := *model
	return &result, nil
}

func (c *CloudCardSource) GetUnitByCardID(cardID int) (string, error) {
	card, err := c.GetCardByID(cardID)
	if err != nil {
		return "", err
	}
	char, err := c.GetCharacterByID(card.CharacterID)
	if err != nil || char == nil {
		return "", fmt.Errorf("character not found for card %d", cardID)
	}
	if char.Unit != "" && char.Unit != "piapro" {
		return char.Unit, nil
	}
	if card.SupportUnit != "" && card.SupportUnit != "none" {
		return card.SupportUnit, nil
	}
	return "piapro", nil
}

func (c *CloudCardSource) GetCardSupplyType(cardModel *masterdata.Card) string {
	if cardModel == nil {
		return formatSupplyType("")
	}
	if cardModel.CardRarityType == "rarity_birthday" {
		return formatSupplyType("birthday")
	}
	if cardModel.CardSupplyID == 0 {
		return formatSupplyType("")
	}
	c.supplyMu.RLock()
	if val, ok := c.supplyByID[cardModel.CardSupplyID]; ok {
		c.supplyMu.RUnlock()
		return val
	}
	c.supplyMu.RUnlock()

	ctx := c.context()
	entity, err := c.client.Cardsupplie.
		Query().
		Where(
			cardsupplie.ServerRegionEQ(c.queryRegion),
			cardsupplie.IDEQ(cardModel.CardSupplyID),
		).
		Only(ctx)
	if err != nil {
		return formatSupplyType("")
	}
	translated := formatSupplyType(entity.CardSupplyType)
	c.supplyMu.Lock()
	c.supplyByID[cardModel.CardSupplyID] = translated
	c.supplyMu.Unlock()
	return translated
}

func (c *CloudCardSource) GetSkillByID(id int) (*masterdata.Skill, error) {
	if id == 0 {
		return nil, nil
	}
	c.skillMu.RLock()
	if cached, ok := c.skillCache[id]; ok {
		c.skillMu.RUnlock()
		return cloneSkill(cached), nil
	}
	c.skillMu.RUnlock()

	ctx := c.context()
	entity, err := c.client.Skill.
		Query().
		Where(
			skill.ServerRegionEQ(c.queryRegion),
			skill.GameIDEQ(int64(id)),
		).
		Only(ctx)
	if err != nil {
		return nil, err
	}
	model, err := convertSkillEntity(entity)
	if err != nil {
		return nil, err
	}
	c.skillMu.Lock()
	c.skillCache[id] = model
	c.skillMu.Unlock()
	return cloneSkill(model), nil
}

func (c *CloudCardSource) FormatSkillDescription(skill *masterdata.Skill, cardCharID int) string {
	if skill == nil {
		return ""
	}
	return skillPlaceholder.ReplaceAllStringFunc(skill.Description, func(match string) string {
		content := match[2 : len(match)-2]
		parts := strings.Split(content, ";")
		if len(parts) != 2 {
			return match
		}
		idStr := parts[0]
		op := parts[1]

		var ids []int
		for _, sub := range strings.Split(idStr, ",") {
			if v, err := strconv.Atoi(strings.TrimSpace(sub)); err == nil {
				ids = append(ids, v)
			}
		}
		if len(ids) == 0 {
			return match
		}

		getValues := func(eff *masterdata.SkillEffect) []int {
			if eff == nil || len(eff.SkillEffectDetails) == 0 {
				return []int{0}
			}
			values := make([]int, 0, len(eff.SkillEffectDetails))
			for _, detail := range eff.SkillEffectDetails {
				values = append(values, detail.ActivateEffectValue)
			}
			return values
		}
		formatValues := func(vals []int) string {
			if len(vals) == 0 {
				return ""
			}
			allSame := true
			first := vals[0]
			for _, v := range vals {
				if v != first {
					allSame = false
					break
				}
			}
			if allSame {
				return fmt.Sprintf("%d", first)
			}
			unique := make([]string, 0, len(vals))
			seen := make(map[int]struct{})
			for _, v := range vals {
				if _, ok := seen[v]; ok {
					continue
				}
				seen[v] = struct{}{}
				unique = append(unique, fmt.Sprintf("%d", v))
			}
			return strings.Join(unique, "/")
		}

		if op == "c" {
			if name := c.lookupCharacterName(cardCharID); name != "" {
				return name
			}
			return "???"
		}

		var effects []*masterdata.SkillEffect
		for _, effID := range ids {
			for i := range skill.SkillEffects {
				if skill.SkillEffects[i].ID == effID {
					effects = append(effects, &skill.SkillEffects[i])
					break
				}
			}
		}
		if len(effects) != len(ids) {
			return "?"
		}

		if len(ids) == 1 {
			eff := effects[0]
			switch op {
			case "d":
				if len(eff.SkillEffectDetails) > 0 {
					return fmt.Sprintf("%.1f", eff.SkillEffectDetails[0].ActivateEffectDuration)
				}
				return "0.0"
			case "v":
				return formatValues(getValues(eff))
			case "e":
				return fmt.Sprintf("%d", eff.SkillEnhance.ActivateEffectValue)
			case "m":
				vals := getValues(eff)
				for i := range vals {
					vals[i] += eff.SkillEnhance.ActivateEffectValue * 5
				}
				return formatValues(vals)
			case "c":
				if name := c.lookupCharacterName(cardCharID); name != "" {
					return name
				}
				return "???"
			}
		}

		if len(ids) == 2 {
			e1, e2 := effects[0], effects[1]
			vals1 := getValues(e1)
			vals2 := getValues(e2)
			switch op {
			case "v":
				var sums []int
				for i := 0; i < len(vals1) && i < len(vals2); i++ {
					sums = append(sums, vals1[i]+vals2[i])
				}
				return formatValues(sums)
			case "u", "o":
				getEnhanced := func(e *masterdata.SkillEffect, base []int) []int {
					if e == nil {
						return nil
					}
					var res []int
					for i, detail := range e.SkillEffectDetails {
						if detail.ActivateEffectValue2 != nil {
							res = append(res, *detail.ActivateEffectValue2)
						} else if i < len(base) {
							res = append(res, base[i])
						}
					}
					return res
				}
				values1 := getEnhanced(e1, vals1)
				values2 := getEnhanced(e2, vals2)
				var sums []int
				for i := 0; i < len(values1) && i < len(values2); i++ {
					sums = append(sums, values1[i]+values2[i])
				}
				return formatValues(sums)
			case "r", "s":
				return "..."
			}
		}

		return match
	})
}

func (c *CloudCardSource) GetGachaByCardID(cardID int) (*masterdata.Gacha, error) {
	if cardID == 0 {
		return nil, fmt.Errorf("invalid card id")
	}
	c.gachaMu.RLock()
	if cached, ok := c.gachaByCard[cardID]; ok {
		c.gachaMu.RUnlock()
		copy := *cached
		return &copy, nil
	}
	c.gachaMu.RUnlock()

	cardModel, err := c.GetCardByID(cardID)
	if err != nil {
		return nil, err
	}
	ctx := c.context()
	candidates, err := c.client.Gacha.
		Query().
		Where(
			gacha.ServerRegionEQ(c.queryRegion),
			gacha.StartAtLTE(cardModel.ReleaseAt),
			gacha.EndAtGTE(cardModel.ReleaseAt),
		).
		Order(gacha.ByStartAt()).
		All(ctx)
	if err != nil {
		return nil, err
	}
	if len(candidates) == 0 {
		candidates, err = c.client.Gacha.
			Query().
			Where(gacha.ServerRegionEQ(c.queryRegion)).
			Order(gacha.ByStartAt(sql.OrderDesc())).
			Limit(30).
			All(ctx)
		if err != nil {
			return nil, err
		}
	}

	for _, entity := range candidates {
		model, err := convertGachaEntity(entity)
		if err != nil {
			continue
		}
		if containsPickup(model, cardID) {
			c.gachaMu.Lock()
			c.gachaByCard[cardID] = model
			c.gachaCache[model.ID] = model
			c.gachaMu.Unlock()
			copy := *model
			return &copy, nil
		}
	}
	return nil, fmt.Errorf("gacha not found for card: %d", cardID)
}

func (c *CloudCardSource) GetCostume3dsByCardID(cardID int) ([]*masterdata.Costume3d, error) {
	if cardID == 0 {
		return nil, nil
	}
	c.costumeMu.RLock()
	if cached, ok := c.costumeByCard[cardID]; ok {
		c.costumeMu.RUnlock()
		return cloneCostumes(cached), nil
	}
	c.costumeMu.RUnlock()

	ctx := c.context()
	links, err := c.client.Cardcostume3D.
		Query().
		Where(
			cardcostume3d.ServerRegionEQ(c.queryRegion),
			cardcostume3d.CardIDEQ(int64(cardID)),
		).
		Order(cardcostume3d.ByCostume3DID()).
		All(ctx)
	if err != nil {
		return nil, err
	}
	if len(links) == 0 {
		return nil, nil
	}

	var costumes []*masterdata.Costume3d
	for _, link := range links {
		costume, err := c.client.Costume3D.
			Query().
			Where(
				costume3d.ServerRegionEQ(c.queryRegion),
				costume3d.GameIDEQ(link.Costume3DID),
			).
			Only(ctx)
		if err != nil {
			continue
		}
		costumes = append(costumes, convertCostumeEntity(costume))
	}
	if len(costumes) == 0 {
		return nil, nil
	}

	c.costumeMu.Lock()
	c.costumeByCard[cardID] = costumes
	c.costumeMu.Unlock()
	return cloneCostumes(costumes), nil
}

// Helpers

func convertCardEntity(entity *sekai.Card) (*masterdata.Card, error) {
	if entity == nil {
		return nil, fmt.Errorf("card entity is nil")
	}
	model := &masterdata.Card{
		ID:                              int(entity.GameID),
		CharacterID:                     int(entity.CharacterID),
		CardRarityType:                  entity.CardRarityType,
		Attr:                            entity.Attr,
		Prefix:                          entity.Prefix,
		AssetBundleName:                 entity.AssetbundleName,
		ReleaseAt:                       entity.ReleaseAt,
		SkillID:                         int(entity.SkillID),
		CardSkillName:                   entity.CardSkillName,
		SupportUnit:                     entity.SupportUnit,
		SpecialTrainingPower1BonusFixed: int(entity.SpecialTrainingPower1BonusFixed),
		SpecialTrainingPower2BonusFixed: int(entity.SpecialTrainingPower2BonusFixed),
		SpecialTrainingPower3BonusFixed: int(entity.SpecialTrainingPower3BonusFixed),
		SpecialTrainingSkillId:          int(entity.SpecialTrainingSkillID),
		SpecialTrainingSkillName:        entity.SpecialTrainingSkillName,
		CardSupplyID:                    int(entity.CardSupplyID),
	}
	if strings.TrimSpace(entity.CardParameters) != "" {
		var params []masterdata.CardParameter
		if err := json.Unmarshal([]byte(entity.CardParameters), &params); err == nil {
			model.CardParameters = params
		}
	}
	return model, nil
}

func convertSkillEntity(entity *sekai.Skill) (*masterdata.Skill, error) {
	if entity == nil {
		return nil, fmt.Errorf("skill entity is nil")
	}
	model := &masterdata.Skill{
		ID:                    int(entity.GameID),
		ShortDescription:      entity.ShortDescription,
		Description:           entity.Description,
		DescriptionSpriteName: entity.DescriptionSpriteName,
	}
	if len(entity.SkillEffects) > 0 {
		raw, err := json.Marshal(entity.SkillEffects)
		if err != nil {
			return nil, fmt.Errorf("marshal skill effects failed: %w", err)
		}
		if err := json.Unmarshal(raw, &model.SkillEffects); err != nil {
			return nil, fmt.Errorf("unmarshal skill effects failed: %w", err)
		}
	}
	return model, nil
}

func convertGachaEntity(entity *sekai.Gacha) (*masterdata.Gacha, error) {
	if entity == nil {
		return nil, fmt.Errorf("gacha entity is nil")
	}
	model := &masterdata.Gacha{
		ID:              int(entity.GameID),
		GachaType:       entity.GachaType,
		Name:            entity.Name,
		Seq:             int(entity.Seq),
		AssetBundleName: entity.AssetbundleName,
		StartAt:         entity.StartAt,
		EndAt:           entity.EndAt,
		IsShowPeriod:    entity.IsShowPeriod,
	}
	parseJSON := func(src interface{}, target interface{}) {
		if src == nil {
			return
		}
		raw, err := json.Marshal(src)
		if err != nil {
			return
		}
		_ = json.Unmarshal(raw, target)
	}
	var pickups []masterdata.GachaPickup
	parseJSON(entity.GachaPickups, &pickups)
	model.GachaPickups = pickups
	var details []masterdata.GachaDetail
	parseJSON(entity.GachaDetails, &details)
	model.GachaDetails = details
	var rates []masterdata.GachaCardRarityRate
	parseJSON(entity.GachaCardRarityRates, &rates)
	model.GachaCardRarityRates = rates
	var behaviors []masterdata.GachaBehavior
	parseJSON(entity.GachaBehaviors, &behaviors)
	model.GachaBehaviors = behaviors
	if len(entity.GachaInformation) > 0 {
		raw, err := json.Marshal(entity.GachaInformation)
		if err == nil {
			var info masterdata.GachaInformation
			if err := json.Unmarshal(raw, &info); err == nil {
				model.GachaInformation = info
			}
		}
	}
	return model, nil
}

func convertCostumeEntity(entity *sekai.Costume3D) *masterdata.Costume3d {
	return &masterdata.Costume3d{
		ID:              int(entity.GameID),
		CharacterID:     int(entity.CharacterID),
		AssetBundleName: entity.AssetbundleName,
		Description:     entity.Name,
	}
}

func (c *CloudCardSource) lookupCharacterName(id int) string {
	char, err := c.GetCharacterByID(id)
	if err != nil || char == nil {
		return ""
	}
	return char.FirstName + char.GivenName
}

func containsPickup(g *masterdata.Gacha, cardID int) bool {
	for _, pickup := range g.GachaPickups {
		if pickup.CardID == cardID {
			return true
		}
	}
	return false
}

func cloneCard(src *masterdata.Card) *masterdata.Card {
	if src == nil {
		return nil
	}
	dup := *src
	if len(src.CardParameters) > 0 {
		dup.CardParameters = make([]masterdata.CardParameter, len(src.CardParameters))
		copy(dup.CardParameters, src.CardParameters)
	}
	return &dup
}

func cloneSkill(src *masterdata.Skill) *masterdata.Skill {
	if src == nil {
		return nil
	}
	dup := *src
	if len(src.SkillEffects) > 0 {
		dup.SkillEffects = make([]masterdata.SkillEffect, len(src.SkillEffects))
		for i := range src.SkillEffects {
			dup.SkillEffects[i] = src.SkillEffects[i]
			if len(src.SkillEffects[i].SkillEffectDetails) > 0 {
				dup.SkillEffects[i].SkillEffectDetails = make([]masterdata.SkillEffectDetail, len(src.SkillEffects[i].SkillEffectDetails))
				copy(dup.SkillEffects[i].SkillEffectDetails, src.SkillEffects[i].SkillEffectDetails)
			}
		}
	}
	return &dup
}

func cloneCostumes(items []*masterdata.Costume3d) []*masterdata.Costume3d {
	if len(items) == 0 {
		return nil
	}
	result := make([]*masterdata.Costume3d, len(items))
	for i, item := range items {
		if item == nil {
			continue
		}
		dup := *item
		result[i] = &dup
	}
	return result
}
