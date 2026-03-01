package model

// CardDetailRequest DrawingAPI 卡牌详情请求
// 这个结构体需要与 Haruki-Drawing-API 的 Pydantic 模型完全匹配
type CardDetailRequest struct {
	CardInfo            CardBasic      `json:"card_info"`
	Region              string         `json:"region"`
	EventInfo           *CardEventInfo `json:"event_info,omitempty"`
	GachaInfo           *CardGachaInfo `json:"gacha_info,omitempty"`
	CardImagesPath      []string       `json:"card_images_path"`
	CostumeImagesPath   []string       `json:"costume_images_path"`
	CharacterIconPath   string         `json:"character_icon_path"`
	UnitLogoPath        string         `json:"unit_logo_path"`
	BackgroundImagePath *string        `json:"background_image_path,omitempty"`
	EventAttrIconPath   string         `json:"event_attr_icon_path,omitempty"`
	EventUnitIconPath   string         `json:"event_unit_icon_path,omitempty"`
	EventCharaIconPath  string         `json:"event_chara_icon_path,omitempty"`
}

// CardBasic 卡牌基础信息
type CardBasic struct {
	CardID           int                        `json:"card_id"`
	CharacterID      int                        `json:"character_id"`
	CharacterName    string                     `json:"character_name,omitempty"`
	Unit             string                     `json:"unit,omitempty"`
	ReleaseAt        int64                      `json:"release_at,omitempty"`
	SupplyType       string                     `json:"supply_type,omitempty"`
	Rare             string                     `json:"rare,omitempty"`
	Attr             string                     `json:"attr,omitempty"`
	Prefix           string                     `json:"prefix,omitempty"`
	AssetBundleName  string                     `json:"asset_bundle_name,omitempty"`
	Skill            *CardSkill                 `json:"skill,omitempty"`
	SpecialSkillInfo *CardSkill                 `json:"special_skill_info,omitempty"`
	ThumbnailInfo    []CardFullThumbnailRequest `json:"thumbnail_info"`
	IsAfterTraining  bool                       `json:"is_after_training"`
	Power            *CardPower                 `json:"power,omitempty"`
}

// CardSkill 卡牌技能信息
type CardSkill struct {
	SkillID           int    `json:"skill_id"`
	SkillName         string `json:"skill_name"`
	SkillType         string `json:"skill_type"`
	SkillDetail       string `json:"skill_detail"`
	SkillTypeIconPath string `json:"skill_type_icon_path"`
	SkillDetailCN     string `json:"skill_detail_cn,omitempty"`
}

// CardPower 卡牌数值
type CardPower struct {
	PowerTotal int `json:"power_total"`
	Power1     int `json:"power1"`
	Power2     int `json:"power2"`
	Power3     int `json:"power3"`
}

// CardFullThumbnailRequest 卡牌缩略图信息
type CardFullThumbnailRequest struct {
	CardID            int         `json:"card_id"`
	CardThumbnailPath string      `json:"card_thumbnail_path"`
	Rare              string      `json:"rare"`
	FrameImgPath      string      `json:"frame_img_path"`
	AttrImgPath       string      `json:"attr_img_path"`
	RareImgPath       string      `json:"rare_img_path"`
	TrainRank         *int        `json:"train_rank,omitempty"`
	TrainRankImgPath  *string     `json:"train_rank_img_path,omitempty"`
	Level             *int        `json:"level,omitempty"`
	BirthdayIconPath  *string     `json:"birthday_icon_path,omitempty"`
	IsAfterTraining   *bool       `json:"is_after_training,omitempty"`
	CustomText        *string     `json:"custom_text,omitempty"`
	CardLevel         interface{} `json:"card_level,omitempty"` // Python端是 dict
	IsPcard           bool        `json:"is_pcard"`
}

// CardEventInfo 活动信息
type CardEventInfo struct {
	EventID         int    `json:"event_id"`
	EventName       string `json:"event_name"`
	StartAt         int64  `json:"start_at"`
	EndAt           int64  `json:"end_at"`
	EventBannerPath string `json:"event_banner_path"`
	BonusAttr       string `json:"bonus_attr,omitempty"`
	Unit            string `json:"unit,omitempty"`
	BannerCID       int    `json:"banner_cid,omitempty"`
}

