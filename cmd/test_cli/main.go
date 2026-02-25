package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"Haruki-Service-API/internal/config"
	"Haruki-Service-API/internal/controller"
	"Haruki-Service-API/internal/model"
	"Haruki-Service-API/internal/service"
	"Haruki-Service-API/pkg/asset"
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
	cardParser          *service.CardParser
	eventParser         *service.EventParser
	eventSearch         *service.EventSearchService
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
	modePtr := flag.String("mode", "auto", "Mode: auto/detail/card-detail, card-list, music (detail), music-brief, music-list, music-progress, music-chart, music-reward-detail, music-reward-basic, gacha-list, gacha-detail, event-detail, event-list, event-record, education-* (challenge/power/area/bonds/leader), honor, profile")
	cmdPtr := flag.String("cmd", "", "Command payload, e.g. '/查卡 190'")
	scenarioPtr := flag.String("scenario", "", "Run multiple commands; use 'all' for built-in regression or provide a JSON file path")
	flag.Parse()

	configPath := "../../configs/configs.yaml"
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		configPath = "configs/configs.yaml"
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		slog.Error("Failed to load config", "error", err)
		os.Exit(1)
	}

	globalOutputDir = cfg.DrawingAPI.OutputDir

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

	nicknames := masterdata.GetNicknames()
	cardParser := service.NewCardParser(nicknames)
	cardSearchService := service.NewCardSearchService(masterdata, cardParser)
	eventParser := service.NewEventParser(nicknames)
	eventSearch := service.NewEventSearchService(masterdata, eventParser)

	env := &cliEnv{
		masterdata:          masterdata,
		cardController:      controller.NewCardController(masterdata, drawing, cardSearchService, cfg.DrawingAPI.BaseURL, assetHelper, userData),
		musicController:     controller.NewMusicController(masterdata, drawing, cfg.DrawingAPI.BaseURL, assetHelper, userData),
		gachaController:     controller.NewGachaController(masterdata, drawing, cfg.DrawingAPI.BaseURL, assetHelper),
		honorController:     controller.NewHonorController(masterdata, drawing, assetHelper),
		profileController:   controller.NewProfileController(masterdata, drawing, assetHelper),
		cardParser:          cardParser,
		eventController:     controller.NewEventController(masterdata, drawing, cfg.DrawingAPI.BaseURL, assetHelper),
		educationController: controller.NewEducationController(masterdata, drawing, assetHelper, userData),
		eventParser:         eventParser,
		eventSearch:         eventSearch,
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
		{"card-detail", "card-detail", "卡牌详情 (默认卡ID)"},
		{"card-list", "card-list", "卡牌列表查询"},
		{"card-box", "card-box", "卡牌盒子一览"},
		{"music-detail", "music", "单曲详情"},
		{"music-brief", "music-brief", "曲目概览"},
		{"music-list", "music-list", "谱面等级列表"},
		{"music-progress", "music-progress", "谱面完成度"},
		{"music-chart", "music-chart", "谱面预览"},
		{"gacha-list", "gacha-list", "卡池列表"},
		{"gacha-detail", "gacha-detail", "最新卡池详情"},
		{"event-detail", "event-detail", "活动详情"},
		{"event-list", "event-list", "活动列表"},
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

	// 如果是 auto 模式，先用 GlobalResolver 解析
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
		return testMusicDetail(env.musicController, payload)
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
			return fmt.Errorf("music-reward-detail 需要 -cmd 指向 JSON 文件")
		}
		return testMusicRewardsDetail(env.musicController, cmd)
	case "music-reward-basic":
		if strings.TrimSpace(cmd) == "" {
			return fmt.Errorf("music-reward-basic 需要 -cmd 指向 JSON 文件")
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
		return testGachaDetail(env.gachaController, payload)
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
			return fmt.Errorf("event-record 模式需要 -cmd 指向 JSON 文件")
		}
		return testEventRecord(env.eventController, cmd)
	case "education-challenge":
		return testEducationChallengeLive(env.educationController, cmd)
	case "education-power":
		if strings.TrimSpace(cmd) == "" {
			return fmt.Errorf("education-power 模式需要 -cmd 指向 JSON 文件")
		}
		return testEducationPowerBonus(env.educationController, cmd)
	case "education-area":
		if strings.TrimSpace(cmd) == "" {
			return fmt.Errorf("education-area 模式需要 -cmd 指向 JSON 文件")
		}
		return testEducationAreaItem(env.educationController, cmd)
	case "education-bonds":
		if strings.TrimSpace(cmd) == "" {
			return fmt.Errorf("education-bonds 模式需要 -cmd 指向 JSON 文件")
		}
		return testEducationBonds(env.educationController, cmd)
	case "education-leader":
		if strings.TrimSpace(cmd) == "" {
			return fmt.Errorf("education-leader 模式需要 -cmd 指向 JSON 文件")
		}
		return testEducationLeaderCount(env.educationController, cmd)
	case "honor":
		if strings.TrimSpace(cmd) == "" {
			return fmt.Errorf("honor 模式需要 -cmd 指向 JSON 文件")
		}
		return testHonorGenerate(env.honorController, cmd)
	case "profile":
		return testProfileGenerate(env.profileController, env.userData, cmd)
	default:
		return fmt.Errorf("unknown mode: %s", mode)
	}
}

