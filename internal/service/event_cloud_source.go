package service

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"Haruki-Service-API/pkg/masterdata"

	sekai "haruki-cloud/database/sekai"
	"haruki-cloud/database/sekai/card"
	"haruki-cloud/database/sekai/cardsupplie"
	"haruki-cloud/database/sekai/event"
	"haruki-cloud/database/sekai/eventcard"
	"haruki-cloud/database/sekai/eventdeckbonuse"
	"haruki-cloud/database/sekai/gamecharacter"
	"haruki-cloud/database/sekai/gamecharacterunit"
	"haruki-cloud/database/sekai/worldbloom"
)

// CloudEventSource implements EventDataSource backed by Haruki-Cloud database.
type CloudEventSource struct {
	client      *sekai.Client
	region      string
	queryRegion string

	cardMu       sync.RWMutex
	cardCache    map[int]*masterdata.Card
	supplyMu     sync.RWMutex
	supplyCache  map[int]string
	gcuMu        sync.RWMutex
	gcuCache     map[int]*masterdata.GameCharacterUnit
	characterMu  sync.RWMutex
	characterMap map[int]*masterdata.Character
	eventMu      sync.RWMutex
	eventCache   map[int]*masterdata.Event
}

// NewCloudEventSource creates a Cloud-backed data source.
func NewCloudEventSource(client *sekai.Client, defaultRegion string) *CloudEventSource {
	region := strings.TrimSpace(defaultRegion)
	if region == "" {
		region = "JP"
	}
	queryRegion := strings.ToLower(region)
	return &CloudEventSource{
		client:       client,
		region:       region,
		queryRegion:  queryRegion,
		cardCache:    make(map[int]*masterdata.Card),
		supplyCache:  make(map[int]string),
		gcuCache:     make(map[int]*masterdata.GameCharacterUnit),
		characterMap: make(map[int]*masterdata.Character),
		eventCache:   make(map[int]*masterdata.Event),
	}
}

func (c *CloudEventSource) DefaultRegion() string {
	return c.region
}

func (c *CloudEventSource) context() context.Context {
	return context.Background()
}

func (c *CloudEventSource) GetEventByID(id int) (*masterdata.Event, error) {
	c.eventMu.RLock()
	if evt, ok := c.eventCache[id]; ok {
		c.eventMu.RUnlock()
		return cloneEvent(evt), nil
	}
	c.eventMu.RUnlock()

	ctx := c.context()
	evt, err := c.client.Event.Query().
		Where(
			event.ServerRegionEQ(c.queryRegion),
			event.GameIDEQ(int64(id)),
		).
		Only(ctx)
	if err != nil {
		return nil, err
	}
	model := convertEventEntity(evt)

	c.eventMu.Lock()
	c.eventCache[id] = model
	c.eventMu.Unlock()

	return cloneEvent(model), nil
}

func (c *CloudEventSource) GetEventByCardID(cardID int) (*masterdata.Event, error) {
	ctx := c.context()
	link, err := c.client.Eventcard.Query().
		Where(
			eventcard.ServerRegionEQ(c.queryRegion),
			eventcard.CardIDEQ(int64(cardID)),
		).
		Order(eventcard.ByEventID()).
		First(ctx)
	if err != nil {
		return nil, err
	}
	return c.GetEventByID(int(link.EventID))
}

func (c *CloudEventSource) GetEvents() []*masterdata.Event {
	ctx := c.context()
	events, err := c.client.Event.Query().
		Where(event.ServerRegionEQ(c.queryRegion)).
		Order(event.ByStartAt()).
		All(ctx)
	if err != nil {
		return nil
	}
	result := make([]*masterdata.Event, 0, len(events))
	for _, evt := range events {
		model := convertEventEntity(evt)
		c.eventMu.Lock()
		c.eventCache[model.ID] = model
		c.eventMu.Unlock()
		result = append(result, cloneEvent(model))
	}
	return result
}

func (c *CloudEventSource) GetEventCards(eventID int) ([]*masterdata.Card, error) {
	ctx := c.context()
	links, err := c.client.Eventcard.Query().
		Where(
			eventcard.ServerRegionEQ(c.queryRegion),
			eventcard.EventIDEQ(int64(eventID)),
		).
		All(ctx)
	if err != nil {
		return nil, err
	}
	if len(links) == 0 {
		return nil, fmt.Errorf("no cards found for event %d", eventID)
	}
	cardIDs := make([]int, 0, len(links))
	for _, link := range links {
		cardIDs = append(cardIDs, int(link.CardID))
	}

	cards, err := c.getCardsByIDs(cardIDs)
	if err != nil {
		return nil, err
	}
	return cards, nil
}

func (c *CloudEventSource) GetEventBannerCharacterID(eventID int) (int, error) {
	cards, err := c.GetEventCards(eventID)
	if err != nil {
		return 0, err
	}
	minCardID := -1
	var selected *masterdata.Card
	for _, card := range cards {
		supplyType := c.getCardSupplyType(card.CardSupplyID)
		isFestival := supplyType == "colorful_festival_limited" || supplyType == "bloom_festival_limited"
		if isFestival {
			continue
		}
		if minCardID == -1 || card.ID < minCardID {
			minCardID = card.ID
			selected = card
		}
	}
	if selected == nil {
		return 0, fmt.Errorf("no valid banner card found for event %d", eventID)
	}
	return selected.CharacterID, nil
}