// CardGachaInfo 卡池信息
type CardGachaInfo struct {
	GachaID         int    `json:"gacha_id"`
	GachaName       string `json:"gacha_name"`
	StartAt         int64  `json:"start_at"`
	EndAt           int64  `json:"end_at"`
	GachaBannerPath string `json:"gacha_banner_path"`
}

// CardListRequest DrawingAPI 卡牌列表请求
type CardListRequest struct {
	Cards               []CardBasic `json:"cards"`
	Region              string      `json:"region"`
	UserInfo            interface{} `json:"user_info,omitempty"`
	BackgroundImagePath *string     `json:"background_img_path,omitempty"` // Python端是 background_img_path
}

// CardBoxRequest DrawingAPI 卡牌一览请求
type CardBoxRequest struct {
	Cards               []UserCard        `json:"cards"`
	Region              string            `json:"region"`
	UserInfo            interface{}       `json:"user_info,omitempty"`
	ShowID              bool              `json:"show_id"`
	ShowBox             bool              `json:"show_box"`
	UseAfterTraining    bool              `json:"use_after_training"`
	BackgroundImagePath *string           `json:"background_img_path,omitempty"` // Python端是 background_img_path
	CharacterIconPaths  map[string]string `json:"character_icon_paths,omitempty"`
	TermLimitedIconPath string            `json:"term_limited_icon_path,omitempty"`
	FesLimitedIconPath  string            `json:"fes_limited_icon_path,omitempty"`
}

// UserCard 用户卡牌信息
type UserCard struct {
	Card    CardBasic `json:"card"`
	HasCard bool      `json:"has_card"`
}

// MusicDetailRequest DrawingAPI 音乐详情请求
type MusicDetailRequest struct {
	Region          string         `json:"region"`
	MusicInfo       MusicMD        `json:"music_info"`
	Difficulty      DifficultyInfo `json:"difficulty"`
	Vocal           MusicVocalInfo `json:"vocal"`
	MusicJacketPath string         `json:"music_jacket_path"`
	Alias           []string       `json:"alias"` // 新增字段
	EventID         *int           `json:"event_id,omitempty"`
	EventBannerPath string         `json:"event_banner_path,omitempty"`
	LimitedTimes    [][2]string    `json:"limited_times,omitempty"`
}

// MusicBriefListRequest 绠€鐣ユ瓕鏇插垪琛ㄥ疄渚嬭姹?
type MusicBriefListRequest struct {
	MusicList          []MusicBriefListItem `json:"music_list"`
	Region             string               `json:"region"`
	RequiredDifficulty string               `json:"required_difficulty"`
	Title              *string              `json:"title,omitempty"`
	TitleStyle         interface{}          `json:"title_style,omitempty"`
	TitleShadow        bool                 `json:"title_shadow,omitempty"`
}

// MusicMD 音乐元数据
type MusicMD struct {
	ID           int      `json:"id"`
	Title        string   `json:"title"`
	Composer     string   `json:"composer"`
	Lyricist     string   `json:"lyricist"`
	Arranger     string   `json:"arranger"`
	Categories   []string `json:"categories"`
	ReleaseAt    int64    `json:"release_at"`
	IsFullLength bool     `json:"is_full_length"`
}

// DifficultyInfo 难度信息
type DifficultyInfo struct {
	Level     []int    `json:"level"`
	NoteCount []int    `json:"note_count"`
	HasAppend bool     `json:"has_append"`
	Order     []string `json:"order,omitempty"`
}

// MusicVocalInfo Vocal信息
type MusicVocalInfo struct {
	VocalInfo   map[string]interface{} `json:"vocal_info"`
	VocalAssets map[string]string      `json:"vocal_assets"`
}

// MusicBriefListItem 绠€鐣ユ瓕鏇插垪琛ㄥ崟椤?
type MusicBriefListItem struct {
	ID              int    `json:"id"`
	Level           int    `json:"level"`
	MusicJacketPath string `json:"music_jacket_path"`
	PlayResult      string `json:"play_result,omitempty"`
}

