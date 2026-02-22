package controller

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"Haruki-Service-API/internal/builder"
	"Haruki-Service-API/internal/model"
	"Haruki-Service-API/internal/service"
	"Haruki-Service-API/pkg/asset"
	"Haruki-Service-API/pkg/masterdata"
)

// MusicController 音乐控制器
// 定义在这里以便 builder 方法挂载
type MusicController struct {
	masterdata *service.MasterDataService
	drawing    *service.DrawingService
	drawingURL string
	assetDir   string
	assets     *asset.AssetHelper
	userData   *service.UserDataService
}

// NewMusicController 创建音乐控制器
func NewMusicController(
	masterdata *service.MasterDataService,
	drawing *service.DrawingService,
	drawingURL string,
	assetHelper *asset.AssetHelper,
	userData *service.UserDataService,
) *MusicController {
	assetDir := ""
	if assetHelper != nil {
		assetDir = assetHelper.Primary()
	}
	return &MusicController{
		masterdata: masterdata,
		drawing:    drawing,
		drawingURL: drawingURL,
		assetDir:   assetDir,
		assets:     assetHelper,
		userData:   userData,
	}
}

// RenderMusicDetail 渲染音乐详情
func (c *MusicController) RenderMusicDetail(query model.MusicQuery) ([]byte, error) {
	drawingReq, err := c.BuildMusicDetail(query)
	if err != nil {
		return nil, err
	}

	// Need to add GenerateMusicDetail to DrawingService
	return c.drawing.GenerateMusicDetail(drawingReq.Body)
}

// BuildMusicChart 构建谱面预览请求
func (c *MusicController) BuildMusicChart(query model.MusicChartQuery) (*model.DrawingRequest, error) {
	req, err := c.buildMusicChartRequest(query)
	if err != nil {
		return nil, err
	}
	return &model.DrawingRequest{
		URL:    c.drawingURL + "/api/pjsk/chart",
		Method: "POST",
		Body:   req,
	}, nil
}

// RenderMusicChart 直接渲染谱面预览
func (c *MusicController) RenderMusicChart(query model.MusicChartQuery) ([]byte, error) {
	req, err := c.buildMusicChartRequest(query)
	if err != nil {
		return nil, err
	}
	return c.drawing.GenerateMusicChart(req)
}

func (c *MusicController) buildMusicChartRequest(query model.MusicChartQuery) (*model.MusicChartRequest, error) {
	music, err := c.masterdata.SearchMusic(query.Query)
	if err != nil {
		return nil, fmt.Errorf("failed to search music: %w", err)
	}

	b := builder.NewMusicBuilder(c.masterdata, c.assets, c.assetDir, c.userData)
	return b.BuildMusicChartRequest(query, music)
}

// BuildMusicDetail 鏋勫缓闊充箰璇︽儏璇锋眰锛堟ā寮?1锛?
func (c *MusicController) BuildMusicDetail(query model.MusicQuery) (*model.DrawingRequest, error) {
	music, err := c.masterdata.SearchMusic(query.Query)
	if err != nil {
		return nil, fmt.Errorf("failed to search music: %w", err)
	}

	region := query.Region
	if region == "" {
		region = c.masterdata.GetRegion()
	}

	b := builder.NewMusicBuilder(c.masterdata, c.assets, c.assetDir, c.userData)
	req, err := b.BuildMusicDetailRequest(music, region)
	if err != nil {
		return nil, fmt.Errorf("failed to build music request: %w", err)
	}

	return &model.DrawingRequest{
		URL:    c.drawingURL + "/api/pjsk/music/detail",
		Method: "POST",
		Body:   req,
	}, nil
}

// RenderMusicBriefList 渲染音乐简要列表
func (c *MusicController) RenderMusicBriefList(musicIDs []int, difficulty, region string) ([]byte, error) {
	req, err := c.BuildMusicBriefListRequest(musicIDs, difficulty, region)
	if err != nil {
		return nil, err
	}
	return c.drawing.GenerateMusicBriefList(req)
}

