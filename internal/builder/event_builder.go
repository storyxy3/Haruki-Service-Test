package builder

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"Haruki-Service-API/internal/model"
	"Haruki-Service-API/internal/service"
	"Haruki-Service-API/pkg/asset"
	"Haruki-Service-API/pkg/masterdata"
)

// EventBuilder 构建活动相关的数据请求
type EventBuilder struct {
	masterdata *service.MasterDataService
	assets     *asset.AssetHelper
	assetDir   string
}

// NewEventBuilder 创建
func NewEventBuilder(masterdata *service.MasterDataService, assets *asset.AssetHelper, assetDir string) *EventBuilder {
	return &EventBuilder{
		masterdata: masterdata,
		assets:     assets,
		assetDir:   assetDir,
	}
}

func (b *EventBuilder) displayEventType(code string) string {
	eventTypeDisplay := map[string]string{
		"marathon":          "马拉松",
		"cheerful_carnival": "5v5",
		"world_bloom":       "世界连结",
	}
	if label, ok := eventTypeDisplay[strings.ToLower(code)]; ok {
		return label
	}
	return code
}

// BuildEventDetailRequest 生成请求体
func (b *EventBuilder) BuildEventDetailRequest(query model.EventDetailQuery) (*model.EventDetailRequest, error) {
	if query.EventID == 0 {
		return nil, fmt.Errorf("event id is required")
	}
	event, err := b.masterdata.GetEventByID(query.EventID)
	if err != nil {
		return nil, err
	}
	region := query.Region
	if region == "" {
		region = b.masterdata.GetRegion()
	}
	cards, err := b.masterdata.GetEventCards(event.ID)
	if err != nil {
		return nil, err
	}
	cardThumbs := make([]model.CardFullThumbnailRequest, 0, len(cards))
	for _, card := range cards {
		cardThumbs = append(cardThumbs, BuildCardThumbnail(b.assets, b.assetDir, card, ThumbnailOptions{AfterTraining: false}))
	}
	info, err := b.buildEventInfo(event)
	if err != nil {
		return nil, err
	}
	assets := b.buildEventAssets(event, info)

	return &model.EventDetailRequest{
		Region:      region,
		EventInfo:   info,
		EventAssets: assets,
		EventCards:  cardThumbs,
	}, nil
}

// BuildEventListRequest 构建活动列表请求体
func (b *EventBuilder) BuildEventListRequest(query model.EventListQuery) (*model.EventListRequest, error) {
	region := query.Region
	if region == "" {
		region = b.masterdata.GetRegion()
	}
	events := b.filterEvents(query)
	if len(events) == 0 {
		return nil, fmt.Errorf("no events matched filters")
	}
	briefs := make([]model.EventBrief, 0, len(events))
	for _, ev := range events {
		brief, err := b.buildEventBrief(ev)
		if err != nil {
			return nil, err
		}
		briefs = append(briefs, brief)
	}
	return &model.EventListRequest{
		Region:    region,
		EventInfo: briefs,
	}, nil
}

func (b *EventBuilder) buildEventInfo(event *masterdata.Event) (model.EventInfo, error) {
	info := model.EventInfo{
		ID:        event.ID,
		EventType: b.displayEventType(event.EventType),
		StartAt:   event.StartAt,
		EndAt:     event.AggregateAt + 1000,
		IsWLEvent: strings.EqualFold(event.EventType, "world_bloom"),
	}
	if bannerCID, err := b.masterdata.GetEventBannerCharacterID(event.ID); err == nil && bannerCID != 0 {
		info.BannerCID = &bannerCID
		if idx := b.getBannerIndex(bannerCID, event.ID); idx != nil {
			info.BannerIndex = idx
		}
	}
	if attr, chars := b.extractEventBonuses(event.ID); attr != "" {
		info.BonusAttr = attr
		info.BonusCharaIDs = chars
	}
	if wlTimeline := b.buildWorldBloomTimeline(event.ID); len(wlTimeline) > 0 {
		info.WLTimeList = wlTimeline
	}
	return info, nil
}

