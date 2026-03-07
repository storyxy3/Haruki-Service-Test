package service

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"sync/atomic"
	"time"

	"Haruki-Service-API/internal/config"
)

type DeckRecommender interface {
	Enabled() bool
	ExpandAlgorithms(option map[string]interface{}) []map[string]interface{}
	Recommend(req DeckRecommendRequest) (*DeckRecommendResult, error)
}

type DeckRecommenderService struct {
	enabled     bool
	servers     []string
	defaultAlgs []string
	httpClient  *http.Client
	counter     atomic.Uint64
}

type DeckRecommendRequest struct {
	Region      string                   `json:"region"`
	UserData    []byte                   `json:"user_data"`
	MusicMeta   []byte                   `json:"music_meta"`
	BatchOption []map[string]interface{} `json:"batch_options"`
}

type DeckRecommendResult struct {
	Decks     []DeckRecommendDeck `json:"decks"`
	CostTimes map[string]float64
	WaitTimes map[string]float64
	DeckAlgs  []string
}

type DeckRecommendDeck struct {
	Cards                []DeckRecommendCard `json:"cards"`
	Score                int                 `json:"score"`
	LiveScore            int                 `json:"live_score"`
	MysekaiEventPoint    int                 `json:"mysekai_event_point"`
	TotalPower           int                 `json:"total_power"`
	EventBonusRate       float64             `json:"event_bonus_rate"`
	SupportDeckBonusRate float64             `json:"support_deck_bonus_rate"`
	MultiLiveScoreUp     float64             `json:"multi_live_score_up"`
	ChallengeScoreDelta  int                 `json:"challenge_score_delta"`
	Algs                 []string            `json:"-"`
}

type DeckRecommendCard struct {
	CardID         int     `json:"card_id"`
	Level          int     `json:"level"`
	MasterRank     int     `json:"master_rank"`
	DefaultImage   string  `json:"default_image"`
	SkillLevel     int     `json:"skill_level"`
	SkillRate      float64 `json:"skill_rate"`
	EventBonusRate float64 `json:"event_bonus_rate"`
	IsBeforeStory  bool    `json:"is_before_story"`
	IsAfterStory   bool    `json:"is_after_story"`
	HasCanvasBonus bool    `json:"has_canvas_bonus"`
}

type deckRecommendAPIItem struct {
	Result struct {
		Decks []DeckRecommendDeck `json:"decks"`
	} `json:"result"`
	Alg      string  `json:"alg"`
	CostTime float64 `json:"cost_time"`
	WaitTime float64 `json:"wait_time"`
}

func NewDeckRecommenderService(cfg config.DeckRecommendConfig) *DeckRecommenderService {
	servers := make([]string, 0, len(cfg.Servers))
	for _, s := range cfg.Servers {
		url := strings.TrimSpace(s.URL)
		if url == "" {
			continue
		}
		weight := s.Weight
		if weight <= 0 {
			weight = 1
		}
		for i := 0; i < weight; i++ {
			servers = append(servers, strings.TrimRight(url, "/"))
		}
	}
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	algs := cfg.DefaultAlgs
	if len(algs) == 0 {
		algs = []string{"dfs", "sa", "ga"}
	}
	return &DeckRecommenderService{
		enabled:     cfg.Enabled && len(servers) > 0,
		servers:     servers,
		defaultAlgs: algs,
		httpClient:  &http.Client{Timeout: timeout},
	}
}

func (s *DeckRecommenderService) Enabled() bool {
	return s != nil && s.enabled && len(s.servers) > 0
}

func (s *DeckRecommenderService) ExpandAlgorithms(option map[string]interface{}) []map[string]interface{} {
	if option == nil {
		return nil
	}
	alg, _ := option["algorithm"].(string)
	alg = strings.ToLower(strings.TrimSpace(alg))
	if alg != "all" {
		return []map[string]interface{}{option}
	}
	result := make([]map[string]interface{}, 0, len(s.defaultAlgs))
	for _, a := range s.defaultAlgs {
		copied := make(map[string]interface{}, len(option))
		for k, v := range option {
			copied[k] = v
		}
		copied["algorithm"] = a
		result = append(result, copied)
	}
	return result
}