// BuildMusicBriefListRequest 构建音乐简要列表请求
func (c *MusicController) BuildMusicBriefListRequest(musicIDs []int, difficulty, region string) (*model.MusicBriefListRequest, error) {
	if len(musicIDs) == 0 {
		return nil, fmt.Errorf("no music ids provided")
	}

	diff := strings.ToLower(strings.TrimSpace(difficulty))
	if diff == "" {
		diff = "master"
	}

	if region == "" {
		region = c.masterdata.GetRegion()
	}

	var list []model.MusicBriefListItem
	for _, id := range musicIDs {
		music, err := c.masterdata.GetMusicByID(id)
		if err != nil {
			return nil, fmt.Errorf("music %d not found: %w", id, err)
		}

		level := c.getDifficultyLevel(music.ID, diff)
		jacket := builder.NewMusicBuilder(c.masterdata, c.assets, c.assetDir, c.userData).BuildMusicJacketPath(music.AssetBundleName)

		list = append(list, model.MusicBriefListItem{
			ID:              music.ID,
			Level:           level,
			MusicJacketPath: jacket,
		})
	}

	if len(list) == 0 {
		return nil, fmt.Errorf("no valid music data")
	}

	return &model.MusicBriefListRequest{
		MusicList:          list,
		Region:             region,
		RequiredDifficulty: diff,
	}, nil
}

func (c *MusicController) buildUserResults(diff string) map[int]string {
	if c.userData == nil {
		return nil
	}
	return c.userData.MusicResults(diff)
}

func (c *MusicController) getDifficultyLevel(musicID int, diff string) int {
	b := builder.NewMusicBuilder(c.masterdata, c.assets, c.assetDir, c.userData)
	return b.GetDifficultyLevel(musicID, diff)
}

// RenderMusicList 渲染歌曲列表
func (c *MusicController) RenderMusicList(query model.MusicListQuery) ([]byte, error) {
	req, err := c.BuildMusicListRequest(query)
	if err != nil {
		return nil, err
	}
	return c.drawing.GenerateMusicList(req)
}

// BuildMusicList 构建歌曲列表 Drawing 请求
func (c *MusicController) BuildMusicList(query model.MusicListQuery) (*model.DrawingRequest, error) {
	req, err := c.BuildMusicListRequest(query)
	if err != nil {
		return nil, err
	}

	return &model.DrawingRequest{
		URL:    c.drawingURL + "/api/pjsk/music/list",
		Method: "POST",
		Body:   req,
	}, nil
}

