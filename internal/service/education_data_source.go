package service

import "Haruki-Service-API/pkg/masterdata"

// EducationDataSource abstracts masterdata access used by education module.
type EducationDataSource interface {
	DefaultRegion() string
	GetChallengeRewardsByCharacter(charID int) []*masterdata.ChallengeLiveHighScoreReward
	GetResourceBoxByPurpose(purpose string, id int) *masterdata.ResourceBox
}

// MasterDataEducationSource adapts MasterDataService to EducationDataSource.
type MasterDataEducationSource struct {
	svc *MasterDataService
}

func NewMasterDataEducationSource(svc *MasterDataService) *MasterDataEducationSource {
	return &MasterDataEducationSource{svc: svc}
}

func (m *MasterDataEducationSource) DefaultRegion() string {
	return m.svc.GetRegion()
}

func (m *MasterDataEducationSource) GetChallengeRewardsByCharacter(charID int) []*masterdata.ChallengeLiveHighScoreReward {
	return m.svc.GetChallengeRewardsByCharacter(charID)
}

func (m *MasterDataEducationSource) GetResourceBoxByPurpose(purpose string, id int) *masterdata.ResourceBox {
	return m.svc.GetResourceBoxByPurpose(purpose, id)
}
