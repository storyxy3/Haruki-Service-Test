package service

import "Haruki-Service-API/pkg/masterdata"

// ProfileDataSource abstracts profile module masterdata access.
// It embeds HonorDataSource because profile rendering composes honor cards.
type ProfileDataSource interface {
	HonorDataSource
	GetPlayerFrameByID(id int) (*masterdata.PlayerFrame, error)
	GetPlayerFrameGroupByID(id int) (*masterdata.PlayerFrameGroup, error)
	GetCardByID(id int) (*masterdata.Card, error)
	GetEventIDByHonorID(honorID int) int
}

// MasterDataProfileSource adapts MasterDataService to ProfileDataSource.
type MasterDataProfileSource struct {
	svc *MasterDataService
}

func NewMasterDataProfileSource(svc *MasterDataService) *MasterDataProfileSource {
	return &MasterDataProfileSource{svc: svc}
}

func (m *MasterDataProfileSource) DefaultRegion() string {
	return m.svc.GetRegion()
}

func (m *MasterDataProfileSource) GetPlayerFrameByID(id int) (*masterdata.PlayerFrame, error) {
	return m.svc.GetPlayerFrameByID(id)
}

func (m *MasterDataProfileSource) GetPlayerFrameGroupByID(id int) (*masterdata.PlayerFrameGroup, error) {
	return m.svc.GetPlayerFrameGroupByID(id)
}

func (m *MasterDataProfileSource) GetCardByID(id int) (*masterdata.Card, error) {
	return m.svc.GetCardByID(id)
}

func (m *MasterDataProfileSource) GetEventIDByHonorID(honorID int) int {
	return m.svc.GetEventIDByHonorID(honorID)
}

func (m *MasterDataProfileSource) GetHonorByID(id int) (*masterdata.Honor, error) {
	return m.svc.GetHonorByID(id)
}

func (m *MasterDataProfileSource) GetHonorGroupByID(id int) (*masterdata.HonorGroup, error) {
	return m.svc.GetHonorGroupByID(id)
}

func (m *MasterDataProfileSource) GetBondsHonorByID(id int) (*masterdata.BondsHonor, error) {
	return m.svc.GetBondsHonorByID(id)
}

func (m *MasterDataProfileSource) GetGameCharacterUnitByID(id int) (*masterdata.GameCharacterUnit, bool) {
	return m.svc.GetGameCharacterUnitByID(id)
}
