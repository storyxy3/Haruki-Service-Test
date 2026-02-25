package masterdata

// Card MasterData 中的卡牌数据结构
type Card struct {
	ID                              int             `json:"id"`
	CharacterID                     int             `json:"characterId"`
	CardRarityType                  string          `json:"cardRarityType"`
	Attr                            string          `json:"attr"`
	Prefix                          string          `json:"prefix"`
	AssetBundleName                 string          `json:"assetbundleName"`
	ReleaseAt                       int64           `json:"releaseAt"`
	SkillID                         int             `json:"skillId"`
	CardSkillName                   string          `json:"cardSkillName"`
	SupportUnit                     string          `json:"supportUnit"`
	CardParameters                  []CardParameter `json:"cardParameters"`
	SpecialTrainingPower1BonusFixed int             `json:"specialTrainingPower1BonusFixed"`
	SpecialTrainingPower2BonusFixed int             `json:"specialTrainingPower2BonusFixed"`
	SpecialTrainingPower3BonusFixed int             `json:"specialTrainingPower3BonusFixed"`
	SpecialTrainingSkillId          int             `json:"specialTrainingSkillId"`
	SpecialTrainingSkillName        string          `json:"specialTrainingSkillName"`
	CardSupplyID                    int             `json:"cardSupplyId"`
}

// CardParameter 卡牌参数（用于判断限定类型等）
type CardParameter struct {
	ID                int    `json:"id"`
	CardID            int    `json:"cardId"`
	CardParameterType string `json:"cardParameterType"`
	Power             int    `json:"power"`
}

// Skill 技能数据
type Skill struct {
	ID                    int           `json:"id"`
	ShortDescription      string        `json:"shortDescription"`
	Description           string        `json:"description"`
	DescriptionSpriteName string        `json:"descriptionSpriteName"`
	SkillEffects          []SkillEffect `json:"skillEffects"`
}

// SkillEffect 技能效果
type SkillEffect struct {
	ID                        int                 `json:"id"`
	SkillEffectType           string              `json:"skillEffectType"`
	ActivateEffectDuration    int                 `json:"activateEffectDuration"`
	ActivateEffectValueType   string              `json:"activateEffectValueType"`
	ActivateEffectValue       float64             `json:"activateEffectValue"`
	SkillEffectDetails        []SkillEffectDetail `json:"skillEffectDetails"`
	SkillEnhance              SkillEnhance        `json:"skillEnhance"`
	ConditionType             string              `json:"conditionType"`
	ActivateNotesJudgmentType string              `json:"activateNotesJudgmentType"`
	ActivateUnitCount         int                 `json:"activateUnitCount"`
	ActivateCharacterRank     int                 `json:"activateCharacterRank"`
}

// SkillEffectDetail 技能效果详情
type SkillEffectDetail struct {
	ID                      int     `json:"id"`
	ActivateEffectDuration  float64 `json:"activateEffectDuration"`
	ActivateEffectValueType string  `json:"activateEffectValueType"`
	ActivateEffectValue     int     `json:"activateEffectValue"`
	ActivateEffectValue2    *int    `json:"activateEffectValue2,omitempty"` // Optional
}

// SkillEnhance 技能增强
type SkillEnhance struct {
	ActivateEffectValue int `json:"activateEffectValue"`
}

// Character 角色数据
type Character struct {
	ID        int    `json:"id"`
	FirstName string `json:"firstName"`
	GivenName string `json:"givenName"`
	Unit      string `json:"unit"`
	// 添加其他需要的字段
}

// Event 活动数据
type Event struct {
	ID                       int                       `json:"id"`
	EventType                string                    `json:"eventType"`
	Name                     string                    `json:"name"`
	AssetBundleName          string                    `json:"assetbundleName"`
	StartAt                  int64                     `json:"startAt"`
	AggregateAt              int64                     `json:"aggregateAt"`
	ClosedAt                 int64                     `json:"closedAt"`
	EventRankingRewardRanges []EventRankingRewardRange `json:"eventRankingRewardRanges"`
}

type EventRankingRewardRange struct {
	FromRank int `json:"fromRank"`
	ToRank   int `json:"toRank"`
	// 我们只需要获取 honorId
	EventRankingRewardDetails []EventRankingRewardDetail `json:"eventRankingRewardDetails"`
}

type EventRankingRewardDetail struct {
	ID                 int    `json:"id"`
	ResourceType       string `json:"resourceType"` // "honor"
	ResourceID         int    `json:"resourceId"`   // honorId
	ResourceQuantity   int    `json:"resourceQuantity"`
	ResourceLevel      int    `json:"resourceLevel"`
	EventRankingReward int    `json:"eventRankingRewardId"`
}