func (env *cliEnv) handleResolvedCommand(res *service.ResolvedCommand) error {
	if res.IsHelp {
		fmt.Println("Haruki Command Help:")
		fmt.Println("  /查卡 <mnr> [-r jp/en/cn] - 查卡详情")
		fmt.Println("  /查曲 <ID/名称> [-r jp/en/cn] - 查曲详情")
		fmt.Println("  /活动 [current/ID/名称] - 查活动详情")
		fmt.Println("  /sk [UID/排名/@用户] - 查活动排名详情")
		return nil
	}

	// 统一处理 Region
	if res.Region != "" {
		slog.Info("Switching region", "target", res.Region)
		// 这里简单处理：如果不是 JP，重新加载 MasterData
		// 实际上更好的做法是 Controller 内部处理，目前先打印
	}

	var err error
	switch res.Module {
	case service.ModuleCard:
		switch res.Mode {
		case "gacha-list":
			err = testGachaList(env.gachaController, res.Query)
		case "card-box":
			err = testCardBox(env.cardController, res.Query)
		default:
			// 包含 card-detail 和任何未定义的单卡模式
			err = testCardDetail(env.cardController, env.cardParser, res.Query)
		}
	case service.ModuleMusic:
		switch res.Mode {
		case "music-chart":
			err = testMusicChart(env.musicController, res.Query)
		default:
			err = testMusicDetail(env.musicController, res.Query)
		}
	case service.ModuleEvent:
		switch res.Mode {
		case "event-list":
			err = testEventList(env.eventController, env.eventParser, res.Query)
		default:
			err = testEventDetail(env.eventController, env.eventSearch, res.Query)
		}
	case service.ModuleProfile:
		// sk 默认路由到 profile 或者特定的 sk 解析逻辑
		err = testProfileGenerate(env.profileController, env.userData, res.Query)
	default:
		return fmt.Errorf("解析结果无法直接运行: %v", res)
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
		return "/查曲 1", nil
	case "music-brief":
		return "master:1,2,3", nil
	case "music-list":
		return "/乐曲列表 ma 32", nil
	case "music-progress":
		return "/谱面进度 ma", nil
	case "music-chart":
		return "/谱面预览 1 ma", nil
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
		return "1", nil // 默认用户 ID
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

func testMusicDetail(ctrl *controller.MusicController, cmd string) error {
	raw := preprocessCommand(cmd, "/查曲", "查曲", "查歌", "查乐", "查询乐曲", "查音乐")
	query := model.MusicQuery{Query: raw, UserID: "test_user", Region: "jp"}

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
	payload := raw
	if strings.Contains(raw, ":") {
		parts := strings.SplitN(raw, ":", 2)
		if len(parts) == 2 {
			if strings.TrimSpace(parts[0]) != "" {
				diff = strings.TrimSpace(parts[0])
			}
			payload = strings.TrimSpace(parts[1])
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

func testGachaDetail(ctrl *controller.GachaController, cmd string) error {
	query, err := parseGachaDetailCommand(cmd)
	if err != nil {
		return fmt.Errorf("parse gacha detail command failed: %w", err)
	}
	start := time.Now()
	imageData, err := ctrl.RenderGachaDetail(query)
	if err != nil {
		return fmt.Errorf("render gacha detail failed: %w", err)
	}
	fmt.Printf("Render gacha detail success! Took %v. Image size: %d bytes\n", time.Since(start), len(imageData))
	return saveImage("gacha_detail", query.GachaID, imageData)
}

func testEventDetail(ctrl *controller.EventController, search *service.EventSearchService, cmd string) error {
	raw := preprocessEventCommand(cmd)
	if raw == "" {
		raw = "current"
	}
	event, err := search.Search(raw)
	if err != nil {
		return fmt.Errorf("failed to find event: %w", err)
	}
	query := model.EventDetailQuery{
		Region:  "jp",
		EventID: event.ID,
	}
	start := time.Now()
	imageData, err := ctrl.RenderEventDetail(query)
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
	region := "jp" // 默认

	start := time.Now()
	imageData, err := ctrl.RenderProfile(userID, region, userData)
	if err != nil {
		return fmt.Errorf("render profile failed: %w", err)
	}
	fmt.Printf("Render profile success! Took %v. Image size: %d bytes\n", time.Since(start), len(imageData))
	return saveImage("profile", 0, imageData)
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
		return fmt.Errorf("failed to write image: %w", err)
	}

	fmt.Printf("鉁?Image saved to: %s\n", outputPath)
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
	fields := strings.Fields(raw)
	diff := "master"
	if len(fields) > 0 {
		if normalized, ok := normalizeDifficultyAlias(fields[0]); ok {
			diff = normalized
		}
	}

	return model.MusicProgressQuery{
		Difficulty: diff,
		Region:     "jp",
	}, nil
}

func parseMusicChartCommand(cmd string) (model.MusicChartQuery, error) {
	raw := preprocessCommand(cmd, "/谱面预览", "谱面预览", "查谱面", "查谱", "谱面", "查谱图")
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
		Region:     "jp",
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
		Region:       "jp",
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
	return json.Unmarshal(data, target)
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
		"pjsk gacha", "卡池列表", "卡池一览", "卡池",
	}
	lower := strings.ToLower(raw)
	for _, rep := range replacements {
		lower = strings.ReplaceAll(lower, strings.ToLower(rep), "")
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
		case lt == "current":
			query.IncludeFuture = false
			query.IncludePast = false
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
	}
	for _, rep := range replacements {
		if rep == "" {
			continue
		}
		raw = strings.ReplaceAll(raw, rep, "")
	}
	raw = strings.TrimSpace(raw)
	id, err := strconv.Atoi(raw)
	if err != nil {
		return model.GachaDetailQuery{}, fmt.Errorf("invalid gacha id: %s", raw)
	}
	return model.GachaDetailQuery{
		Region:  "jp",
		GachaID: id,
	}, nil
}

func parseEventListCommand(cmd string, parser *service.EventParser) (model.EventListQuery, error) {
	query := model.EventListQuery{
		Region:        "jp",
		IncludeFuture: true,
		Limit:         6,
	}
	raw := preprocessEventCommand(cmd)
	if raw == "" {
		return query, nil
	}
	tokens := strings.Fields(raw)
	var filtered []string
	for _, token := range tokens {
		lower := strings.ToLower(token)
		switch {
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
		return query, fmt.Errorf("请输入过滤条件，例如 'wl' 或 '25h 2024'")
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
