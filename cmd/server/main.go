package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"Haruki-Service-API/internal/apiutils"
	"Haruki-Service-API/internal/config"
	"Haruki-Service-API/internal/controller"
	"Haruki-Service-API/internal/model"
	"Haruki-Service-API/internal/service"
	"Haruki-Service-API/pkg/asset"

	_ "github.com/lib/pq"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)
	slog.Info("Starting Lunabot Service...")

	cfg, err := config.Load("configs.yaml")
	if err != nil {
		slog.Error("Failed to load config", "error", err)
		os.Exit(1)
	}

	cloudClients, err := apiutils.InitCloudClients(cfg.HarukiCloud, logger)
	if err != nil {
		slog.Error("Failed to initialize Haruki Cloud clients", "error", err)
		os.Exit(1)
	}
	defer cloudClients.Close()
	cloudService := service.NewCloudService(cloudClients.Sekai)

	slog.Info("Initializing services...")
	masterdata := service.NewMasterDataService(cfg.MasterData.Dir, "JP")
	if err := masterdata.LoadAll(); err != nil {
		slog.Error("Failed to load masterdata", "error", err)
		os.Exit(1)
	}
	slog.Info("MasterData loaded successfully")

	assetHelper := asset.NewAssetHelper(cfg.Assets.Dir, cfg.Assets.LegacyDirs)

	var userData *service.UserDataService
	if cfg.UserData.Path != "" {
		data, err := service.NewUserDataService(cfg.UserData.Path, assetHelper.Primary(), masterdata, masterdata.GetRegion())
		if err != nil {
			slog.Warn("Failed to load user data", "path", cfg.UserData.Path, "error", err)
		} else {
			userData = data
		}
	}

	drawing := service.NewDrawingService(
		cfg.DrawingAPI.BaseURL,
		cfg.DrawingAPI.Timeout,
		cfg.DrawingAPI.RetryCount,
		assetHelper.Roots(),
	)
	deckRecommender := service.NewDeckRecommenderService(cfg.DeckRecommend)

	nicknames := masterdata.GetNicknames()
	cardParser := service.NewCardParser(nicknames)
	cloudRegion := strings.TrimSpace(cfg.HarukiCloud.Region)
	if cloudRegion == "" {
		cloudRegion = masterdata.GetRegion()
	}
	masterCardSource := service.NewMasterDataCardSource(masterdata)
	var cardSource service.CardDataSource
	if cloudClients.Sekai != nil {
		cardSource = service.NewCloudCardSource(cloudClients.Sekai, cloudRegion)
	}
	if cardSource == nil && cfg.HarukiCloud.UseLocalCardSrc {
		cardSource = masterCardSource
	}
	if cardSource == nil {
		slog.Error("No card data source available; please enable local source or configure Sekai DB")
		os.Exit(1)
	}

	secondaryRegion := strings.TrimSpace(cfg.HarukiCloud.SecondaryRegion)
	if secondaryRegion == "" {
		secondaryRegion = "jp"
	}
	var secondaryCardSource service.CardDataSource
	if cloudClients.Sekai != nil && secondaryRegion != "" {
		secondaryCardSource = service.NewCloudCardSource(cloudClients.Sekai, secondaryRegion)
	}
	if secondaryCardSource == nil {
		secondaryCardSource = masterCardSource
	}

	cardSearchService := service.NewCardSearchService(cardSource, cardParser)
	var eventSource service.EventDataSource
	if cloudClients.Sekai != nil {
		eventSource = service.NewCloudEventSource(cloudClients.Sekai, cloudRegion)
	}
	if eventSource == nil && cfg.HarukiCloud.UseLocalEventSrc {
		eventSource = service.NewMasterDataEventSource(masterdata)
	}
	if eventSource == nil {
		slog.Error("No event data source available; please enable local source or configure Sekai DB")
		os.Exit(1)
	}
	var secondaryEventSource service.EventDataSource
	if cloudClients.Sekai != nil && secondaryRegion != "" && !strings.EqualFold(secondaryRegion, cloudRegion) {
		secondaryEventSource = service.NewCloudEventSource(cloudClients.Sekai, secondaryRegion)
	}

	cardController := controller.NewCardController(cardSource, secondaryCardSource, eventSource, masterdata, drawing, cardSearchService, cfg.DrawingAPI.BaseURL, assetHelper, userData)
	if secondaryEventSource != nil {
		cardController.RegisterEventSource(secondaryEventSource)
	}
	musicSource := service.MusicDataSource(service.NewMasterDataMusicSource(masterdata))
	if cloudClients.Sekai != nil {
		if cloudMusicSource := service.NewCloudMusicSource(cloudClients.Sekai, cloudRegion); cloudMusicSource != nil {
			musicSource = cloudMusicSource
		}
	}
	musicController := controller.NewMusicController(musicSource, drawing, cfg.DrawingAPI.BaseURL, assetHelper, userData)
	if cloudClients.Sekai != nil {
		for _, region := range []string{"jp", "en", "tw", "kr", "cn", secondaryRegion} {
			normalized := strings.ToLower(strings.TrimSpace(region))
			if normalized == "" || strings.EqualFold(normalized, cloudRegion) {
				continue
			}
			if source := service.NewCloudMusicSource(cloudClients.Sekai, normalized); source != nil {
				musicController.RegisterSource(source)
			}
		}
	}
	masterGachaSource := service.NewMasterDataGachaSource(masterdata)
	var gachaSource service.GachaDataSource
	if cloudClients.Sekai != nil {
		gachaSource = service.NewCloudGachaSource(cloudClients.Sekai, cloudRegion)
	}
	if gachaSource == nil {
		gachaSource = masterGachaSource
	}
	gachaController := controller.NewGachaController(gachaSource, drawing, cfg.DrawingAPI.BaseURL, assetHelper)
	if cloudClients.Sekai != nil && secondaryRegion != "" && !strings.EqualFold(secondaryRegion, cloudRegion) {
		if secondaryGachaSource := service.NewCloudGachaSource(cloudClients.Sekai, secondaryRegion); secondaryGachaSource != nil {
			gachaController.RegisterSource(secondaryGachaSource)
		}
	}
	masterHonorSource := service.NewMasterDataHonorSource(masterdata)
	var honorSource service.HonorDataSource
	if cloudClients.Sekai != nil {
		honorSource = service.NewCloudHonorSource(cloudClients.Sekai, cloudRegion)
	}
	if honorSource == nil {
		honorSource = masterHonorSource
	}
	honorController := controller.NewHonorController(honorSource, drawing, assetHelper)
	if cloudClients.Sekai != nil && secondaryRegion != "" && !strings.EqualFold(secondaryRegion, cloudRegion) {
		if secondaryHonorSource := service.NewCloudHonorSource(cloudClients.Sekai, secondaryRegion); secondaryHonorSource != nil {
			honorController.RegisterSource(secondaryHonorSource)
		}
	}
	eventController := controller.NewEventController(eventSource, drawing, cfg.DrawingAPI.BaseURL, assetHelper, cloudService)
	if secondaryEventSource != nil {
		eventController.RegisterSource(secondaryEventSource)
	}
	masterProfileSource := service.NewMasterDataProfileSource(masterdata)
	var profileSource service.ProfileDataSource
	if cloudClients.Sekai != nil {
		profileSource = service.NewCloudProfileSource(cloudClients.Sekai, cloudRegion)
	}
	if profileSource == nil {
		profileSource = masterProfileSource
	}
	profileController := controller.NewProfileController(profileSource, drawing, assetHelper)
	if cloudClients.Sekai != nil && secondaryRegion != "" && !strings.EqualFold(secondaryRegion, cloudRegion) {
		if secondaryProfileSource := service.NewCloudProfileSource(cloudClients.Sekai, secondaryRegion); secondaryProfileSource != nil {
			profileController.RegisterSource(secondaryProfileSource)
		}
	}
	masterEducationSource := service.NewMasterDataEducationSource(masterdata)
	var educationSource service.EducationDataSource
	if cloudClients.Sekai != nil {
		educationSource = service.NewCloudEducationSource(cloudClients.Sekai, cloudRegion)
	}
	if educationSource == nil {
		educationSource = masterEducationSource
	}
	educationController := controller.NewEducationController(educationSource, drawing, assetHelper, userData)
	if cloudClients.Sekai != nil && secondaryRegion != "" && !strings.EqualFold(secondaryRegion, cloudRegion) {
		if secondaryEducationSource := service.NewCloudEducationSource(cloudClients.Sekai, secondaryRegion); secondaryEducationSource != nil {
			educationController.RegisterSource(secondaryEducationSource)
		}
	}
	masterStampSource := service.NewMasterDataStampSource(masterdata)
	var stampSource service.StampDataSource
	if cloudClients.Sekai != nil {
		stampSource = service.NewCloudStampSource(cloudClients.Sekai, cloudRegion)
	}
	if stampSource == nil {
		stampSource = masterStampSource
	}
	stampController := controller.NewStampController(stampSource, drawing, assetHelper)
	if cloudClients.Sekai != nil && secondaryRegion != "" && !strings.EqualFold(secondaryRegion, cloudRegion) {
		if secondaryStampSource := service.NewCloudStampSource(cloudClients.Sekai, secondaryRegion); secondaryStampSource != nil {
			stampController.RegisterSource(secondaryStampSource)
		}
	}
	miscController := controller.NewMiscController(drawing)
	scoreController := controller.NewScoreController(drawing)
	deckController := controller.NewDeckController(drawing, cardSource, eventSource, assetHelper, userData, deckRecommender)
	skController := controller.NewSkController(drawing)
	mysekaiController := controller.NewMysekaiController(drawing)

	mux := http.NewServeMux()

	mux.HandleFunc("/health", handleHealth)

	mux.HandleFunc("/api/card/detail/build", handleCardDetailBuild(cardController))
	mux.HandleFunc("/api/card/detail/render", handleCardDetailRender(cardController))
	mux.HandleFunc("/api/card/list/build", handleCardListBuild(cardController))
	mux.HandleFunc("/api/card/list/render", handleCardListRender(cardController))
	mux.HandleFunc("/api/card/box/build", handleCardBoxBuild(cardController))
	mux.HandleFunc("/api/card/box/render", handleCardBoxRender(cardController))

	mux.HandleFunc("/api/music/detail/build", handleMusicDetailBuild(musicController))
	mux.HandleFunc("/api/music/detail/render", handleMusicDetailRender(musicController))
	mux.HandleFunc("/api/music/brief-list/build", handleMusicBriefListBuild(musicController))
	mux.HandleFunc("/api/music/brief-list/render", handleMusicBriefListRender(musicController))
	mux.HandleFunc("/api/music/list/build", handleMusicListBuild(musicController))
	mux.HandleFunc("/api/music/list/render", handleMusicListRender(musicController))
	mux.HandleFunc("/api/music/progress/build", handleMusicProgressBuild(musicController))
	mux.HandleFunc("/api/music/progress/render", handleMusicProgressRender(musicController))
	mux.HandleFunc("/api/music/chart/build", handleMusicChartBuild(musicController))
	mux.HandleFunc("/api/music/chart/render", handleMusicChartRender(musicController))
	mux.HandleFunc("/api/music/rewards/detail/build", handleMusicRewardsDetailBuild(musicController))
	mux.HandleFunc("/api/music/rewards/detail/render", handleMusicRewardsDetailRender(musicController))
	mux.HandleFunc("/api/music/rewards/basic/build", handleMusicRewardsBasicBuild(musicController))
	mux.HandleFunc("/api/music/rewards/basic/render", handleMusicRewardsBasicRender(musicController))
	mux.HandleFunc("/api/gacha/list/build", handleGachaListBuild(gachaController))
	mux.HandleFunc("/api/gacha/list/render", handleGachaListRender(gachaController))
	mux.HandleFunc("/api/gacha/detail/build", handleGachaDetailBuild(gachaController))
	mux.HandleFunc("/api/gacha/detail/render", handleGachaDetailRender(gachaController))
	mux.HandleFunc("/api/event/detail/build", handleEventDetailBuild(eventController))
	mux.HandleFunc("/api/event/detail/render", handleEventDetailRender(eventController))
	mux.HandleFunc("/api/event/list/build", handleEventListBuild(eventController))
	mux.HandleFunc("/api/event/list/render", handleEventListRender(eventController))
	mux.HandleFunc("/api/event/record/build", handleEventRecordBuild(eventController))
	mux.HandleFunc("/api/event/record/render", handleEventRecordRender(eventController))
	mux.HandleFunc("/api/education/challenge/build", handleEducationChallengeBuild(educationController))
	mux.HandleFunc("/api/education/challenge/render", handleEducationChallengeRender(educationController))
	mux.HandleFunc("/api/education/challenge-live/build", handleEducationChallengeBuild(educationController))
	mux.HandleFunc("/api/education/challenge-live/render", handleEducationChallengeRender(educationController))
	mux.HandleFunc("/api/education/power/build", handleEducationPowerBuild(educationController))
	mux.HandleFunc("/api/education/power/render", handleEducationPowerRender(educationController))
	mux.HandleFunc("/api/education/power-bonus/build", handleEducationPowerBuild(educationController))
	mux.HandleFunc("/api/education/power-bonus/render", handleEducationPowerRender(educationController))
	mux.HandleFunc("/api/education/area/build", handleEducationAreaBuild(educationController))
	mux.HandleFunc("/api/education/area/render", handleEducationAreaRender(educationController))
	mux.HandleFunc("/api/education/area-item/build", handleEducationAreaBuild(educationController))
	mux.HandleFunc("/api/education/area-item/render", handleEducationAreaRender(educationController))
	mux.HandleFunc("/api/education/bonds/build", handleEducationBondsBuild(educationController))
	mux.HandleFunc("/api/education/bonds/render", handleEducationBondsRender(educationController))
	mux.HandleFunc("/api/education/leader/build", handleEducationLeaderBuild(educationController))
	mux.HandleFunc("/api/education/leader/render", handleEducationLeaderRender(educationController))
	mux.HandleFunc("/api/education/leader-count/build", handleEducationLeaderBuild(educationController))
	mux.HandleFunc("/api/education/leader-count/render", handleEducationLeaderRender(educationController))
	mux.HandleFunc("/api/stamp/list/build", handleStampListBuild(stampController))
	mux.HandleFunc("/api/stamp/list/render", handleStampListRender(stampController))
	mux.HandleFunc("/api/misc/chara-birthday/build", handleMiscCharaBirthdayBuild(miscController))
	mux.HandleFunc("/api/misc/chara-birthday/render", handleMiscCharaBirthdayRender(miscController))
	mux.HandleFunc("/api/score/control/build", handleScoreControlBuild(scoreController))
	mux.HandleFunc("/api/score/control/render", handleScoreControlRender(scoreController))
	mux.HandleFunc("/api/score/custom-room/build", handleScoreCustomRoomBuild(scoreController))
	mux.HandleFunc("/api/score/custom-room/render", handleScoreCustomRoomRender(scoreController))
	mux.HandleFunc("/api/score/music-meta/build", handleScoreMusicMetaBuild(scoreController))
	mux.HandleFunc("/api/score/music-meta/render", handleScoreMusicMetaRender(scoreController))
	mux.HandleFunc("/api/score/music-board/build", handleScoreMusicBoardBuild(scoreController))
	mux.HandleFunc("/api/score/music-board/render", handleScoreMusicBoardRender(scoreController))
	mux.HandleFunc("/api/deck/recommend/build", handleDeckRecommendBuild(deckController))
	mux.HandleFunc("/api/deck/recommend/render", handleDeckRecommendRender(deckController))
	mux.HandleFunc("/api/deck/recommend/auto/build", handleDeckRecommendAutoBuild(deckController))
	mux.HandleFunc("/api/deck/recommend/auto/render", handleDeckRecommendAutoRender(deckController))
	mux.HandleFunc("/api/sk/line/build", handleSKBuild(skController))
	mux.HandleFunc("/api/sk/line/render", handleSKLineRender(skController))
	mux.HandleFunc("/api/sk/query/build", handleSKBuild(skController))
	mux.HandleFunc("/api/sk/query/render", handleSKQueryRender(skController))
	mux.HandleFunc("/api/sk/check-room/build", handleSKBuild(skController))
	mux.HandleFunc("/api/sk/check-room/render", handleSKCheckRoomRender(skController))
	mux.HandleFunc("/api/sk/speed/build", handleSKBuild(skController))
	mux.HandleFunc("/api/sk/speed/render", handleSKSpeedRender(skController))
	mux.HandleFunc("/api/sk/player-trace/build", handleSKBuild(skController))
	mux.HandleFunc("/api/sk/player-trace/render", handleSKPlayerTraceRender(skController))
	mux.HandleFunc("/api/sk/rank-trace/build", handleSKBuild(skController))
	mux.HandleFunc("/api/sk/rank-trace/render", handleSKRankTraceRender(skController))
	mux.HandleFunc("/api/sk/winrate/build", handleSKBuild(skController))
	mux.HandleFunc("/api/sk/winrate/render", handleSKWinrateRender(skController))
	mux.HandleFunc("/api/mysekai/resource/build", handleMysekaiBuild(mysekaiController))
	mux.HandleFunc("/api/mysekai/resource/render", handleMysekaiResourceRender(mysekaiController))
	mux.HandleFunc("/api/mysekai/fixture-list/build", handleMysekaiBuild(mysekaiController))
	mux.HandleFunc("/api/mysekai/fixture-list/render", handleMysekaiFixtureListRender(mysekaiController))
	mux.HandleFunc("/api/mysekai/fixture-detail/build", handleMysekaiBuild(mysekaiController))
	mux.HandleFunc("/api/mysekai/fixture-detail/render", handleMysekaiFixtureDetailRender(mysekaiController))
	mux.HandleFunc("/api/mysekai/door-upgrade/build", handleMysekaiBuild(mysekaiController))
	mux.HandleFunc("/api/mysekai/door-upgrade/render", handleMysekaiDoorUpgradeRender(mysekaiController))
	mux.HandleFunc("/api/mysekai/music-record/build", handleMysekaiBuild(mysekaiController))
	mux.HandleFunc("/api/mysekai/music-record/render", handleMysekaiMusicRecordRender(mysekaiController))
	mux.HandleFunc("/api/mysekai/talk-list/build", handleMysekaiBuild(mysekaiController))
	mux.HandleFunc("/api/mysekai/talk-list/render", handleMysekaiTalkListRender(mysekaiController))

	mux.HandleFunc("/api/honor/build", handleHonorBuild(honorController))
	mux.HandleFunc("/api/honor/render", handleHonorRender(honorController))

	mux.HandleFunc("/api/profile/build", handleProfileBuild(profileController, userData))
	mux.HandleFunc("/api/profile/render", handleProfileRender(profileController, userData))

	// ── Unified render dispatch (for Haruki-Command-Parser integration) ──────────
	// POST /api/render  accepts ParsedCommand JSON from Part1 and routes to the
	// appropriate controller. This is the main integration point in the dev branch.
	renderDispatchEnv := &renderEnv{
		card:      cardController,
		music:     musicController,
		gacha:     gachaController,
		event:     eventController,
		deck:      deckController,
		sk:        skController,
		mysekai:   mysekaiController,
		honor:     honorController,
		profile:   profileController,
		education: educationController,
		stamp:     stampController,
		misc:      miscController,
		score:     scoreController,
		userData:  userData,
	}
	mux.HandleFunc("/api/render", handleRenderDispatch(renderDispatchEnv))
	// ─────────────────────────────────────────────────────────────────────────────

	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	slog.Info("Server starting", "address", addr)

	if err := http.ListenAndServe(addr, loggingMiddleware(mux)); err != nil {
		slog.Error("Server failed", "error", err)
		os.Exit(1)
	}
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "ok",
		"service": "lunabot-service",
	})
}