func (b *EventBuilder) buildEventAssets(event *masterdata.Event, info model.EventInfo) model.EventAssets {
	assetName := event.AssetBundleName
	assets := model.EventAssets{
		EventBgPath: asset.ResolveAssetPath(b.assets, b.assetDir,
			filepath.Join("event", assetName, "screen", "bg.png"),
			filepath.Join("event", assetName+"_rip", "screen", "bg.png"),
			filepath.Join("event", assetName, "bg.png"),
		),
		EventLogoPath: asset.ResolveAssetPath(b.assets, b.assetDir,
			filepath.Join("event", assetName, "logo", "logo.png"),
			filepath.Join("event", assetName+"_rip", "logo", "logo.png"),
			filepath.Join("event", assetName, "logo.png"),
		),
	}
	if !strings.EqualFold(event.EventType, "world_bloom") {
		assets.EventStoryBgPath = asset.ResolveAssetPath(b.assets, b.assetDir,
			filepath.Join("event_story", assetName, "screen_image", "story_bg.png"),
			filepath.Join("event_story", assetName+"_rip", "screen_image", "story_bg.png"),
		)
		assets.EventBanCharaImg = asset.ResolveAssetPath(b.assets, b.assetDir,
			filepath.Join("event", assetName, "screen", "character.png"),
			filepath.Join("event", assetName+"_rip", "screen", "character.png"),
		)
	}
	if info.BonusAttr != "" {
		assets.EventAttrImagePath = asset.ResolveAssetPath(b.assets, b.assetDir, filepath.Join("card", fmt.Sprintf("attr_icon_%s.png", strings.ToLower(info.BonusAttr))))
	}
	if info.BannerCID != nil {
		path := b.characterIconPath(*info.BannerCID)
		assets.BanCharaIconPath = path
	}
	for _, cid := range info.BonusCharaIDs {
		assets.BonusCharaPath = append(assets.BonusCharaPath, b.characterIconPath(cid))
	}
	if assets.BanCharaIconPath == "" && len(info.BonusCharaIDs) > 0 {
		assets.BanCharaIconPath = b.characterIconPath(info.BonusCharaIDs[0])
	}
	return assets
}

func (b *EventBuilder) buildEventBrief(event *masterdata.Event) (model.EventBrief, error) {
	brief := model.EventBrief{
		ID:        event.ID,
		EventName: event.Name,
		EventType: b.displayEventType(event.EventType),
		StartAt:   event.StartAt,
		EndAt:     event.AggregateAt + 1000,
		EventBannerPath: asset.ResolveAssetPath(b.assets, b.assetDir,
			filepath.Join("home", "banner", fmt.Sprintf("%s_rip", event.AssetBundleName), event.AssetBundleName+".png"),
			filepath.Join("home", "banner", event.AssetBundleName, event.AssetBundleName+".png"),
			filepath.Join("event", event.AssetBundleName, "banner.png"),
		),
	}
	cards, err := b.masterdata.GetEventCards(event.ID)
	if err == nil && len(cards) > 0 {
		maxCards := len(cards)
		if maxCards > 6 {
			maxCards = 6
		}
		for i := 0; i < maxCards; i++ {
			brief.EventCards = append(brief.EventCards, BuildCardThumbnail(b.assets, b.assetDir, cards[i], ThumbnailOptions{}))
		}
	}
	if attr, chars := b.extractEventBonuses(event.ID); attr != "" {
		path := filepath.ToSlash(filepath.Join(b.assetDir, "card", fmt.Sprintf("attr_%s.png", strings.ToLower(attr))))
		brief.EventAttrPath = &path
		if len(chars) > 0 {
			chID := chars[0]
			charaPath := b.characterIconPath(chID)
			brief.EventCharaPath = &charaPath
			if unitPath := b.unitIconPathByCharacter(chID); unitPath != "" {
				brief.EventUnitPath = &unitPath
			}
		}
	}
	if bannerCID, err := b.masterdata.GetEventBannerCharacterID(event.ID); err == nil && bannerCID != 0 {
		if brief.EventCharaPath == nil {
			path := b.characterIconPath(bannerCID)
			brief.EventCharaPath = &path
		}
		if brief.EventUnitPath == nil {
			if unit := b.unitIconPathByCharacter(bannerCID); unit != "" {
				brief.EventUnitPath = &unit
			}
		}
	}
	return brief, nil
}

func (b *EventBuilder) filterEvents(query model.EventListQuery) []*masterdata.Event {
	events := b.masterdata.GetEvents()
	now := time.Now()
	result := make([]*masterdata.Event, 0, len(events))
	includePast := query.IncludePast
	includeFuture := query.IncludeFuture
	if query.OnlyFuture {
		includeFuture = true
		includePast = false
	}
	for _, ev := range events {
		start := time.UnixMilli(ev.StartAt)
		end := time.UnixMilli(ev.AggregateAt + 1000)
		if !includePast && end.Before(now) {
			continue
		}
		if !includeFuture && start.After(now) {
			continue
		}
		if query.EventType != "" && !strings.EqualFold(ev.EventType, query.EventType) {
			continue
		}
		if query.Year != 0 && start.Year() != query.Year {
			continue
		}
		if query.Unit != "" || query.Attr != "" || query.CharacterID != 0 {
			if !b.matchEventBonus(ev.ID, query.Unit, query.Attr, query.CharacterID) {
				continue
			}
		}
		if query.BannerCharID != nil {
			bid, err := b.masterdata.GetEventBannerCharacterID(ev.ID)
			if err != nil || bid != *query.BannerCharID {
				continue
			}
		}
		result = append(result, ev)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].StartAt > result[j].StartAt
	})
	if query.Limit > 0 && len(result) > query.Limit {
		result = result[:query.Limit]
	}
	return result
}

