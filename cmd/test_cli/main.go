package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
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

type cliEnv struct {
	masterdata          *service.MasterDataService
	cardController      *controller.CardController
	musicController     *controller.MusicController
	gachaController     *controller.GachaController
	eventController     *controller.EventController
	educationController *controller.EducationController
	honorController     *controller.HonorController
	profileController   *controller.ProfileController
	stampController     *controller.StampController
	miscController      *controller.MiscController
	scoreController     *controller.ScoreController
	deckController      *controller.DeckController
	skController        *controller.SkController
	mysekaiController   *controller.MysekaiController
	cardParser          *service.CardParser
	eventParser         *service.EventParser
	eventSearch         *service.EventSearchService
	musicSearch         *service.MusicSearchService
	userData            *service.UserDataService
	resolver            *service.GlobalCommandResolver
}

type scenario struct {
	Name        string `json:"name"`
	Mode        string `json:"mode"`
	Cmd         string `json:"cmd"`
	Description string `json:"description"`
}

var globalOutputDir string

func main() {
	modePtr := flag.String("mode", "auto", "Mode: auto/detail/card-detail, card-list, card-box, music (detail), music-brief, music-list, music-progress, music-chart, music-reward-detail, music-reward-basic, gacha-list, gacha-detail, event-detail, event-list, event-record, education-* (challenge/power/area/bonds/leader), honor, profile, stamp-list, misc-chara-birthday, score-control/score-custom-room/score-music-meta/score-music-board, deck-recommend/deck-recommend-auto, sk-*, mysekai-*")
	cmdPtr := flag.String("cmd", "", "Command payload, e.g. '/查卡 190'、'/查曲 112'、'/卡池 17'、'/活动列表 wl'")
	scenarioPtr := flag.String("scenario", "", "Run multiple commands; use 'all' for built-in regression or provide a JSON file path")
	flag.Parse()

	configPath := "../../configs/configs.yaml"
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		configPath = "configs/configs.yaml"
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			configPath = "configs.yaml"
		}
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		slog.Error("Failed to load config", "error", err)
		os.Exit(1)
	}

	globalOutputDir = cfg.DrawingAPI.OutputDir

	cloudClients, err := apiutils.InitCloudClients(cfg.HarukiCloud, slog.Default())
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

	drawing := service.NewDrawingService(cfg.DrawingAPI.BaseURL, cfg.DrawingAPI.Timeout, cfg.DrawingAPI.RetryCount, assetHelper.Roots())
	deckRecommender := service.NewDeckRecommenderService(cfg.DeckRecommend)

	nicknames := masterdata.GetNicknames()
	cardParser := service.NewCardParser(nicknames)
	cloudRegion := strings.TrimSpace(cfg.HarukiCloud.Region)
	if cloudRegion == "" {
		cloudRegion = masterdata.GetRegion()
	}
	secondaryRegion := strings.TrimSpace(cfg.HarukiCloud.SecondaryRegion)
	if secondaryRegion == "" {
		secondaryRegion = "jp"
	}

	masterCardSource := service.NewMasterDataCardSource(masterdata)
	var cardSource service.CardDataSource
	if cloudClients.Sekai != nil {
		cardSource = service.NewCloudCardSource(cloudClients.Sekai, cloudRegion)
	}
	if cardSource == nil {
		cardSource = masterCardSource
	}
	var secondaryCardSource service.CardDataSource
	if cloudClients.Sekai != nil && secondaryRegion != "" {
		secondaryCardSource = service.NewCloudCardSource(cloudClients.Sekai, secondaryRegion)
	}
	if secondaryCardSource == nil {
		secondaryCardSource = masterCardSource
	}

	var eventSource service.EventDataSource
	if cloudClients.Sekai != nil {
		eventSource = service.NewCloudEventSource(cloudClients.Sekai, cloudRegion)
	}
	if eventSource == nil {
		eventSource = service.NewMasterDataEventSource(masterdata)
	}
	var secondaryEventSource service.EventDataSource
	if cloudClients.Sekai != nil && secondaryRegion != "" && !strings.EqualFold(secondaryRegion, cloudRegion) {
		secondaryEventSource = service.NewCloudEventSource(cloudClients.Sekai, secondaryRegion)
	}

	cardSearchService := service.NewCardSearchService(cardSource, cardParser)
	eventParser := service.NewEventParser(nicknames)
	eventSearch := service.NewEventSearchService(eventSource, eventParser)
	musicParser := service.NewMusicParser(masterdata)
	musicSearch := service.NewMusicSearchService(masterdata, musicParser)

	musicSource := service.MusicDataSource(service.NewMasterDataMusicSource(masterdata))
	if cloudClients.Sekai != nil {
		if src := service.NewCloudMusicSource(cloudClients.Sekai, cloudRegion); src != nil {
			musicSource = src
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
	masterHonorSource := service.NewMasterDataHonorSource(masterdata)
	var honorSource service.HonorDataSource
	if cloudClients.Sekai != nil {
		honorSource = service.NewCloudHonorSource(cloudClients.Sekai, cloudRegion)
	}
	if honorSource == nil {
		honorSource = masterHonorSource
	}
	masterProfileSource := service.NewMasterDataProfileSource(masterdata)
	var profileSource service.ProfileDataSource
	if cloudClients.Sekai != nil {
		profileSource = service.NewCloudProfileSource(cloudClients.Sekai, cloudRegion)
	}
	if profileSource == nil {
		profileSource = masterProfileSource
	}
	masterEducationSource := service.NewMasterDataEducationSource(masterdata)
	var educationSource service.EducationDataSource
	if cloudClients.Sekai != nil {
		educationSource = service.NewCloudEducationSource(cloudClients.Sekai, cloudRegion)
	}
	if educationSource == nil {
		educationSource = masterEducationSource
	}

	cardController := controller.NewCardController(cardSource, secondaryCardSource, eventSource, masterdata, drawing, cardSearchService, cfg.DrawingAPI.BaseURL, assetHelper, userData)
	if secondaryEventSource != nil {
		cardController.RegisterEventSource(secondaryEventSource)
	}
	musicController := controller.NewMusicController(musicSource, drawing, cfg.DrawingAPI.BaseURL, assetHelper, userData)
	if cloudClients.Sekai != nil {
		for _, region := range []string{"jp", "en", "tw", "kr", "cn", secondaryRegion} {
			normalized := strings.ToLower(strings.TrimSpace(region))
			if normalized == "" || strings.EqualFold(normalized, cloudRegion) {
				continue
			}
			if src := service.NewCloudMusicSource(cloudClients.Sekai, normalized); src != nil {
				musicController.RegisterSource(src)
			}
		}
	}
	gachaController := controller.NewGachaController(gachaSource, drawing, cfg.DrawingAPI.BaseURL, assetHelper)
	if cloudClients.Sekai != nil && secondaryRegion != "" && !strings.EqualFold(secondaryRegion, cloudRegion) {
		if src := service.NewCloudGachaSource(cloudClients.Sekai, secondaryRegion); src != nil {
			gachaController.RegisterSource(src)
		}
	}
	honorController := controller.NewHonorController(honorSource, drawing, assetHelper)
	if cloudClients.Sekai != nil && secondaryRegion != "" && !strings.EqualFold(secondaryRegion, cloudRegion) {
		if src := service.NewCloudHonorSource(cloudClients.Sekai, secondaryRegion); src != nil {
			honorController.RegisterSource(src)
		}
	}
	eventController := controller.NewEventController(eventSource, drawing, cfg.DrawingAPI.BaseURL, assetHelper, cloudService)
	if secondaryEventSource != nil {
		eventController.RegisterSource(secondaryEventSource)
	}
	profileController := controller.NewProfileController(profileSource, drawing, assetHelper)
	if cloudClients.Sekai != nil && secondaryRegion != "" && !strings.EqualFold(secondaryRegion, cloudRegion) {
		if src := service.NewCloudProfileSource(cloudClients.Sekai, secondaryRegion); src != nil {
			profileController.RegisterSource(src)
		}
	}
	educationController := controller.NewEducationController(educationSource, drawing, assetHelper, userData)
	if cloudClients.Sekai != nil && secondaryRegion != "" && !strings.EqualFold(secondaryRegion, cloudRegion) {
		if src := service.NewCloudEducationSource(cloudClients.Sekai, secondaryRegion); src != nil {
			educationController.RegisterSource(src)
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
		if src := service.NewCloudStampSource(cloudClients.Sekai, secondaryRegion); src != nil {
			stampController.RegisterSource(src)
		}
	}
	miscController := controller.NewMiscController(drawing)
	scoreController := controller.NewScoreController(drawing)
	deckController := controller.NewDeckController(drawing, cardSource, eventSource, assetHelper, userData, deckRecommender)
	skController := controller.NewSkController(drawing)
	mysekaiController := controller.NewMysekaiController(drawing)

	env := &cliEnv{
		masterdata:          masterdata,
		cardController:      cardController,
		musicController:     musicController,
		gachaController:     gachaController,
		honorController:     honorController,
		profileController:   profileController,
		stampController:     stampController,
		miscController:      miscController,
		scoreController:     scoreController,
		deckController:      deckController,
		skController:        skController,
		mysekaiController:   mysekaiController,
		cardParser:          cardParser,
		eventController:     eventController,
		educationController: educationController,
		eventParser:         eventParser,
		eventSearch:         eventSearch,
		musicSearch:         musicSearch,
		userData:            userData,
		resolver:            service.NewGlobalCommandResolver(nicknames),
	}

	if *scenarioPtr != "" {
		if err := env.runScenario(*scenarioPtr); err != nil {
			slog.Error("Scenario failed", "scenario", *scenarioPtr, "error", err)
			os.Exit(1)
		}
		return
	}

	if err := env.runMode(*modePtr, *cmdPtr); err != nil {
		slog.Error("Mode execution failed", "mode", *modePtr, "cmd", *cmdPtr, "error", err)
		os.Exit(1)
	}
}

func (env *cliEnv) runScenario(name string) error {
	scenarios, err := env.resolveScenario(name)
	if err != nil {
		return err
	}
	fmt.Printf("Running %d scenario(s)\n", len(scenarios))
	for idx, sc := range scenarios {
		fmt.Printf("\n[%d/%d] %s - %s\n", idx+1, len(scenarios), sc.Name, sc.Description)
		if err := env.runMode(sc.Mode, sc.Cmd); err != nil {
			return fmt.Errorf("scenario %s failed: %w", sc.Name, err)
		}
	}
	fmt.Println("\nAll scenarios finished successfully.")
	return nil
}

func (env *cliEnv) resolveScenario(name string) ([]scenario, error) {
	lower := strings.ToLower(strings.TrimSpace(name))
	switch {
	case lower == "all":
		return defaultScenarios(env)
	case strings.HasSuffix(lower, ".json"):
		return loadScenarioFile(name)
	default:
		return nil, fmt.Errorf("unknown scenario: %s", name)
	}
}

func defaultScenarios(env *cliEnv) ([]scenario, error) {
	slots := []struct {
		Name string
		Mode string
		Desc string
	}{
		{"card-detail", "card-detail", "Card detail"},
		{"card-list", "card-list", "Card list query"},
		{"card-box", "card-box", "Card box"},
		{"music-detail", "music", "Music detail"},
		{"music-brief", "music-brief", "Music brief"},
		{"music-list", "music-list", "Music list"},
		{"music-progress", "music-progress", "Music progress"},
		{"music-chart", "music-chart", "Music chart"},
		{"gacha-list", "gacha-list", "Gacha list"},
		{"gacha-detail", "gacha-detail", "Gacha detail"},
		{"event-detail", "event-detail", "Event detail"},
		{"event-list", "event-list", "Event list"},
	}
	var scenarios []scenario
	for _, slot := range slots {
		cmd, err := env.defaultCommand(slot.Mode)
		if err != nil {
			return nil, err
		}
		scenarios = append(scenarios, scenario{Name: slot.Name, Mode: slot.Mode, Cmd: cmd, Description: slot.Desc})
	}
	return scenarios, nil
}

func loadScenarioFile(path string) ([]scenario, error) {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return nil, err
	}
	var scenarios []scenario
	if err := json.Unmarshal(data, &scenarios); err != nil {
		return nil, err
	}
	if len(scenarios) == 0 {
		return nil, fmt.Errorf("scenario file %s is empty", path)
	}
	return scenarios, nil
}

func (env *cliEnv) runMode(mode string, cmd string) error {
	normalized := strings.ToLower(strings.TrimSpace(mode))

	if normalized == "auto" {
		res, err := env.resolver.Resolve(cmd)
		if err != nil {
			return err
		}
		return env.handleResolvedCommand(res)
	}

	switch normalized {
	case "detail", "card-detail":
		payload, err := env.ensureCommand("card-detail", cmd)
		if err != nil {
			return err
		}
		return testCardDetail(env.cardController, env.cardParser, payload)
	case "list", "card-list":
		if strings.TrimSpace(cmd) == "" {
			return testCardListHardcoded(env.cardController)
		}
		return testCardListDynamic(env.cardController, cmd)
	case "box", "card-box":
		payload, err := env.ensureCommand("card-box", cmd)
		if err != nil {
			return err
		}
		return testCardBox(env.cardController, payload)
	case "music", "music-detail":
		payload, err := env.ensureCommand("music", cmd)
		if err != nil {
			return err
		}
		return testMusicDetail(env.musicController, env.musicSearch, payload)
	case "music-brief":
		payload, err := env.ensureCommand("music-brief", cmd)
		if err != nil {
			return err
		}
		return testMusicBriefList(env.musicController, payload)
	case "music-list":
		payload, err := env.ensureCommand("music-list", cmd)
		if err != nil {
			return err
		}
		return testMusicList(env.musicController, payload)
	case "music-progress":
		payload, err := env.ensureCommand("music-progress", cmd)
		if err != nil {
			return err
		}
		return testMusicProgress(env.musicController, payload)
	case "music-chart":
		payload, err := env.ensureCommand("music-chart", cmd)
		if err != nil {
			return err
		}
		return testMusicChart(env.musicController, payload)
	case "music-reward-detail":
		if strings.TrimSpace(cmd) == "" {
			return fmt.Errorf("music-reward-detail requires -cmd JSON file path")
		}
		return testMusicRewardsDetail(env.musicController, cmd)
	case "music-reward-basic":
		if strings.TrimSpace(cmd) == "" {
			return fmt.Errorf("music-reward-basic requires -cmd JSON file path")
		}
		return testMusicRewardsBasic(env.musicController, cmd)
	case "gacha-list":
		payload, err := env.ensureCommand("gacha-list", cmd)
		if err != nil {
			return err
		}
		return testGachaList(env.gachaController, payload)
	case "gacha-detail":
		payload, err := env.ensureCommand("gacha-detail", cmd)
		if err != nil {
			return err
		}
		return testGachaDetail(env, payload)
	case "event-detail":
		payload, err := env.ensureCommand("event-detail", cmd)
		if err != nil {
			return err
		}
		return testEventDetail(env.eventController, env.eventSearch, payload)
	case "event-list":
		payload, err := env.ensureCommand("event-list", cmd)
		if err != nil {
			return err
		}
		return testEventList(env.eventController, env.eventParser, payload)
	case "event-record":
		if strings.TrimSpace(cmd) == "" {
			return fmt.Errorf("event-record mode requires -cmd JSON file path")
		}
		return testEventRecord(env.eventController, cmd)
	case "education-challenge":
		return testEducationChallengeLive(env.educationController, cmd)
	case "education-power":
		if strings.TrimSpace(cmd) == "" {
			return fmt.Errorf("education-power mode requires -cmd JSON file path")
		}
		return testEducationPowerBonus(env.educationController, cmd)
	case "education-area":
		if strings.TrimSpace(cmd) == "" {
			return fmt.Errorf("education-area mode requires -cmd JSON file path")
		}
		return testEducationAreaItem(env.educationController, cmd)
	case "education-bonds":
		if strings.TrimSpace(cmd) == "" {
			return fmt.Errorf("education-bonds mode requires -cmd JSON file path")
		}
		return testEducationBonds(env.educationController, cmd)
	case "education-leader":
		if strings.TrimSpace(cmd) == "" {
			return fmt.Errorf("education-leader mode requires -cmd JSON file path")
		}
		return testEducationLeaderCount(env.educationController, cmd)
	case "honor":
		if strings.TrimSpace(cmd) == "" {
			return fmt.Errorf("honor mode requires -cmd JSON file path")
		}
		return testHonorGenerate(env.honorController, cmd)
	case "profile":
		return testProfileGenerate(env.profileController, env.userData, cmd)
	case "stamp-list":
		if strings.TrimSpace(cmd) == "" {
			return fmt.Errorf("stamp-list mode requires -cmd JSON file path")
		}
		return testStampList(env.stampController, cmd)
	case "misc-chara-birthday":
		if strings.TrimSpace(cmd) == "" {
			return fmt.Errorf("misc-chara-birthday mode requires -cmd JSON file path")
		}
		return testMiscCharaBirthday(env.miscController, cmd)
	case "score-control":
		if strings.TrimSpace(cmd) == "" {
			return fmt.Errorf("score-control mode requires -cmd JSON file path")
		}
		return testScoreControl(env.scoreController, cmd)
	case "score-custom-room":
		if strings.TrimSpace(cmd) == "" {
			return fmt.Errorf("score-custom-room mode requires -cmd JSON file path")
		}
		return testScoreCustomRoom(env.scoreController, cmd)
	case "score-music-meta":
		if strings.TrimSpace(cmd) == "" {
			return fmt.Errorf("score-music-meta mode requires -cmd JSON file path")
		}
		return testScoreMusicMeta(env.scoreController, cmd)
	case "score-music-board":
		if strings.TrimSpace(cmd) == "" {
			return fmt.Errorf("score-music-board mode requires -cmd JSON file path")
		}
		return testScoreMusicBoard(env.scoreController, cmd)
	case "deck-recommend":
		if strings.TrimSpace(cmd) == "" {
			return fmt.Errorf("deck-recommend mode requires -cmd JSON file path")
		}
		return testDeckRecommend(env.deckController, cmd)
	case "deck-recommend-auto":
		if strings.TrimSpace(cmd) == "" {
			return fmt.Errorf("deck-recommend-auto mode requires -cmd JSON file path")
		}
		return testDeckRecommendAuto(env.deckController, cmd)
	case "sk-line":
		if strings.TrimSpace(cmd) == "" {
			return fmt.Errorf("sk-line mode requires -cmd JSON file path")
		}
		return testSKLine(env.skController, cmd)
	case "sk-query":
		if strings.TrimSpace(cmd) == "" {
			return fmt.Errorf("sk-query mode requires -cmd JSON file path")
		}
		return testSKQuery(env.skController, cmd)
	case "sk-check-room":
		if strings.TrimSpace(cmd) == "" {
			return fmt.Errorf("sk-check-room mode requires -cmd JSON file path")
		}
		return testSKCheckRoom(env.skController, cmd)
	case "sk-speed":
		if strings.TrimSpace(cmd) == "" {
			return fmt.Errorf("sk-speed mode requires -cmd JSON file path")
		}
		return testSKSpeed(env.skController, cmd)
	case "sk-player-trace":
		if strings.TrimSpace(cmd) == "" {
			return fmt.Errorf("sk-player-trace mode requires -cmd JSON file path")
		}
		return testSKPlayerTrace(env.skController, cmd)
	case "sk-rank-trace":
		if strings.TrimSpace(cmd) == "" {
			return fmt.Errorf("sk-rank-trace mode requires -cmd JSON file path")
		}
		return testSKRankTrace(env.skController, cmd)
	case "sk-winrate":
		if strings.TrimSpace(cmd) == "" {
			return fmt.Errorf("sk-winrate mode requires -cmd JSON file path")
		}
		return testSKWinrate(env.skController, cmd)
	case "mysekai-resource":
		if strings.TrimSpace(cmd) == "" {
			return fmt.Errorf("mysekai-resource mode requires -cmd JSON file path")
		}
		return testMysekaiResource(env.mysekaiController, cmd)
	case "mysekai-fixture-list":
		if strings.TrimSpace(cmd) == "" {
			return fmt.Errorf("mysekai-fixture-list mode requires -cmd JSON file path")
		}
		return testMysekaiFixtureList(env.mysekaiController, cmd)
	case "mysekai-fixture-detail":
		if strings.TrimSpace(cmd) == "" {
			return fmt.Errorf("mysekai-fixture-detail mode requires -cmd JSON file path")
		}
		return testMysekaiFixtureDetail(env.mysekaiController, cmd)
	case "mysekai-door-upgrade":
		if strings.TrimSpace(cmd) == "" {
			return fmt.Errorf("mysekai-door-upgrade mode requires -cmd JSON file path")
		}
		return testMysekaiDoorUpgrade(env.mysekaiController, cmd)
	case "mysekai-music-record":
		if strings.TrimSpace(cmd) == "" {
			return fmt.Errorf("mysekai-music-record mode requires -cmd JSON file path")
		}
		return testMysekaiMusicRecord(env.mysekaiController, cmd)
	case "mysekai-talk-list":
		if strings.TrimSpace(cmd) == "" {
			return fmt.Errorf("mysekai-talk-list mode requires -cmd JSON file path")
		}
		return testMysekaiTalkList(env.mysekaiController, cmd)
	default:
		return fmt.Errorf("unknown mode: %s", mode)
	}
}

func (env *cliEnv) handleResolvedCommand(res *service.ResolvedCommand) error {
	if res.IsHelp {
		fmt.Println("Haruki Command Help:")
		fmt.Println("  /card <mnr> [-r jp/en/cn] - card detail")
		fmt.Println("  /music <id/name> [-r jp/en/cn] - music detail")
		fmt.Println("  /event [current/id/name] - event detail")
		fmt.Println("  /sk [uid/rank/@user] - event record detail")
		return nil
	}

	if res.Region != "" {
		slog.Info("Switching region", "target", res.Region)
	}

	var err error
	switch res.Module {
	case service.ModuleCard:
		switch res.Mode {
		case "gacha-list":
			err = testGachaList(env.gachaController, res.Query)
		case "card-box":
			err = testCardBox(env.cardController, res.Query)
		case "card-list":
			err = testCardListDynamic(env.cardController, res.Query)
		default:
			err = testCardDetail(env.cardController, env.cardParser, res.Query)
		}
	case service.ModuleMusic:
		switch res.Mode {
		case "music-chart":
			err = testMusicChart(env.musicController, res.Query)
		case "music-list":
			err = testMusicList(env.musicController, res.Query)
		case "music-progress":
			err = testMusicProgress(env.musicController, res.Query)
		default:
			err = testMusicDetail(env.musicController, env.musicSearch, res.Query)
		}
	case service.ModuleEvent:
		switch res.Mode {
		case "event-list":
			err = testEventList(env.eventController, env.eventParser, res.Query)
		default:
			err = testEventDetail(env.eventController, env.eventSearch, res.Query)
		}
	case service.ModuleProfile:
		err = testProfileGenerate(env.profileController, env.userData, res.Query)
	case service.ModuleGacha:
		switch res.Mode {
		case "gacha":
			err = testGachaDetail(env, res.Query)
			if err != nil {
				err = testGachaList(env.gachaController, res.Query)
			}
		default:
			err = testGachaList(env.gachaController, res.Query)
		}
	case service.ModuleDeck:
		switch res.Mode {
		case "deck-event":
			err = testDeckRecommendAuto(env.deckController, res.Query)
		case "deck-no-event":
			err = testDeckRecommendAuto(env.deckController, res.Query)
		case "deck-bonus":
			err = testDeckRecommendAuto(env.deckController, res.Query)
		case "deck-challenge":
			err = testDeckRecommendAuto(env.deckController, res.Query)
		case "deck-mysekai":
			err = testDeckRecommendAuto(env.deckController, res.Query)
		default:
			err = testDeckRecommendAuto(env.deckController, res.Query)
		}
	case service.ModuleSK:
		return fmt.Errorf("sk module requires JSON input file, cannot be run from auto parsing alone")
	case service.ModuleMysekai:
		return fmt.Errorf("mysekai module requires JSON input file, cannot be run from auto parsing alone")
	default:
		return fmt.Errorf("cannot execute resolved command directly: %v", res)
	}

	if err != nil {
		slog.Error("Module execution failed", "module", res.Module, "mode", res.Mode, "error", err)
		return err
	}
	return nil
}

func (env *cliEnv) ensureCommand(mode, cmd string) (string, error) {
	if strings.TrimSpace(cmd) != "" {
		return cmd, nil
	}
	return env.defaultCommand(mode)
}

func (env *cliEnv) defaultCommand(mode string) (string, error) {
	switch strings.ToLower(mode) {
	case "card-detail":
		return "/查卡 190", nil
	case "card-box":
		return "/查卡 mnr", nil
	case "music", "music-detail":
		return "/查曲 112", nil
	case "music-brief":
		return "master:1,2,3", nil
	case "music-list":
		return "/查曲列表 ma 32", nil
	case "music-progress":
		return "/歌曲进度 ma", nil
	case "music-chart":
		return "/谱面信息 1 ma", nil
	case "gacha-list":
		return "/卡池列表 p1", nil
	case "gacha-detail":
		latestID, err := env.pickLatestGachaID()
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("/卡池 %d", latestID), nil
	case "event-detail":
		return "/活动 current", nil
	case "event-list":
		return "/events wl", nil
	case "education-challenge":
		return "/挑战信息", nil
	case "profile":
		return "1", nil
	case "stamp-list":
		return "D:/github/testfile/stamp_list.json", nil
	case "misc-chara-birthday":
		return "D:/github/testfile/misc_birthday.json", nil
	case "score-control":
		return "D:/github/testfile/score_control.json", nil
	case "score-custom-room":
		return "D:/github/testfile/score_custom_room.json", nil
	case "score-music-meta":
		return "D:/github/testfile/score_music_meta.json", nil
	case "score-music-board":
		return "D:/github/testfile/score_music_board.json", nil
	case "deck-recommend":
		return "D:/github/testfile/deck_recommend.json", nil
	case "deck-recommend-auto":
		return "D:/github/testfile/deck_recommend.json", nil
	case "sk-line":
		return "D:/github/testfile/sk_line.json", nil
	case "sk-query":
		return "D:/github/testfile/sk_query.json", nil
	case "sk-check-room":
		return "D:/github/testfile/sk_check_room.json", nil
	case "sk-speed":
		return "D:/github/testfile/sk_speed.json", nil
	case "sk-player-trace":
		return "D:/github/testfile/sk_player_trace.json", nil
	case "sk-rank-trace":
		return "D:/github/testfile/sk_rank_trace.json", nil
	case "sk-winrate":
		return "D:/github/testfile/sk_winrate.json", nil
	case "mysekai-resource":
		return "D:/github/testfile/mysekai_resource.json", nil
	case "mysekai-fixture-list":
		return "D:/github/testfile/mysekai_fixture_list.json", nil
	case "mysekai-fixture-detail":
		return "D:/github/testfile/mysekai_fixture_detail.json", nil
	case "mysekai-door-upgrade":
		return "D:/github/testfile/mysekai_door_upgrade.json", nil
	case "mysekai-music-record":
		return "D:/github/testfile/mysekai_music_record.json", nil
	case "mysekai-talk-list":
		return "D:/github/testfile/mysekai_talk_list.json", nil
	default:
		return "", fmt.Errorf("mode %s requires -cmd", mode)
	}
}
func (env *cliEnv) pickLatestGachaID() (int, error) {
	gachas := env.masterdata.GetGachas()
	if len(gachas) == 0 {
		return 0, fmt.Errorf("no gacha data available")
	}
	latest := gachas[0]
	for _, g := range gachas {
		if g.StartAt > latest.StartAt {
			latest = g
		}
	}
	return latest.ID, nil
}

func preprocessCommand(cmd string, keywords ...string) string {
	raw := strings.TrimSpace(strings.TrimPrefix(cmd, "/"))
	for _, kw := range keywords {
		if kw == "" {
			continue
		}
		raw = strings.ReplaceAll(raw, kw, "")
	}
	return strings.TrimSpace(raw)
}

func stripLeadingRegionToken(raw string) string {
	_, rest := extractLeadingRegionToken(raw)
	return rest
}

func extractLeadingRegionToken(raw string) (string, string) {
	parts := strings.Fields(strings.TrimSpace(raw))
	if len(parts) == 0 {
		return "", ""
	}
	regionSet := map[string]struct{}{
		"jp": {}, "en": {}, "cn": {}, "tw": {}, "kr": {},
	}
	first := strings.ToLower(strings.TrimSpace(parts[0]))
	if _, ok := regionSet[first]; ok {
		return first, strings.TrimSpace(strings.Join(parts[1:], " "))
	}
	return "", strings.TrimSpace(strings.Join(parts, " "))
}

func testMusicDetail(ctrl *controller.MusicController, svc *service.MusicSearchService, cmd string) error {
	raw := preprocessCommand(cmd, "/查曲", "查曲", "查歌", "查乐", "查询乐曲", "查音乐")
	region, cleaned := extractLeadingRegionToken(raw)
	if region == "" {
		region = "jp"
	}
	if svc != nil {
		if music, err := svc.Search(cleaned); err == nil {
			cleaned = strconv.Itoa(music.ID)
		}
	}
	query := model.MusicQuery{Query: cleaned, UserID: "test_user", Region: region}

	start := time.Now()
	imageData, err := ctrl.RenderMusicDetail(query)
	if err != nil {
		return fmt.Errorf("render music failed: %w", err)
	}
	fmt.Printf("Render music success! Took %v. Image size: %d bytes\n", time.Since(start), len(imageData))
	return saveImage("music_detail", 0, imageData)
}

func testMusicBriefList(ctrl *controller.MusicController, cmd string) error {
	raw := strings.TrimSpace(strings.TrimPrefix(cmd, "/"))
	diff := "master"
	region := "jp"
	parsedRegion, payload := extractLeadingRegionToken(raw)
	if parsedRegion != "" {
		region = parsedRegion
	}
	if strings.Contains(raw, ":") {
		parts := strings.SplitN(raw, ":", 2)
		if len(parts) == 2 {
			if strings.TrimSpace(parts[0]) != "" {
				diff = strings.TrimSpace(parts[0])
			}
			payload = strings.TrimSpace(parts[1])
			parsedRegion, trimmed := extractLeadingRegionToken(payload)
			if parsedRegion != "" {
				region = parsedRegion
			}
			payload = trimmed
		}
	}

	tokens := strings.FieldsFunc(payload, func(r rune) bool {
		return r == ',' || r == ';' || r == ' '
	})
	var ids []int
	for _, t := range tokens {
		if t == "" {
			continue
		}
		id, err := strconv.Atoi(t)
		if err != nil {
			fmt.Printf("Skip invalid id %s\n", t)
			continue
		}
		ids = append(ids, id)
	}

	if len(ids) == 0 {
		return fmt.Errorf("no valid music IDs to render")
	}

	fmt.Printf("Testing Music Brief List: diff=%s, ids=%v\n", diff, ids)
	start := time.Now()
	imageData, err := ctrl.RenderMusicBriefList(ids, diff, region)
	if err != nil {
		return fmt.Errorf("render music brief list failed: %w", err)
	}
	fmt.Printf("Render music brief list success! Took %v. Image size: %d bytes\n", time.Since(start), len(imageData))
	return saveImage("music_brief_list", 0, imageData)
}

func testMusicList(ctrl *controller.MusicController, cmd string) error {
	query, err := parseMusicListCommand(cmd)
	if err != nil {
		return fmt.Errorf("parse music list command failed: %w", err)
	}

	start := time.Now()
	imageData, err := ctrl.RenderMusicList(query)
	if err != nil {
		return fmt.Errorf("render music list failed: %w", err)
	}

	fmt.Printf("Render music list success! Took %v. Image size: %d bytes\n", time.Since(start), len(imageData))
	return saveImage("music_list", 0, imageData)
}

func testMusicProgress(ctrl *controller.MusicController, cmd string) error {
	query, err := parseMusicProgressCommand(cmd)
	if err != nil {
		return fmt.Errorf("parse music progress command failed: %w", err)
	}

	start := time.Now()
	imageData, err := ctrl.RenderMusicProgress(query)
	if err != nil {
		return fmt.Errorf("render music progress failed: %w", err)
	}

	fmt.Printf("Render music progress success! Took %v. Image size: %d bytes\n", time.Since(start), len(imageData))
	return saveImage("music_progress", 0, imageData)
}

func testMusicChart(ctrl *controller.MusicController, cmd string) error {
	query, err := parseMusicChartCommand(cmd)
	if err != nil {
		return fmt.Errorf("parse music chart command failed: %w", err)
	}
	start := time.Now()
	imageData, err := ctrl.RenderMusicChart(query)
	if err != nil {
		return fmt.Errorf("render music chart failed: %w", err)
	}
	fmt.Printf("Render music chart success! Took %v. Image size: %d bytes\n", time.Since(start), len(imageData))
	return saveImage("music_chart", 0, imageData)
}

func testMusicRewardsDetail(ctrl *controller.MusicController, cmd string) error {
	query, err := parseMusicRewardsDetailCommand(cmd)
	if err != nil {
		return fmt.Errorf("parse music rewards detail command failed: %w", err)
	}

	start := time.Now()
	imageData, err := ctrl.RenderMusicRewardsDetail(query)
	if err != nil {
		return fmt.Errorf("render music rewards detail failed: %w", err)
	}

	fmt.Printf("Render music rewards detail success! Took %v. Image size: %d bytes\n", time.Since(start), len(imageData))
	return saveImage("music_rewards_detail", 0, imageData)
}

func testMusicRewardsBasic(ctrl *controller.MusicController, cmd string) error {
	query, err := parseMusicRewardsBasicCommand(cmd)
	if err != nil {
		return fmt.Errorf("parse music rewards basic command failed: %w", err)
	}

	start := time.Now()
	imageData, err := ctrl.RenderMusicRewardsBasic(query)
	if err != nil {
		return fmt.Errorf("render music rewards basic failed: %w", err)
	}

	fmt.Printf("Render music rewards basic success! Took %v. Image size: %d bytes\n", time.Since(start), len(imageData))
	return saveImage("music_rewards_basic", 0, imageData)
}

func testGachaList(ctrl *controller.GachaController, cmd string) error {
	query := parseGachaListCommand(cmd)
	start := time.Now()
	imageData, err := ctrl.RenderGachaList(query)
	if err != nil {
		return fmt.Errorf("render gacha list failed: %w", err)
	}
	fmt.Printf("Render gacha list success! Took %v. Image size: %d bytes\n", time.Since(start), len(imageData))
	return saveImage("gacha_list", query.Page, imageData)
}

func testGachaDetail(env *cliEnv, cmd string) error {
	query, err := parseGachaDetailCommand(cmd)
	if err != nil {
		return fmt.Errorf("parse gacha detail command failed: %w", err)
	}
	if query.GachaID < 0 && env.masterdata != nil {
		gachas := env.masterdata.GetGachas()
		if len(gachas) > 0 {
			if query.GachaID == -1 && query.NegIndex > 0 {
				idx := len(gachas) - query.NegIndex
				if idx >= 0 && idx < len(gachas) {
					query.GachaID = gachas[idx].ID
				}
			} else if query.GachaID == -2 && query.EventID > 0 {
				if event, err := env.masterdata.GetEventByID(query.EventID); err == nil && event != nil {
					// Find gacha that starts around the same time
					for _, g := range gachas {
						if strings.Contains(strings.ToLower(g.Name), "it's back") || strings.Contains(strings.ToLower(g.Name), "复刻") {
							continue
						}
						if getAbsDiff(g.StartAt, event.StartAt) < int64(time.Hour*48/time.Millisecond) {
							query.GachaID = g.ID
							break
						}
					}
				}
			}
		}
	}

	start := time.Now()
	imageData, err := env.gachaController.RenderGachaDetail(query)
	if err != nil {
		return fmt.Errorf("render gacha detail failed: %w", err)
	}
	fmt.Printf("Render gacha detail success! Took %v. Image size: %d bytes\n", time.Since(start), len(imageData))
	return saveImage("gacha_detail", query.GachaID, imageData)
}

func getAbsDiff(a, b int64) int64 {
	if a > b {
		return a - b
	}
	return b - a
}

func testEventDetail(ctrl *controller.EventController, search *service.EventSearchService, cmd string) error {
	raw := preprocessEventCommand(cmd)
	region, cleaned := extractLeadingRegionToken(raw)
	if region == "" {
		region = "jp"
	}
	raw = cleaned
	if raw == "" {
		raw = "current"
	}
	event, err := search.Search(raw)
	if err != nil {
		return fmt.Errorf("failed to find event: %w", err)
	}
	query := model.EventDetailQuery{
		Region:  region,
		EventID: event.ID,
	}
	start := time.Now()
	imageData, err := ctrl.RenderEventDetail(context.Background(), query)
	if err != nil {
		return fmt.Errorf("render event detail failed: %w", err)
	}
	fmt.Printf("Render event detail success! Took %v. Image size: %d bytes\n", time.Since(start), len(imageData))
	return saveImage("event_detail", event.ID, imageData)
}

func testEventList(ctrl *controller.EventController, parser *service.EventParser, cmd string) error {
	query, err := parseEventListCommand(cmd, parser)
	if err != nil {
		return fmt.Errorf("parse event list command failed: %w", err)
	}
	start := time.Now()
	imageData, err := ctrl.RenderEventList(query)
	if err != nil {
		return fmt.Errorf("render event list failed: %w", err)
	}
	fmt.Printf("Render event list success! Took %v. Image size: %d bytes\n", time.Since(start), len(imageData))
	return saveImage("event_list", 0, imageData)
}

func testEventRecord(ctrl *controller.EventController, cmd string) error {
	raw := strings.TrimSpace(cmd)
	var req model.EventRecordRequest
	if err := loadQueryFromFile(raw, &req); err != nil {
		return err
	}
	start := time.Now()
	imageData, err := ctrl.RenderEventRecord(req)
	if err != nil {
		return fmt.Errorf("render event record failed: %w", err)
	}
	fmt.Printf("Render event record success! Took %v. Image size: %d bytes\n", time.Since(start), len(imageData))
	return saveImage("event_record", 0, imageData)
}

func testEducationChallengeLive(ctrl *controller.EducationController, cmd string) error {
	raw := strings.TrimSpace(cmd)
	start := time.Now()
	var (
		imageData []byte
		err       error
	)
	switch determineEducationInputType(raw) {
	case educationInputJSON:
		var req model.ChallengeLiveDetailsRequest
		if err := loadQueryFromFile(strings.TrimSpace(raw), &req); err != nil {
			return err
		}
		imageData, err = ctrl.RenderChallengeLiveDetail(req)
	case educationInputRegion:
		region := parseEducationRegion(raw)
		imageData, err = ctrl.RenderChallengeLiveDetailFromUser(region)
	default:
		imageData, err = ctrl.RenderChallengeLiveDetailFromUser("jp")
	}
	if err != nil {
		return fmt.Errorf("render education challenge failed: %w", err)
	}
	fmt.Printf("Render education challenge success! Took %v. Image size: %d bytes\n", time.Since(start), len(imageData))
	return saveImage("education_challenge", 0, imageData)
}

func testEducationPowerBonus(ctrl *controller.EducationController, cmd string) error {
	var req model.PowerBonusDetailRequest
	if err := loadQueryFromFile(strings.TrimSpace(cmd), &req); err != nil {
		return err
	}
	start := time.Now()
	imageData, err := ctrl.RenderPowerBonusDetail(req)
	if err != nil {
		return fmt.Errorf("render education power bonus failed: %w", err)
	}
	fmt.Printf("Render education power bonus success! Took %v. Image size: %d bytes\n", time.Since(start), len(imageData))
	return saveImage("education_power_bonus", 0, imageData)
}

func testEducationAreaItem(ctrl *controller.EducationController, cmd string) error {
	var req model.AreaItemUpgradeMaterialsRequest
	if err := loadQueryFromFile(strings.TrimSpace(cmd), &req); err != nil {
		return err
	}
	start := time.Now()
	imageData, err := ctrl.RenderAreaItemMaterials(req)
	if err != nil {
		return fmt.Errorf("render education area item failed: %w", err)
	}
	fmt.Printf("Render education area item success! Took %v. Image size: %d bytes\n", time.Since(start), len(imageData))
	return saveImage("education_area_item", 0, imageData)
}

func testEducationBonds(ctrl *controller.EducationController, cmd string) error {
	var req model.BondsRequest
	if err := loadQueryFromFile(strings.TrimSpace(cmd), &req); err != nil {
		return err
	}
	start := time.Now()
	imageData, err := ctrl.RenderBonds(req)
	if err != nil {
		return fmt.Errorf("render education bonds failed: %w", err)
	}
	fmt.Printf("Render education bonds success! Took %v. Image size: %d bytes\n", time.Since(start), len(imageData))
	return saveImage("education_bonds", 0, imageData)
}

func testEducationLeaderCount(ctrl *controller.EducationController, cmd string) error {
	var req model.LeaderCountRequest
	if err := loadQueryFromFile(strings.TrimSpace(cmd), &req); err != nil {
		return err
	}
	start := time.Now()
	imageData, err := ctrl.RenderLeaderCount(req)
	if err != nil {
		return fmt.Errorf("render education leader count failed: %w", err)
	}
	fmt.Printf("Render education leader count success! Took %v. Image size: %d bytes\n", time.Since(start), len(imageData))
	return saveImage("education_leader_count", 0, imageData)
}

func testHonorGenerate(ctrl *controller.HonorController, cmd string) error {
	raw := strings.TrimSpace(cmd)
	var query model.HonorQuery
	if err := loadQueryFromFile(raw, &query); err != nil {
		return err
	}
	req, err := ctrl.BuildHonorRequest(query)
	if err != nil {
		return fmt.Errorf("build honor request failed: %w", err)
	}
	start := time.Now()
	imageData, err := ctrl.RenderHonorImage(req)
	if err != nil {
		return fmt.Errorf("render honor failed: %w", err)
	}
	fmt.Printf("Render honor success! Took %v. Image size: %d bytes\n", time.Since(start), len(imageData))
	return saveImage("honor", 0, imageData)
}

func testProfileGenerate(ctrl *controller.ProfileController, userData *service.UserDataService, cmd string) error {
	userID := strings.TrimSpace(cmd)
	if userID == "" {
		userID = "1"
	}
	region := "jp" // 默认使用 JP 区服

	start := time.Now()
	imageData, err := ctrl.RenderProfile(userID, region, userData)
	if err != nil {
		return fmt.Errorf("render profile failed: %w", err)
	}
	fmt.Printf("Render profile success! Took %v. Image size: %d bytes\n", time.Since(start), len(imageData))
	return saveImage("profile", 0, imageData)
}

func testSKLine(ctrl *controller.SkController, cmd string) error {
	var req map[string]interface{}
	if err := loadQueryFromFile(strings.TrimSpace(cmd), &req); err != nil {
		return err
	}
	start := time.Now()
	imageData, err := ctrl.RenderLine(req)
	if err != nil {
		return fmt.Errorf("render sk-line failed: %w", err)
	}
	fmt.Printf("Render sk-line success! Took %v. Image size: %d bytes\n", time.Since(start), len(imageData))
	return saveImage("sk_line", 0, imageData)
}

func testStampList(ctrl *controller.StampController, cmd string) error {
	var query model.StampListQuery
	if err := loadQueryFromFile(strings.TrimSpace(cmd), &query); err != nil {
		return err
	}
	req, err := ctrl.BuildStampListRequest(query)
	if err != nil {
		return fmt.Errorf("build stamp-list request failed: %w", err)
	}
	start := time.Now()
	imageData, err := ctrl.RenderStampList(req)
	if err != nil {
		return fmt.Errorf("render stamp-list failed: %w", err)
	}
	fmt.Printf("Render stamp-list success! Took %v. Image size: %d bytes\n", time.Since(start), len(imageData))
	return saveImage("stamp_list", 0, imageData)
}

func testMiscCharaBirthday(ctrl *controller.MiscController, cmd string) error {
	var query model.CharaBirthdayRequest
	if err := loadQueryFromFile(strings.TrimSpace(cmd), &query); err != nil {
		return err
	}
	req, err := ctrl.BuildCharaBirthdayRequest(query)
	if err != nil {
		return fmt.Errorf("build misc-chara-birthday request failed: %w", err)
	}
	start := time.Now()
	imageData, err := ctrl.RenderCharaBirthday(req)
	if err != nil {
		return fmt.Errorf("render misc-chara-birthday failed: %w", err)
	}
	fmt.Printf("Render misc-chara-birthday success! Took %v. Image size: %d bytes\n", time.Since(start), len(imageData))
	return saveImage("misc_chara_birthday", 0, imageData)
}

func testScoreControl(ctrl *controller.ScoreController, cmd string) error {
	var query model.ScoreControlRequest
	if err := loadQueryFromFile(strings.TrimSpace(cmd), &query); err != nil {
		return err
	}
	req, err := ctrl.BuildScoreControlRequest(query)
	if err != nil {
		return fmt.Errorf("build score-control request failed: %w", err)
	}
	start := time.Now()
	imageData, err := ctrl.RenderScoreControl(req)
	if err != nil {
		return fmt.Errorf("render score-control failed: %w", err)
	}
	fmt.Printf("Render score-control success! Took %v. Image size: %d bytes\n", time.Since(start), len(imageData))
	return saveImage("score_control", 0, imageData)
}

func testScoreCustomRoom(ctrl *controller.ScoreController, cmd string) error {
	var query model.CustomRoomScoreRequest
	if err := loadQueryFromFile(strings.TrimSpace(cmd), &query); err != nil {
		return err
	}
	req, err := ctrl.BuildCustomRoomScoreRequest(query)
	if err != nil {
		return fmt.Errorf("build score-custom-room request failed: %w", err)
	}
	start := time.Now()
	imageData, err := ctrl.RenderCustomRoomScore(req)
	if err != nil {
		return fmt.Errorf("render score-custom-room failed: %w", err)
	}
	fmt.Printf("Render score-custom-room success! Took %v. Image size: %d bytes\n", time.Since(start), len(imageData))
	return saveImage("score_custom_room", 0, imageData)
}

func testScoreMusicMeta(ctrl *controller.ScoreController, cmd string) error {
	var query []model.MusicMetaRequest
	if err := loadQueryFromFile(strings.TrimSpace(cmd), &query); err != nil {
		return err
	}
	req, err := ctrl.BuildMusicMetaRequest(query)
	if err != nil {
		return fmt.Errorf("build score-music-meta request failed: %w", err)
	}
	start := time.Now()
	imageData, err := ctrl.RenderMusicMeta(req)
	if err != nil {
		return fmt.Errorf("render score-music-meta failed: %w", err)
	}
	fmt.Printf("Render score-music-meta success! Took %v. Image size: %d bytes\n", time.Since(start), len(imageData))
	return saveImage("score_music_meta", 0, imageData)
}

func testScoreMusicBoard(ctrl *controller.ScoreController, cmd string) error {
	var query model.MusicBoardRequest
	if err := loadQueryFromFile(strings.TrimSpace(cmd), &query); err != nil {
		return err
	}
	req, err := ctrl.BuildMusicBoardRequest(query)
	if err != nil {
		return fmt.Errorf("build score-music-board request failed: %w", err)
	}
	start := time.Now()
	imageData, err := ctrl.RenderMusicBoard(req)
	if err != nil {
		return fmt.Errorf("render score-music-board failed: %w", err)
	}
	fmt.Printf("Render score-music-board success! Took %v. Image size: %d bytes\n", time.Since(start), len(imageData))
	return saveImage("score_music_board", 0, imageData)
}

func testDeckRecommend(ctrl *controller.DeckController, cmd string) error {
	var req map[string]interface{}
	if err := loadQueryFromFile(strings.TrimSpace(cmd), &req); err != nil {
		return err
	}
	start := time.Now()
	imageData, err := ctrl.RenderDeckRecommend(req)
	if err != nil {
		return fmt.Errorf("render deck-recommend failed: %w", err)
	}
	fmt.Printf("Render deck-recommend success! Took %v. Image size: %d bytes\n", time.Since(start), len(imageData))
	return saveImage("deck_recommend", 0, imageData)
}

func testDeckRecommendAuto(ctrl *controller.DeckController, cmd string) error {
	var query model.DeckAutoQuery
	if err := loadQueryFromFile(strings.TrimSpace(cmd), &query); err != nil {
		return err
	}
	query.RecommendType = normalizeDeckAutoType(query.RecommendType)
	if strings.TrimSpace(query.RecommendType) == "" {
		var fallback map[string]interface{}
		if err := loadQueryFromFile(strings.TrimSpace(cmd), &fallback); err == nil {
			if region, ok := fallback["region"].(string); ok && strings.TrimSpace(query.Region) == "" {
				query.Region = strings.TrimSpace(region)
			}
			if rt, ok := fallback["recommend_type"].(string); ok {
				query.RecommendType = normalizeDeckAutoType(rt)
			}
			if ev, ok := fallback["event_id"].(float64); ok {
				id := int(ev)
				if id > 0 {
					query.EventID = &id
				}
			}
		}
	}
	if strings.TrimSpace(query.RecommendType) == "" {
		query.RecommendType = "event"
	}
	start := time.Now()
	imageData, err := ctrl.RenderDeckRecommendAuto(query)
	if err != nil {
		return fmt.Errorf("render deck-recommend-auto failed: %w", err)
	}
	fmt.Printf("Render deck-recommend-auto success! Took %v. Image size: %d bytes\n", time.Since(start), len(imageData))
	return saveImage("deck_recommend_auto", 0, imageData)
}

func testSKQuery(ctrl *controller.SkController, cmd string) error {
	var req map[string]interface{}
	if err := loadQueryFromFile(strings.TrimSpace(cmd), &req); err != nil {
		return err
	}
	start := time.Now()
	imageData, err := ctrl.RenderQuery(req)
	if err != nil {
		return fmt.Errorf("render sk-query failed: %w", err)
	}
	fmt.Printf("Render sk-query success! Took %v. Image size: %d bytes\n", time.Since(start), len(imageData))
	return saveImage("sk_query", 0, imageData)
}

func testSKCheckRoom(ctrl *controller.SkController, cmd string) error {
	var req map[string]interface{}
	if err := loadQueryFromFile(strings.TrimSpace(cmd), &req); err != nil {
		return err
	}
	start := time.Now()
	imageData, err := ctrl.RenderCheckRoom(req)
	if err != nil {
		return fmt.Errorf("render sk-check-room failed: %w", err)
	}
	fmt.Printf("Render sk-check-room success! Took %v. Image size: %d bytes\n", time.Since(start), len(imageData))
	return saveImage("sk_check_room", 0, imageData)
}

func testSKSpeed(ctrl *controller.SkController, cmd string) error {
	var req map[string]interface{}
	if err := loadQueryFromFile(strings.TrimSpace(cmd), &req); err != nil {
		return err
	}
	start := time.Now()
	imageData, err := ctrl.RenderSpeed(req)
	if err != nil {
		return fmt.Errorf("render sk-speed failed: %w", err)
	}
	fmt.Printf("Render sk-speed success! Took %v. Image size: %d bytes\n", time.Since(start), len(imageData))
	return saveImage("sk_speed", 0, imageData)
}

func testSKPlayerTrace(ctrl *controller.SkController, cmd string) error {
	var req map[string]interface{}
	if err := loadQueryFromFile(strings.TrimSpace(cmd), &req); err != nil {
		return err
	}
	start := time.Now()
	imageData, err := ctrl.RenderPlayerTrace(req)
	if err != nil {
		return fmt.Errorf("render sk-player-trace failed: %w", err)
	}
	fmt.Printf("Render sk-player-trace success! Took %v. Image size: %d bytes\n", time.Since(start), len(imageData))
	return saveImage("sk_player_trace", 0, imageData)
}

func testSKRankTrace(ctrl *controller.SkController, cmd string) error {
	var req map[string]interface{}
	if err := loadQueryFromFile(strings.TrimSpace(cmd), &req); err != nil {
		return err
	}
	start := time.Now()
	imageData, err := ctrl.RenderRankTrace(req)
	if err != nil {
		return fmt.Errorf("render sk-rank-trace failed: %w", err)
	}
	fmt.Printf("Render sk-rank-trace success! Took %v. Image size: %d bytes\n", time.Since(start), len(imageData))
	return saveImage("sk_rank_trace", 0, imageData)
}

func testSKWinrate(ctrl *controller.SkController, cmd string) error {
	var req map[string]interface{}
	if err := loadQueryFromFile(strings.TrimSpace(cmd), &req); err != nil {
		return err
	}
	start := time.Now()
	imageData, err := ctrl.RenderWinrate(req)
	if err != nil {
		return fmt.Errorf("render sk-winrate failed: %w", err)
	}
	fmt.Printf("Render sk-winrate success! Took %v. Image size: %d bytes\n", time.Since(start), len(imageData))
	return saveImage("sk_winrate", 0, imageData)
}

func testMysekaiResource(ctrl *controller.MysekaiController, cmd string) error {
	var req interface{}
	if err := loadQueryFromFile(strings.TrimSpace(cmd), &req); err != nil {
		return err
	}
	start := time.Now()
	imageData, err := ctrl.RenderResource(req)
	if err != nil {
		return fmt.Errorf("render mysekai-resource failed: %w", err)
	}
	fmt.Printf("Render mysekai-resource success! Took %v. Image size: %d bytes\n", time.Since(start), len(imageData))
	return saveImage("mysekai_resource", 0, imageData)
}

func testMysekaiFixtureList(ctrl *controller.MysekaiController, cmd string) error {
	var req interface{}
	if err := loadQueryFromFile(strings.TrimSpace(cmd), &req); err != nil {
		return err
	}
	start := time.Now()
	imageData, err := ctrl.RenderFixtureList(req)
	if err != nil {
		return fmt.Errorf("render mysekai-fixture-list failed: %w", err)
	}
	fmt.Printf("Render mysekai-fixture-list success! Took %v. Image size: %d bytes\n", time.Since(start), len(imageData))
	return saveImage("mysekai_fixture_list", 0, imageData)
}

func testMysekaiFixtureDetail(ctrl *controller.MysekaiController, cmd string) error {
	var req interface{}
	if err := loadQueryFromFile(strings.TrimSpace(cmd), &req); err != nil {
		return err
	}
	start := time.Now()
	imageData, err := ctrl.RenderFixtureDetail(req)
	if err != nil {
		return fmt.Errorf("render mysekai-fixture-detail failed: %w", err)
	}
	fmt.Printf("Render mysekai-fixture-detail success! Took %v. Image size: %d bytes\n", time.Since(start), len(imageData))
	return saveImage("mysekai_fixture_detail", 0, imageData)
}

func testMysekaiDoorUpgrade(ctrl *controller.MysekaiController, cmd string) error {
	var req interface{}
	if err := loadQueryFromFile(strings.TrimSpace(cmd), &req); err != nil {
		return err
	}
	start := time.Now()
	imageData, err := ctrl.RenderDoorUpgrade(req)
	if err != nil {
		return fmt.Errorf("render mysekai-door-upgrade failed: %w", err)
	}
	fmt.Printf("Render mysekai-door-upgrade success! Took %v. Image size: %d bytes\n", time.Since(start), len(imageData))
	return saveImage("mysekai_door_upgrade", 0, imageData)
}

func testMysekaiMusicRecord(ctrl *controller.MysekaiController, cmd string) error {
	var req interface{}
	if err := loadQueryFromFile(strings.TrimSpace(cmd), &req); err != nil {
		return err
	}
	start := time.Now()
	imageData, err := ctrl.RenderMusicRecord(req)
	if err != nil {
		return fmt.Errorf("render mysekai-music-record failed: %w", err)
	}
	fmt.Printf("Render mysekai-music-record success! Took %v. Image size: %d bytes\n", time.Since(start), len(imageData))
	return saveImage("mysekai_music_record", 0, imageData)
}

func testMysekaiTalkList(ctrl *controller.MysekaiController, cmd string) error {
	var req interface{}
	if err := loadQueryFromFile(strings.TrimSpace(cmd), &req); err != nil {
		return err
	}
	start := time.Now()
	imageData, err := ctrl.RenderTalkList(req)
	if err != nil {
		return fmt.Errorf("render mysekai-talk-list failed: %w", err)
	}
	fmt.Printf("Render mysekai-talk-list success! Took %v. Image size: %d bytes\n", time.Since(start), len(imageData))
	return saveImage("mysekai_talk_list", 0, imageData)
}

func testCardListHardcoded(ctrl *controller.CardController) error {
	ids := []int{190, 1252, 1309, 17}
	region := "jp"
	start := time.Now()
	imageData, err := ctrl.RenderCardListFromIDs(ids, region)
	if err != nil {
		return fmt.Errorf("render list failed: %w", err)
	}
	fmt.Printf("Render list success! Took %v. Image size: %d bytes\n", time.Since(start), len(imageData))
	return saveImage("card_list_hardcoded", 0, imageData)
}

func testCardListDynamic(ctrl *controller.CardController, cmd string) error {
	raw := preprocessCommand(cmd, "/查卡", "查卡", "查牌", "查卡片", "查询卡片")
	raw = stripLeadingRegionToken(raw)
	queries := []model.CardQuery{{Query: raw, UserID: "test_user"}}

	start := time.Now()
	imageData, err := ctrl.RenderCardList(queries)
	if err != nil {
		return fmt.Errorf("render list failed: %w", err)
	}
	fmt.Printf("Render list success! Took %v. Image size: %d bytes\n", time.Since(start), len(imageData))
	return saveImage("card_list_search", 0, imageData)
}

func testCardBox(ctrl *controller.CardController, cmd string) error {
	raw := preprocessCommand(cmd, "/查卡", "查卡", "查牌", "查卡片", "查询卡片")
	raw = stripLeadingRegionToken(raw)
	queries := []model.CardQuery{{Query: raw, UserID: "test_user"}}

	start := time.Now()
	imageData, err := ctrl.RenderCardBox(queries)
	if err != nil {
		return fmt.Errorf("render card box failed: %w", err)
	}
	fmt.Printf("Render card box success! Took %v. Image size: %d bytes\n", time.Since(start), len(imageData))
	return saveImage("card_box", 0, imageData)
}

func testCardDetail(ctrl *controller.CardController, parser *service.CardParser, cmd string) error {
	raw := preprocessCommand(cmd, "/查卡", "查卡", "查牌", "查卡片", "查询卡片")
	raw = stripLeadingRegionToken(raw)
	fmt.Printf("Processing command: '%s'\n", raw)

	if _, err := parser.Parse(raw); err != nil {
		return fmt.Errorf("parser failed: %w", err)
	}

	start := time.Now()
	imageData, err := ctrl.RenderCardDetail(model.CardQuery{Query: raw, UserID: "test_user"})
	if err != nil {
		return fmt.Errorf("render failed: %w", err)
	}
	fmt.Printf("Render success! Took %v. Image size: %d bytes\n", time.Since(start), len(imageData))
	return saveImage("card_detail", 0, imageData)
}

func saveImage(prefix string, id int, data []byte) error {
	outputDir := globalOutputDir
	if outputDir == "" {
		outputDir = "D:/github/testfile"
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return fmt.Errorf("failed to create output dir: %w", err)
	}

	filename := fmt.Sprintf("%s_%s_%d.png", prefix, time.Now().Format("20060102_150405"), id)
	outputPath := filepath.Join(outputDir, filename)
	if err := os.WriteFile(outputPath, data, 0o644); err != nil {
		fallback := filepath.Join(os.TempDir(), filename)
		if writeErr := os.WriteFile(fallback, data, 0o644); writeErr != nil {
			return fmt.Errorf("failed to write image: %w (fallback: %v)", err, writeErr)
		}
		fmt.Printf("Image saved to fallback path: %s\n", fallback)
		return nil
	}

	fmt.Printf("Image saved to: %s\n", outputPath)
	return nil
}

func parseMusicProgressCommand(cmd string) (model.MusicProgressQuery, error) {
	raw := strings.TrimSpace(cmd)
	if strings.HasSuffix(strings.ToLower(raw), ".json") {
		var query model.MusicProgressQuery
		if err := loadQueryFromFile(raw, &query); err != nil {
			return model.MusicProgressQuery{}, err
		}
		if query.Difficulty == "" {
			query.Difficulty = "master"
		}
		if query.Region == "" {
			query.Region = "jp"
		}
		return query, nil
	}

	raw = strings.TrimSpace(strings.TrimPrefix(raw, "/"))
	raw = strings.Replace(raw, "谱面进度", "", 1)
	raw = strings.Replace(raw, "查谱面进度", "", 1)
	raw = strings.Replace(raw, "打歌进度", "", 1)
	raw = strings.Replace(raw, "pjsk进度", "", 1)
	region := "jp"
	fields := strings.Fields(raw)
	if len(fields) > 0 {
		if parsedRegion, ok := map[string]bool{"jp": true, "en": true, "cn": true, "tw": true, "kr": true}[strings.ToLower(fields[0])]; ok && parsedRegion {
			region = strings.ToLower(fields[0])
			fields = fields[1:]
		}
	}
	diff := "master"
	if len(fields) > 0 {
		if normalized, ok := normalizeDifficultyAlias(fields[0]); ok {
			diff = normalized
		}
	}

	return model.MusicProgressQuery{
		Difficulty: diff,
		Region:     region,
	}, nil
}

func parseMusicChartCommand(cmd string) (model.MusicChartQuery, error) {
	raw := preprocessCommand(cmd, "/谱面预览", "谱面预览", "查谱面", "查谱", "谱面", "查谱图", "chart")
	region, cleaned := extractLeadingRegionToken(raw)
	if region == "" {
		region = "jp"
	}
	raw = cleaned
	if raw == "" {
		return model.MusicChartQuery{}, fmt.Errorf("please provide music keyword")
	}
	diff := "master"
	skill := false
	style := ""
	var terms []string
	for _, token := range strings.Fields(raw) {
		if normalized, ok := normalizeDifficultyAlias(token); ok {
			diff = normalized
			continue
		}
		lt := strings.ToLower(token)
		switch lt {
		case "skill", "技能", "withskill":
			skill = true
			continue
		}
		if strings.HasPrefix(lt, "style=") {
			style = strings.TrimPrefix(token, "style=")
			continue
		}
		terms = append(terms, token)
	}
	if len(terms) == 0 {
		return model.MusicChartQuery{}, fmt.Errorf("music keyword missing")
	}
	return model.MusicChartQuery{
		Query:      strings.Join(terms, " "),
		Region:     region,
		Difficulty: diff,
		Skill:      skill,
		Style:      style,
	}, nil
}

func parseMusicRewardsDetailCommand(cmd string) (model.MusicRewardsDetailQuery, error) {
	raw := strings.TrimSpace(cmd)
	if !strings.HasSuffix(strings.ToLower(raw), ".json") {
		return model.MusicRewardsDetailQuery{}, fmt.Errorf("please provide a JSON file path for rewards detail data")
	}
	var query model.MusicRewardsDetailQuery
	if err := loadQueryFromFile(raw, &query); err != nil {
		return model.MusicRewardsDetailQuery{}, err
	}
	if query.Region == "" {
		query.Region = "jp"
	}
	query.ComboRewards = ensureDetailComboRewardsMap(query.ComboRewards)
	return query, nil
}

func parseMusicRewardsBasicCommand(cmd string) (model.MusicRewardsBasicQuery, error) {
	raw := strings.TrimSpace(cmd)
	if !strings.HasSuffix(strings.ToLower(raw), ".json") {
		return model.MusicRewardsBasicQuery{}, fmt.Errorf("please provide a JSON file path for rewards basic data")
	}
	var query model.MusicRewardsBasicQuery
	if err := loadQueryFromFile(raw, &query); err != nil {
		return model.MusicRewardsBasicQuery{}, err
	}
	if query.Region == "" {
		query.Region = "jp"
	}
	if query.ComboRewards == nil {
		query.ComboRewards = map[string]string{
			"hard":   "0",
			"expert": "0",
			"master": "0",
			"append": "0",
		}
	}
	return query, nil
}

func parseMusicListCommand(cmd string) (model.MusicListQuery, error) {
	raw := strings.TrimSpace(strings.TrimPrefix(cmd, "/"))
	replacements := []string{
		"pjsk song list", "pjsk music list", "pjsk music constant",
		"乐曲列表", "乐曲一览", "难度排行", "查乐曲",
	}
	lower := strings.ToLower(raw)
	for _, rep := range replacements {
		lower = strings.ReplaceAll(lower, strings.ToLower(rep), "")
	}

	includeLeaks := strings.Contains(lower, "leak")
	if includeLeaks {
		lower = strings.ReplaceAll(lower, "leak", "")
	}

	tokens := strings.Fields(lower)
	region := "jp"
	if len(tokens) > 0 {
		if _, ok := map[string]struct{}{"jp": {}, "en": {}, "cn": {}, "tw": {}, "kr": {}}[tokens[0]]; ok {
			region = tokens[0]
			tokens = tokens[1:]
		}
	}
	diff := "master"
	if len(tokens) > 0 {
		if normalized, ok := normalizeDifficultyAlias(tokens[0]); ok {
			diff = normalized
			tokens = tokens[1:]
		}
	}

	var levels []int
	for _, token := range tokens {
		if n, err := strconv.Atoi(token); err == nil {
			levels = append(levels, n)
		}
	}

	query := model.MusicListQuery{
		Difficulty:   diff,
		Region:       region,
		IncludeLeaks: includeLeaks,
	}

	switch len(levels) {
	case 1:
		query.Level = levels[0]
	case 2:
		query.LevelMin = levels[0]
		query.LevelMax = levels[1]
	}

	return query, nil
}

func normalizeDifficultyAlias(token string) (string, bool) {
	token = strings.TrimSpace(strings.ToLower(token))
	switch token {
	case "easy", "ez":
		return "easy", true
	case "normal", "nm":
		return "normal", true
	case "hard", "hd":
		return "hard", true
	case "expert", "exp", "ex":
		return "expert", true
	case "master", "mas", "ma":
		return "master", true
	case "append", "apd":
		return "append", true
	default:
		return "", false
	}
}

func loadQueryFromFile(path string, target interface{}) error {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return err
	}
	data = bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF})
	return json.Unmarshal(data, target)
}