// BuildMusicListRequest 构建歌曲列表请求体
func (c *MusicController) BuildMusicListRequest(query model.MusicListQuery) (*model.MusicListRequest, error) {
	diff := strings.ToLower(strings.TrimSpace(query.Difficulty))
	if diff == "" {
		diff = "master"
	}

	region := query.Region
	if region == "" {
		region = c.masterdata.GetRegion()
	}

	bannedIDs := map[int]struct{}{
		241: {},
		290: {},
	}

	minLevel := query.LevelMin
	maxLevel := query.LevelMax
	if query.Level > 0 {
		minLevel = query.Level
		maxLevel = query.Level
	}
	if minLevel > 0 && maxLevel > 0 && minLevel > maxLevel {
		minLevel, maxLevel = maxLevel, minLevel
	}

	musics := c.masterdata.GetMusics()
	now := time.Now()
	keyword := strings.ToLower(strings.TrimSpace(query.Keyword))
	var list []model.MusicListItem
	jackets := make(map[int]string)

	for _, music := range musics {
		if keyword != "" {
			title := strings.ToLower(music.Title)
			pron := strings.ToLower(strings.TrimSpace(music.Pronunciation))
			matched := strings.Contains(title, keyword) || (pron != "" && strings.Contains(pron, keyword))
			if !matched {
				if tags, err := c.masterdata.GetMusicTags(music.ID); err == nil {
					for _, tag := range tags {
						if strings.Contains(strings.ToLower(tag), keyword) {
							matched = true
							break
						}
					}
				}
			}
			if !matched {
				continue
			}
		}
		if _, banned := bannedIDs[music.ID]; banned {
			continue
		}
		if !query.IncludeLeaks && time.UnixMilli(music.PublishedAt).After(now) {
			continue
		}

		level := c.getDifficultyLevel(music.ID, diff)
		if level == 0 {
			continue
		}
		if minLevel > 0 && level < minLevel {
			continue
		}
		if maxLevel > 0 && level > maxLevel {
			continue
		}

		jackets[music.ID] = builder.NewMusicBuilder(c.masterdata, c.assets, c.assetDir, c.userData).BuildMusicJacketPath(music.AssetBundleName)
		list = append(list, model.MusicListItem{
			ID:         music.ID,
			Difficulty: level,
			ReleaseAt:  music.PublishedAt,
		})
	}

	if len(list) == 0 {
		return nil, fmt.Errorf("no music matched the current filters")
	}

	userResults := query.UserResults
	if userResults == nil {
		userResults = c.buildUserResults(diff)
		if userResults == nil {
			userResults = make(map[int]string)
		}
	}

	req := &model.MusicListRequest{
		UserResults:          userResults,
		MusicList:            list,
		JacketsPathList:      jackets,
		RequiredDifficulties: diff,
		Profile:              c.detailedProfile(region),
		TitleShadow:          query.TitleShadow,
	}
	if len(userResults) > 0 {
		req.PlayResultIconPath = c.buildPlayResultIconMap()
	}

	if query.Title != nil {
		req.Title = query.Title
	}
	if len(query.TitleStyle) > 0 {
		req.TitleStyle = query.TitleStyle
	}

	return req, nil
}

func (c *MusicController) buildPlaceholderProfile(region string) model.DetailedProfileCardRequest {
	if region == "" {
		region = c.masterdata.GetRegion()
	}
	region = strings.ToUpper(region)

	defaultLeader := asset.ResolveAssetPath(c.assets, c.assetDir, filepath.Join("user", "leader.png"))
	if defaultLeader == "" {
		defaultLeader = asset.ResolveAssetPath(c.assets, c.assetDir, filepath.Join("chara_icon", "miku.png"))
	}

	return model.DetailedProfileCardRequest{
		ID:              "service",
		Region:          region,
		Nickname:        "Lunabot",
		Source:          "lunabot-service",
		UpdateTime:      time.Now().Unix(),
		Mode:            "service",
		IsHideUID:       true,
		LeaderImagePath: defaultLeader,
		HasFrame:        false,
		UserCards:       []map[string]interface{}{},
	}
}

func (c *MusicController) detailedProfile(region string) model.DetailedProfileCardRequest {
	if c.userData != nil {
		if profile := c.userData.DetailedProfile(region); profile != nil {
			return *profile
		}
	}
	return c.buildPlaceholderProfile(region)
}

// BuildMusicProgress 构建打歌进度 DrawingRequest
func (c *MusicController) BuildMusicProgress(query model.MusicProgressQuery) (*model.DrawingRequest, error) {
	req, err := c.BuildMusicProgressRequest(query)
	if err != nil {
		return nil, err
	}

	return &model.DrawingRequest{
		URL:    c.drawingURL + "/api/pjsk/music/progress",
		Method: "POST",
		Body:   req,
	}, nil
}

// RenderMusicProgress 渲染打歌进度
func (c *MusicController) RenderMusicProgress(query model.MusicProgressQuery) ([]byte, error) {
	req, err := c.BuildMusicProgressRequest(query)
	if err != nil {
		return nil, err
	}
	return c.drawing.GenerateMusicProgress(req)
}

