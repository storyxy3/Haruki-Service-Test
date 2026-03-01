package model

// ScoreData mirrors score control row item in DrawingAPI.
type ScoreData struct {
	EventBonus int `json:"event_bonus"`
	Boost      int `json:"boost"`
	ScoreMin   int `json:"score_min"`
	ScoreMax   int `json:"score_max"`
}

// ScoreControlRequest mirrors /score/control request.
type ScoreControlRequest struct {
	MusicCoverPath  string      `json:"music_cover_path"`
	MusicID         int         `json:"music_id"`
	MusicTitle      string      `json:"music_title"`
	MusicBasicPoint int         `json:"music_basic_point"`
	TargetPoint     int         `json:"target_point"`
	ValidScores     []ScoreData `json:"valid_scores"`
}

// CustomRoomScoreRequest mirrors /score/custom-room request.
type CustomRoomScoreRequest struct {
	TargetPoint    int                         `json:"target_point"`
	CandidatePairs [][2]int                    `json:"candidate_pairs"`
	MusicListMap   map[string][]map[string]any `json:"music_list_map"`
}

// MusicMetaInfo mirrors metadata item for one difficulty.
type MusicMetaInfo struct {
	Difficulty      string    `json:"difficulty"`
	MusicTime       float64   `json:"music_time"`
	TapCount        int       `json:"tap_count"`
	EventRate       float64   `json:"event_rate"`
	BaseScore       float64   `json:"base_score"`
	BaseScoreAuto   float64   `json:"base_score_auto"`
	SkillScoreSolo  []float64 `json:"skill_score_solo"`
	SkillScoreAuto  []float64 `json:"skill_score_auto"`
	SkillScoreMulti []float64 `json:"skill_score_multi"`
	FeverScore      float64   `json:"fever_score"`
}

// MusicMetaRequest mirrors /score/music-meta request item.
type MusicMetaRequest struct {
	MusicID        int             `json:"music_id"`
	MusicTitle     string          `json:"music_title"`
	MusicCoverPath string          `json:"music_cover_path"`
	Metas          []MusicMetaInfo `json:"metas"`
}

// MusicBoardItem mirrors /score/music-board row item.
type MusicBoardItem struct {
	Rank              int      `json:"rank"`
	MusicID           int      `json:"music_id"`
	Difficulty        string   `json:"difficulty"`
	Level             int      `json:"level"`
	MusicTitle        string   `json:"music_title"`
	MusicCoverPath    string   `json:"music_cover_path"`
	LiveTypePT        *float64 `json:"live_type_pt,omitempty"`
	LiveTypeRealScore *float64 `json:"live_type_real_score,omitempty"`
	LiveTypeScore     *float64 `json:"live_type_score,omitempty"`
	LiveTypeSkillAcc  *float64 `json:"live_type_skill_account,omitempty"`
	LiveTypePTPerHour *float64 `json:"live_type_pt_per_hour,omitempty"`
	PlayCountPerHour  *float64 `json:"play_count_per_hour,omitempty"`
	EventRate         float64  `json:"event_rate"`
	MusicTime         float64  `json:"music_time"`
	TPS               float64  `json:"tps"`
}

// MusicBoardRequest mirrors /score/music-board request.
type MusicBoardRequest struct {
	LiveType     string           `json:"live_type"`
	Target       string           `json:"target"`
	Ascend       bool             `json:"ascend"`
	Page         int              `json:"page"`
	TotalPage    int              `json:"total_page"`
	TitleText    string           `json:"title_text"`
	Items        []MusicBoardItem `json:"items"`
	SpecMidDiffs [][2]interface{} `json:"spec_mid_diffs,omitempty"`
	Description  string           `json:"description,omitempty"`
}