func normalizeDeckAutoType(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "event_pt", "event":
		return "event"
	case "bonus", "event_bonus":
		return "bonus"
	case "challenge", "no_event", "mysekai":
		return strings.ToLower(strings.TrimSpace(raw))
	default:
		return ""
	}
}

func ensureDetailComboRewardsMap(combo map[string][]model.MusicComboReward) map[string][]model.MusicComboReward {
	if combo == nil {
		combo = make(map[string][]model.MusicComboReward)
	}
	for _, diff := range []string{"hard", "expert", "master", "append"} {
		if _, ok := combo[diff]; !ok {
			combo[diff] = []model.MusicComboReward{}
		}
	}
	return combo
}

func parseGachaListCommand(cmd string) model.GachaListQuery {
	query := model.GachaListQuery{
		Region:      "jp",
		Page:        1,
		PageSize:    6,
		IncludePast: true,
	}

	raw := strings.TrimSpace(strings.TrimPrefix(cmd, "/"))
	replacements := []string{
		"pjsk gacha", "卡池列表", "卡池一览", "查卡池", "卡池",
	}
	lower := strings.ToLower(raw)
	for _, rep := range replacements {
		lower = strings.ReplaceAll(lower, strings.ToLower(rep), "")
	}
	if parsedRegion, rest := extractLeadingRegionToken(lower); parsedRegion != "" {
		query.Region = parsedRegion
		lower = rest
	}

	for _, token := range strings.Fields(lower) {
		t := strings.TrimSpace(token)
		if t == "" {
			continue
		}
		lt := strings.ToLower(t)
		switch {
		case strings.HasPrefix(lt, "p"):
			if val, err := strconv.Atoi(strings.TrimPrefix(lt, "p")); err == nil && val > 0 {
				query.Page = val
			}
		case lt == "leak":
			query.IncludeFuture = true
		case lt == "当前" || lt == "current":
			query.OnlyCurrent = true
			query.IncludeFuture = false
			query.IncludePast = false
		case lt == "复刻" || lt == "rerelease" || lt == "back":
			query.IsRerelease = true
		case lt == "回响" || lt == "recall":
			query.IsRecall = true
		case lt == "past":
			query.IncludePast = true
		case lt == "nopast":
			query.IncludePast = false
		case strings.HasPrefix(lt, "card"):
			if val, err := strconv.Atoi(lt[4:]); err == nil {
				query.CardID = val
			}
		default:
			if val, err := strconv.Atoi(lt); err == nil {
				if val >= 2000 && val <= 2100 {
					query.Year = val
				} else {
					query.CardID = val
				}
			} else {
				query.Keyword = strings.TrimSpace(t)
			}
		}
	}
	return query
}