// GachaFilter 用于列表请求的翻页参数
type GachaFilter struct {
	Page int `json:"page"`
}

// GachaBrief 缩略信息
type GachaBrief struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	GachaType string `json:"gacha_type"`
	StartAt   int64  `json:"start_at"`
	EndAt     int64  `json:"end_at"`
	AssetName string `json:"asset_name"`
}

// GachaListRequest DrawingAPI 卡池列表请求
type GachaListRequest struct {
	Gachas     []GachaBrief   `json:"gachas"`
	PageSize   int            `json:"page_size"`
	Region     string         `json:"region"`
	GachaLogos map[int]string `json:"gacha_logos"`
	Filter     GachaFilter    `json:"filter"`
}

// GachaBehavior 定义抽卡行为
type GachaBehavior struct {
	Type         string  `json:"type"`
	SpinCount    int     `json:"spin_count"`
	CostType     *string `json:"cost_type,omitempty"`
	CostIconPath *string `json:"cost_icon_path,omitempty"`
	CostQuantity *int    `json:"cost_quantity,omitempty"`
	ExecuteLimit *int    `json:"execute_limit,omitempty"`
	ColorfulPass bool    `json:"colorful_pass"`
}

// GachaInfo 详情信息
type GachaInfo struct {
	ID                int             `json:"id"`
	Name              string          `json:"name"`
	GachaType         string          `json:"gacha_type"`
	Summary           string          `json:"summary"`
	Desc              string          `json:"desc"`
	StartAt           int64           `json:"start_at"`
	EndAt             int64           `json:"end_at"`
	AssetName         string          `json:"asset_name"`
	CeilItemImgPath   *string         `json:"ceil_item_img_path,omitempty"`
	Behaviors         []GachaBehavior `json:"behaviors"`
	Rarity1Count      int             `json:"rarity_1_count"`
	Rarity2Count      int             `json:"rarity_2_count"`
	Rarity3Count      int             `json:"rarity_3_count"`
	Rarity4Count      int             `json:"rarity_4_count"`
	RarityBirthdayCnt int             `json:"rarity_birthday_count"`
	PickupCount       int             `json:"pickup_count"`
}

// GachaCardWeight 卡池卡片权重
type GachaCardWeight struct {
	ID               int                      `json:"id"`
	Rarity           string                   `json:"rarity"`
	Rate             float64                  `json:"rate"`
	IsPickup         bool                     `json:"is_pickup"`
	ThumbnailRequest CardFullThumbnailRequest `json:"thumbnail_request"`
}

// GachaWeight 概率信息
type GachaWeight struct {
	Rarity1Rate        float64            `json:"rarity_1_rate"`
	Rarity2Rate        float64            `json:"rarity_2_rate"`
	Rarity3Rate        float64            `json:"rarity_3_rate"`
	Rarity4Rate        float64            `json:"rarity_4_rate"`
	RarityBirthdayRate float64            `json:"rarity_birthday_rate"`
	GuaranteedRates    map[string]float64 `json:"guaranteed_rates"`
}

// GachaDetailRequest DrawingAPI 卡池详情请求
type GachaDetailRequest struct {
	Gacha         GachaInfo         `json:"gacha"`
	WeightInfo    GachaWeight       `json:"weight_info"`
	PickupCards   []GachaCardWeight `json:"pickup_cards"`
	LogoImgPath   *string           `json:"logo_img_path,omitempty"`
	BannerImgPath *string           `json:"banner_img_path,omitempty"`
	BgImgPath     *string           `json:"bg_img_path,omitempty"`
	Region        string            `json:"region"`
}

// EventInfo 描述活动基本信息
type EventInfo struct {
	ID            int             `json:"id"`
	EventType     string          `json:"event_type"`
	StartAt       int64           `json:"start_at"`
	EndAt         int64           `json:"end_at"`
	IsWLEvent     bool            `json:"is_wl_event"`
	BannerCID     *int            `json:"banner_cid,omitempty"`
	BannerIndex   *int            `json:"banner_index,omitempty"`
	BonusAttr     string          `json:"bonus_attr,omitempty"`
	BonusCharaIDs []int           `json:"bonus_chara_id,omitempty"`
	WLTimeList    []EventWlTiming `json:"wl_time_list,omitempty"`
}

