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
	source   service.EventDataSource
	assets   *asset.AssetHelper
	assetDir string
}

// NewEventBuilder 创建
func NewEventBuilder(source service.EventDataSource, assets *asset.AssetHelper, assetDir string) *EventBuilder {
	return &EventBuilder{
		source:   source,
		assets:   assets,
		assetDir: assetDir,
	}
}

func (b *EventBuilder) displayEventType(code string) string {
	eventTypeDisplay := map[string]string{
		"marathon":          "马拉松",
		"cheerful_carnival": "5v5",
		"world_bloom":       "WorldLink",
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
	event, err := b.source.GetEventByID(query.EventID)
	if err != nil {
		return nil, err
	}
	region := query.Region
	if region == "" {
		region = b.source.DefaultRegion()
	}
	cards, err := b.source.GetEventCards(event.ID)
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
		region = b.source.DefaultRegion()
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
	isWLEvent := strings.EqualFold(event.EventType, "world_bloom")
	info := model.EventInfo{
		ID:            event.ID,
		EventType:     b.displayEventType(event.EventType),
		StartAt:       event.StartAt,
		EndAt:         event.AggregateAt + 1000,
		IsWLEvent:     isWLEvent,
		BonusCharaIDs: []int{},
	}
	// Lunabot behavior: WL does not expose banner character/avatar in detail.
	if !isWLEvent {
		if bannerCID, err := b.source.GetEventBannerCharacterID(event.ID); err == nil && bannerCID != 0 {
			info.BannerCID = &bannerCID
			if idx := b.getBannerIndex(bannerCID, event.ID); idx != nil {
				info.BannerIndex = *idx
			} else {
				info.BannerIndex = 1 // default fallback
			}
		}
	}
	if attr, chars := b.extractEventBonuses(event.ID); attr != "" {
		info.BonusAttr = attr
		if chars != nil {
			info.BonusCharaIDs = chars
		}
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
			filepath.Join("event", assetName, "bg.png"),
		),
		EventLogoPath: asset.ResolveAssetPath(b.assets, b.assetDir,
			filepath.Join("event", assetName, "logo", "logo.png"),
			filepath.Join("event", assetName, "logo.png"),
		),
		BonusCharaPath: []string{},
	}
	if !strings.EqualFold(event.EventType, "world_bloom") {
		assets.EventStoryBgPath = asset.ResolveAssetPath(b.assets, b.assetDir,
			filepath.Join("event_story", assetName, "screen_image", "story_bg.png"),
		)
		assets.EventBanCharaImg = asset.ResolveAssetPath(b.assets, b.assetDir,
			filepath.Join("event", assetName, "screen", "character.png"),
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
			filepath.Join("home", "banner", event.AssetBundleName, event.AssetBundleName+".png"),
			filepath.Join("event", event.AssetBundleName, "banner.png"),
		),
	}
	cards, err := b.source.GetEventCards(event.ID)
	if err == nil && len(cards) > 0 {
		maxCards := len(cards)
		if maxCards > 6 {
			maxCards = 6
		}
		for i := 0; i < maxCards; i++ {
			brief.EventCards = append(brief.EventCards, BuildCardThumbnail(b.assets, b.assetDir, cards[i], ThumbnailOptions{}))
		}
	}
	if attr, _ := b.extractEventBonuses(event.ID); attr != "" {
		path := filepath.ToSlash(filepath.Join(b.assetDir, "card", fmt.Sprintf("attr_%s.png", strings.ToLower(attr))))
		brief.EventAttrPath = &path
	}

	isWLEvent := strings.EqualFold(event.EventType, "world_bloom")
	if !isWLEvent {
		if bannerCID, err := b.source.GetEventBannerCharacterID(event.ID); err == nil && bannerCID != 0 {
			path := b.characterIconPath(bannerCID)
			brief.EventCharaPath = &path
			if unit := b.unitIconPathByCharacter(bannerCID); unit != "" {
				brief.EventUnitPath = &unit
			}
		}
		return brief, nil
	}

	// Lunabot behavior: WL list only keeps unit icon and does not show character avatar.
	if len(cards) > 0 && len(cards) <= 6 {
		if unit := b.unitIconPathByCharacter(cards[0].CharacterID); unit != "" {
			brief.EventUnitPath = &unit
		}
	}
	return brief, nil
}