func (c *CloudEventSource) GetEventDeckBonuses(eventID int) ([]*masterdata.EventDeckBonus, error) {
	ctx := c.context()
	items, err := c.client.Eventdeckbonuse.Query().
		Where(
			eventdeckbonuse.ServerRegionEQ(c.queryRegion),
			eventdeckbonuse.EventIDEQ(int64(eventID)),
		).
		All(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*masterdata.EventDeckBonus, 0, len(items))
	for _, item := range items {
		result = append(result, &masterdata.EventDeckBonus{
			ID:                  item.ID,
			EventID:             int(item.EventID),
			GameCharacterUnitID: int(item.GameCharacterUnitID),
			CardAttr:            item.CardAttr,
			BonusRate:           item.BonusRate,
		})
	}
	return result, nil
}

func (c *CloudEventSource) GetGameCharacterUnit(id int) (*masterdata.GameCharacterUnit, error) {
	c.gcuMu.RLock()
	if val, ok := c.gcuCache[id]; ok {
		c.gcuMu.RUnlock()
		result := *val
		return &result, nil
	}
	c.gcuMu.RUnlock()

	ctx := c.context()
	entity, err := c.client.Gamecharacterunit.Query().
		Where(
			gamecharacterunit.ServerRegionEQ(c.queryRegion),
			gamecharacterunit.IDEQ(id),
		).
		Only(ctx)
	if err != nil {
		return nil, err
	}
	model := &masterdata.GameCharacterUnit{
		ID:              entity.ID,
		GameCharacterID: int(entity.GameCharacterID),
		Unit:            entity.Unit,
		ColorCode:       entity.ColorCode,
	}
	c.gcuMu.Lock()
	c.gcuCache[id] = model
	c.gcuMu.Unlock()
	result := *model
	return &result, nil
}

func (c *CloudEventSource) GetBanEvents(charID int) []*masterdata.Event {
	ctx := c.context()
	events, err := c.client.Event.Query().
		Where(
			event.ServerRegionEQ(c.queryRegion),
			event.EventTypeIn("marathon", "cheerful_carnival"),
		).
		Order(event.ByStartAt()).
		All(ctx)
	if err != nil {
		return nil
	}
	var result []*masterdata.Event
	for _, evt := range events {
		cards, err := c.GetEventCards(evt.ID)
		if err != nil || len(cards) == 0 {
			continue
		}
		var banner *masterdata.Card
		for _, card := range cards {
			if c.isFestivalCard(card.CardSupplyID) {
				continue
			}
			if banner == nil || card.ID < banner.ID {
				banner = card
			}
		}
		if banner != nil && banner.CharacterID == charID {
			result = append(result, convertEventEntity(evt))
		}
	}
	return result
}

func (c *CloudEventSource) GetWorldBloomChapters(eventID int) []*masterdata.WorldBloom {
	ctx := c.context()
	items, err := c.client.Worldbloom.Query().
		Where(
			worldbloom.ServerRegionEQ(c.queryRegion),
			worldbloom.EventIDEQ(int64(eventID)),
		).
		Order(worldbloom.ByChapterNo()).
		All(ctx)
	if err != nil || len(items) == 0 {
		return nil
	}
	result := make([]*masterdata.WorldBloom, 0, len(items))
	for _, item := range items {
		var charID *int
		if item.GameCharacterID != 0 {
			id := int(item.GameCharacterID)
			charID = &id
		}
		result = append(result, &masterdata.WorldBloom{
			ID:              item.ID,
			EventID:         int(item.EventID),
			GameCharacterID: charID,
			ChapterNo:       int(item.ChapterNo),
			ChapterStartAt:  item.ChapterStartAt,
			AggregateAt:     item.AggregateAt,
			ChapterEndAt:    item.ChapterEndAt,
			IsSupplemental:  item.IsSupplemental,
		})
	}
	return result
}

func (c *CloudEventSource) GetCharacterByID(id int) (*masterdata.Character, error) {
	c.characterMu.RLock()
	if val, ok := c.characterMap[id]; ok {
		c.characterMu.RUnlock()
		result := *val
		return &result, nil
	}
	c.characterMu.RUnlock()

	ctx := c.context()
	entity, err := c.client.Gamecharacter.Query().
		Where(
			gamecharacter.ServerRegionEQ(c.queryRegion),
			gamecharacter.IDEQ(id),
		).
		Only(ctx)
	if err != nil {
		return nil, err
	}
	model := &masterdata.Character{
		ID:        entity.ID,
		FirstName: entity.FirstName,
		GivenName: entity.GivenName,
		Unit:      entity.Unit,
	}
	c.characterMu.Lock()
	c.characterMap[id] = model
	c.characterMu.Unlock()
	result := *model
	return &result, nil
}