// Gacha 卡池数据
type Gacha struct {
	ID                     int                   `json:"id"`
	GachaType              string                `json:"gachaType"`
	Name                   string                `json:"name"`
	Seq                    int                   `json:"seq"`
	AssetBundleName        string                `json:"assetbundleName"`
	StartAt                int64                 `json:"startAt"`
	EndAt                  int64                 `json:"endAt"`
	IsShowPeriod           bool                  `json:"isShowPeriod"`
	GachaCeilItemID        *int                  `json:"gachaCeilItemId"`
	WishSelectCount        int                   `json:"wishSelectCount"`
	WishFixedSelectCount   int                   `json:"wishFixedSelectCount"`
	WishLimitedSelectCount int                   `json:"wishLimitedSelectCount"`
	GachaCardRarityRates   []GachaCardRarityRate `json:"gachaCardRarityRates"`
	GachaPickups           []GachaPickup         `json:"gachaPickups"`
	GachaDetails           []GachaDetail         `json:"gachaDetails"`
	GachaBehaviors         []GachaBehavior       `json:"gachaBehaviors"`
	GachaInformation       GachaInformation      `json:"gachaInformation"`
}

type GachaPickup struct {
	ID              int    `json:"id"`
	GachaID         int    `json:"gachaId"`
	CardID          int    `json:"cardId"`
	GachaPickupType string `json:"gachaPickupType"`
}

type GachaDetail struct {
	ID      int  `json:"id"`
	GachaID int  `json:"gachaId"`
	CardID  int  `json:"cardId"`
	Weight  int  `json:"weight"`
	IsWish  bool `json:"isWish"`
}

type GachaCardRarityRate struct {
	ID             int     `json:"id"`
	GroupID        int     `json:"groupId"`
	CardRarityType string  `json:"cardRarityType"`
	LotteryType    string  `json:"lotteryType"`
	Rate           float64 `json:"rate"`
}

type GachaBehavior struct {
	ID                   int    `json:"id"`
	GachaID              int    `json:"gachaId"`
	GachaBehaviorType    string `json:"gachaBehaviorType"`
	CostResourceType     string `json:"costResourceType"`
	CostResourceQuantity int    `json:"costResourceQuantity"`
	SpinCount            int    `json:"spinCount"`
	ExecuteLimit         *int   `json:"executeLimit"`
	GroupID              int    `json:"groupId"`
	Priority             int    `json:"priority"`
	ResourceCategory     string `json:"resourceCategory"`
	GachaSpinnableType   string `json:"gachaSpinnableType"`
}

type GachaInformation struct {
	GachaID     int    `json:"gachaId"`
	Summary     string `json:"summary"`
	Description string `json:"description"`
}

// Music 音乐数据
type Music struct {
	ID                 int      `json:"id"`
	Seq                int      `json:"seq"`
	ReleaseConditionId int      `json:"releaseConditionId"`
	Categories         []string `json:"categories"`
	Title              string   `json:"title"`
	Pronunciation      string   `json:"pronunciation"`
	Lyricist           string   `json:"lyricist"`
	Composer           string   `json:"composer"`
	Arranger           string   `json:"arranger"`
	DancerCount        int      `json:"dancerCount"`
	SelfDancerCount    int      `json:"selfDancerCount"`
	AssetBundleName    string   `json:"assetbundleName"`
	PublishedAt        int64    `json:"publishedAt"`
	DigitizedAt        int64    `json:"digitizedAt"`
}

// MusicDifficulty 音乐难度数据
type MusicDifficulty struct {
	ID              int    `json:"id"`
	MusicID         int    `json:"musicId"`
	MusicDifficulty string `json:"musicDifficulty"` // easy, normal, hard, expert, master
	PlayLevel       int    `json:"playLevel"`
	TotalNoteCount  int    `json:"totalNoteCount"`
}

// MusicVocal 音乐 Vocal 数据
type MusicVocal struct {
	ID              int                   `json:"id"`
	MusicID         int                   `json:"musicId"`
	MusicVocalType  string                `json:"musicVocalType"` // original_song, sekai
	Caption         string                `json:"caption"`
	Characters      []MusicVocalCharacter `json:"characters"`
	AssetBundleName string                `json:"assetbundleName"`
}

// MusicVocalCharacter Vocal 角色关联
type MusicVocalCharacter struct {
	ID            int    `json:"id"`
	MusicID       int    `json:"musicId"`
	MusicVocalID  int    `json:"musicVocalId"`
	CharacterType string `json:"characterType"`
	CharacterID   int    `json:"characterId"`
}

// MusicTag 音乐标签数据
type MusicTag struct {
	ID       int    `json:"id"`
	MusicID  int    `json:"musicId"`
	MusicTag string `json:"musicTag"`
}

// LimitedTimeMusic represents limited-time availability for a music
type LimitedTimeMusic struct {
	ID      int   `json:"id"`
	MusicID int   `json:"musicId"`
	StartAt int64 `json:"startAt"`
	EndAt   int64 `json:"endAt"`
}

// ChallengeLiveHighScoreReward stores reward thresholds for challenge live.
type ChallengeLiveHighScoreReward struct {
	ID            int `json:"id"`
	CharacterID   int `json:"characterId"`
	HighScore     int `json:"highScore"`
	ResourceBoxID int `json:"resourceBoxId"`
}

