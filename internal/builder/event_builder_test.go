package builder

import (
	"fmt"
	"testing"

	"Haruki-Service-API/internal/model"
	"Haruki-Service-API/internal/service"
	"Haruki-Service-API/pkg/masterdata"
)

type testEventSource struct {
	region          string
	events          []*masterdata.Event
	eventsByID      map[int]*masterdata.Event
	cardsByEvent    map[int][]*masterdata.Card
	bannerByEvent   map[int]int
	bonusesByEvent  map[int][]*masterdata.EventDeckBonus
	gcuByID         map[int]*masterdata.GameCharacterUnit
	worldByEvent    map[int][]*masterdata.WorldBloom
	characterByID   map[int]*masterdata.Character
}

func newTestEventSource(region string) *testEventSource {
	return &testEventSource{
		region:         region,
		eventsByID:     make(map[int]*masterdata.Event),
		cardsByEvent:   make(map[int][]*masterdata.Card),
		bannerByEvent:  make(map[int]int),
		bonusesByEvent: make(map[int][]*masterdata.EventDeckBonus),
		gcuByID:        make(map[int]*masterdata.GameCharacterUnit),
		worldByEvent:   make(map[int][]*masterdata.WorldBloom),
		characterByID:  make(map[int]*masterdata.Character),
	}
}

func (s *testEventSource) DefaultRegion() string { return s.region }

func (s *testEventSource) GetEventByID(id int) (*masterdata.Event, error) {
	if ev, ok := s.eventsByID[id]; ok {
		copy := *ev
		return &copy, nil
	}
	return nil, fmt.Errorf("event not found: %d", id)
}

func (s *testEventSource) GetEventByCardID(cardID int) (*masterdata.Event, error) {
	return nil, fmt.Errorf("not implemented")
}

func (s *testEventSource) GetEvents() []*masterdata.Event {
	out := make([]*masterdata.Event, 0, len(s.events))
	for _, ev := range s.events {
		copy := *ev
		out = append(out, &copy)
	}
	return out
}

func (s *testEventSource) GetEventCards(eventID int) ([]*masterdata.Card, error) {
	cards := s.cardsByEvent[eventID]
	out := make([]*masterdata.Card, 0, len(cards))
	for _, c := range cards {
		copy := *c
		out = append(out, &copy)
	}
	return out, nil
}

func (s *testEventSource) GetEventBannerCharacterID(eventID int) (int, error) {
	if v, ok := s.bannerByEvent[eventID]; ok {
		return v, nil
	}
	return 0, fmt.Errorf("banner not found: %d", eventID)
}

func (s *testEventSource) GetEventDeckBonuses(eventID int) ([]*masterdata.EventDeckBonus, error) {
	items := s.bonusesByEvent[eventID]
	out := make([]*masterdata.EventDeckBonus, 0, len(items))
	for _, item := range items {
		copy := *item
		out = append(out, &copy)
	}
	return out, nil
}

func (s *testEventSource) GetGameCharacterUnit(id int) (*masterdata.GameCharacterUnit, error) {
	if item, ok := s.gcuByID[id]; ok {
		copy := *item
		return &copy, nil
	}
	return nil, fmt.Errorf("gcu not found: %d", id)
}

func (s *testEventSource) GetBanEvents(charID int) []*masterdata.Event { return nil }

func (s *testEventSource) GetWorldBloomChapters(eventID int) []*masterdata.WorldBloom {
	items := s.worldByEvent[eventID]
	out := make([]*masterdata.WorldBloom, 0, len(items))
	for _, item := range items {
		copy := *item
		out = append(out, &copy)
	}
	return out
}

func (s *testEventSource) GetCharacterByID(id int) (*masterdata.Character, error) {
	if c, ok := s.characterByID[id]; ok {
		copy := *c
		return &copy, nil
	}
	return nil, fmt.Errorf("character not found: %d", id)
}

func (s *testEventSource) FilterEvents(filter service.EventFilter) []*masterdata.Event { return nil }

func TestBuildEventListRequest_WorldBloomNoCharacterAvatar(t *testing.T) {
	source := newTestEventSource("jp")
	ev := &masterdata.Event{ID: 101, EventType: "world_bloom", Name: "JP_WL", AssetBundleName: "wl_101", StartAt: 100, AggregateAt: 200}
	source.events = []*masterdata.Event{ev}
	source.eventsByID[ev.ID] = ev
	source.cardsByEvent[ev.ID] = []*masterdata.Card{{ID: 1001, CharacterID: 5, Attr: "cool", AssetBundleName: "card_1001"}}
	source.bannerByEvent[ev.ID] = 5 // WL should ignore this in list/detail avatar fields.
	source.bonusesByEvent[ev.ID] = []*masterdata.EventDeckBonus{{ID: 1, EventID: ev.ID, GameCharacterUnitID: 501, CardAttr: "cool"}}
	source.gcuByID[501] = &masterdata.GameCharacterUnit{ID: 501, GameCharacterID: 5, Unit: "idol"}
	source.characterByID[5] = &masterdata.Character{ID: 5, Unit: "idol"}

	b := NewEventBuilder(source, nil, "assets")
	req, err := b.BuildEventListRequest(model.EventListQuery{Region: "jp", EventType: "world_bloom", IncludePast: true, IncludeFuture: true})
	if err != nil {
		t.Fatalf("BuildEventListRequest failed: %v", err)
	}
	if len(req.EventInfo) != 1 {
		t.Fatalf("expected 1 event, got %d", len(req.EventInfo))
	}
	brief := req.EventInfo[0]
	if brief.EventType != "WorldLink" {
		t.Fatalf("expected WorldLink event type, got %q", brief.EventType)
	}
	if brief.EventCharaPath != nil {
		t.Fatalf("WL should not expose character avatar in list, got %q", *brief.EventCharaPath)
	}
	if brief.EventUnitPath == nil || *brief.EventUnitPath == "" {
		t.Fatalf("WL should keep unit icon path")
	}
}

func TestBuildEventListRequest_OrderByStartAtAscending(t *testing.T) {
	source := newTestEventSource("jp")
	ev1 := &masterdata.Event{ID: 201, EventType: "marathon", Name: "Later", AssetBundleName: "later", StartAt: 200, AggregateAt: 300}
	ev2 := &masterdata.Event{ID: 202, EventType: "marathon", Name: "Earlier", AssetBundleName: "earlier", StartAt: 100, AggregateAt: 150}
	source.events = []*masterdata.Event{ev1, ev2}
	source.eventsByID[ev1.ID] = ev1
	source.eventsByID[ev2.ID] = ev2

	b := NewEventBuilder(source, nil, "assets")
	req, err := b.BuildEventListRequest(model.EventListQuery{Region: "jp", IncludePast: true, IncludeFuture: true})
	if err != nil {
		t.Fatalf("BuildEventListRequest failed: %v", err)
	}
	if len(req.EventInfo) != 2 {
		t.Fatalf("expected 2 events, got %d", len(req.EventInfo))
	}
	if req.EventInfo[0].ID != 202 || req.EventInfo[1].ID != 201 {
		t.Fatalf("unexpected order: got [%d, %d], want [202, 201]", req.EventInfo[0].ID, req.EventInfo[1].ID)
	}
}
