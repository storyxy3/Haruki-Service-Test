package service

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"Haruki-Service-API/internal/model"
)

// isNumeric 判断字符串是否全为数字
func isNumeric(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return len(s) > 0
}

// UserDataService 负责加载本地 suite 导出的 user.json，并为需要玩家数据的模块提供便捷访问
type UserDataService struct {
	baseProfile    *model.DetailedProfileCardRequest
	musicResult    map[string]map[int]string
	challenge      *ChallengeLiveData
	rawData        *RawUserData // Keep raw for detailed mapping
	musicMetaBytes []byte       // Raw bytes from music_metas.json
	rawJSON        []byte       // Original user.json bytes (preserved for C++ engine)
}

// RawUserData JSON structures (only keep fields we need)
type RawUserData struct {
	Now            int64            `json:"now"`
	UserGamedata   RawUserGamedata  `json:"userGamedata"`
	UserProfile    RawUserProfile   `json:"userProfile"`
	UserDecks      []RawUserDeck    `json:"userDecks"`
	UserCards      []RawUserCard    `json:"userCards"`
	UserMusicStats []RawMusicResult `json:"userMusicResults"`
	// Education related snapshots
	UserChallengeLiveSoloResults          []RawChallengeLiveResult `json:"userChallengeLiveSoloResults"`
	UserChallengeLiveSoloStages           []RawChallengeLiveStage  `json:"userChallengeLiveSoloStages"`
	UserChallengeLiveSoloHighScoreRewards []RawChallengeLiveReward `json:"userChallengeLiveSoloHighScoreRewards"`
	UserCharacters                        []RawUserCharacter       `json:"userCharacters"`
	UserMusicClear                        []RawMusicClear          `json:"userMusicDifficultyClearCounts"`
	UserHonors                            []RawUserHonor           `json:"userHonors"`
	UserProfileHonors                     []RawUserProfileHonor    `json:"userProfileHonors"`
	UserFrames                            []RawUserFrame           `json:"userPlayerFrames"`
	UserEvents                            []RawUserEvent           `json:"userEvents"`
	UserEventResults                      []RawUserEventResult     `json:"userEventResults"`
}

type RawUserCharacter struct {
	CharacterID   int `json:"characterId"`
	CharacterRank int `json:"characterRank"`
}

type RawMusicClear struct {
	MusicDifficultyType string `json:"musicDifficultyType"`
	LiveClear           int    `json:"liveClear"`
	FullCombo           int    `json:"fullCombo"`
	AllPerfect          int    `json:"allPerfect"`
}

type RawUserEvent struct {
	EventID    int `json:"eventId"`
	EventPoint int `json:"eventPoint"`
}

type RawUserEventResult struct {
	EventID int `json:"eventId"`
	Rank    int `json:"rank"`
}

type RawUserHonor struct {
	Seq           int    `json:"seq"`
	HonorID       int    `json:"honorId"`
	HonorLevel    int    `json:"level"`
	ProfilePlayer bool   `json:"profilePlayer"`
	HonorRarity   string `json:"honorRarity"`
}

type RawUserFrame struct {
	PlayerFrameID           int    `json:"playerFrameId"`
	PlayerFrameAttachStatus string `json:"playerFrameAttachStatus"`
}

type RawUserProfileHonor struct {
	Seq              int    `json:"seq"`
	ProfileHonorType string `json:"profileHonorType"` // "normal" or "bonds"
	HonorID          int    `json:"honorId"`          // if normal
	HonorLevel       int    `json:"honorLevel"`       // if normal
	HonorId2         int    `json:"honorId2"`         // if bonds
	BondsHonorWordId int    `json:"bondsHonorWordId"` // if bonds
}

type RawUserGamedata struct {
	UserID int64  `json:"userId"`
	Name   string `json:"name"`
	Deck   int    `json:"deck"`
	Rank   int    `json:"rank"`
}

type RawUserProfile struct {
	ProfileImageType string `json:"profileImageType"`
	Word             string `json:"word"`
	TwitterID        string `json:"twitterId"`
}

type RawUserDeck struct {
	DeckID    int `json:"deckId"`
	Leader    int `json:"leader"`
	SubLeader int `json:"subLeader"`
	Member1   int `json:"member1"`
	Member2   int `json:"member2"`
	Member3   int `json:"member3"`
	Member4   int `json:"member4"`
	Member5   int `json:"member5"`
}

