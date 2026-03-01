package service

import "Haruki-Service-API/pkg/masterdata"

// CardDataSource 抽象卡牌数据访问能力，便于在 MasterData 与 Cloud 之间切换。
type CardDataSource interface {
	DefaultRegion() string
	GetCardByID(id int) (*masterdata.Card, error)
	GetCardByCharacterAndSeq(charID, seq int) (*masterdata.Card, error)
	FilterCards(info *CardQueryInfo) ([]*masterdata.Card, error)
	GetCharacterByID(id int) (*masterdata.Character, error)
	GetUnitByCardID(cardID int) (string, error)
	GetCardSupplyType(card *masterdata.Card) string
	GetSkillByID(id int) (*masterdata.Skill, error)
	FormatSkillDescription(skill *masterdata.Skill, cardCharID int) string
	GetGachaByCardID(cardID int) (*masterdata.Gacha, error)
	GetCostume3dsByCardID(cardID int) ([]*masterdata.Costume3d, error)
}

// MasterDataCardSource 适配 MasterDataService 以满足 CardDataSource。
type MasterDataCardSource struct {
	svc *MasterDataService
}

// NewMasterDataCardSource 创建一个基于本地 MasterData 的卡牌数据源。
func NewMasterDataCardSource(svc *MasterDataService) *MasterDataCardSource {
	return &MasterDataCardSource{svc: svc}
}

func (m *MasterDataCardSource) DefaultRegion() string {
	return m.svc.GetRegion()
}

func (m *MasterDataCardSource) GetCardByID(id int) (*masterdata.Card, error) {
	return m.svc.GetCardByID(id)
}

func (m *MasterDataCardSource) GetCardByCharacterAndSeq(charID, seq int) (*masterdata.Card, error) {
	return m.svc.GetCardByCharacterAndSeq(charID, seq)
}

func (m *MasterDataCardSource) FilterCards(info *CardQueryInfo) ([]*masterdata.Card, error) {
	return m.svc.FilterCards(info), nil
}

func (m *MasterDataCardSource) GetCharacterByID(id int) (*masterdata.Character, error) {
	return m.svc.GetCharacterByID(id)
}

func (m *MasterDataCardSource) GetUnitByCardID(cardID int) (string, error) {
	return m.svc.GetUnitByCardID(cardID)
}

func (m *MasterDataCardSource) GetCardSupplyType(card *masterdata.Card) string {
	return m.svc.GetCardSupplyType(card)
}

func (m *MasterDataCardSource) GetSkillByID(id int) (*masterdata.Skill, error) {
	return m.svc.GetSkillByID(id)
}

func (m *MasterDataCardSource) FormatSkillDescription(skill *masterdata.Skill, cardCharID int) string {
	return m.svc.FormatSkillDescription(skill, cardCharID)
}

func (m *MasterDataCardSource) GetGachaByCardID(cardID int) (*masterdata.Gacha, error) {
	return m.svc.GetGachaByCardID(cardID)
}

func (m *MasterDataCardSource) GetCostume3dsByCardID(cardID int) ([]*masterdata.Costume3d, error) {
	return m.svc.GetCostume3dsByCardID(cardID)
}