func parseGachaDetailCommand(cmd string) (model.GachaDetailQuery, error) {
	raw := strings.TrimSpace(strings.TrimPrefix(cmd, "/"))
	replacements := []string{
		"卡池",
		"查卡池",
		"抽卡",
		"查看卡池",
		"gacha",
		"gacha-detail",
		"gachadetail",
		"pool",
		"banner",
	}
	for _, rep := range replacements {
		if rep == "" {
			continue
		}
		raw = strings.ReplaceAll(raw, rep, "")
	}
	raw = strings.TrimSpace(raw)
	region, cleaned := extractLeadingRegionToken(raw)
	if region == "" {
		region = "jp"
	}
	raw = cleaned
	if strings.HasPrefix(raw, "-") {
		if idx, err := strconv.Atoi(raw); err == nil && idx < 0 {
			return model.GachaDetailQuery{
				Region:   region,
				GachaID:  -1,
				NegIndex: -idx,
			}, nil
		}
	}
	if strings.HasPrefix(raw, "event") {
		if eid, err := strconv.Atoi(raw[5:]); err == nil {
			return model.GachaDetailQuery{
				Region:  region,
				GachaID: -2,
				EventID: eid,
			}, nil
		}
	}
	id, err := strconv.Atoi(raw)
	if err != nil {
		return model.GachaDetailQuery{}, fmt.Errorf("invalid gacha id: %s", raw)
	}
	return model.GachaDetailQuery{
		Region:  region,
		GachaID: id,
	}, nil
}