func handleCardDetailBuild(ctrl *controller.CardController) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var query model.CardQuery
		if err := json.NewDecoder(r.Body).Decode(&query); err != nil {
			http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
			return
		}

		drawingReq, err := ctrl.BuildCardDetailRequest(query)
		if err != nil {
			slog.Error("Failed to build card request", "error", err, "query", query.Query)
			http.Error(w, fmt.Sprintf("Failed to build request: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(drawingReq)
	}
}

func handleCardDetailRender(ctrl *controller.CardController) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var query model.CardQuery
		if err := json.NewDecoder(r.Body).Decode(&query); err != nil {
			http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
			return
		}

		imageData, err := ctrl.RenderCardDetail(query)
		if err != nil {
			slog.Error("Failed to render card detail", "error", err, "query", query.Query)
			http.Error(w, fmt.Sprintf("Failed to render image: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "image/png")
		w.Write(imageData)
	}
}

func handleCardListBuild(ctrl *controller.CardController) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			CardIDs []int  `json:"card_ids"`
			Region  string `json:"region"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
			return
		}

		if req.Region == "" {
			req.Region = "jp"
		}

		payload, err := ctrl.BuildCardListRequestFromIDs(req.CardIDs, req.Region)
		if err != nil {
			slog.Error("Failed to build card list", "error", err, "ids", req.CardIDs)
			http.Error(w, fmt.Sprintf("Failed to build request: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(payload)
	}
}

func handleCardListRender(ctrl *controller.CardController) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			CardIDs []int  `json:"card_ids"`
			Region  string `json:"region"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
			return
		}

		if req.Region == "" {
			req.Region = "jp"
		}

		imageData, err := ctrl.RenderCardListFromIDs(req.CardIDs, req.Region)
		if err != nil {
			slog.Error("Failed to render card list", "error", err, "ids", req.CardIDs)
			http.Error(w, fmt.Sprintf("Failed to render image: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "image/png")
		w.Write(imageData)
	}
}

func handleCardBoxBuild(ctrl *controller.CardController) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var queries []model.CardQuery
		if err := json.NewDecoder(r.Body).Decode(&queries); err != nil {
			http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
			return
		}

		payload, err := ctrl.BuildCardBoxRequest(queries)
		if err != nil {
			slog.Error("Failed to build card box", "error", err)
			http.Error(w, fmt.Sprintf("Failed to build request: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(payload)
	}
}

func handleCardBoxRender(ctrl *controller.CardController) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var queries []model.CardQuery
		if err := json.NewDecoder(r.Body).Decode(&queries); err != nil {
			http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
			return
		}

		imageData, err := ctrl.RenderCardBox(queries)
		if err != nil {
			slog.Error("Failed to render card box", "error", err)
			http.Error(w, fmt.Sprintf("Failed to render image: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "image/png")
		w.Write(imageData)
	}
}

func handleMusicDetailBuild(ctrl *controller.MusicController) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var query model.MusicQuery
		if err := json.NewDecoder(r.Body).Decode(&query); err != nil {
			http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
			return
		}

		drawingReq, err := ctrl.BuildMusicDetail(query)
		if err != nil {
			slog.Error("Failed to build music request", "error", err, "query", query.Query)
			http.Error(w, fmt.Sprintf("Failed to build request: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(drawingReq)
	}
}

func handleMusicDetailRender(ctrl *controller.MusicController) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var query model.MusicQuery
		if err := json.NewDecoder(r.Body).Decode(&query); err != nil {
			http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
			return
		}

		imageData, err := ctrl.RenderMusicDetail(query)
		if err != nil {
			slog.Error("Failed to render music detail", "error", err, "query", query.Query)
			http.Error(w, fmt.Sprintf("Failed to render image: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "image/png")
		w.Write(imageData)
	}
}

func handleMusicBriefListBuild(ctrl *controller.MusicController) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			MusicIDs   []int  `json:"music_ids"`
			Difficulty string `json:"difficulty"`
			Region     string `json:"region"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
			return
		}

		drawingReq, err := ctrl.BuildMusicBriefListRequest(req.MusicIDs, req.Difficulty, req.Region)
		if err != nil {
			slog.Error("Failed to build music brief list request", "error", err)
			http.Error(w, fmt.Sprintf("Failed to build request: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(drawingReq)
	}
}

func handleMusicBriefListRender(ctrl *controller.MusicController) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			MusicIDs   []int  `json:"music_ids"`
			Difficulty string `json:"difficulty"`
			Region     string `json:"region"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
			return
		}

		imageData, err := ctrl.RenderMusicBriefList(req.MusicIDs, req.Difficulty, req.Region)
		if err != nil {
			slog.Error("Failed to render music brief list", "error", err)
			http.Error(w, fmt.Sprintf("Failed to render image: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "image/png")
		w.Write(imageData)
	}
}

func handleMusicListBuild(ctrl *controller.MusicController) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var query model.MusicListQuery
		if err := json.NewDecoder(r.Body).Decode(&query); err != nil {
			http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
			return
		}

		req, err := ctrl.BuildMusicList(query)
		if err != nil {
			slog.Error("Failed to build music list request", "error", err)
			http.Error(w, fmt.Sprintf("Failed to build request: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(req)
	}
}

func handleMusicListRender(ctrl *controller.MusicController) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var query model.MusicListQuery
		if err := json.NewDecoder(r.Body).Decode(&query); err != nil {
			http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
			return
		}

		imageData, err := ctrl.RenderMusicList(query)
		if err != nil {
			slog.Error("Failed to render music list", "error", err)
			http.Error(w, fmt.Sprintf("Failed to render image: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "image/png")
		w.Write(imageData)
	}
}

func handleMusicProgressBuild(ctrl *controller.MusicController) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var query model.MusicProgressQuery
		if err := json.NewDecoder(r.Body).Decode(&query); err != nil {
			http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
			return
		}

		req, err := ctrl.BuildMusicProgress(query)
		if err != nil {
			slog.Error("Failed to build music progress request", "error", err)
			http.Error(w, fmt.Sprintf("Failed to build request: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(req)
	}
}

func handleMusicProgressRender(ctrl *controller.MusicController) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var query model.MusicProgressQuery
		if err := json.NewDecoder(r.Body).Decode(&query); err != nil {
			http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
			return
		}

		imageData, err := ctrl.RenderMusicProgress(query)
		if err != nil {
			slog.Error("Failed to render music progress", "error", err)
			http.Error(w, fmt.Sprintf("Failed to render image: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "image/png")
		w.Write(imageData)
	}
}

func handleMusicChartBuild(ctrl *controller.MusicController) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var query model.MusicChartQuery
		if err := json.NewDecoder(r.Body).Decode(&query); err != nil {
			http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
			return
		}

		req, err := ctrl.BuildMusicChart(query)
		if err != nil {
			slog.Error("Failed to build music chart request", "error", err, "query", query.Query)
			http.Error(w, fmt.Sprintf("Failed to build request: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(req)
	}
}

func handleMusicChartRender(ctrl *controller.MusicController) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var query model.MusicChartQuery
		if err := json.NewDecoder(r.Body).Decode(&query); err != nil {
			http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
			return
		}

		imageData, err := ctrl.RenderMusicChart(query)
		if err != nil {
			slog.Error("Failed to render music chart", "error", err, "query", query.Query)
			http.Error(w, fmt.Sprintf("Failed to render image: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "image/png")
		w.Write(imageData)
	}
}

func handleMusicRewardsDetailBuild(ctrl *controller.MusicController) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var query model.MusicRewardsDetailQuery
		if err := json.NewDecoder(r.Body).Decode(&query); err != nil {
			http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
			return
		}

		req, err := ctrl.BuildMusicRewardsDetail(query)
		if err != nil {
			slog.Error("Failed to build music rewards detail request", "error", err)
			http.Error(w, fmt.Sprintf("Failed to build request: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(req)
	}
}

func handleMusicRewardsDetailRender(ctrl *controller.MusicController) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var query model.MusicRewardsDetailQuery
		if err := json.NewDecoder(r.Body).Decode(&query); err != nil {
			http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
			return
		}

		imageData, err := ctrl.RenderMusicRewardsDetail(query)
		if err != nil {
			slog.Error("Failed to render music rewards detail", "error", err)
			http.Error(w, fmt.Sprintf("Failed to render image: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "image/png")
		w.Write(imageData)
	}
}

func handleMusicRewardsBasicBuild(ctrl *controller.MusicController) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var query model.MusicRewardsBasicQuery
		if err := json.NewDecoder(r.Body).Decode(&query); err != nil {
			http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
			return
		}

		req, err := ctrl.BuildMusicRewardsBasic(query)
		if err != nil {
			slog.Error("Failed to build music rewards basic request", "error", err)
			http.Error(w, fmt.Sprintf("Failed to build request: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(req)
	}
}

func handleMusicRewardsBasicRender(ctrl *controller.MusicController) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var query model.MusicRewardsBasicQuery
		if err := json.NewDecoder(r.Body).Decode(&query); err != nil {
			http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
			return
		}

		imageData, err := ctrl.RenderMusicRewardsBasic(query)
		if err != nil {
			slog.Error("Failed to render music rewards basic", "error", err)
			http.Error(w, fmt.Sprintf("Failed to render image: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "image/png")
		w.Write(imageData)
	}
}

func handleGachaListBuild(ctrl *controller.GachaController) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var query model.GachaListQuery
		if err := json.NewDecoder(r.Body).Decode(&query); err != nil {
			http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
			return
		}

		req, err := ctrl.BuildGachaList(query)
		if err != nil {
			slog.Error("Failed to build gacha list request", "error", err)
			http.Error(w, fmt.Sprintf("Failed to build request: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(req)
	}
}

func handleGachaListRender(ctrl *controller.GachaController) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var query model.GachaListQuery
		if err := json.NewDecoder(r.Body).Decode(&query); err != nil {
			http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
			return
		}

		imageData, err := ctrl.RenderGachaList(query)
		if err != nil {
			slog.Error("Failed to render gacha list", "error", err)
			http.Error(w, fmt.Sprintf("Failed to render image: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "image/png")
		w.Write(imageData)
	}
}

func handleGachaDetailBuild(ctrl *controller.GachaController) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var query model.GachaDetailQuery
		if err := json.NewDecoder(r.Body).Decode(&query); err != nil {
			http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
			return
		}

		req, err := ctrl.BuildGachaDetail(query)
		if err != nil {
			slog.Error("Failed to build gacha detail request", "error", err)
			http.Error(w, fmt.Sprintf("Failed to build request: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(req)
	}
}

func handleGachaDetailRender(ctrl *controller.GachaController) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var query model.GachaDetailQuery
		if err := json.NewDecoder(r.Body).Decode(&query); err != nil {
			http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
			return
		}

		imageData, err := ctrl.RenderGachaDetail(query)
		if err != nil {
			slog.Error("Failed to render gacha detail", "error", err)
			http.Error(w, fmt.Sprintf("Failed to render image: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "image/png")
		w.Write(imageData)
	}
}

func handleEventDetailBuild(ctrl *controller.EventController) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var query model.EventDetailQuery
		if err := json.NewDecoder(r.Body).Decode(&query); err != nil {
			http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
			return
		}

		req, err := ctrl.BuildEventDetail(r.Context(), query)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to build request: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(req)
	}
}

func handleEventDetailRender(ctrl *controller.EventController) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var query model.EventDetailQuery
		if err := json.NewDecoder(r.Body).Decode(&query); err != nil {
			http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
			return
		}

		imageData, err := ctrl.RenderEventDetail(r.Context(), query)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to render image: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "image/png")
		w.Write(imageData)
	}
}

func handleEventListBuild(ctrl *controller.EventController) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var query model.EventListQuery
		if err := json.NewDecoder(r.Body).Decode(&query); err != nil {
			http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
			return
		}

		req, err := ctrl.BuildEventList(query)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to build request: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(req)
	}
}

func handleEventListRender(ctrl *controller.EventController) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var query model.EventListQuery
		if err := json.NewDecoder(r.Body).Decode(&query); err != nil {
			http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
			return
		}

		imageData, err := ctrl.RenderEventList(query)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to render image: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "image/png")
		w.Write(imageData)
	}
}

func handleEventRecordBuild(ctrl *controller.EventController) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req model.EventRecordRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
			return
		}

		drawingReq, err := ctrl.BuildEventRecord(req)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to build request: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(drawingReq)
	}
}

func handleEventRecordRender(ctrl *controller.EventController) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req model.EventRecordRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
			return
		}

		imageData, err := ctrl.RenderEventRecord(req)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to render image: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "image/png")
		w.Write(imageData)
	}
}

func handleEducationChallengeBuild(ctrl *controller.EducationController) http.HandlerFunc {
	type request struct {
		Region string `json:"region"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
			return
		}

		payload, err := ctrl.BuildChallengeLiveRequest(req.Region)
		if err != nil {
			slog.Error("Failed to build education challenge request", "error", err, "region", req.Region)
			http.Error(w, fmt.Sprintf("Failed to build request: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(payload)
	}
}

func handleEducationChallengeRender(ctrl *controller.EducationController) http.HandlerFunc {
	type request struct {
		Region string `json:"region"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
			return
		}

		imageData, err := ctrl.RenderChallengeLiveDetailFromUser(req.Region)
		if err != nil {
			slog.Error("Failed to render education challenge", "error", err, "region", req.Region)
			http.Error(w, fmt.Sprintf("Failed to render image: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "image/png")
		w.Write(imageData)
	}
}

func handleEducationPowerRender(ctrl *controller.EducationController) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req model.PowerBonusDetailRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
			return
		}

		imageData, err := ctrl.RenderPowerBonusDetail(req)
		if err != nil {
			slog.Error("Failed to render education power bonus", "error", err)
			http.Error(w, fmt.Sprintf("Failed to render image: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "image/png")
		w.Write(imageData)
	}
}

func handleEducationPowerBuild(ctrl *controller.EducationController) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req model.PowerBonusDetailRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
			return
		}

		payload, err := ctrl.BuildPowerBonusRequest(req)
		if err != nil {
			slog.Error("Failed to build education power bonus", "error", err)
			http.Error(w, fmt.Sprintf("Failed to build request: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(payload)
	}
}

func handleEducationAreaRender(ctrl *controller.EducationController) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req model.AreaItemUpgradeMaterialsRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
			return
		}

		imageData, err := ctrl.RenderAreaItemMaterials(req)
		if err != nil {
			slog.Error("Failed to render education area item", "error", err)
			http.Error(w, fmt.Sprintf("Failed to render image: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "image/png")
		w.Write(imageData)
	}
}

func handleEducationAreaBuild(ctrl *controller.EducationController) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req model.AreaItemUpgradeMaterialsRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
			return
		}

		payload, err := ctrl.BuildAreaItemMaterialsRequest(req)
		if err != nil {
			slog.Error("Failed to build education area item", "error", err)
			http.Error(w, fmt.Sprintf("Failed to build request: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(payload)
	}
}

func handleEducationBondsRender(ctrl *controller.EducationController) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req model.BondsRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
			return
		}

		imageData, err := ctrl.RenderBonds(req)
		if err != nil {
			slog.Error("Failed to render education bonds", "error", err)
			http.Error(w, fmt.Sprintf("Failed to render image: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "image/png")
		w.Write(imageData)
	}
}

func handleEducationBondsBuild(ctrl *controller.EducationController) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req model.BondsRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
			return
		}

		payload, err := ctrl.BuildBondsRequest(req)
		if err != nil {
			slog.Error("Failed to build education bonds", "error", err)
			http.Error(w, fmt.Sprintf("Failed to build request: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(payload)
	}
}

func handleEducationLeaderRender(ctrl *controller.EducationController) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req model.LeaderCountRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
			return
		}

		imageData, err := ctrl.RenderLeaderCount(req)
		if err != nil {
			slog.Error("Failed to render education leader count", "error", err)
			http.Error(w, fmt.Sprintf("Failed to render image: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "image/png")
		w.Write(imageData)
	}
}

func handleEducationLeaderBuild(ctrl *controller.EducationController) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req model.LeaderCountRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
			return
		}

		payload, err := ctrl.BuildLeaderCountRequest(req)
		if err != nil {
			slog.Error("Failed to build education leader count", "error", err)
			http.Error(w, fmt.Sprintf("Failed to build request: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(payload)
	}
}

func handleStampListRender(ctrl *controller.StampController) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req model.StampListRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
			return
		}

		imageData, err := ctrl.RenderStampList(req)
		if err != nil {
			slog.Error("Failed to render stamp list", "error", err)
			http.Error(w, fmt.Sprintf("Failed to render image: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "image/png")
		w.Write(imageData)
	}
}

func handleStampListBuild(ctrl *controller.StampController) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var query model.StampListQuery
		if err := json.NewDecoder(r.Body).Decode(&query); err != nil {
			http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
			return
		}

		req, err := ctrl.BuildStampListRequest(query)
		if err != nil {
			slog.Error("Failed to build stamp list", "error", err)
			http.Error(w, fmt.Sprintf("Failed to build request: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(req)
	}
}

func handleMiscCharaBirthdayRender(ctrl *controller.MiscController) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req model.CharaBirthdayRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
			return
		}

		imageData, err := ctrl.RenderCharaBirthday(req)
		if err != nil {
			slog.Error("Failed to render chara birthday", "error", err)
			http.Error(w, fmt.Sprintf("Failed to render image: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "image/png")
		w.Write(imageData)
	}
}

func handleMiscCharaBirthdayBuild(ctrl *controller.MiscController) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req model.CharaBirthdayRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
			return
		}

		payload, err := ctrl.BuildCharaBirthdayRequest(req)
		if err != nil {
			slog.Error("Failed to build chara birthday", "error", err)
			http.Error(w, fmt.Sprintf("Failed to build request: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(payload)
	}
}

func handleScoreControlBuild(ctrl *controller.ScoreController) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req model.ScoreControlRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
			return
		}
		payload, err := ctrl.BuildScoreControlRequest(req)
		if err != nil {
			slog.Error("Failed to build score control request", "error", err)
			http.Error(w, fmt.Sprintf("Failed to build request: %v", err), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(payload)
	}
}

func handleScoreControlRender(ctrl *controller.ScoreController) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req model.ScoreControlRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
			return
		}
		imageData, err := ctrl.RenderScoreControl(req)
		if err != nil {
			slog.Error("Failed to render score control", "error", err)
			http.Error(w, fmt.Sprintf("Failed to render image: %v", err), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "image/png")
		w.Write(imageData)
	}
}

func handleScoreCustomRoomBuild(ctrl *controller.ScoreController) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req model.CustomRoomScoreRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
			return
		}
		payload, err := ctrl.BuildCustomRoomScoreRequest(req)
		if err != nil {
			slog.Error("Failed to build custom-room score request", "error", err)
			http.Error(w, fmt.Sprintf("Failed to build request: %v", err), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(payload)
	}
}

func handleScoreCustomRoomRender(ctrl *controller.ScoreController) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req model.CustomRoomScoreRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
			return
		}
		imageData, err := ctrl.RenderCustomRoomScore(req)
		if err != nil {
			slog.Error("Failed to render custom-room score", "error", err)
			http.Error(w, fmt.Sprintf("Failed to render image: %v", err), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "image/png")
		w.Write(imageData)
	}
}