// EventWlTiming 记录 WL 章节时间
type EventWlTiming struct {
	StartAt       int64 `json:"start_at"`
	AggregateAt   int64 `json:"aggregate_at"`
	ChapterID     *int  `json:"chapter_id,omitempty"`
	GameCharacter *int  `json:"game_character_id,omitempty"`
}

// EventAssets 描述活动素材路径
type EventAssets struct {
	EventBgPath        string   `json:"event_bg_path"`
	EventLogoPath      string   `json:"event_logo_path"`
	EventStoryBgPath   string   `json:"event_story_bg_path,omitempty"`
	EventAttrImagePath string   `json:"event_attr_image_path,omitempty"`
	EventBanCharaImg   string   `json:"event_ban_chara_img,omitempty"`
	BanCharaIconPath   string   `json:"ban_chara_icon_path,omitempty"`
	BonusCharaPath     []string `json:"bonus_chara_path,omitempty"`
}

// EventDetailRequest 描述活动详情请求
type EventDetailRequest struct {
	Region      string                     `json:"region"`
	EventInfo   EventInfo                  `json:"event_info"`
	EventAssets EventAssets                `json:"event_assets"`
	EventCards  []CardFullThumbnailRequest `json:"event_cards"`
}

// EventHistory 描述活动历史记录
type EventHistory struct {
	ID               int     `json:"id"`
	EventName        string  `json:"event_name"`
	StartAt          int64   `json:"start_at"`
	EndAt            int64   `json:"end_at"`
	Rank             *int    `json:"rank,omitempty"`
	EventPoint       int     `json:"event_point"`
	IsWLEvent        bool    `json:"is_wl_event"`
	BannerPath       string  `json:"banner_path"`
	WLCharaIconPath  *string `json:"wl_chara_icon_path,omitempty"`
	GameCharacterID  *int    `json:"game_character_id,omitempty"`
	WorldBloomBanner *string `json:"world_bloom_banner_path,omitempty"`
}

// EventRecordRequest 绘制活动记录
type EventRecordRequest struct {
	EventInfo   []EventHistory             `json:"event_info"`
	WLEventInfo []EventHistory             `json:"wl_event_info"`
	UserInfo    DetailedProfileCardRequest `json:"user_info"`
}

// EventBrief 摘要
type EventBrief struct {
	ID              int                        `json:"id"`
	EventName       string                     `json:"event_name"`
	EventType       string                     `json:"event_type"`
	StartAt         int64                      `json:"start_at"`
	EndAt           int64                      `json:"end_at"`
	EventBannerPath string                     `json:"event_banner_path"`
	EventCards      []CardFullThumbnailRequest `json:"event_cards,omitempty"`
	EventAttrPath   *string                    `json:"event_attr_path,omitempty"`
	EventCharaPath  *string                    `json:"event_chara_path,omitempty"`
	EventUnitPath   *string                    `json:"event_unit_path,omitempty"`
}

// EventListRequest 活动列表绘制请求
type EventListRequest struct {
	Region    string       `json:"region"`
	EventInfo []EventBrief `json:"event_info"`
}

// MusicListItem represents a single entry inside the music list image
type MusicListItem struct {
	ID         int   `json:"id"`
	Difficulty int   `json:"difficulty"`
	ReleaseAt  int64 `json:"release_at"`
}

// DetailedProfileCardRequest mirrors DrawingAPI's profile payload
type DetailedProfileCardRequest struct {
	ID              string                   `json:"id"`
	Region          string                   `json:"region"`
	Nickname        string                   `json:"nickname"`
	Source          string                   `json:"source"`
	UpdateTime      int64                    `json:"update_time"`
	Mode            string                   `json:"mode,omitempty"`
	IsHideUID       bool                     `json:"is_hide_uid"`
	LeaderImagePath string                   `json:"leader_image_path"`
	HasFrame        bool                     `json:"has_frame"`
	FramePath       *string                  `json:"frame_path,omitempty"`
	UserCards       []map[string]interface{} `json:"user_cards,omitempty"`
}

