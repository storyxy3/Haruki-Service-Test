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
	"runtime"
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

	pool, err := deck_cgo.NewPool(
		masterdataDir,
		nil, // no in-memory masterdata map; load from dir
		"",  // no musicmetas file path; inject bytes below
		musicmetasData,
		region,
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
	seen := make(map[string]struct{})

	for range req.BatchOption {
		p := <-results
		if p.err != nil {
			return nil, fmt.Errorf("LocalDeckRecommender[%s]: %w", p.alg, p.err)
		}
		if p.alg != "" {
			agg.CostTimes[p.alg] = p.cost
			agg.WaitTimes[p.alg] = 0 // no queue wait in local mode
		}
		for _, deck := range p.decks {
			h := deckHash(deck)
			if _, ok := seen[h]; ok {
				continue
			}
			seen[h] = struct{}{}
			agg.Decks = append(agg.Decks, deck)
			agg.DeckAlgs = append(agg.DeckAlgs, p.alg)
		}
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
		v, ok := get(key).(float64) // JSON numbers decode as float64
		if !ok {
			return nil
		}
		n := int(v)
		return &n
	}
	boolPtr := func(key string) *bool {
		v, ok := get(key).(bool)
		if !ok {
			return nil
		}
		return &v
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
	}

	if mid := intPtr("music_id"); mid != nil {
		o.MusicID = *mid
	}
	if limit := intPtr("limit"); limit != nil {
		o.Limit = *limit
	}

	// Fixed cards / characters
	if v, ok := opt["fixed_cards"].([]interface{}); ok {
		for _, c := range v {
			if n, ok := c.(float64); ok {
				o.FixedCards = append(o.FixedCards, int(n))
			}
		}
	}
	if v, ok := opt["fixed_characters"].([]interface{}); ok {
		for _, c := range v {
			if n, ok := c.(float64); ok {
				o.FixedCharacters = append(o.FixedCharacters, int(n))
			}
		}
	}

	return o
}

// convertCgoDecks maps CGo result types to the service's shared result types.
func convertCgoDecks(src []deck_cgo.ResultDeck) []DeckRecommendDeck {
	out := make([]DeckRecommendDeck, 0, len(src))
	for _, d := range src {
		cards := make([]DeckRecommendCard, 0, len(d.Cards))
		for _, c := range d.Cards {
			cards = append(cards, DeckRecommendCard{
				CardID:         c.CardID,
				DefaultImage:   c.DefaultImage,
				SkillLevel:     c.SkillLevel,
				EventBonusRate: c.EventBonusRate,
				IsAfterStory:   c.Episode2Read,
				IsBeforeStory:  !c.Episode1Read,
				HasCanvasBonus: c.HasCanvasBonus,
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