func handleScoreMusicMetaBuild(ctrl *controller.ScoreController) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req []model.MusicMetaRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
			return
		}
		payload, err := ctrl.BuildMusicMetaRequest(req)
		if err != nil {
			slog.Error("Failed to build music-meta request", "error", err)
			http.Error(w, fmt.Sprintf("Failed to build request: %v", err), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(payload)
	}
}

func handleScoreMusicMetaRender(ctrl *controller.ScoreController) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req []model.MusicMetaRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
			return
		}
		imageData, err := ctrl.RenderMusicMeta(req)
		if err != nil {
			slog.Error("Failed to render music-meta", "error", err)
			http.Error(w, fmt.Sprintf("Failed to render image: %v", err), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "image/png")
		w.Write(imageData)
	}
}

func handleScoreMusicBoardBuild(ctrl *controller.ScoreController) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req model.MusicBoardRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
			return
		}
		payload, err := ctrl.BuildMusicBoardRequest(req)
		if err != nil {
			slog.Error("Failed to build music-board request", "error", err)
			http.Error(w, fmt.Sprintf("Failed to build request: %v", err), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(payload)
	}
}

func handleScoreMusicBoardRender(ctrl *controller.ScoreController) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req model.MusicBoardRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
			return
		}
		imageData, err := ctrl.RenderMusicBoard(req)
		if err != nil {
			slog.Error("Failed to render music-board", "error", err)
			http.Error(w, fmt.Sprintf("Failed to render image: %v", err), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "image/png")
		w.Write(imageData)
	}
}