// BasicProfile contains minimum profile info for generic profile cards
type BasicProfile struct {
	ID              string  `json:"id"`
	Region          string  `json:"region"`
	Nickname        string  `json:"nickname"`
	IsHideUID       bool    `json:"is_hide_uid"`
	LeaderImagePath string  `json:"leader_image_path"`
	HasFrame        bool    `json:"has_frame"`
	FramePath       *string `json:"frame_path,omitempty"`
}

// ProfileDataSource records provenance of the profile data
type ProfileDataSource struct {
	Name       string  `json:"name"`
	Source     *string `json:"source,omitempty"`
	UpdateTime *int64  `json:"update_time,omitempty"`
	Mode       *string `json:"mode,omitempty"`
}

// ProfileCardRequest is consumed by multiple DrawingAPI endpoints
type ProfileCardRequest struct {
	Profile     *BasicProfile       `json:"profile,omitempty"`
	DataSources []ProfileDataSource `json:"data_sources"`
	ErrorMsg    *string             `json:"error_message,omitempty"`
}

// MusicListRequest represents /api/pjsk/music/list payload
type MusicListRequest struct {
	UserResults          map[int]string             `json:"user_results"`
	MusicList            []MusicListItem            `json:"music_list"`
	JacketsPathList      map[int]string             `json:"jackets_path_list"`
	RequiredDifficulties string                     `json:"required_difficulties"`
	Profile              DetailedProfileCardRequest `json:"profile"`
	PlayResultIconPath   map[string]string          `json:"play_result_icon_path_map,omitempty"`
	Title                *string                    `json:"title,omitempty"`
	TitleStyle           interface{}                `json:"title_style,omitempty"`
	TitleShadow          bool                       `json:"title_shadow,omitempty"`
}

// MusicChartRequest mirrors DrawingAPI generate_chart payload
type MusicChartRequest struct {
	MusicID              interface{}            `json:"music_id"`
	Title                string                 `json:"title"`
	Artist               string                 `json:"artist"`
	Difficulty           string                 `json:"difficulty"`
	PlayLevel            interface{}            `json:"play_level"`
	Skill                bool                   `json:"skill"`
	JacketPath           string                 `json:"jacket_path"`
	SusPath              string                 `json:"sus_path"`
	StylePath            *string                `json:"style_path,omitempty"`
	NoteHost             string                 `json:"note_host"`
	MusicMeta            map[string]interface{} `json:"music_meta,omitempty"`
	TargetSegmentSeconds *float64               `json:"target_segment_seconds,omitempty"`
}

// PlayProgressCount mirrors DrawingAPI struct for difficulty statistics
type PlayProgressCount struct {
	Level    int `json:"level"`
	Total    int `json:"total"`
	NotClear int `json:"not_clear"`
	Clear    int `json:"clear"`
	FC       int `json:"fc"`
	AP       int `json:"ap"`
}

// PlayProgressRequest is used by /api/pjsk/music/progress
type PlayProgressRequest struct {
	Counts     []PlayProgressCount `json:"counts"`
	Difficulty string              `json:"difficulty"`
	Profile    *ProfileCardRequest `json:"profile"`
}

// MusicComboReward defines reward info per level per diff
type MusicComboReward struct {
	Level  int `json:"level"`
	Reward int `json:"reward"`
}

// DetailMusicRewardsRequest is consumed by /rewards/detail
type DetailMusicRewardsRequest struct {
	RankRewards  int                           `json:"rank_rewards"`
	ComboRewards map[string][]MusicComboReward `json:"combo_rewards"`
	Profile      *ProfileCardRequest           `json:"profile"`
	JewelIcon    *string                       `json:"jewel_icon_path,omitempty"`
	ShardIcon    *string                       `json:"shard_icon_path,omitempty"`
}

