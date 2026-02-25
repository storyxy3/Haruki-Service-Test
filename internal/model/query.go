package model

// DrawingRequest 表示发送给 DrawingAPI 的请求信息
type DrawingRequest struct {
	URL    string      `json:"url"`    // DrawingAPI 的端点 URL
	Method string      `json:"method"` // HTTP 方法（通常是 POST）
	Body   interface{} `json:"body"`   // 请求体（具体的 Request 对象）
}

// DrawingResponse 表示 DrawingAPI 的响应
type DrawingResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Data    []byte `json:"data,omitempty"` // 图片的二进制数据（Base64 或直接二进制）
}

// CardQuery 卡牌查询参数
type CardQuery struct {
	Query  string `json:"query"`             // 查询字符串，如 "mnr-7"
	Region string `json:"region"`            // 服务器区域
	UserID string `json:"user_id,omitempty"` // 用户 ID（可选）
	Mode   string `json:"mode,omitempty"`    // 模式（可选）
}

// MusicQuery 音乐查询参数
type MusicQuery struct {
	Query      string `json:"query"`
	Region     string `json:"region"`
	Difficulty string `json:"difficulty,omitempty"`
	UserID     string `json:"user_id,omitempty"` // 用户 ID
}

// MusicChartQuery 表示谱面预览请求
type MusicChartQuery struct {
	Query      string `json:"query"`
	Region     string `json:"region"`
	Difficulty string `json:"difficulty,omitempty"`
	Skill      bool   `json:"skill,omitempty"`
	Style      string `json:"style,omitempty"`
}

// EventDetailQuery 表示活动详情查询
type EventDetailQuery struct {
	Region  string `json:"region"`
	EventID int    `json:"event_id"`
}

// EventListQuery 表示活动列表筛选条件
type EventListQuery struct {
	Region        string `json:"region"`
	EventType     string `json:"event_type,omitempty"`
	Unit          string `json:"unit,omitempty"`
	Attr          string `json:"attr,omitempty"`
	Year          int    `json:"year,omitempty"`
	CharacterID   int    `json:"character_id,omitempty"`
	BannerCharID  *int   `json:"banner_char_id,omitempty"`
	IncludePast   bool   `json:"include_past,omitempty"`
	IncludeFuture bool   `json:"include_future,omitempty"`
	OnlyFuture    bool   `json:"only_future,omitempty"`
	Limit         int    `json:"limit,omitempty"`
}

// MusicListQuery 表示歌曲列表查询条件
type MusicListQuery struct {
	Difficulty   string            `json:"difficulty"`
	Level        int               `json:"level,omitempty"`
	LevelMin     int               `json:"level_min,omitempty"`
	LevelMax     int               `json:"level_max,omitempty"`
	Region       string            `json:"region"`
	IncludeLeaks bool              `json:"include_leaks,omitempty"`
	UserResults  map[int]string    `json:"user_results,omitempty"`
	Title        *string           `json:"title,omitempty"`
	TitleStyle   map[string]string `json:"title_style,omitempty"`
	TitleShadow  bool              `json:"title_shadow,omitempty"`
	Keyword      string            `json:"keyword,omitempty"`
}

// MusicProgressQuery 表示打歌进度请求
type MusicProgressQuery struct {
	Difficulty string                 `json:"difficulty"`
	Region     string                 `json:"region"`
	Counts     []PlayProgressCount    `json:"counts,omitempty"`
	Title      *string                `json:"title,omitempty"`
	TitleStyle map[string]interface{} `json:"title_style,omitempty"`
}

// MusicRewardsDetailQuery 表示详细奖励请求
type MusicRewardsDetailQuery struct {
	Region       string                        `json:"region"`
	RankRewards  int                           `json:"rank_rewards"`
	ComboRewards map[string][]MusicComboReward `json:"combo_rewards"`
	Title        *string                       `json:"title,omitempty"`
	TitleStyle   map[string]interface{}        `json:"title_style,omitempty"`
	JewelIcon    *string                       `json:"jewel_icon_path,omitempty"`
	ShardIcon    *string                       `json:"shard_icon_path,omitempty"`
}

// MusicRewardsBasicQuery 表示基础奖励请求
type MusicRewardsBasicQuery struct {
	Region       string                 `json:"region"`
	RankRewards  string                 `json:"rank_rewards"`
	ComboRewards map[string]string      `json:"combo_rewards"`
	Title        *string                `json:"title,omitempty"`
	TitleStyle   map[string]interface{} `json:"title_style,omitempty"`
	JewelIcon    *string                `json:"jewel_icon_path,omitempty"`
	ShardIcon    *string                `json:"shard_icon_path,omitempty"`
}

// GachaListQuery 卡池列表筛选
type GachaListQuery struct {
	Region        string `json:"region"`
	Page          int    `json:"page"`
	PageSize      int    `json:"page_size"`
	Year          int    `json:"year,omitempty"`
	IncludeFuture bool   `json:"include_future,omitempty"`
	IncludePast   bool   `json:"include_past,omitempty"`
	CardID        int    `json:"card_id,omitempty"`
	Keyword       string `json:"keyword,omitempty"`
}

// GachaDetailQuery 卡池详情
type GachaDetailQuery struct {
	Region  string `json:"region"`
	GachaID int    `json:"gacha_id"`
}

// DeckQuery 组队推荐参数
type DeckQuery struct {
	EventID       *int   `json:"event_id,omitempty"`
	Region        string `json:"region"`
	Unit          string `json:"unit,omitempty"`
	Attr          string `json:"attr,omitempty"`
	FixedCards    []int  `json:"fixed_cards,omitempty"`
	ExcludedCards []int  `json:"excluded_cards,omitempty"`
}

// HonorQuery 称号绘制查询
type HonorQuery struct {
	Region     string `json:"region"`
	HonorID    int    `json:"honor_id"`
	HonorLevel int    `json:"honor_level,omitempty"`
	IsMain     bool   `json:"is_main,omitempty"`
	Rank       int    `json:"rank,omitempty"` // 用于绘制排名数字，未来活动记录指令可能使用
}

// ProfileQuery 玩家名片查询
type ProfileQuery struct {
	UserID string `json:"user_id"`
	Region string `json:"region"`
}
