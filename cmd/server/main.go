package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"Haruki-Service-API/internal/config"
	"Haruki-Service-API/internal/controller"
	"Haruki-Service-API/internal/model"
	"Haruki-Service-API/internal/service"
	"Haruki-Service-API/pkg/asset"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)
	slog.Info("Starting Lunabot Service...")

	cfg, err := config.Load("configs/configs.yaml")
	if err != nil {
		slog.Error("Failed to load config", "error", err)
		os.Exit(1)
	}

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

	nicknames := masterdata.GetNicknames()
	cardParser := service.NewCardParser(nicknames)
	cardSearchService := service.NewCardSearchService(masterdata, cardParser)

	cardController := controller.NewCardController(masterdata, drawing, cardSearchService, cfg.DrawingAPI.BaseURL, assetHelper, userData)
	musicController := controller.NewMusicController(masterdata, drawing, cfg.DrawingAPI.BaseURL, assetHelper, userData)
	gachaController := controller.NewGachaController(masterdata, drawing, cfg.DrawingAPI.BaseURL, assetHelper)
	honorController := controller.NewHonorController(masterdata, drawing, assetHelper)
	eventController := controller.NewEventController(masterdata, drawing, cfg.DrawingAPI.BaseURL, assetHelper)
	profileController := controller.NewProfileController(masterdata, drawing, assetHelper)

	mux := http.NewServeMux()

	mux.HandleFunc("/health", handleHealth)

	mux.HandleFunc("/api/card/detail/build", handleCardDetailBuild(cardController))
	mux.HandleFunc("/api/card/detail/render", handleCardDetailRender(cardController))
	mux.HandleFunc("/api/card/list/render", handleCardListRender(cardController))

	mux.HandleFunc("/api/music/detail/build", handleMusicDetailBuild(musicController))
	mux.HandleFunc("/api/music/detail/render", handleMusicDetailRender(musicController))
	mux.HandleFunc("/api/music/brief-list/build", handleMusicBriefListBuild(musicController))
	mux.HandleFunc("/api/music/brief-list/render", handleMusicBriefListRender(musicController))
	mux.HandleFunc("/api/music/list/build", handleMusicListBuild(musicController))
	mux.HandleFunc("/api/music/list/render", handleMusicListRender(musicController))
	mux.HandleFunc("/api/music/progress/build", handleMusicProgressBuild(musicController))
	mux.HandleFunc("/api/music/progress/render", handleMusicProgressRender(musicController))
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

	mux.HandleFunc("/api/honor/build", handleHonorBuild(honorController))
	mux.HandleFunc("/api/honor/render", handleHonorRender(honorController))

	mux.HandleFunc("/api/profile/render", handleProfileRender(profileController, userData))

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

		req, err := ctrl.BuildEventDetail(query)
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

		imageData, err := ctrl.RenderEventDetail(query)
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