// BasicMusicRewardsRequest is consumed by /rewards/basic
type BasicMusicRewardsRequest struct {
	RankRewards  string              `json:"rank_rewards"`
	ComboRewards map[string]string   `json:"combo_rewards"`
	Profile      *ProfileCardRequest `json:"profile"`
	JewelIcon    *string             `json:"jewel_icon_path,omitempty"`
	ShardIcon    *string             `json:"shard_icon_path,omitempty"`
}

// ===== Education Requests =====

// CharacterChallengeInfo mirrors DrawingAPI character challenge payload.
type CharacterChallengeInfo struct {
	CharaID       int    `json:"chara_id"`
	Rank          int    `json:"rank"`
	Score         int    `json:"score"`
	Jewel         int    `json:"jewel"`
	Shard         int    `json:"shard"`
	CharaIconPath string `json:"chara_icon_path"`
}

// ChallengeLiveDetailsRequest mirrors /education/challenge-live request.
type ChallengeLiveDetailsRequest struct {
	Profile             DetailedProfileCardRequest `json:"profile"`
	CharacterChallenges []CharacterChallengeInfo   `json:"character_challenges"`
	MaxScore            int                        `json:"max_score"`
	JewelIconPath       *string                    `json:"jewel_icon_path,omitempty"`
	ShardIconPath       *string                    `json:"shard_icon_path,omitempty"`
}

// CharacterBonus mirrors DrawingAPI character bonus payload.
type CharacterBonus struct {
	CharaID       int     `json:"chara_id"`
	CharaIconPath string  `json:"chara_icon_path"`
	AreaItem      float64 `json:"area_item"`
	Rank          float64 `json:"rank"`
	Fixture       float64 `json:"fixture"`
	Total         float64 `json:"total"`
}

// UnitBonus mirrors DrawingAPI unit bonus payload.
type UnitBonus struct {
	Unit         string  `json:"unit"`
	UnitIconPath string  `json:"unit_icon_path"`
	AreaItem     float64 `json:"area_item"`
	Gate         float64 `json:"gate"`
	Total        float64 `json:"total"`
}

// AttrBonus mirrors DrawingAPI attr bonus payload.
type AttrBonus struct {
	Attr         string  `json:"attr"`
	AttrIconPath string  `json:"attr_icon_path"`
	AreaItem     float64 `json:"area_item"`
	Total        float64 `json:"total"`
}

// PowerBonusDetailRequest mirrors /education/power-bonus request.
type PowerBonusDetailRequest struct {
	Profile      DetailedProfileCardRequest `json:"profile"`
	CharaBonuses []CharacterBonus           `json:"chara_bonuses"`
	UnitBonuses  []UnitBonus                `json:"unit_bonuses"`
	AttrBonuses  []AttrBonus                `json:"attr_bonuses"`
}

// AreaItemMaterial mirrors DrawingAPI material payload.
type AreaItemMaterial struct {
	MaterialID   int    `json:"material_id"`
	MaterialIcon string `json:"material_icon_path"`
	Quantity     int    `json:"quantity"`
	HaveQuantity int    `json:"have_quantity"`
	SumQuantity  int    `json:"sum_quantity"`
	IsEnough     bool   `json:"is_enough"`
}

// AreaItemLevel mirrors DrawingAPI area item level payload.
type AreaItemLevel struct {
	Level      int                `json:"level"`
	Bonus      float64            `json:"bonus"`
	CanUpgrade bool               `json:"can_upgrade"`
	Materials  []AreaItemMaterial `json:"materials"`
}

// AreaItemInfo mirrors DrawingAPI area item info payload.
type AreaItemInfo struct {
	ItemID         int             `json:"item_id"`
	CurrentLevel   int             `json:"current_level"`
	ItemIconPath   string          `json:"item_icon_path"`
	TargetIconPath *string         `json:"target_icon_path,omitempty"`
	Levels         []AreaItemLevel `json:"levels"`
}

// AreaItemUpgradeMaterialsRequest mirrors /education/area-item request.
type AreaItemUpgradeMaterialsRequest struct {
	Profile    *DetailedProfileCardRequest `json:"profile,omitempty"`
	AreaItems  []AreaItemInfo              `json:"area_items"`
	HasProfile bool                        `json:"has_profile"`
}