func parseEventListCommand(cmd string, parser *service.EventParser) (model.EventListQuery, error) {
	query := model.EventListQuery{
		Region:        "jp",
		IncludePast:   true,
		IncludeFuture: true,
		Limit:         6,
	}
	raw := preprocessEventCommand(cmd)
	if parsedRegion, rest := extractLeadingRegionToken(raw); parsedRegion != "" {
		query.Region = parsedRegion
		raw = rest
	}
	if raw == "" {
		return query, nil
	}
	tokens := strings.Fields(raw)
	var filtered []string
	for _, token := range tokens {
		lower := strings.ToLower(token)
		switch {
		case lower == "list" || lower == "列表" || lower == "一览":
			continue
		case lower == "past":
			query.IncludePast = true
		case lower == "future":
			query.IncludeFuture = true
		case lower == "onlyfuture":
			query.OnlyFuture = true
			query.IncludeFuture = true
			query.IncludePast = false
		case strings.HasPrefix(lower, "limit"):
			if v, err := strconv.Atoi(strings.TrimPrefix(lower, "limit")); err == nil && v > 0 {
				query.Limit = v
			}
		default:
			filtered = append(filtered, token)
		}
	}
	filteredRaw := strings.TrimSpace(strings.Join(filtered, " "))
	if filteredRaw == "" {
		return query, nil
	}
	info, err := parser.Parse(filteredRaw)
	if err != nil {
		return query, err
	}
	if info.Type != service.QueryTypeEventFilter {
		return query, fmt.Errorf("please provide filters like 'wl' or '25h 2024'")
	}
	query.EventType = info.Filter.EventType
	query.Unit = info.Filter.Unit
	query.Attr = info.Filter.Attr
	query.Year = info.Filter.Year
	query.CharacterID = info.Filter.CharacterID
	return query, nil
}