// ResourceBoxDetail describes a single entry within a resource box.
type ResourceBoxDetail struct {
	ResourceBoxPurpose string `json:"resourceBoxPurpose"`
	ResourceBoxID      int    `json:"resourceBoxId"`
	Seq                int    `json:"seq"`
	ResourceType       string `json:"resourceType"`
	ResourceID         int    `json:"resourceId"`
	ResourceQuantity   int    `json:"resourceQuantity"`
}

// ResourceBox describes a reward bundle.
type ResourceBox struct {
	ResourceBoxPurpose string              `json:"resourceBoxPurpose"`
	ID                 int                 `json:"id"`
	ResourceBoxType    string              `json:"resourceBoxType"`
	Description        string              `json:"description"`
	Details            []ResourceBoxDetail `json:"details"`
}

// EventCard 活动卡牌关联
type EventCard struct {
	ID      int `json:"id"`
	EventID int `json:"eventId"`
	CardID  int `json:"cardId"`
}

// EventMusic 活动-乐曲关联
type EventMusic struct {
	EventID            int `json:"eventId"`
	MusicID            int `json:"musicId"`
	Seq                int `json:"seq"`
	ReleaseConditionID int `json:"releaseConditionId"`
}

// Costume3d 3D服装数据
type Costume3d struct {
	ID              int    `json:"id"`
	CharacterID     int    `json:"characterId"`
	AssetBundleName string `json:"assetbundleName"`
	Description     string `json:"description"`
}

// CardCostume3d 卡牌与3D服装关联
type CardCostume3d struct {
	CardID      int `json:"cardId"`
	Costume3dID int `json:"costume3dId"`
}

// EventDeckBonus 活动加成数据
type EventDeckBonus struct {
	ID                  int     `json:"id"`
	EventID             int     `json:"eventId"`
	GameCharacterUnitID int     `json:"gameCharacterUnitId"`
	CardAttr            string  `json:"cardAttr"`
	BonusRate           float64 `json:"bonusRate"`
}

// GameCharacterUnit 角色与组合关联数据
type GameCharacterUnit struct {
	ID              int    `json:"id"`
	GameCharacterID int    `json:"gameCharacterId"`
	Unit            string `json:"unit"`
	ColorCode       string `json:"colorCode"`
}

// CardSupply 卡牌供给类型数据
type CardSupply struct {
	ID             int    `json:"id"`
	CardSupplyType string `json:"cardSupplyType"`
	Seq            int    `json:"seq"`
}

// WorldBloom 描述 WL 章节
type WorldBloom struct {
	ID              int    `json:"id"`
	EventID         int    `json:"eventId"`
	GameCharacterID *int   `json:"gameCharacterId,omitempty"`
	ChapterNo       int    `json:"chapterNo"`
	ChapterStartAt  int64  `json:"chapterStartAt"`
	AggregateAt     int64  `json:"aggregateAt"`
	ChapterEndAt    int64  `json:"chapterEndAt"`
	IsSupplemental  bool   `json:"isSupplemental"`
	ChapterType     string `json:"worldBloomChapterType"`
}

// Honor 称号数据
type Honor struct {
	ID              int          `json:"id"`
	GroupID         int          `json:"groupId"`
	HonorType       string       `json:"honorType"`
	HonorRarity     string       `json:"honorRarity"`
	Name            string       `json:"name"`
	Description     string       `json:"description"`
	AssetbundleName string       `json:"assetbundleName"`
	Levels          []HonorLevel `json:"levels"`
}

// HonorLevel 称号等级信息
type HonorLevel struct {
	Level           int    `json:"level"`
	HonorRarity     string `json:"honorRarity"`
	Description     string `json:"description"`
	AssetbundleName string `json:"assetbundleName"`
}

// HonorGroup 称号组
type HonorGroup struct {
	ID                        int     `json:"id"`
	HonorType                 string  `json:"honorType"`
	Name                      string  `json:"name"`
	Description               string  `json:"description"`
	BackgroundAssetbundleName *string `json:"backgroundAssetbundleName"`
	FrameName                 *string `json:"frameName"`
}

// BondsHonor 羁绊称号
type BondsHonor struct {
	ID                   int    `json:"id"`
	GameCharacterUnitId1 int    `json:"gameCharacterUnitId1"`
	GameCharacterUnitId2 int    `json:"gameCharacterUnitId2"`
	HonorRarity          string `json:"honorRarity"`
	Name                 string `json:"name"`
	Description          string `json:"description"`
	BondsGroupId         int    `json:"bondsGroupId"`
}

// PlayerFrame 玩家头像框
type PlayerFrame struct {
	ID                 int    `json:"id"`
	Seq                int    `json:"seq"`
	PlayerFrameGroupID int    `json:"playerFrameGroupId"`
	Description        string `json:"description"`
	GameCharacterID    int    `json:"gameCharacterId"`
}

// PlayerFrameGroup 玩家头像框组
type PlayerFrameGroup struct {
	ID              int    `json:"id"`
	Seq             int    `json:"seq"`
	Name            string `json:"name"`
	AssetbundleName string `json:"assetbundleName"`
}
