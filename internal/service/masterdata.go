package service

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"Haruki-Service-API/pkg/masterdata"
)

// MasterDataService 游戏主数据服务
type MasterDataService struct {
	dataDir string
	region  string

	// 数据缓存
	cards              []masterdata.Card
	characters         []masterdata.Character
	events             []masterdata.Event
	skills             []masterdata.Skill
	musics             []masterdata.Music // Cached Musics
	gachas             []masterdata.Gacha
	eventCards         []masterdata.EventCard
	eventDeckBonuses   []masterdata.EventDeckBonus
	eventMusics        []masterdata.EventMusic
	gameCharacterUnits []masterdata.GameCharacterUnit
	costume3ds         []masterdata.Costume3d
	cardCostume3ds     []masterdata.CardCostume3d
	cardSupplies       []masterdata.CardSupply
	musicDifficulties  []masterdata.MusicDifficulty
	musicVocals        []masterdata.MusicVocal
	musicTags          []masterdata.MusicTag
	limitedTimeMusics  []masterdata.LimitedTimeMusic
	challengeRewards   []masterdata.ChallengeLiveHighScoreReward
	resourceBoxes      []masterdata.ResourceBox
	worldBlooms        []masterdata.WorldBloom

	// 索引
	cardByID          map[int]*masterdata.Card
	charByID          map[int]*masterdata.Character
	skillByID         map[int]*masterdata.Skill
	musicByID         map[int]*masterdata.Music // Music Index
	eventByID         map[int]*masterdata.Event
	gachaByID         map[int]*masterdata.Gacha
	eventCardByID     map[int]*masterdata.EventCard
	costume3dByID     map[int]*masterdata.Costume3d
	costume3dByCardID map[int][]int // cardID -> []costume3dID
	cardSupplyByID    map[int]*masterdata.CardSupply
	gameCharUnitByID  map[int]*masterdata.GameCharacterUnit
	eventIDsByMusicID map[int][]int

	// 关联索引
	cardsByEventID        map[int][]int // eventID -> []cardID
	cardsByGachaID        map[int][]int // gachaID -> []cardID
	deckBonusesByEventID  map[int][]*masterdata.EventDeckBonus
	gameCharUnitByCharID  map[int][]*masterdata.GameCharacterUnit
	difficultiesByMusicID map[int][]*masterdata.MusicDifficulty
	vocalsByMusicID       map[int][]*masterdata.MusicVocal
	tagsByMusicID         map[int][]string // musicID -> []tagName
	limitedTimesByMusicID map[int][]*masterdata.LimitedTimeMusic
	challengeRewardsByCID map[int][]*masterdata.ChallengeLiveHighScoreReward
	resourceBoxByID       map[int]*masterdata.ResourceBox
	resourceBoxesByPurpose map[string]map[int]*masterdata.ResourceBox
	worldBloomsByEventID  map[int][]*masterdata.WorldBloom

	// 角色昵称映射
	charNicknames map[string]int
}

// GetNicknames 获取角色昵称映射
func (s *MasterDataService) GetNicknames() map[string]int {
	return s.charNicknames
}

// GetDataDir 获取数据目录
func (s *MasterDataService) GetDataDir() string {
	return s.dataDir
}

// GetRegion 获取当前数据区服
func (s *MasterDataService) GetRegion() string {
	return s.region
}

// NewMasterDataService 创建 MasterData 服务
func NewMasterDataService(dataDir string, region string) *MasterDataService {
	nicknames := initCharacterNicknames()
	return &MasterDataService{
		dataDir:               dataDir,
		region:                region,
		cardByID:              make(map[int]*masterdata.Card),
		charByID:              make(map[int]*masterdata.Character),
		skillByID:             make(map[int]*masterdata.Skill),
		musicByID:             make(map[int]*masterdata.Music),
		eventByID:             make(map[int]*masterdata.Event),
		gachaByID:             make(map[int]*masterdata.Gacha),
		cardsByEventID:        make(map[int][]int),
		cardsByGachaID:        make(map[int][]int),
		eventCardByID:         make(map[int]*masterdata.EventCard),
		costume3dByID:         make(map[int]*masterdata.Costume3d),
		costume3dByCardID:     make(map[int][]int),
		cardSupplyByID:        make(map[int]*masterdata.CardSupply),
		gameCharUnitByID:      make(map[int]*masterdata.GameCharacterUnit),
		eventIDsByMusicID:     make(map[int][]int),
		deckBonusesByEventID:  make(map[int][]*masterdata.EventDeckBonus),
		difficultiesByMusicID: make(map[int][]*masterdata.MusicDifficulty),
		vocalsByMusicID:       make(map[int][]*masterdata.MusicVocal),
		tagsByMusicID:         make(map[int][]string),
		limitedTimesByMusicID: make(map[int][]*masterdata.LimitedTimeMusic),
		challengeRewardsByCID: make(map[int][]*masterdata.ChallengeLiveHighScoreReward),
		resourceBoxByID:       make(map[int]*masterdata.ResourceBox),
		resourceBoxesByPurpose: make(map[string]map[int]*masterdata.ResourceBox),
		worldBloomsByEventID:  make(map[int][]*masterdata.WorldBloom),
		charNicknames:         nicknames,
	}
}