type RawUserCardEpisode struct {
	CardEpisodeID  int    `json:"cardEpisodeId"`
	ScenarioStatus string `json:"scenarioStatus"`
}

type RawUserCard struct {
	CardID                int                  `json:"cardId"`
	Level                 int                  `json:"level"`
	MasterRank            int                  `json:"masterRank"`
	SpecialTrainingStatus string               `json:"specialTrainingStatus"`
	DefaultImage          string               `json:"defaultImage"`
	Episodes              []RawUserCardEpisode `json:"episodes"`
}

type RawMusicResult struct {
	MusicID             int    `json:"musicId"`
	MusicDifficulty     string `json:"musicDifficulty"`
	MusicDifficultyType string `json:"musicDifficultyType"`
	PlayResult          string `json:"playResult"`
	FullComboFlg        bool   `json:"fullComboFlg"`
	FullPerfectFlg      bool   `json:"fullPerfectFlg"`
}

type RawChallengeLiveResult struct {
	CharacterID int `json:"characterId"`
	HighScore   int `json:"highScore"`
}

type RawChallengeLiveStage struct {
	CharacterID int `json:"characterId"`
	Rank        int `json:"rank"`
}

type RawChallengeLiveReward struct {
	ChallengeLiveHighScoreRewardID int `json:"challengeLiveHighScoreRewardId"`
	CharacterID                    int `json:"characterId"`
}

// NewUserDataService loads user.json and prepares derived structures. If path is empty, returns nil.
func NewUserDataService(path string, musicMetaPath string, assetDir string, masterdata *MasterDataService, defaultRegion string) (*UserDataService, error) {
	if strings.TrimSpace(path) == "" {
		return nil, nil
	}
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return nil, err
	}
	var raw RawUserData
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	if raw.UserGamedata.UserID == 0 {
		return nil, fmt.Errorf("userId is missing in user.json")
	}

	activeDeck := findActiveDeck(raw.UserDecks, raw.UserGamedata.Deck)
	leaderCardID := activeDeck.Leader
	leaderCard := findUserCard(raw.UserCards, leaderCardID)
	isAfter := IsAfterTraining(leaderCard)
	leaderPath := resolveCardPortraitPath(assetDir, leaderCardID, isAfter, masterdata)
	if leaderPath == "" {
		fallback := filepath.Join(assetDir, "user", "leader.png")
		if _, err := os.Stat(fallback); err == nil {
			leaderPath = relativeAssetPath(assetDir, fallback)
		}
	}

	profile := &model.DetailedProfileCardRequest{
		ID:              fmt.Sprintf("%d", raw.UserGamedata.UserID),
		Region:          strings.ToUpper(defaultRegion),
		Nickname:        raw.UserGamedata.Name,
		Source:          "suite_dump",
		UpdateTime:      raw.Now,
		Mode:            strings.TrimSpace(raw.UserProfile.ProfileImageType),
		IsHideUID:       true,
		LeaderImagePath: leaderPath,
		HasFrame:        false,
		UserCards:       buildUserCardEntries(activeDeck),
	}

	resultMap := buildMusicResultMap(raw.UserMusicStats)

	var musicMeta []byte
	if strings.TrimSpace(musicMetaPath) != "" {
		if mBytes, err := os.ReadFile(filepath.Clean(musicMetaPath)); err == nil {
			musicMeta = injectOmakaseMusicMeta(mBytes)
		} else {
			fmt.Printf("[WARN] Failed to load music meta from %s: %v\n", musicMetaPath, err)
		}
	}

	return &UserDataService{
		baseProfile: profile,
		musicResult: resultMap,
		challenge: &ChallengeLiveData{
			Results: convertChallengeResults(raw.UserChallengeLiveSoloResults),
			Stages:  convertChallengeStages(raw.UserChallengeLiveSoloStages),
			Rewards: convertChallengeRewards(raw.UserChallengeLiveSoloHighScoreRewards),
		},
		rawData:        &raw,
		rawJSON:        data, // save original bytes - preserves ALL fields including userAreas
		musicMetaBytes: musicMeta,
	}, nil
}

// GetRawData returns the raw user data for deeper inspection
func (s *UserDataService) GetRawData() *RawUserData {
	if s == nil {
		return nil
	}
	return s.rawData
}

// DetailedProfile returns a copy of the profile with region overridden.
func (s *UserDataService) DetailedProfile(region string) *model.DetailedProfileCardRequest {
	if s == nil || s.baseProfile == nil {
		return nil
	}
	profile := *s.baseProfile
	if strings.TrimSpace(region) != "" {
		profile.Region = strings.ToUpper(region)
	}
	return &profile
}