// BuildMusicProgressRequest 构建打歌进度请求体
func (c *MusicController) BuildMusicProgressRequest(query model.MusicProgressQuery) (*model.PlayProgressRequest, error) {
	diff := strings.ToLower(strings.TrimSpace(query.Difficulty))
	if diff == "" {
		diff = "master"
	}

	region := query.Region
	if region == "" {
		region = c.masterdata.GetRegion()
	}

	counts := query.Counts
	if len(counts) == 0 {
		if userCounts := c.buildUserProgressCounts(diff); len(userCounts) > 0 {
			counts = userCounts
		} else {
			counts = c.buildDefaultProgressCounts(diff)
		}
	}

	req := &model.PlayProgressRequest{
		Counts:     counts,
		Difficulty: diff,
		Profile:    c.buildProfileCard(region),
	}
	return req, nil
}

// BuildMusicRewardsDetail 构建详细奖励 DrawingRequest
func (c *MusicController) BuildMusicRewardsDetail(query model.MusicRewardsDetailQuery) (*model.DrawingRequest, error) {
	req, err := c.BuildMusicRewardsDetailRequest(query)
	if err != nil {
		return nil, err
	}

	return &model.DrawingRequest{
		URL:    c.drawingURL + "/api/pjsk/music/rewards/detail",
		Method: "POST",
		Body:   req,
	}, nil
}

// RenderMusicRewardsDetail 渲染详细奖励图
func (c *MusicController) RenderMusicRewardsDetail(query model.MusicRewardsDetailQuery) ([]byte, error) {
	req, err := c.BuildMusicRewardsDetailRequest(query)
	if err != nil {
		return nil, err
	}
	return c.drawing.GenerateMusicRewardsDetail(req)
}

// BuildMusicRewardsDetailRequest 构建详细奖励请求体
func (c *MusicController) BuildMusicRewardsDetailRequest(query model.MusicRewardsDetailQuery) (*model.DetailMusicRewardsRequest, error) {
	region := query.Region
	if region == "" {
		region = c.masterdata.GetRegion()
	}

	combo := ensureDetailComboRewards(query.ComboRewards)

	return &model.DetailMusicRewardsRequest{
		RankRewards:  query.RankRewards,
		ComboRewards: combo,
		Profile:      c.buildProfileCard(region),
		JewelIcon:    query.JewelIcon,
		ShardIcon:    query.ShardIcon,
	}, nil
}

// BuildMusicRewardsBasic 构建基础奖励 DrawingRequest
func (c *MusicController) BuildMusicRewardsBasic(query model.MusicRewardsBasicQuery) (*model.DrawingRequest, error) {
	req, err := c.BuildMusicRewardsBasicRequest(query)
	if err != nil {
		return nil, err
	}

	return &model.DrawingRequest{
		URL:    c.drawingURL + "/api/pjsk/music/rewards/basic",
		Method: "POST",
		Body:   req,
	}, nil
}

// RenderMusicRewardsBasic 渲染基础奖励图
func (c *MusicController) RenderMusicRewardsBasic(query model.MusicRewardsBasicQuery) ([]byte, error) {
	req, err := c.BuildMusicRewardsBasicRequest(query)
	if err != nil {
		return nil, err
	}
	return c.drawing.GenerateMusicRewardsBasic(req)
}

// BuildMusicRewardsBasicRequest 构建基础奖励请求体
func (c *MusicController) BuildMusicRewardsBasicRequest(query model.MusicRewardsBasicQuery) (*model.BasicMusicRewardsRequest, error) {
	region := query.Region
	if region == "" {
		region = c.masterdata.GetRegion()
	}

	combo := query.ComboRewards
	if combo == nil {
		combo = map[string]string{
			"hard":   "0",
			"expert": "0",
			"master": "0",
			"append": "0",
		}
	}

	return &model.BasicMusicRewardsRequest{
		RankRewards:  query.RankRewards,
		ComboRewards: combo,
		Profile:      c.buildProfileCard(region),
		JewelIcon:    query.JewelIcon,
		ShardIcon:    query.ShardIcon,
	}, nil
}