func handleDeckRecommendBuild(ctrl *controller.DeckController) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
			return
		}
		payload, err := ctrl.BuildDeckRecommendRequest(req)
		if err != nil {
			slog.Error("Failed to build deck recommend request", "error", err)
			http.Error(w, fmt.Sprintf("Failed to build request: %v", err), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(payload)
	}
}

func handleDeckRecommendRender(ctrl *controller.DeckController) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
			return
		}
		imageData, err := ctrl.RenderDeckRecommend(req)
		if err != nil {
			slog.Error("Failed to render deck recommend", "error", err)
			http.Error(w, fmt.Sprintf("Failed to render image: %v", err), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "image/png")
		w.Write(imageData)
	}
}

func handleDeckRecommendAutoBuild(ctrl *controller.DeckController) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var query model.DeckAutoQuery
		if err := json.NewDecoder(r.Body).Decode(&query); err != nil {
			http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
			return
		}
		payload, err := ctrl.BuildDeckRecommendAutoRequest(query)
		if err != nil {
			slog.Error("Failed to build deck auto recommend request", "error", err)
			http.Error(w, fmt.Sprintf("Failed to build request: %v", err), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(payload)
	}
}

func handleDeckRecommendAutoRender(ctrl *controller.DeckController) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var query model.DeckAutoQuery
		if err := json.NewDecoder(r.Body).Decode(&query); err != nil {
			http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
			return
		}
		imageData, err := ctrl.RenderDeckRecommendAuto(query)
		if err != nil {
			slog.Error("Failed to render deck auto recommend", "error", err)
			http.Error(w, fmt.Sprintf("Failed to render image: %v", err), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "image/png")
		w.Write(imageData)
	}
}

