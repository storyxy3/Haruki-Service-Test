package service

import "Haruki-Service-API/pkg/masterdata"

// EventDataSource provides event-centric data used by builders/controllers.
type EventDataSource interface {
	DefaultRegion() string
	GetEventByID(id int) (*masterdata.Event, error)
	GetEventByCardID(cardID int) (*masterdata.Event, error)
	GetEvents() []*masterdata.Event
	GetEventCards(eventID int) ([]*masterdata.Card, error)
	GetEventBannerCharacterID(eventID int) (int, error)
	GetEventDeckBonuses(eventID int) ([]*masterdata.EventDeckBonus, error)
	GetGameCharacterUnit(id int) (*masterdata.GameCharacterUnit, error)
	GetBanEvents(charID int) []*masterdata.Event
	GetWorldBloomChapters(eventID int) []*masterdata.WorldBloom
	GetCharacterByID(id int) (*masterdata.Character, error)
	FilterEvents(filter EventFilter) []*masterdata.Event
}

// MasterDataEventSource adapts MasterDataService to the EventDataSource interface.
type MasterDataEventSource struct {
	svc *MasterDataService
}

// NewMasterDataEventSource creates an EventDataSource backed by MasterDataService.
func NewMasterDataEventSource(svc *MasterDataService) *MasterDataEventSource {
	return &MasterDataEventSource{svc: svc}
}

func (m *MasterDataEventSource) DefaultRegion() string {
	return m.svc.GetRegion()
}

func (m *MasterDataEventSource) GetEventByID(id int) (*masterdata.Event, error) {
	return m.svc.GetEventByID(id)
}

func (m *MasterDataEventSource) GetEventByCardID(cardID int) (*masterdata.Event, error) {
	return m.svc.GetEventByCardID(cardID)
}

func (m *MasterDataEventSource) GetEvents() []*masterdata.Event {
	return m.svc.GetEvents()
}

func (m *MasterDataEventSource) GetEventCards(eventID int) ([]*masterdata.Card, error) {
	return m.svc.GetEventCards(eventID)
}

func (m *MasterDataEventSource) GetEventBannerCharacterID(eventID int) (int, error) {
	return m.svc.GetEventBannerCharacterID(eventID)
}

func (m *MasterDataEventSource) GetEventDeckBonuses(eventID int) ([]*masterdata.EventDeckBonus, error) {
	return m.svc.GetEventDeckBonuses(eventID)
}

func (m *MasterDataEventSource) GetGameCharacterUnit(id int) (*masterdata.GameCharacterUnit, error) {
	return m.svc.GetGameCharacterUnit(id)
}

func (m *MasterDataEventSource) GetBanEvents(charID int) []*masterdata.Event {
	return m.svc.GetBanEvents(charID)
}

func (m *MasterDataEventSource) GetWorldBloomChapters(eventID int) []*masterdata.WorldBloom {
	return m.svc.GetWorldBloomChapters(eventID)
}

func (m *MasterDataEventSource) GetCharacterByID(id int) (*masterdata.Character, error) {
	return m.svc.GetCharacterByID(id)
}

func (m *MasterDataEventSource) FilterEvents(filter EventFilter) []*masterdata.Event {
	return m.svc.FilterEvents(filter)
}
