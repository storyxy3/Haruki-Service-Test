//go:build cgo

// deck_local.go — local CGo-based deck recommendation service.
//
// This file is included only when CGO_ENABLED=1 AND the build tag "cgo"
// resolves (which is the default when CGo is available).
//
// When the sekai_deck_recommend_c shared library is present in
// pkg/deck_cgo/lib/<GOOS>_<GOARCH>/, this service runs the full C++ engine
// in-process: zero HTTP overhead, zero Python runtime.
//
// It implements the same Recommend() signature as DeckRecommenderService so
// the controller can swap between them transparently.

package service

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"Haruki-Service-API/pkg/deck_cgo"
)

// LocalDeckRecommender wraps the CGo engine pool.
// Instantiated only when DeckRecommendConfig.UseLocalEngine == true.
type LocalDeckRecommender struct {
	pool        *deck_cgo.Pool
	defaultAlgs []string
	timeout     time.Duration
}

// NewLocalDeckRecommender initialises the CGo engine pool.
//
//   - masterdataDir   — base directory of masterdata JSONs (e.g. D:/github/haruki-sekai-master/master)
//   - musicmetasData  — raw bytes of music_metas.json; may be nil
//   - region          — "jp" | "cn" | etc.
//   - algs            — default algorithms, e.g. ["dfs","sa","ga"]
//   - poolSize        — number of concurrent C++ engine instances
//     (0 = auto, uses runtime.NumCPU())
func NewLocalDeckRecommender(
	masterdataDir string,
	musicmetasData []byte,
	region string,
	algs []string,
	poolSize int,
	timeout time.Duration,
) (*LocalDeckRecommender, error) {
	if poolSize <= 0 {
		poolSize = runtime.NumCPU()
		if poolSize > 4 {
			poolSize = 4 // cap at 4; each instance holds significant memory
		}
	}
	if len(algs) == 0 {
		algs = []string{"dfs", "sa", "ga"}
	}

	if staticDataDir := resolveDeckStaticDataDir(); staticDataDir != "" {
		if err := deck_cgo.SetStaticDataDir(staticDataDir); err != nil {
			return nil, fmt.Errorf("LocalDeckRecommender: set static data dir: %w", err)
		}
	}

	pool, err := deck_cgo.NewPool(
		masterdataDir,
		nil, // no in-memory masterdata map; load from dir
		"",  // no musicmetas file path; inject bytes below
		musicmetasData,
		strings.ToLower(region),
		poolSize,
	)
	if err != nil {
		return nil, fmt.Errorf("LocalDeckRecommender: init pool: %w", err)
	}

	return &LocalDeckRecommender{
		pool:        pool,
		defaultAlgs: algs,
		timeout:     timeout,
	}, nil
}