func (c *CloudEventSource) FilterEvents(filter EventFilter) []*masterdata.Event {
	events := c.GetEvents()
	if len(events) == 0 {
		return nil
	}
	var result []*masterdata.Event
	for _, event := range events {
		if filter.EventType != "" && event.EventType != filter.EventType {
			continue
		}
		if filter.Year != 0 {
			t := time.Unix(event.StartAt/1000, 0)
			if t.Year() != filter.Year {
				continue
			}
		}
		if filter.Unit != "" || filter.Attr != "" {
			bonuses, err := c.GetEventDeckBonuses(event.ID)
			if err != nil {
				continue
			}
			matched := false
			for _, bonus := range bonuses {
				matchUnit := true
				matchAttr := true
				if filter.Attr != "" && bonus.CardAttr != filter.Attr {
					matchAttr = false
				}
				if filter.Unit != "" {
					unit, err := c.GetGameCharacterUnit(bonus.GameCharacterUnitID)
					if err != nil || unit.Unit != filter.Unit {
						matchUnit = false
					}
				}
				if matchUnit && matchAttr {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}
		if filter.CharacterID != 0 {
			cards, err := c.GetEventCards(event.ID)
			if err != nil {
				continue
			}
			found := false
			for _, card := range cards {
				if card.CharacterID == filter.CharacterID {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		result = append(result, event)
	}
	return result
}

// Helpers

func (c *CloudEventSource) getCardsByIDs(ids []int) ([]*masterdata.Card, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	result := make([]*masterdata.Card, len(ids))
	var missing []int64
	var missingIndices []int

	c.cardMu.RLock()
	for i, id := range ids {
		if card, ok := c.cardCache[id]; ok {
			copy := *card
			result[i] = &copy
		} else {
			missing = append(missing, int64(id))
			missingIndices = append(missingIndices, i)
		}
	}
	c.cardMu.RUnlock()

	if len(missing) == 0 {
		return result, nil
	}

	ctx := c.context()
	entities, err := c.client.Card.Query().
		Where(
			card.ServerRegionEQ(c.queryRegion),
			card.GameIDIn(missing...),
		).
		All(ctx)
	if err != nil {
		return nil, err
	}

	entityMap := make(map[int64]*sekai.Card)
	for _, e := range entities {
		entityMap[e.GameID] = e
	}

	c.cardMu.Lock()
	defer c.cardMu.Unlock()

	for k, idx := range missingIndices {
		id64 := missing[k]
		e, ok := entityMap[id64]
		if !ok {
			return nil, fmt.Errorf("card %d not found in DB", id64)
		}
		model, err := convertCardEntity(e)
		if err != nil {
			return nil, err
		}
		c.cardCache[int(id64)] = model
		copy := *model
		result[idx] = &copy
	}

	return result, nil
}

func (c *CloudEventSource) getCardByID(id int) (*masterdata.Card, error) {
	c.cardMu.RLock()
	if card, ok := c.cardCache[id]; ok {
		c.cardMu.RUnlock()
		copy := *card
		return &copy, nil
	}
	c.cardMu.RUnlock()

	ctx := c.context()
	entity, err := c.client.Card.Query().
		Where(
			card.ServerRegionEQ(c.queryRegion),
			card.GameIDEQ(int64(id)),
		).
		Only(ctx)
	if err != nil {
		return nil, err
	}
	model, err := convertCardEntity(entity)
	if err != nil {
		return nil, err
	}
	c.cardMu.Lock()
	c.cardCache[id] = model
	c.cardMu.Unlock()
	copy := *model
	return &copy, nil
}

func (c *CloudEventSource) isFestivalCard(supplyID int) bool {
	if supplyID == 0 {
		return false
	}
	typ := c.getCardSupplyType(supplyID)
	return typ == "colorful_festival_limited" || typ == "bloom_festival_limited"
}

func (c *CloudEventSource) getCardSupplyType(id int) string {
	if id == 0 {
		return ""
	}
	c.supplyMu.RLock()
	if val, ok := c.supplyCache[id]; ok {
		c.supplyMu.RUnlock()
		return val
	}
	c.supplyMu.RUnlock()

	ctx := c.context()
	supply, err := c.client.Cardsupplie.Query().
		Where(cardsupplie.ServerRegionEQ(c.queryRegion), cardsupplie.IDEQ(id)).
		Only(ctx)
	if err != nil {
		return ""
	}
	c.supplyMu.Lock()
	c.supplyCache[id] = supply.CardSupplyType
	c.supplyMu.Unlock()
	return supply.CardSupplyType
}

func convertEventEntity(evt *sekai.Event) *masterdata.Event {
	return &masterdata.Event{
		ID:              int(evt.GameID),
		EventType:       evt.EventType,
		Name:            evt.Name,
		AssetBundleName: evt.AssetbundleName,
		StartAt:         evt.StartAt,
		AggregateAt:     evt.AggregateAt,
		ClosedAt:        evt.ClosedAt,
	}
}

func cloneEvent(evt *masterdata.Event) *masterdata.Event {
	if evt == nil {
		return nil
	}
	copy := *evt
	return &copy
}
