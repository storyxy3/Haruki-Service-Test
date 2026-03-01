package service

import "Haruki-Service-API/pkg/masterdata"

// GachaDataSource abstracts gacha-related data access so callers can switch
// between local masterdata and cloud-backed providers.
type GachaDataSource interface {
	DefaultRegion() string
	GetGachaByID(id int) (*masterdata.Gacha, error)
	GetGachas() []*masterdata.Gacha
	GetCardByID(id int) (*masterdata.Card, error)
}

// MasterDataGachaSource adapts MasterDataService to GachaDataSource.
type MasterDataGachaSource struct {
	svc *MasterDataService
}

func NewMasterDataGachaSource(svc *MasterDataService) *MasterDataGachaSource {
	return &MasterDataGachaSource{svc: svc}
}

func (m *MasterDataGachaSource) DefaultRegion() string {
	return m.svc.GetRegion()
}

func (m *MasterDataGachaSource) GetGachaByID(id int) (*masterdata.Gacha, error) {
	return m.svc.GetGachaByID(id)
}

func (m *MasterDataGachaSource) GetGachas() []*masterdata.Gacha {
	return m.svc.GetGachas()
}

func (m *MasterDataGachaSource) GetCardByID(id int) (*masterdata.Card, error) {
	return m.svc.GetCardByID(id)
}
