package controller

import (
	"fmt"
	"testing"

	"Haruki-Service-API/internal/model"
	"Haruki-Service-API/internal/service"
	"Haruki-Service-API/pkg/masterdata"
)

type testControllerEventSource struct {
	region string
	events []*masterdata.Event
}

func (s *testControllerEventSource) DefaultRegion() string { return s.region }

func (s *testControllerEventSource) GetEventByID(id int) (*masterdata.Event, error) {
	for _, ev := range s.events {
		if ev.ID == id {
			copy := *ev
			return &copy, nil
		}
	}
	return nil, fmt.Errorf("event not found: %d", id)
}

func (s *testControllerEventSource) GetEventByCardID(cardID int) (*masterdata.Event, error) {
	return nil, fmt.Errorf("not implemented")
}

func (s *testControllerEventSource) GetEvents() []*masterdata.Event {
	out := make([]*masterdata.Event, 0, len(s.events))
	for _, ev := range s.events {
		copy := *ev
		out = append(out, &copy)
	}
	return out
}

func (s *testControllerEventSource) GetEventCards(eventID int) ([]*masterdata.Card, error) {
	return nil, nil
}

func (s *testControllerEventSource) GetEventBannerCharacterID(eventID int) (int, error) {
	return 0, fmt.Errorf("banner not found")
}

func (s *testControllerEventSource) GetEventDeckBonuses(eventID int) ([]*masterdata.EventDeckBonus, error) {
	return nil, nil
}

func (s *testControllerEventSource) GetGameCharacterUnit(id int) (*masterdata.GameCharacterUnit, error) {
	return nil, fmt.Errorf("gcu not found")
}

func (s *testControllerEventSource) GetBanEvents(charID int) []*masterdata.Event { return nil }

func (s *testControllerEventSource) GetWorldBloomChapters(eventID int) []*masterdata.WorldBloom {
	return nil
}

func (s *testControllerEventSource) GetCharacterByID(id int) (*masterdata.Character, error) {
	return nil, fmt.Errorf("character not found")
}

func (s *testControllerEventSource) FilterEvents(filter service.EventFilter) []*masterdata.Event {
	return nil
}

func TestEventController_BuildEventListRequest_UsesRequestedRegionSource(t *testing.T) {
	cn := &testControllerEventSource{
		region: "cn",
		events: []*masterdata.Event{
			{ID: 1, EventType: "world_bloom", Name: "CN_NAME", AssetBundleName: "cn_ev", StartAt: 100, AggregateAt: 200},
		},
	}
	jp := &testControllerEventSource{
		region: "jp",
		events: []*masterdata.Event{
			{ID: 1, EventType: "world_bloom", Name: "JP_NAME", AssetBundleName: "jp_ev", StartAt: 100, AggregateAt: 200},
		},
	}

	ctrl := NewEventController(cn, nil, "", nil, nil)
	ctrl.RegisterSource(jp)

	req, err := ctrl.BuildEventListRequest(model.EventListQuery{
		Region:        "jp",
		EventType:     "world_bloom",
		IncludePast:   true,
		IncludeFuture: true,
	})
	if err != nil {
		t.Fatalf("BuildEventListRequest failed: %v", err)
	}
	if req.Region != "jp" {
		t.Fatalf("unexpected region: %s", req.Region)
	}
	if len(req.EventInfo) != 1 {
		t.Fatalf("expected 1 event, got %d", len(req.EventInfo))
	}
	if req.EventInfo[0].EventName != "JP_NAME" {
		t.Fatalf("expected JP source event name, got %q", req.EventInfo[0].EventName)
	}
}