// ProfileCard converts the stored detailed profile to ProfileCardRequest.
func (s *UserDataService) ProfileCard(region string) *model.ProfileCardRequest {
	detail := s.DetailedProfile(region)
	if detail == nil {
		return nil
	}
	source := detail.Source
	mode := detail.Mode
	update := detail.UpdateTime
	return &model.ProfileCardRequest{
		Profile: &model.BasicProfile{
			ID:              detail.ID,
			Region:          detail.Region,
			Nickname:        detail.Nickname,
			IsHideUID:       detail.IsHideUID,
			LeaderImagePath: detail.LeaderImagePath,
			HasFrame:        detail.HasFrame,
			FramePath:       detail.FramePath,
		},
		DataSources: []model.ProfileDataSource{
			{
				Name:       "User Data",
				Source:     &source,
				UpdateTime: &update,
				Mode:       &mode,
			},
		},
	}
}

// MusicResults returns a copy of the result map for a given difficulty (ap/fc/clear/not_clear).
func (s *UserDataService) MusicResults(diff string) map[int]string {
	if s == nil {
		return nil
	}
	diffKey := strings.ToLower(strings.TrimSpace(diff))
	if diffKey == "" {
		return nil
	}
	source := s.musicResult[diffKey]
	copied := make(map[int]string, len(source))
	for k, v := range source {
		copied[k] = v
	}
	return copied
}

// GetMusicResult returns the best play result for (music, diff).
func (s *UserDataService) GetMusicResult(musicID int, diff string) string {
	if s == nil {
		return ""
	}
	diffKey := strings.ToLower(strings.TrimSpace(diff))
	if diffKey == "" {
		return ""
	}
	if result, ok := s.musicResult[diffKey][musicID]; ok {
		return result
	}
	return ""
}

func buildUserCardEntries(deck RawUserDeck) []map[string]interface{} {
	ids := []int{deck.Leader, deck.SubLeader, deck.Member1, deck.Member2, deck.Member3, deck.Member4, deck.Member5}
	seen := make(map[int]struct{})
	var result []map[string]interface{}
	for _, id := range ids {
		if id == 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, map[string]interface{}{"card_id": id})
	}
	return result
}

func findActiveDeck(decks []RawUserDeck, activeID int) RawUserDeck {
	for _, deck := range decks {
		if deck.DeckID == activeID {
			return deck
		}
	}
	if len(decks) > 0 {
		return decks[0]
	}
	return RawUserDeck{}
}

func resolveCardPortraitPath(assetDir string, cardID int, afterTraining bool, masterdata *MasterDataService) string {
	if cardID == 0 || masterdata == nil {
		return ""
	}
	card, err := masterdata.GetCardByID(cardID)
	if err != nil || card == nil {
		return ""
	}
	imageType := "normal"
	if afterTraining {
		imageType = "after_training"
	}
	thumbnail := fmt.Sprintf("%s_%s.png", card.AssetBundleName, imageType)
	candidates := []string{
		filepath.Join(assetDir, "thumbnail", "chara", thumbnail),
		filepath.Join(assetDir, "character", "member", card.AssetBundleName, "card_normal.png"),
	}
	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			return relativeAssetPath(assetDir, path)
		}
	}
	return relativeAssetPath(assetDir, candidates[0]) // fallback 用 chara/
}

func relativeAssetPath(assetDir string, absolutePath string) string {
	normBase := filepath.ToSlash(filepath.Clean(assetDir))
	normPath := filepath.ToSlash(filepath.Clean(absolutePath))
	if normBase == "." || normBase == "" {
		return normPath
	}
	normBase = strings.TrimSuffix(normBase, "/")
	prefix := normBase + "/"
	if strings.HasPrefix(normPath, prefix) {
		return strings.TrimPrefix(normPath, prefix)
	}
	return normPath
}

func buildMusicResultMap(rawResults []RawMusicResult) map[string]map[int]string {
	result := make(map[string]map[int]string)
	for _, item := range rawResults {
		diff := strings.ToLower(strings.TrimSpace(item.MusicDifficultyType))
		if diff == "" {
			diff = strings.ToLower(strings.TrimSpace(item.MusicDifficulty))
		}
		if diff == "" {
			continue
		}
		status := normalizePlayResult(item)
		if _, ok := result[diff]; !ok {
			result[diff] = make(map[int]string)
		}
		prev := result[diff][item.MusicID]
		if prioritizePlayResult(status) >= prioritizePlayResult(prev) {
			result[diff][item.MusicID] = status
		}
	}
	return result
}