func resolveDeckStaticDataDir() string {
	if wd, err := os.Getwd(); err == nil {
		candidate := filepath.Join(wd, "data")
		if dirExists(candidate) {
			return candidate
		}
	}

	if exePath, err := os.Executable(); err == nil {
		candidate := filepath.Join(filepath.Dir(exePath), "data")
		if dirExists(candidate) {
			return candidate
		}
	}

	return ""
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// Enabled always returns true for a successfully created LocalDeckRecommender.
func (l *LocalDeckRecommender) Enabled() bool {
	return l != nil && l.pool != nil
}

// Close frees all C++ engine instances.
func (l *LocalDeckRecommender) Close() {
	if l != nil && l.pool != nil {
		l.pool.Close()
	}
}

// ExpandAlgorithms mirrors DeckRecommenderService.ExpandAlgorithms.
func (l *LocalDeckRecommender) ExpandAlgorithms(option map[string]interface{}) []map[string]interface{} {
	if option == nil {
		return nil
	}
	alg, _ := option["algorithm"].(string)
	alg = strings.ToLower(strings.TrimSpace(alg))
	if alg != "all" {
		return []map[string]interface{}{option}
	}
	result := make([]map[string]interface{}, 0, len(l.defaultAlgs))
	for _, a := range l.defaultAlgs {
		copied := make(map[string]interface{}, len(option))
		for k, v := range option {
			copied[k] = v
		}
		copied["algorithm"] = a
		result = append(result, copied)
	}
	return result
}

// Recommend runs the deck recommendation engine locally via CGo.
// It fans out all batch options in parallel (one goroutine per option) and
// aggregates duplicates — identical to the HTTP DeckRecommenderService contract.
func (l *LocalDeckRecommender) Recommend(req DeckRecommendRequest) (*DeckRecommendResult, error) {
	if len(req.BatchOption) == 0 {
		return nil, fmt.Errorf("LocalDeckRecommender: requires batch_options")
	}
	if len(req.UserData) == 0 {
		return nil, fmt.Errorf("LocalDeckRecommender: requires user_data bytes")
	}

	type partial struct {
		alg   string
		decks []DeckRecommendDeck
		cost  float64
		err   error
	}

	results := make(chan partial, len(req.BatchOption))

	for _, option := range req.BatchOption {
		opt := option // capture
		go func() {
			alg, _ := opt["algorithm"].(string)
			start := time.Now()

			cgoOpts := mapOptionsToCgo(opt, req.Region)
			var cgoResult *deck_cgo.Result
			var err error

			poolErr := l.pool.Do(func(eng *deck_cgo.Engine) error {
				cgoResult, err = eng.Recommend(cgoOpts, req.UserData)
				return err
			})
			if poolErr != nil {
				results <- partial{alg: alg, err: poolErr}
				return
			}

			decks := convertCgoDecks(cgoResult.Decks)
			results <- partial{
				alg:   alg,
				decks: decks,
				cost:  time.Since(start).Seconds(),
			}
		}()
	}

	agg := &DeckRecommendResult{
		CostTimes: make(map[string]float64),
		WaitTimes: make(map[string]float64),
	}
	seen := make(map[string]*DeckRecommendDeck)
	var order []string

	for range req.BatchOption {
		p := <-results
		if p.err != nil {
			log.Printf("[WARN] LocalDeckRecommender[%s] failed: %v\n", p.alg, p.err)
			continue
		}
		if p.alg != "" {
			agg.CostTimes[p.alg] = p.cost
			agg.WaitTimes[p.alg] = 0 // no queue wait in local mode
		}
		for _, deck := range p.decks {
			h := deckHash(deck)
			if existing, ok := seen[h]; ok {
				if p.alg != "" {
					existing.Algs = append(existing.Algs, p.alg)
				}
				continue
			}
			deckCopy := deck
			if p.alg != "" {
				deckCopy.Algs = []string{p.alg}
			}
			seen[h] = &deckCopy
			order = append(order, h)
		}
	}

	type pair struct {
		Deck DeckRecommendDeck
		Alg  string
	}
	var pairs []pair

	for _, h := range order {
		deck := seen[h]
		algsMap := make(map[string]struct{})
		for _, a := range deck.Algs {
			algsMap[a] = struct{}{}
		}
		var algs []string
		for k := range algsMap {
			algs = append(algs, k)
		}
		sort.Strings(algs)
		pairs = append(pairs, pair{Deck: *deck, Alg: strings.Join(algs, "+")})
	}

	liveType, _ := req.BatchOption[0]["live_type"].(string)
	target, _ := req.BatchOption[0]["target"].(string)

	sort.SliceStable(pairs, func(i, j int) bool {
		d1 := pairs[i].Deck
		d2 := pairs[j].Deck
		if liveType == "mysekai" {
			if d1.MysekaiEventPoint != d2.MysekaiEventPoint {
				return d1.MysekaiEventPoint > d2.MysekaiEventPoint
			}
			return d1.TotalPower > d2.TotalPower
		} else if target == "power" {
			return d1.TotalPower > d2.TotalPower
		} else if target == "skill" {
			return d1.MultiLiveScoreUp > d2.MultiLiveScoreUp
		} else if target == "bonus" {
			if d1.EventBonusRate != d2.EventBonusRate {
				return d1.EventBonusRate < d2.EventBonusRate
			}
			if d1.Score != d2.Score {
				return d1.Score > d2.Score
			}
			return d1.MultiLiveScoreUp > d2.MultiLiveScoreUp
		}
		// default target == "score"
		if d1.Score != d2.Score {
			return d1.Score > d2.Score
		}
		return d1.MultiLiveScoreUp > d2.MultiLiveScoreUp
	})

	limitFloat, _ := req.BatchOption[0]["limit"].(float64)
	limitIntOpt, ok := req.BatchOption[0]["limit"].(int)
	if !ok {
		limitIntOpt = int(limitFloat)
	}
	if limitIntOpt <= 0 {
		limitIntOpt = len(pairs)
	}
	if limitIntOpt > len(pairs) {
		limitIntOpt = len(pairs)
	}

	for i := 0; i < limitIntOpt; i++ {
		agg.Decks = append(agg.Decks, pairs[i].Deck)
		// Usually concatenate algs if the deck comes from multiple.
		agg.DeckAlgs = append(agg.DeckAlgs, pairs[i].Alg)
	}

	return agg, nil
}

// ─── Conversion helpers ───────────────────────────────────────────────────

// mapOptionsToCgo converts the generic map[string]interface{} batch option
// (as produced by the controller) into a typed deck_cgo.Options struct.
func mapOptionsToCgo(opt map[string]interface{}, region string) deck_cgo.Options {
	get := func(key string) interface{} { return opt[key] }
	str := func(key string) string {
		v, _ := get(key).(string)
		return v
	}
	intPtr := func(key string) *int {
		val := get(key)
		if val == nil {
			return nil
		}
		switch v := val.(type) {
		case int:
			return &v
		case float64:
			n := int(v)
			return &n
		case float32:
			n := int(v)
			return &n
		}
		return nil
	}
	boolPtr := func(key string) *bool {
		v, ok := get(key).(bool)
		if !ok {
			return nil
		}
		return &v
	}

	limitInt := 0
	if l := intPtr("limit"); l != nil {
		limitInt = *l
	}

	o := deck_cgo.Options{
		Region:                       strings.ToLower(strings.TrimSpace(region)),
		Algorithm:                    str("algorithm"),
		Target:                       str("target"),
		LiveType:                     str("live_type"),
		MusicDiff:                    str("music_diff"),
		EventAttr:                    str("event_attr"),
		EventUnit:                    str("event_unit"),
		EventType:                    str("event_type"),
		SkillOrderChooseStrategy:     str("skill_order_choose_strategy"),
		SkillReferenceChooseStrategy: str("skill_reference_choose_strategy"),
		EventID:                      intPtr("event_id"),
		WorldBloomCharacterID:        intPtr("world_bloom_character_id"),
		WorldBloomEventTurn:          intPtr("world_bloom_event_turn"),
		ChallengeLiveCharacterID:     intPtr("challenge_live_character_id"),
		TimeoutMs:                    intPtr("timeout_ms"),
		MultiLiveTeammatePower:       intPtr("multi_live_teammate_power"),
		MultiLiveTeammateScoreUp:     intPtr("multi_live_teammate_score_up"),
		BestSkillAsLeader:            boolPtr("best_skill_as_leader"),
		Limit:                        limitInt,
	}

	parseCardConfig := func(key string) *deck_cgo.CardConfig {
		val, ok := opt[key].(map[string]interface{})
		if !ok {
			return nil
		}
		b := func(k string) bool {
			v, _ := val[k].(bool)
			return v
		}
		return &deck_cgo.CardConfig{
			Disable:     b("disable"),
			LevelMax:    b("level_max"),
			EpisodeRead: b("episode_read"),
			MasterMax:   b("master_max"),
			SkillMax:    b("skill_max"),
			Canvas:      b("canvas"),
		}
	}

	o.Rarity1Config = parseCardConfig("rarity_1_config")
	o.Rarity2Config = parseCardConfig("rarity_2_config")
	o.Rarity3Config = parseCardConfig("rarity_3_config")
	o.Rarity4Config = parseCardConfig("rarity_4_config")
	o.RarityBDConfig = parseCardConfig("rarity_birthday_config")

	if rawCfgs, ok := opt["single_card_configs"].([]interface{}); ok {
		for _, raw := range rawCfgs {
			if m, ok := raw.(map[string]interface{}); ok {
				b := func(k string) bool { v, _ := m[k].(bool); return v }
				idVal, _ := m["card_id"].(float64)
				o.SingleCardCfgs = append(o.SingleCardCfgs, deck_cgo.SingleCardConfig{
					CardID:      int(idVal),
					Disable:     b("disable"),
					LevelMax:    b("level_max"),
					EpisodeRead: b("episode_read"),
					MasterMax:   b("master_max"),
					SkillMax:    b("skill_max"),
					Canvas:      b("canvas"),
				})
			}
		}
	}

	if mid := intPtr("music_id"); mid != nil {
		o.MusicID = *mid
	}
	if limit := intPtr("limit"); limit != nil {
		o.Limit = *limit
	}

	// Fixed cards / characters
	parseArray := func(key string, target *[]int) {
		val := opt[key]
		switch v := val.(type) {
		case []int:
			*target = append(*target, v...)
		case []interface{}:
			for _, item := range v {
				switch num := item.(type) {
				case int:
					*target = append(*target, num)
				case float64:
					*target = append(*target, int(num))
				}
			}
		}
	}

	parseArray("fixed_cards", &o.FixedCards)
	parseArray("fixed_characters", &o.FixedCharacters)

	return o
}

// convertCgoDecks maps CGo result types to the service's shared result types.
func convertCgoDecks(src []deck_cgo.ResultDeck) []DeckRecommendDeck {
	out := make([]DeckRecommendDeck, 0, len(src))
	for _, d := range src {
		cards := make([]DeckRecommendCard, 0, len(d.Cards))
		for _, c := range d.Cards {
			cards = append(cards, DeckRecommendCard{
				CardID:          c.CardID,
				Level:           c.Level,
				MasterRank:      c.MasterRank,
				DefaultImage:    c.DefaultImage,
				SkillLevel:      c.SkillLevel,
				SkillRate:       float64(c.SkillScoreUp),
				EventBonusRate:  c.EventBonusRate,
				IsAfterStory:    c.Episode2Read,
				IsBeforeStory:   c.Episode1Read,
				IsAfterTraining: c.AfterTraining,
				HasCanvasBonus:  c.HasCanvasBonus,
			})
		}
		out = append(out, DeckRecommendDeck{
			Cards:                cards,
			Score:                d.Score,
			LiveScore:            d.LiveScore,
			MysekaiEventPoint:    d.MysekaiEventPoint,
			TotalPower:           d.TotalPower,
			EventBonusRate:       d.EventBonusRate,
			SupportDeckBonusRate: d.SupportDeckBonusRate,
			MultiLiveScoreUp:     d.MultiLiveScoreUp,
		})
	}
	return out
}