func (b *EventBuilder) extractEventBonuses(eventID int) (string, []int) {
	bonuses, err := b.masterdata.GetEventDeckBonuses(eventID)
	if err != nil {
		return "", nil
	}
	attr := ""
	charSet := map[int]struct{}{}
	for _, bb := range bonuses {
		if attr == "" && bb.CardAttr != "" {
			attr = strings.ToLower(bb.CardAttr)
		}
		if bb.GameCharacterUnitID != 0 {
			if unit, err := b.masterdata.GetGameCharacterUnit(bb.GameCharacterUnitID); err == nil {
				charSet[unit.GameCharacterID] = struct{}{}
			}
		}
	}
	var chars []int
	for cid := range charSet {
		chars = append(chars, cid)
	}
	sort.Ints(chars)
	return attr, chars
}

func (b *EventBuilder) matchEventBonus(eventID int, unit string, attr string, charID int) bool {
	if unit == "" && attr == "" && charID == 0 {
		return true
	}
	bonuses, err := b.masterdata.GetEventDeckBonuses(eventID)
	if err != nil {
		return false
	}
	for _, bonus := range bonuses {
		matchAttr := attr == "" || strings.EqualFold(bonus.CardAttr, attr)
		matchUnit := true
		if unit != "" {
			gcu, err := b.masterdata.GetGameCharacterUnit(bonus.GameCharacterUnitID)
			if err != nil || !strings.EqualFold(gcu.Unit, unit) {
				matchUnit = false
			}
		}
		matchChar := true
		if charID != 0 {
			gcu, err := b.masterdata.GetGameCharacterUnit(bonus.GameCharacterUnitID)
			if err != nil || gcu.GameCharacterID != charID {
				matchChar = false
			}
		}
		if matchAttr && matchUnit && matchChar {
			return true
		}
	}
	return attr == "" && unit == "" && charID == 0
}

func (b *EventBuilder) getBannerIndex(charID, eventID int) *int {
	events := b.masterdata.GetBanEvents(charID)
	sort.Slice(events, func(i, j int) bool {
		return events[i].StartAt < events[j].StartAt
	})
	for idx, ev := range events {
		if ev.ID == eventID {
			i := idx + 1
			return &i
		}
	}
	return nil
}

func (b *EventBuilder) buildWorldBloomTimeline(eventID int) []model.EventWlTiming {
	chapters := b.masterdata.GetWorldBloomChapters(eventID)
	if len(chapters) == 0 {
		return nil
	}
	timeline := make([]model.EventWlTiming, 0, len(chapters))
	for _, chapter := range chapters {
		item := model.EventWlTiming{
			StartAt:     chapter.ChapterStartAt,
			AggregateAt: chapter.AggregateAt,
		}
		if chapter.ChapterNo != 0 {
			no := chapter.ChapterNo
			item.ChapterID = &no
		}
		if chapter.GameCharacterID != nil && *chapter.GameCharacterID != 0 {
			cid := *chapter.GameCharacterID
			item.GameCharacter = &cid
		}
		timeline = append(timeline, item)
	}
	sort.Slice(timeline, func(i, j int) bool {
		return timeline[i].StartAt < timeline[j].StartAt
	})
	return timeline
}

func (b *EventBuilder) characterIconPath(charID int) string {
	if nickname, ok := asset.CharacterIDToNickname[charID]; ok {
		return asset.ResolveAssetPath(b.assets, b.assetDir, filepath.Join("chara_icon", nickname+".png"))
	}
	return asset.ResolveAssetPath(b.assets, b.assetDir, filepath.Join("chara_icon", fmt.Sprintf("chr_icon_%d.png", charID)))
}

func (b *EventBuilder) unitIconPathByCharacter(charID int) string {
	char, err := b.masterdata.GetCharacterByID(charID)
	if err != nil || char == nil {
		return ""
	}
	unitIcon := asset.UnitIconFilename(char.Unit)
	if unitIcon == "" {
		return ""
	}
	return asset.ResolveAssetPath(b.assets, b.assetDir, filepath.Join("unit", unitIcon+".png"))
}