// BondInfo mirrors DrawingAPI bonds payload.
type BondInfo struct {
	CharaID1       int    `json:"chara_id1"`
	CharaID2       int    `json:"chara_id2"`
	CharaIconPath1 string `json:"chara_icon_path1"`
	CharaIconPath2 string `json:"chara_icon_path2"`
	CharaRank1     int    `json:"chara_rank1"`
	CharaRank2     int    `json:"chara_rank2"`
	BondLevel      int    `json:"bond_level"`
	NeedExp        *int   `json:"need_exp,omitempty"`
	HasBond        bool   `json:"has_bond"`
	Color1         [3]int `json:"color1"`
	Color2         [3]int `json:"color2"`
}

// BondsRequest mirrors /education/bonds request.
type BondsRequest struct {
	Profile  DetailedProfileCardRequest `json:"profile"`
	Bonds    []BondInfo                 `json:"bonds"`
	MaxLevel int                        `json:"max_level"`
}

// LeaderCountInfo mirrors DrawingAPI leader count payload.
type LeaderCountInfo struct {
	CharaID       int    `json:"chara_id"`
	CharaIconPath string `json:"chara_icon_path"`
	PlayCount     int    `json:"play_count"`
	EXLevel       int    `json:"ex_level"`
	EXCount       int    `json:"ex_count"`
}

// LeaderCountRequest mirrors /education/leader-count request.
type LeaderCountRequest struct {
	Profile      DetailedProfileCardRequest `json:"profile"`
	LeaderCounts []LeaderCountInfo          `json:"leader_counts"`
	MaxPlayCount int                        `json:"max_play_count"`
}

// HonorRequest 称号绘制请求（匹配 DrawingAPI 的 `HonorRequest`）
type HonorRequest struct {
	HonorType               *string `json:"honor_type,omitempty"`
	GroupType               *string `json:"group_type,omitempty"`
	HonorRarity             *string `json:"honor_rarity,omitempty"`
	HonorLevel              *int    `json:"honor_level,omitempty"`
	FcOrApLevel             *string `json:"fc_or_ap_level,omitempty"`
	IsEmpty                 bool    `json:"is_empty"`
	IsMainHonor             bool    `json:"is_main_honor"`
	HonorImgPath            *string `json:"honor_img_path,omitempty"`
	RankImgPath             *string `json:"rank_img_path,omitempty"`
	LvImgPath               *string `json:"lv_img_path,omitempty"`
	Lv6ImgPath              *string `json:"lv6_img_path,omitempty"`
	EmptyHonorPath          *string `json:"empty_honor_path,omitempty"`
	ScrollImgPath           *string `json:"scroll_img_path,omitempty"`
	WordImgPath             *string `json:"word_img_path,omitempty"`
	CharaIconPath           *string `json:"chara_icon_path,omitempty"`
	CharaIconPath2          *string `json:"chara_icon_path2,omitempty"`
	CharaID                 *string `json:"chara_id,omitempty"`
	CharaID2                *string `json:"chara_id2,omitempty"`
	BondsBgPath             *string `json:"bonds_bg_path,omitempty"`
	BondsBgPath2            *string `json:"bonds_bg_path2,omitempty"`
	MaskImgPath             *string `json:"mask_img_path,omitempty"`
	FrameImgPath            *string `json:"frame_img_path,omitempty"`
	FrameDegreeLevelImgPath *string `json:"frame_degree_level_img_path,omitempty"`
}

// ProfileRequest 合成个人信息图片所需数据
type ProfileRequest struct {
	Profile              BasicProfile               `json:"profile"`
	Rank                 int                        `json:"rank"`
	TwitterID            string                     `json:"twitter_id"`
	Word                 string                     `json:"word"`
	PCards               []CardFullThumbnailRequest `json:"pcards"`
	BgSettings           *ProfileBgSettings         `json:"bg_settings,omitempty"`
	Honors               []HonorRequest             `json:"honors"`
	MusicDifficultyCount []MusicClearCount          `json:"music_difficulty_count"`
	CharacterRank        []CharacterRank            `json:"character_rank"`
	SoloLive             *SoloLiveRank              `json:"solo_live,omitempty"`
	UpdateTime           int64                      `json:"update_time,omitempty"`
	LvRankBgPath         string                     `json:"lv_rank_bg_path"`
	XIconPath            string                     `json:"x_icon_path"`
	IconClearPath        string                     `json:"icon_clear_path"`
	IconFcPath           string                     `json:"icon_fc_path"`
	IconApPath           string                     `json:"icon_ap_path"`
	CharaRankIconPathMap map[int]string             `json:"chara_rank_icon_path_map"`
	FramePaths           *PlayerFramePaths          `json:"frame_paths,omitempty"`
}

