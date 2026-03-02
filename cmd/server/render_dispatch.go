package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"Haruki-Service-API/internal/controller"
	"Haruki-Service-API/internal/model"
	"Haruki-Service-API/internal/service"
)

// ParsedCommand mirrors the response from Haruki-Command-Parser's /api/parse.
// When the bot calls /api/render with this payload, we dispatch to the right controller.
type ParsedCommand struct {
	Module string          `json:"module"` // card, music, event, gacha, deck, sk, mysekai, profile, education, score, stamp, misc, help
	Mode   string          `json:"mode"`   // card-detail, card-list, card-box, music-detail, ...
	Region string          `json:"region"` // jp, cn, en, tw, kr
	Query  string          `json:"query"`  // raw query string (after prefix stripping)
	Flags  map[string]bool `json:"flags,omitempty"`
	UserID string          `json:"user_id,omitempty"`
	// Additional structured params for modules that need them
	Params json.RawMessage `json:"params,omitempty"`
}

// renderEnv holds all controllers needed by the render dispatcher.
type renderEnv struct {
	card      *controller.CardController
	music     *controller.MusicController
	gacha     *controller.GachaController
	event     *controller.EventController
	deck      *controller.DeckController
	sk        *controller.SkController
	mysekai   *controller.MysekaiController
	honor     *controller.HonorController
	profile   *controller.ProfileController
	education *controller.EducationController
	stamp     *controller.StampController
	misc      *controller.MiscController
	score     *controller.ScoreController
	userData  *service.UserDataService
}