func (b *EventBuilder) filterEvents(query model.EventListQuery) []*masterdata.Event {
	events := b.source.GetEvents()
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
		if query.Unit != "" || query.Blend || query.Attr != "" || query.CharacterID != 0 || len(query.CharacterIDs) > 0 {
			if !b.matchEventBonus(ev.ID, query.Unit, query.Blend, query.Attr, query.CharacterID, query.CharacterIDs) {
				continue
			}
		}
		if query.BannerCharID != nil {
			bid, err := b.source.GetEventBannerCharacterID(ev.ID)
			if err != nil || bid != *query.BannerCharID {
				continue
			}
		}
		result = append(result, ev)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].StartAt < result[j].StartAt
	})
	if query.Limit > 0 && len(result) > query.Limit {
		result = result[:query.Limit]
	}
	return result
}

func (b *EventBuilder) extractEventBonuses(eventID int) (string, []int) {
	bonuses, err := b.source.GetEventDeckBonuses(eventID)
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
			if unit, err := b.source.GetGameCharacterUnit(bb.GameCharacterUnitID); err == nil {
				charSet[unit.GameCharacterID] = struct{}{}
			}
		} else if bb.GameCharacterID != 0 {
			// Fallback for older servers/early events without CUID directly to CharacterID
			charSet[bb.GameCharacterID] = struct{}{}
		}
	}
	var chars []int
	for cid := range charSet {
		chars = append(chars, cid)
	}
	sort.Ints(chars)
	return attr, chars
}

func (b *EventBuilder) matchEventBonus(eventID int, unit string, blend bool, attr string, charID int, charIDs []int) bool {
	if unit == "" && !blend && attr == "" && charID == 0 && len(charIDs) == 0 {
		return true
	}
	bonuses, err := b.source.GetEventDeckBonuses(eventID)
	if err != nil {
		return false
	}

	attrMatched := attr == ""
	units := make(map[string]struct{})
	charSet := make(map[int]struct{})

	for _, bonus := range bonuses {
		if !attrMatched && strings.EqualFold(bonus.CardAttr, attr) {
			attrMatched = true
		}
		if bonus.GameCharacterUnitID != 0 {
			gcu, gcuErr := b.source.GetGameCharacterUnit(bonus.GameCharacterUnitID)
			if gcuErr == nil && gcu != nil {
				units[strings.ToLower(strings.TrimSpace(gcu.Unit))] = struct{}{}
				charSet[gcu.GameCharacterID] = struct{}{}
			}
		} else if bonus.GameCharacterID != 0 {
			// Fallback for older events: we don't have unit info accurately matched, but we know the character ID
			charSet[bonus.GameCharacterID] = struct{}{}
		}
	}

	unitMatched := unit == ""
	if unit != "" {
		_, unitMatched = units[strings.ToLower(strings.TrimSpace(unit))]
	}
	if blend || strings.EqualFold(strings.TrimSpace(unit), "blend") {
		unitMatched = len(units) > 1
	}

	charMatched := true
	if charID != 0 {
		_, charMatched = charSet[charID]
	}
	for _, cid := range charIDs {
		if cid == 0 {
			continue
		}
		if _, ok := charSet[cid]; !ok {
			charMatched = false
			break
		}
	}

	return attrMatched && unitMatched && charMatched
}

func (b *EventBuilder) getBannerIndex(charID, eventID int) *int {
	events := b.source.GetBanEvents(charID)
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
	chapters := b.source.GetWorldBloomChapters(eventID)
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
	char, err := b.source.GetCharacterByID(charID)
	if err != nil || char == nil {
		return ""
	}
	unitIcon := asset.UnitIconFilename(char.Unit)
	if unitIcon == "" {
		return ""
	}
	return asset.ResolveAssetPath(b.assets, b.assetDir, filepath.Join("unit", unitIcon+".png"))
}