func handleSKBuild(ctrl *controller.SkController) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
			return
		}
		payload, err := ctrl.Build(req)
		if err != nil {
			slog.Error("Failed to build sk request", "error", err)
			http.Error(w, fmt.Sprintf("Failed to build request: %v", err), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(payload)
	}
}

func handleSKLineRender(ctrl *controller.SkController) http.HandlerFunc {
	return handleSKRender(ctrl.RenderLine)
}

func handleSKQueryRender(ctrl *controller.SkController) http.HandlerFunc {
	return handleSKRender(ctrl.RenderQuery)
}

func handleSKCheckRoomRender(ctrl *controller.SkController) http.HandlerFunc {
	return handleSKRender(ctrl.RenderCheckRoom)
}

func handleSKSpeedRender(ctrl *controller.SkController) http.HandlerFunc {
	return handleSKRender(ctrl.RenderSpeed)
}

func handleSKPlayerTraceRender(ctrl *controller.SkController) http.HandlerFunc {
	return handleSKRender(ctrl.RenderPlayerTrace)
}

func handleSKRankTraceRender(ctrl *controller.SkController) http.HandlerFunc {
	return handleSKRender(ctrl.RenderRankTrace)
}

func handleSKWinrateRender(ctrl *controller.SkController) http.HandlerFunc {
	return handleSKRender(ctrl.RenderWinrate)
}