// LoadAll 加载所有数据
func (s *MasterDataService) LoadAll() error {
	if err := s.loadCards(); err != nil {
		return fmt.Errorf("failed to load cards: %w", err)
	}
	if err := s.loadCharacters(); err != nil {
		return fmt.Errorf("failed to load characters: %w", err)
	}
	if err := s.loadSkills(); err != nil {
		return fmt.Errorf("failed to load skills: %w", err)
	}
	if err := s.loadMusics(); err != nil {
		return fmt.Errorf("failed to load musics: %w", err)
	}
	if err := s.loadMusicDifficulties(); err != nil {
		return fmt.Errorf("failed to load music difficulties: %w", err)
	}
	if err := s.loadMusicVocals(); err != nil {
		return fmt.Errorf("failed to load music vocals: %w", err)
	}
	if err := s.loadMusicTags(); err != nil {
		return fmt.Errorf("failed to load music tags: %w", err)
	}
	if err := s.loadLimitedTimeMusics(); err != nil {
		return fmt.Errorf("failed to load limited time musics: %w", err)
	}
	if err := s.loadChallengeLiveRewards(); err != nil {
		return fmt.Errorf("failed to load challenge live rewards: %w", err)
	}
	if err := s.loadResourceBoxes(); err != nil {
		return fmt.Errorf("failed to load resource boxes: %w", err)
	}
	if err := s.loadEvents(); err != nil {
		return fmt.Errorf("failed to load events: %w", err)
	}
	if err := s.loadGachas(); err != nil {
		return fmt.Errorf("failed to load gachas: %w", err)
	}
	if err := s.loadEventCards(); err != nil {
		return fmt.Errorf("failed to load event cards: %w", err)
	}
	if err := s.loadEventMusics(); err != nil {
		return fmt.Errorf("failed to load event musics: %w", err)
	}
	if err := s.loadEventDeckBonuses(); err != nil {
		return fmt.Errorf("failed to load event deck bonuses: %w", err)
	}
	if err := s.loadGameCharacterUnits(); err != nil {
		return fmt.Errorf("failed to load game character units: %w", err)
	}
	if err := s.loadCostume3ds(); err != nil {
		return fmt.Errorf("failed to load costume3ds: %w", err)
	}
	if err := s.loadCardCostume3ds(); err != nil {
		return fmt.Errorf("failed to load card costume 3ds: %w", err)
	}
	if err := s.loadCardSupplies(); err != nil {
		return fmt.Errorf("failed to load card supplies: %w", err)
	}
	if err := s.loadWorldBlooms(); err != nil {
		return fmt.Errorf("failed to load world blooms: %w", err)
	}

	// 构建索引
	s.buildIndexes()

	return nil
}

