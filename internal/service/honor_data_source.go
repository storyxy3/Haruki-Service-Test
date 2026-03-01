package service

import "Haruki-Service-API/pkg/masterdata"

// HonorDataSource abstracts honor module masterdata access.
type HonorDataSource interface {
	DefaultRegion() string
	GetHonorByID(id int) (*masterdata.Honor, error)
	GetHonorGroupByID(id int) (*masterdata.HonorGroup, error)
	GetBondsHonorByID(id int) (*masterdata.BondsHonor, error)
	GetGameCharacterUnitByID(id int) (*masterdata.GameCharacterUnit, bool)
}

// MasterDataHonorSource adapts MasterDataService to HonorDataSource.
type MasterDataHonorSource struct {
	svc *MasterDataService
}

func NewMasterDataHonorSource(svc *MasterDataService) *MasterDataHonorSource {
	return &MasterDataHonorSource{svc: svc}
}

func (m *MasterDataHonorSource) DefaultRegion() string {
	return m.svc.GetRegion()
}

func (m *MasterDataHonorSource) GetHonorByID(id int) (*masterdata.Honor, error) {
	return m.svc.GetHonorByID(id)
}

func (m *MasterDataHonorSource) GetHonorGroupByID(id int) (*masterdata.HonorGroup, error) {
	return m.svc.GetHonorGroupByID(id)
}

func (m *MasterDataHonorSource) GetBondsHonorByID(id int) (*masterdata.BondsHonor, error) {
	return m.svc.GetBondsHonorByID(id)
}

func (m *MasterDataHonorSource) GetGameCharacterUnitByID(id int) (*masterdata.GameCharacterUnit, bool) {
	return m.svc.GetGameCharacterUnitByID(id)
}