func findUserCard(cards []RawUserCard, cardID int) *RawUserCard {
	for i := range cards {
		if cards[i].CardID == cardID {
			return &cards[i]
		}
	}
	return nil
}

func IsAfterTraining(card *RawUserCard) bool {
	if card == nil {
		return false
	}
	// We only care if the card actually has the training status "done".
	// We DO NOT check DefaultImage because the user might have trained the card
	// but intentionally switched the art back to "normal" art.
	return strings.EqualFold(card.SpecialTrainingStatus, "done")
}

func normalizePlayResult(item RawMusicResult) string {
	switch {
	case item.FullPerfectFlg:
		return "ap"
	case item.FullComboFlg:
		return "fc"
	case strings.EqualFold(item.PlayResult, "not_clear") || item.PlayResult == "":
		return "not_clear"
	default:
		return "clear"
	}
}

func prioritizePlayResult(result string) int {
	switch result {
	case "ap":
		return 3
	case "fc":
		return 2
	case "clear":
		return 1
	default:
		return 0
	}
}

// ChallengeLive aggregates challenge-live data for downstream modules.
func (s *UserDataService) ChallengeLive() *ChallengeLiveData {
	if s == nil {
		return nil
	}
	return s.challenge
}

func convertChallengeResults(src []RawChallengeLiveResult) []ChallengeLiveResult {
	out := make([]ChallengeLiveResult, 0, len(src))
	for _, item := range src {
		out = append(out, ChallengeLiveResult{
			CharacterID: item.CharacterID,
			HighScore:   item.HighScore,
		})
	}
	return out
}

func convertChallengeStages(src []RawChallengeLiveStage) []ChallengeLiveStage {
	out := make([]ChallengeLiveStage, 0, len(src))
	for _, item := range src {
		out = append(out, ChallengeLiveStage{
			CharacterID: item.CharacterID,
			Rank:        item.Rank,
		})
	}
	return out
}

func convertChallengeRewards(src []RawChallengeLiveReward) []ChallengeLiveReward {
	out := make([]ChallengeLiveReward, 0, len(src))
	for _, item := range src {
		out = append(out, ChallengeLiveReward{
			RewardID:    item.ChallengeLiveHighScoreRewardID,
			CharacterID: item.CharacterID,
		})
	}
	return out
}

// ChallengeLiveData aggregates challenge-live progress extracted from suite dump.
type ChallengeLiveData struct {
	Results []ChallengeLiveResult
	Stages  []ChallengeLiveStage
	Rewards []ChallengeLiveReward
}

// ChallengeLiveResult contains per-character high score.
type ChallengeLiveResult struct {
	CharacterID int
	HighScore   int
}

// ChallengeLiveStage stores rank progress for a character.
type ChallengeLiveStage struct {
	CharacterID int
	Rank        int
}

// ChallengeLiveReward tracks claimed high-score rewards.
type ChallengeLiveReward struct {
	RewardID    int
	CharacterID int
}

// GetUserSuite returns the base profile card request
func (s *UserDataService) GetUserSuite() *model.DetailedProfileCardRequest {
	if s == nil {
		return nil
	}
	return s.baseProfile
}

// GetUserCards returns the user's cards
func (s *UserDataService) GetUserCards() []map[string]interface{} {
	if s == nil || s.baseProfile == nil {
		return nil
	}
	return s.baseProfile.UserCards
}

// RawBytes returns the original user.json bytes unchanged so the C++ engine
// receives ALL fields (including userAreas, userMysekaiCards, etc.) that are
// not mapped into our Go struct but are required by the engine.
func (s *UserDataService) RawBytes() ([]byte, error) {
	if s == nil {
		return nil, fmt.Errorf("raw user data unavailable")
	}
	if len(s.rawJSON) > 0 {
		return s.rawJSON, nil
	}
	// Fallback: re-serialize from struct (loses unmapped fields)
	if s.rawData == nil {
		return nil, fmt.Errorf("raw user data unavailable")
	}
	return json.Marshal(s.rawData)
}

// MusicMetaBytes returns the raw bytes from music_metas.json.
func (s *UserDataService) MusicMetaBytes() []byte {
	if s == nil || s.musicMetaBytes == nil {
		return nil
	}
	return s.musicMetaBytes
}