// handleRenderDispatch is the unified entry point called by the Bot after
// receiving a ParsedCommand from Haruki-Command-Parser.
//
// POST /api/render
// Body: ParsedCommand JSON
// Response: image/png bytes
func handleRenderDispatch(env *renderEnv) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var cmd ParsedCommand
		if err := json.NewDecoder(r.Body).Decode(&cmd); err != nil {
			http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
			return
		}

		if cmd.Region == "" {
			cmd.Region = "jp"
		}
		module := strings.ToLower(strings.TrimSpace(cmd.Module))
		mode := strings.ToLower(strings.TrimSpace(cmd.Mode))

		slog.Info("render dispatch", "module", module, "mode", mode, "region", cmd.Region, "query", cmd.Query)

		var imageData []byte
		var err error

		switch module {
		case "card":
			imageData, err = dispatchCard(env.card, mode, cmd)
		case "music":
			imageData, err = dispatchMusic(env.music, mode, cmd)
		case "gacha":
			imageData, err = dispatchGacha(env.gacha, mode, cmd)
		case "event":
			imageData, err = dispatchEvent(env.event, mode, cmd)
		case "deck":
			imageData, err = dispatchDeck(env.deck, mode, cmd)
		case "education":
			imageData, err = dispatchEducation(env.education, mode, cmd)
		case "score":
			imageData, err = dispatchScore(env.score, mode, cmd)
		case "sk":
			imageData, err = dispatchSK(env.sk, mode, cmd)
		case "mysekai":
			imageData, err = dispatchMysekai(env.mysekai, mode, cmd)
		case "profile":
			imageData, err = dispatchProfile(env.profile, cmd, env.userData)
		case "honor":
			imageData, err = dispatchHonor(env.honor, cmd)
		case "stamp":
			imageData, err = dispatchStamp(env.stamp, cmd)
		case "misc":
			imageData, err = dispatchMisc(env.misc, mode, cmd)
		case "help":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]string{"message": "send /help to see available commands"})
			return
		default:
			http.Error(w, fmt.Sprintf("unknown module: %s", module), http.StatusBadRequest)
			return
		}

		if err != nil {
			slog.Error("render failed", "module", module, "mode", mode, "error", err)
			http.Error(w, fmt.Sprintf("render failed: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "image/png")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(imageData)
	}
}

// --- Card ---

func dispatchCard(ctrl *controller.CardController, mode string, cmd ParsedCommand) ([]byte, error) {
	q := model.CardQuery{
		Query:  cmd.Query,
		Region: cmd.Region,
		UserID: cmd.UserID,
		Mode:   mode,
	}
	switch mode {
	case "card-detail":
		return ctrl.RenderCardDetail(q)
	case "card-list":
		// card-list requires []model.CardQuery; pass single query and let controller handle it
		return ctrl.RenderCardList([]model.CardQuery{q})
	case "card-box":
		return ctrl.RenderCardBox([]model.CardQuery{q})
	default:
		return ctrl.RenderCardDetail(q)
	}
}

// --- Music ---

func dispatchMusic(ctrl *controller.MusicController, mode string, cmd ParsedCommand) ([]byte, error) {
	region := cmd.Region
	query := cmd.Query

	switch mode {
	case "music-detail":
		return ctrl.RenderMusicDetail(model.MusicQuery{Query: query, Region: region})
	case "music-list":
		q := model.MusicListQuery{Region: region, Difficulty: "master"}
		if len(cmd.Params) > 0 {
			_ = json.Unmarshal(cmd.Params, &q)
		}
		if q.Keyword == "" {
			q.Keyword = query
		}
		return ctrl.RenderMusicList(q)
	case "music-progress":
		q := model.MusicProgressQuery{Region: region, Difficulty: "master"}
		if len(cmd.Params) > 0 {
			_ = json.Unmarshal(cmd.Params, &q)
		}
		return ctrl.RenderMusicProgress(q)
	case "music-chart":
		q := model.MusicChartQuery{Query: query, Region: region, Difficulty: "master"}
		if len(cmd.Params) > 0 {
			_ = json.Unmarshal(cmd.Params, &q)
		}
		return ctrl.RenderMusicChart(q)
	case "music-rewards":
		// Bot 层需在 Params 中提供完整的 MusicRewardsDetailQuery（含 combo_rewards / rank_rewards 等）
		// music-rewards 不支持仅靠 query 文本查询，必须由 Bot 组装好奖励数据后传入
		if len(cmd.Params) == 0 {
			return nil, fmt.Errorf("music-rewards: Params is required (must contain MusicRewardsDetailQuery)")
		}
		var q model.MusicRewardsDetailQuery
		if err := json.Unmarshal(cmd.Params, &q); err != nil {
			return nil, fmt.Errorf("music-rewards: invalid params: %w", err)
		}
		if q.Region == "" {
			q.Region = region
		}
		return ctrl.RenderMusicRewardsDetail(q)
	default:
		return ctrl.RenderMusicDetail(model.MusicQuery{Query: query, Region: region})
	}
}

// --- Gacha ---

func dispatchGacha(ctrl *controller.GachaController, mode string, cmd ParsedCommand) ([]byte, error) {
	switch mode {
	case "gacha-list", "gacha":
		q := model.GachaListQuery{Region: cmd.Region, IncludePast: true}
		if len(cmd.Params) > 0 {
			_ = json.Unmarshal(cmd.Params, &q)
		}
		return ctrl.RenderGachaList(q)
	case "gacha-detail":
		q := model.GachaDetailQuery{Region: cmd.Region}
		if len(cmd.Params) > 0 {
			_ = json.Unmarshal(cmd.Params, &q)
		}
		if q.GachaID == 0 && cmd.Query != "" {
			var id int
			if _, err := fmt.Sscanf(cmd.Query, "%d", &id); err == nil {
				q.GachaID = id
			}
		}
		return ctrl.RenderGachaDetail(q)
	default:
		// default: list
		q := model.GachaListQuery{Region: cmd.Region, IncludePast: true}
		if len(cmd.Params) > 0 {
			_ = json.Unmarshal(cmd.Params, &q)
		}
		return ctrl.RenderGachaList(q)
	}
}

// --- Event ---

func dispatchEvent(ctrl *controller.EventController, mode string, cmd ParsedCommand) ([]byte, error) {
	region := cmd.Region
	switch mode {
	case "event-list":
		q := model.EventListQuery{Region: region, IncludePast: true}
		if len(cmd.Params) > 0 {
			_ = json.Unmarshal(cmd.Params, &q)
		}
		return ctrl.RenderEventList(q)
	case "event-detail":
		q := model.EventDetailQuery{Region: region}
		if len(cmd.Params) > 0 {
			_ = json.Unmarshal(cmd.Params, &q)
		}
		if q.EventID == 0 && cmd.Query != "" {
			var id int
			if _, err := fmt.Sscanf(cmd.Query, "%d", &id); err == nil {
				q.EventID = id
			}
		}
		return ctrl.RenderEventDetail(context.Background(), q)
	default:
		q := model.EventDetailQuery{Region: region}
		if len(cmd.Params) > 0 {
			_ = json.Unmarshal(cmd.Params, &q)
		}
		return ctrl.RenderEventDetail(context.Background(), q)
	}
}

// --- Deck ---

func dispatchDeck(ctrl *controller.DeckController, mode string, cmd ParsedCommand) ([]byte, error) {
	switch mode {
	case "deck-event", "deck-challenge", "deck-no-event", "deck-bonus", "deck-mysekai":
		recommendType := strings.TrimPrefix(mode, "deck-")
		q := model.DeckAutoQuery{
			Region:        cmd.Region,
			RecommendType: recommendType,
		}
		if len(cmd.Params) > 0 {
			_ = json.Unmarshal(cmd.Params, &q)
		}
		return ctrl.RenderDeckRecommendAuto(q)
	default:
		q := model.DeckAutoQuery{Region: cmd.Region, RecommendType: "event"}
		if len(cmd.Params) > 0 {
			_ = json.Unmarshal(cmd.Params, &q)
		}
		return ctrl.RenderDeckRecommendAuto(q)
	}
}

// --- Profile ---

func dispatchProfile(ctrl *controller.ProfileController, cmd ParsedCommand, userData *service.UserDataService) ([]byte, error) {
	q := model.ProfileQuery{Region: cmd.Region, UserID: cmd.UserID}
	if len(cmd.Params) > 0 {
		_ = json.Unmarshal(cmd.Params, &q)
	}
	if q.UserID == "" {
		q.UserID = cmd.Query
	}
	return ctrl.RenderProfile(q.UserID, q.Region, userData)
}

// --- Honor ---

func dispatchHonor(ctrl *controller.HonorController, cmd ParsedCommand) ([]byte, error) {
	q := model.HonorQuery{Region: cmd.Region}
	if len(cmd.Params) > 0 {
		_ = json.Unmarshal(cmd.Params, &q)
	}
	req, err := ctrl.BuildHonorRequest(q)
	if err != nil {
		return nil, err
	}
	return ctrl.RenderHonorImage(req)
}

// --- Stamp ---

func dispatchStamp(ctrl *controller.StampController, cmd ParsedCommand) ([]byte, error) {
	q := model.StampListQuery{Region: cmd.Region}
	if len(cmd.Params) > 0 {
		_ = json.Unmarshal(cmd.Params, &q)
	}
	req, err := ctrl.BuildStampListRequest(q)
	if err != nil {
		return nil, err
	}
	return ctrl.RenderStampList(req)
}

// --- Education ---
// Education 模块需要用户数据；Bot 层需在 Params 中附带 user.json 相关字段。
// 各子模式对应 DrawingAPI 的 challenge-live / power-bonus / area-item / bonds / leader-count。

func dispatchEducation(ctrl *controller.EducationController, mode string, cmd ParsedCommand) ([]byte, error) {
	switch mode {
	case "education-challenge", "challenge":
		// Prefer structured params; fall back to region-based auto build
		if len(cmd.Params) > 0 {
			var req model.ChallengeLiveDetailsRequest
			if err := json.Unmarshal(cmd.Params, &req); err == nil {
				return ctrl.RenderChallengeLiveDetail(req)
			}
		}
		return ctrl.RenderChallengeLiveDetailFromUser(cmd.Region)
	case "education-power", "power-bonus":
		var req model.PowerBonusDetailRequest
		if len(cmd.Params) > 0 {
			_ = json.Unmarshal(cmd.Params, &req)
		}
		return ctrl.RenderPowerBonusDetail(req)
	case "education-area", "area-item":
		var req model.AreaItemUpgradeMaterialsRequest
		if len(cmd.Params) > 0 {
			_ = json.Unmarshal(cmd.Params, &req)
		}
		return ctrl.RenderAreaItemMaterials(req)
	case "education-bonds", "bonds":
		var req model.BondsRequest
		if len(cmd.Params) > 0 {
			_ = json.Unmarshal(cmd.Params, &req)
		}
		return ctrl.RenderBonds(req)
	case "education-leader", "leader-count":
		var req model.LeaderCountRequest
		if len(cmd.Params) > 0 {
			_ = json.Unmarshal(cmd.Params, &req)
		}
		return ctrl.RenderLeaderCount(req)
	default:
		// default: challenge-live
		return ctrl.RenderChallengeLiveDetailFromUser(cmd.Region)
	}
}

// --- Score ---
// Score 模块全部为结构化 JSON 输入，由 Bot 层组装后放入 Params。

func dispatchScore(ctrl *controller.ScoreController, mode string, cmd ParsedCommand) ([]byte, error) {
	switch mode {
	case "score-control", "score":
		var req model.ScoreControlRequest
		if len(cmd.Params) > 0 {
			if err := json.Unmarshal(cmd.Params, &req); err != nil {
				return nil, fmt.Errorf("score-control: invalid params: %w", err)
			}
		}
		return ctrl.RenderScoreControl(req)
	case "score-custom-room":
		var req model.CustomRoomScoreRequest
		if len(cmd.Params) > 0 {
			if err := json.Unmarshal(cmd.Params, &req); err != nil {
				return nil, fmt.Errorf("score-custom-room: invalid params: %w", err)
			}
		}
		return ctrl.RenderCustomRoomScore(req)
	case "score-music-meta":
		var req []model.MusicMetaRequest
		if len(cmd.Params) > 0 {
			if err := json.Unmarshal(cmd.Params, &req); err != nil {
				return nil, fmt.Errorf("score-music-meta: invalid params: %w", err)
			}
		}
		return ctrl.RenderMusicMeta(req)
	case "score-music-board":
		var req model.MusicBoardRequest
		if len(cmd.Params) > 0 {
			if err := json.Unmarshal(cmd.Params, &req); err != nil {
				return nil, fmt.Errorf("score-music-board: invalid params: %w", err)
			}
		}
		return ctrl.RenderMusicBoard(req)
	default:
		return nil, fmt.Errorf("unknown score mode: %s", mode)
	}
}

// --- Misc ---

func dispatchMisc(ctrl *controller.MiscController, mode string, cmd ParsedCommand) ([]byte, error) {
	switch mode {
	case "misc-birthday", "chara-birthday":
		var req model.CharaBirthdayRequest
		if len(cmd.Params) > 0 {
			if err := json.Unmarshal(cmd.Params, &req); err != nil {
				return nil, fmt.Errorf("misc-birthday: invalid params: %w", err)
			}
		}
		return ctrl.RenderCharaBirthday(req)
	default:
		return nil, fmt.Errorf("unknown misc mode: %s", mode)
	}
}

// --- SK ---

func dispatchSK(ctrl *controller.SkController, mode string, cmd ParsedCommand) ([]byte, error) {
	var payload map[string]interface{}
	if len(cmd.Params) > 0 {
		if err := json.Unmarshal(cmd.Params, &payload); err != nil {
			return nil, fmt.Errorf("sk: invalid params: %w", err)
		}
	}
	switch mode {
	case "sk-line":
		return ctrl.RenderLine(payload)
	case "sk-query":
		return ctrl.RenderQuery(payload)
	case "sk-check-room":
		return ctrl.RenderCheckRoom(payload)
	case "sk-speed":
		return ctrl.RenderSpeed(payload)
	case "sk-player-trace":
		return ctrl.RenderPlayerTrace(payload)
	case "sk-rank-trace":
		return ctrl.RenderRankTrace(payload)
	case "sk-winrate":
		return ctrl.RenderWinrate(payload)
	default:
		return nil, fmt.Errorf("unknown sk mode: %s", mode)
	}
}

// --- Mysekai ---

func dispatchMysekai(ctrl *controller.MysekaiController, mode string, cmd ParsedCommand) ([]byte, error) {
	var payload interface{}
	if len(cmd.Params) > 0 {
		var raw interface{}
		if err := json.Unmarshal(cmd.Params, &raw); err != nil {
			return nil, fmt.Errorf("mysekai: invalid params: %w", err)
		}
		payload = raw
	}
	switch mode {
	case "mysekai-resource":
		return ctrl.RenderResource(payload)
	case "mysekai-fixture-list":
		return ctrl.RenderFixtureList(payload)
	case "mysekai-fixture-detail":
		return ctrl.RenderFixtureDetail(payload)
	case "mysekai-door-upgrade":
		return ctrl.RenderDoorUpgrade(payload)
	case "mysekai-music-record":
		return ctrl.RenderMusicRecord(payload)
	case "mysekai-talk-list":
		return ctrl.RenderTalkList(payload)
	default:
		return nil, fmt.Errorf("unknown mysekai mode: %s", mode)
	}
}