// loadJSON 通用 JSON 加载函数
func (s *MasterDataService) loadJSON(filename string, v interface{}) error {
	path := filepath.Join(s.dataDir, filename)
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

// buildIndexes 构建索引
func (s *MasterDataService) buildIndexes() {
	// 卡牌索引
	for i := range s.cards {
		s.cardByID[s.cards[i].ID] = &s.cards[i]
	}

	// 角色索引
	for i := range s.characters {
		s.charByID[s.characters[i].ID] = &s.characters[i]
	}

	// 技能索引
	for i := range s.skills {
		s.skillByID[s.skills[i].ID] = &s.skills[i]
	}

	// 音乐索引 & 排序
	sort.Slice(s.musics, func(i, j int) bool {
		return s.musics[i].PublishedAt < s.musics[j].PublishedAt
	})
	for i := range s.musics {
		s.musicByID[s.musics[i].ID] = &s.musics[i]
	}

	// 活动索引 & 排序 (按时间)
	// Sort events by StartAt (asc)
	sort.Slice(s.events, func(i, j int) bool {
		return s.events[i].StartAt < s.events[j].StartAt
	})
	for i := range s.events {
		s.eventByID[s.events[i].ID] = &s.events[i]
	}

	// 卡池索引 & 排序
	sort.Slice(s.gachas, func(i, j int) bool {
		// Mimic Lunabot/DB natural order (usually ID asc)
		return s.gachas[i].ID < s.gachas[j].ID
	})
	for i := range s.gachas {
		s.gachaByID[s.gachas[i].ID] = &s.gachas[i]
	}

	// 活动卡牌关联索引
	for _, ec := range s.eventCards {
		s.cardsByEventID[ec.EventID] = append(s.cardsByEventID[ec.EventID], ec.CardID)
	}

	// 活动加成关联索引
	for i := range s.eventDeckBonuses {
		bonus := &s.eventDeckBonuses[i]
		s.deckBonusesByEventID[bonus.EventID] = append(s.deckBonusesByEventID[bonus.EventID], bonus)
	}
	for _, em := range s.eventMusics {
		s.eventIDsByMusicID[em.MusicID] = append(s.eventIDsByMusicID[em.MusicID], em.EventID)
	}

	// 角色组合索引
	for i := range s.gameCharacterUnits {
		s.gameCharUnitByID[s.gameCharacterUnits[i].ID] = &s.gameCharacterUnits[i]
	}

	// 服装索引
	for i := range s.costume3ds {
		s.costume3dByID[s.costume3ds[i].ID] = &s.costume3ds[i]
	}
	for _, cc := range s.cardCostume3ds {
		s.costume3dByCardID[cc.CardID] = append(s.costume3dByCardID[cc.CardID], cc.Costume3dID)
	}

	// 卡牌供给类型索引
	for i := range s.cardSupplies {
		s.cardSupplyByID[s.cardSupplies[i].ID] = &s.cardSupplies[i]
	}

	// 音乐关联索引
	for i := range s.musicDifficulties {
		d := &s.musicDifficulties[i]
		s.difficultiesByMusicID[d.MusicID] = append(s.difficultiesByMusicID[d.MusicID], d)
	}
	for i := range s.musicVocals {
		v := &s.musicVocals[i]
		s.vocalsByMusicID[v.MusicID] = append(s.vocalsByMusicID[v.MusicID], v)
	}
	for i := range s.musicTags {
		t := &s.musicTags[i]
		s.tagsByMusicID[t.MusicID] = append(s.tagsByMusicID[t.MusicID], t.MusicTag)
	}
	for i := range s.limitedTimeMusics {
		lt := &s.limitedTimeMusics[i]
		s.limitedTimesByMusicID[lt.MusicID] = append(s.limitedTimesByMusicID[lt.MusicID], lt)
	}
	for i := range s.challengeRewards {
		reward := &s.challengeRewards[i]
		s.challengeRewardsByCID[reward.CharacterID] = append(s.challengeRewardsByCID[reward.CharacterID], reward)
	}
	for i := range s.resourceBoxes {
		box := &s.resourceBoxes[i]
		if _, ok := s.resourceBoxesByPurpose[box.ResourceBoxPurpose]; !ok {
			s.resourceBoxesByPurpose[box.ResourceBoxPurpose] = make(map[int]*masterdata.ResourceBox)
		}
		if _, exists := s.resourceBoxesByPurpose[box.ResourceBoxPurpose][box.ID]; !exists {
			s.resourceBoxesByPurpose[box.ResourceBoxPurpose][box.ID] = box
		}
		if _, exists := s.resourceBoxByID[box.ID]; !exists {
			s.resourceBoxByID[box.ID] = box
		}
	}
	for i := range s.worldBlooms {
		wb := &s.worldBlooms[i]
		s.worldBloomsByEventID[wb.EventID] = append(s.worldBloomsByEventID[wb.EventID], wb)
	}
}

// initCharacterNicknames 初始化角色昵称映射
func initCharacterNicknames() map[string]int {
	return map[string]int{
		"ick": 1, "ichika": 1, "星乃一歌": 1,
		"saki": 2, "咲希": 2, "天马咲希": 2,
		"hnm": 3, "honami": 3, "穗波": 3,
		"shiho": 4, "志步": 4, "日野森志步": 4,
		"mnr": 5, "minori": 5, "实乃理": 5, "花里みのり": 5,
		"hrk": 6, "haruka": 6, "遥": 6,
		"airi": 7, "爱莉": 7, "桃井爱莉": 7,
		"szk": 8, "shizuku": 8, "雫": 8,
		"kohane": 9, "小豆泽心羽": 9,
		"an": 10, "杏": 10, "白石杏": 10,
		"akito": 11, "彰人": 11, "青柳彰人": 11,
		"toya": 12, "冬弥": 12, "天马冬弥": 12,
		"tsks": 13, "tsukasa": 13, "司": 13,
		"emu": 14, "笑梦": 14, "天马笑梦": 14,
		"nene": 15, "宁宁": 15, "楠宁宁": 15,
		"rui": 16, "类": 16, "神代类": 16,
		"knd": 17, "kanade": 17, "奏": 17,
		"mfy": 18, "mafuyu": 18, "真冬": 18,
		"ena": 19, "绘名": 19, "朝比奈绘名": 19,
		"mzk": 20, "mizuki": 20, "瑞希": 20, "晓山瑞希": 20,
	}
}

// SetCards sets internal cards for testing
func (s *MasterDataService) SetCards(cards []masterdata.Card) {
	s.cards = cards
	s.buildIndexes()
}

// SetNicknames sets nicknames for testing
func (s *MasterDataService) SetNicknames(nicks map[string]int) {
	s.charNicknames = nicks
}