func preprocessEventCommand(cmd string) string {
	cmd = strings.TrimSpace(strings.TrimPrefix(cmd, "/"))
	replacements := []string{
		"event-list",
		"event",
		"events",
		"pjsk event",
		"pjsk events",
		"活动",
		"查活动",
		"活動",
		"查活動",
		"活动列表",
		"活动详情",
		"查活动列表",
	}
	lower := strings.ToLower(cmd)
	for _, rep := range replacements {
		if rep == "" {
			continue
		}
		lower = strings.ReplaceAll(lower, strings.ToLower(rep), "")
	}
	return strings.TrimSpace(lower)
}

type educationInputKind int

const (
	educationInputAuto educationInputKind = iota
	educationInputJSON
	educationInputRegion
)

func determineEducationInputType(cmd string) educationInputKind {
	raw := strings.TrimSpace(cmd)
	if raw == "" {
		return educationInputAuto
	}
	lower := strings.ToLower(raw)
	if strings.HasSuffix(lower, ".json") {
		return educationInputJSON
	}
	normalized := parseEducationRegion(raw)
	if normalized == "" {
		return educationInputAuto
	}
	return educationInputRegion
}

func parseEducationRegion(cmd string) string {
	raw := preprocessCommand(cmd,
		"/education-challenge", "education-challenge",
		"/education", "education",
		"/challenge", "challenge",
		"/挑战信息", "挑战信息",
		"/挑战", "挑战",
		"/教育挑战", "教育挑战",
	)
	return strings.ToLower(strings.TrimSpace(raw))
}