func handleSKRender(render func(map[string]interface{}) ([]byte, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
			return
		}
		imageData, err := render(req)
		if err != nil {
			slog.Error("Failed to render sk", "error", err)
			http.Error(w, fmt.Sprintf("Failed to render image: %v", err), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "image/png")
		w.Write(imageData)
	}
}

func handleMysekaiBuild(ctrl *controller.MysekaiController) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req interface{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
			return
		}
		payload, err := ctrl.Build(req)
		if err != nil {
			slog.Error("Failed to build mysekai request", "error", err)
			http.Error(w, fmt.Sprintf("Failed to build request: %v", err), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(payload)
	}
}

func handleMysekaiResourceRender(ctrl *controller.MysekaiController) http.HandlerFunc {
	return handleMysekaiRender(ctrl.RenderResource)
}

func handleMysekaiFixtureListRender(ctrl *controller.MysekaiController) http.HandlerFunc {
	return handleMysekaiRender(ctrl.RenderFixtureList)
}

func handleMysekaiFixtureDetailRender(ctrl *controller.MysekaiController) http.HandlerFunc {
	return handleMysekaiRender(ctrl.RenderFixtureDetail)
}

func handleMysekaiDoorUpgradeRender(ctrl *controller.MysekaiController) http.HandlerFunc {
	return handleMysekaiRender(ctrl.RenderDoorUpgrade)
}