func (s *DeckRecommenderService) Recommend(req DeckRecommendRequest) (*DeckRecommendResult, error) {
	if !s.Enabled() {
		return nil, fmt.Errorf("deck recommender service is disabled")
	}
	if len(req.UserData) == 0 {
		return nil, fmt.Errorf("deck recommender requires user_data bytes")
	}
	if len(req.BatchOption) == 0 {
		return nil, fmt.Errorf("deck recommender requires batch_options")
	}

	var payloadParts [][]byte
	payloadParts = append(payloadParts, req.UserData)
	if len(req.MusicMeta) > 0 {
		payloadParts = append(payloadParts, req.MusicMeta)
	}

	cachePayload, err := buildDeckBinaryPayload(payloadParts)
	if err != nil {
		return nil, err
	}
	userHashResp := struct {
		UserDataHash string `json:"userdata_hash"`
	}{}
	if err := s.callDeckAPI("/cache_userdata", cachePayload, &userHashResp); err != nil {
		return nil, err
	}
	if strings.TrimSpace(userHashResp.UserDataHash) == "" {
		return nil, fmt.Errorf("deck recommender returned empty userdata_hash")
	}

	data := map[string]interface{}{
		"region":        strings.ToLower(strings.TrimSpace(req.Region)),
		"batch_options": req.BatchOption,
		"userdata_hash": userHashResp.UserDataHash,
	}
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	recommendPayload, err := buildDeckBinaryPayload([][]byte{jsonData})
	if err != nil {
		return nil, err
	}
	var items []deckRecommendAPIItem
	if err := s.callDeckAPI("/recommend", recommendPayload, &items); err != nil {
		return nil, err
	}

	agg := &DeckRecommendResult{
		CostTimes: make(map[string]float64),
		WaitTimes: make(map[string]float64),
	}
	seen := make(map[string]struct{})
	for _, item := range items {
		if strings.TrimSpace(item.Alg) != "" {
			agg.CostTimes[item.Alg] = item.CostTime
			agg.WaitTimes[item.Alg] = item.WaitTime
		}
		for _, deck := range item.Result.Decks {
			hash := deckHash(deck)
			if _, ok := seen[hash]; ok {
				continue
			}
			seen[hash] = struct{}{}
			agg.Decks = append(agg.Decks, deck)
			agg.DeckAlgs = append(agg.DeckAlgs, item.Alg)
		}
	}
	return agg, nil
}

func (s *DeckRecommenderService) callDeckAPI(path string, payload []byte, target interface{}) error {
	lastErr := error(nil)
	start := int(s.counter.Add(1)) % len(s.servers)
	for i := 0; i < len(s.servers); i++ {
		base := s.servers[(start+i)%len(s.servers)]
		url := base + path
		resp, err := s.httpClient.Post(url, "application/octet-stream", bytes.NewReader(payload))
		if err != nil {
			lastErr = err
			continue
		}
		body, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			lastErr = readErr
			continue
		}
		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("deck api %s returned %d: %s", path, resp.StatusCode, string(body))
			continue
		}
		if err := json.Unmarshal(body, target); err != nil {
			lastErr = err
			continue
		}
		return nil
	}
	return fmt.Errorf("deck api call failed after trying %d servers: %w", len(s.servers), lastErr)
}

func buildDeckBinaryPayload(segments [][]byte) ([]byte, error) {
	var raw bytes.Buffer
	for _, seg := range segments {
		if err := binary.Write(&raw, binary.BigEndian, uint32(len(seg))); err != nil {
			return nil, err
		}
		if _, err := raw.Write(seg); err != nil {
			return nil, err
		}
	}
	return compressZstdByPython(raw.Bytes())
}

func compressZstdByPython(input []byte) ([]byte, error) {
	// Deck recommender protocol requires zstd-compressed payload;
	// this environment already provides Python + zstandard module.
	cmd := exec.Command("python", "-c", "import sys,zstandard as z;sys.stdout.buffer.write(z.ZstdCompressor().compress(sys.stdin.buffer.read()))")
	cmd.Stdin = bytes.NewReader(input)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("python zstd compression failed: %w (%s)", err, strings.TrimSpace(stderr.String()))
	}
	return out, nil
}

func deckHash(deck DeckRecommendDeck) string {
	first := 0
	if len(deck.Cards) > 0 {
		first = deck.Cards[0].CardID
	}
	return fmt.Sprintf("%d_%d_%d", deck.Score, deck.TotalPower, first)
}