// ProfileBgSettings 个人信息背景设置
type ProfileBgSettings struct {
	ImgPath  string `json:"img_path,omitempty"`
	Blur     int    `json:"blur"`
	Alpha    int    `json:"alpha"`
	Vertical bool   `json:"vertical"`
}

// MusicClearCount 歌曲完成情况
type MusicClearCount struct {
	Difficulty string `json:"difficulty"` // easy, normal, hard, expert, master, append
	Clear      int    `json:"clear"`
	FC         int    `json:"fc"`
	AP         int    `json:"ap"`
}

// CharacterRank 角色等级
type CharacterRank struct {
	CharacterID int `json:"character_id"`
	Rank        int `json:"rank"`
}

// SoloLiveRank 挑战live等级
type SoloLiveRank struct {
	CharacterID int `json:"character_id"`
	Score       int `json:"score"`
	Rank        int `json:"rank"`
}

// PlayerFramePaths 玩家头像框各部件路径
type PlayerFramePaths struct {
	Base        string `json:"base"`
	CenterTop   string `json:"centertop"`
	LeftBottom  string `json:"leftbottom"`
	LeftTop     string `json:"lefttop"`
	RightBottom string `json:"rightbottom"`
	RightTop    string `json:"righttop"`
}

// StampData mirrors DrawingAPI stamp item payload.
type StampData struct {
	ID        int    `json:"id"`
	ImagePath string `json:"image_path"`
	TextColor [4]int `json:"text_color,omitempty"`
}

// StampListRequest mirrors /stamp/list request.
type StampListRequest struct {
	PromptMessage *string     `json:"prompt_message,omitempty"`
	Stamps        []StampData `json:"stamps"`
}

// CharaBirthdayCard mirrors birthday card entry.
type CharaBirthdayCard struct {
	ID            int    `json:"id"`
	ThumbnailPath string `json:"thumbnail_path"`
}

// BirthdayEventTime mirrors birthday event time text span.
type BirthdayEventTime struct {
	StartText string `json:"start_text"`
	EndText   string `json:"end_text"`
}

// CharaBirthdayData mirrors compact character birthday info.
type CharaBirthdayData struct {
	CID      int    `json:"cid"`
	Month    int    `json:"month"`
	Day      int    `json:"day"`
	IconPath string `json:"icon_path"`
}

// CharaBirthdayRequest mirrors /misc/chara-birthday request.
type CharaBirthdayRequest struct {
	CID               int                 `json:"cid"`
	Month             int                 `json:"month"`
	Day               int                 `json:"day"`
	RegionName        string              `json:"region_name"`
	DaysUntilBirthday int                 `json:"days_until_birthday"`
	ColorCode         string              `json:"color_code"`
	SDImagePath       string              `json:"sd_image_path"`
	TitleImagePath    string              `json:"title_image_path"`
	CardImagePath     string              `json:"card_image_path"`
	Cards             []CharaBirthdayCard `json:"cards"`
	IsFifthAnniv      bool                `json:"is_fifth_anniv"`
	GachaTime         BirthdayEventTime   `json:"gacha_time"`
	LiveTime          BirthdayEventTime   `json:"live_time"`
	DropTime          *BirthdayEventTime  `json:"drop_time,omitempty"`
	FlowerTime        *BirthdayEventTime  `json:"flower_time,omitempty"`
	PartyTime         *BirthdayEventTime  `json:"party_time,omitempty"`
	AllCharacters     []CharaBirthdayData `json:"all_characters"`
}