func handleMysekaiMusicRecordRender(ctrl *controller.MysekaiController) http.HandlerFunc {
	return handleMysekaiRender(ctrl.RenderMusicRecord)
}

func handleMysekaiTalkListRender(ctrl *controller.MysekaiController) http.HandlerFunc {
	return handleMysekaiRender(ctrl.RenderTalkList)
}

func handleMysekaiRender(render func(interface{}) ([]byte, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req interface{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
			return
		}
		imageData, err := render(req)
		if err != nil {
			slog.Error("Failed to render mysekai", "error", err)
			http.Error(w, fmt.Sprintf("Failed to render image: %v", err), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "image/png")
		w.Write(imageData)
	}
}

func handleHonorBuild(ctrl *controller.HonorController) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var query model.HonorQuery
		if err := json.NewDecoder(r.Body).Decode(&query); err != nil {
			http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
			return
		}

		req, err := ctrl.BuildHonorRequest(query)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to build request: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(req)
	}
}

func handleHonorRender(ctrl *controller.HonorController) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req model.HonorRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
			return
		}

		imageData, err := ctrl.RenderHonorImage(req)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to render image: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "image/png")
		w.Write(imageData)
	}
}

func handleProfileBuild(ctrl *controller.ProfileController, userData *service.UserDataService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var query model.ProfileQuery
		if err := json.NewDecoder(r.Body).Decode(&query); err != nil {
			http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
			return
		}

		if query.UserID == "" {
			http.Error(w, "UserID is required", http.StatusBadRequest)
			return
		}
		if query.Region == "" {
			query.Region = "jp"
		}

		req, err := ctrl.BuildProfileRequest(query.UserID, query.Region, userData)
		if err != nil {
			slog.Error("Failed to build profile", "error", err, "userID", query.UserID)
			http.Error(w, fmt.Sprintf("Failed to build request: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(req)
	}
}

func handleProfileRender(ctrl *controller.ProfileController, userData *service.UserDataService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var query model.ProfileQuery
		if err := json.NewDecoder(r.Body).Decode(&query); err != nil {
			http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
			return
		}

		// 如果没有传入 UserID，则尝试从 query 获取，如果还是没有，则报错
		if query.UserID == "" {
			http.Error(w, "UserID is required", http.StatusBadRequest)
			return
		}

		if query.Region == "" {
			query.Region = "jp"
		}

		imageData, err := ctrl.RenderProfile(query.UserID, query.Region, userData)
		if err != nil {
			slog.Error("Failed to render profile", "error", err, "userID", query.UserID)
			http.Error(w, fmt.Sprintf("Failed to render image: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "image/png")
		w.Write(imageData)
	}
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		slog.Info("HTTP Request",
			"method", r.Method,
			"path", r.URL.Path,
			"duration", time.Since(start),
		)
	})
}