func (c *MusicController) buildDefaultProgressCounts(diff string) []model.PlayProgressCount {
	musics := c.masterdata.GetMusics()
	counts := make(map[int]int)
	for _, music := range musics {
		level := c.getDifficultyLevel(music.ID, diff)
		if level == 0 {
			continue
		}
		counts[level]++
	}

	if len(counts) == 0 {
		return nil
	}

	var levels []int
	for lv := range counts {
		levels = append(levels, lv)
	}
	sort.Ints(levels)

	result := make([]model.PlayProgressCount, 0, len(levels))
	for _, lv := range levels {
		total := counts[lv]
		result = append(result, model.PlayProgressCount{
			Level:    lv,
			Total:    total,
			NotClear: total,
			Clear:    0,
			FC:       0,
			AP:       0,
		})
	}
	return result
}

func (c *MusicController) buildUserProgressCounts(diff string) []model.PlayProgressCount {
	if c.userData == nil {
		return nil
	}
	musics := c.masterdata.GetMusics()
	countMap := make(map[int]*model.PlayProgressCount)
	now := time.Now()
	for _, music := range musics {
		if time.UnixMilli(music.PublishedAt).After(now) {
			continue
		}
		level := c.getDifficultyLevel(music.ID, diff)
		if level == 0 {
			continue
		}
		entry := countMap[level]
		if entry == nil {
			entry = &model.PlayProgressCount{Level: level}
			countMap[level] = entry
		}
		entry.Total++
		switch c.userData.GetMusicResult(music.ID, diff) {
		case "ap":
			entry.AP++
			entry.FC++
			entry.Clear++
		case "fc":
			entry.FC++
			entry.Clear++
		case "clear":
			entry.Clear++
		default:
			entry.NotClear++
		}
	}
	if len(countMap) == 0 {
		return nil
	}
	var levels []int
	for lv := range countMap {
		levels = append(levels, lv)
	}
	sort.Ints(levels)
	result := make([]model.PlayProgressCount, 0, len(levels))
	for _, lv := range levels {
		result = append(result, *countMap[lv])
	}
	return result
}

func ensureDetailComboRewards(combo map[string][]model.MusicComboReward) map[string][]model.MusicComboReward {
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

func (c *MusicController) buildChartArtist(music *masterdata.Music) string {
	b := builder.NewMusicBuilder(c.masterdata, c.assets, c.assetDir, c.userData)
	return b.BuildChartArtist(music)
}

func (c *MusicController) buildProfileCard(region string) *model.ProfileCardRequest {
	if c.userData != nil {
		if profile := c.userData.ProfileCard(region); profile != nil {
			return profile
		}
	}
	detailed := c.buildPlaceholderProfile(region)
	return convertDetailedProfileToCard(detailed)
}

func convertDetailedProfileToCard(detail model.DetailedProfileCardRequest) *model.ProfileCardRequest {
	source := detail.Source
	if source == "" {
		source = "lunabot-service"
	}
	mode := detail.Mode
	if mode == "" {
		mode = "placeholder"
	}
	update := detail.UpdateTime
	if update == 0 {
		update = time.Now().Unix()
	}
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
				Name:       "Lunabot Service",
				Source:     &source,
				UpdateTime: &update,
				Mode:       &mode,
			},
		},
	}
}

func (c *MusicController) buildPlayResultIconMap() map[string]string {
	iconNames := map[string]string{
		"not_clear": "icon_not_clear.png",
		"clear":     "icon_clear.png",
		"fc":        "icon_fc.png",
		"ap":        "icon_ap.png",
	}
	result := make(map[string]string, len(iconNames))
	for key, file := range iconNames {
		result[key] = filepath.ToSlash(filepath.Join(c.assetDir, file))
	}
	return result
}
