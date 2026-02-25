package builder

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"Haruki-Service-API/internal/model"
	"Haruki-Service-API/internal/service"
	"Haruki-Service-API/pkg/asset"
	"Haruki-Service-API/pkg/masterdata"
)

// MusicBuilder 专门负责构建供 DrawingAPI 消费的 Music 模块 JSON Payload
type MusicBuilder struct {
	masterdata *service.MasterDataService
	assets     *asset.AssetHelper
	assetDir   string
	userData   *service.UserDataService
}

func NewMusicBuilder(m *service.MasterDataService, a *asset.AssetHelper, d string, u *service.UserDataService) *MusicBuilder {
	return &MusicBuilder{
		masterdata: m,
		assets:     a,
		assetDir:   d,
		userData:   u,
	}
}

// BuildMusicDetailRequest 构建音乐详情请求
func (b *MusicBuilder) BuildMusicDetailRequest(music *masterdata.Music, region string) (*model.MusicDetailRequest, error) {
	// 1. 基础信息 MusicMD
	musicInfo := model.MusicMD{
		ID:           music.ID,
		Title:        music.Title,
		Composer:     music.Composer,
		Lyricist:     music.Lyricist,
		Arranger:     music.Arranger,
		Categories:   b.buildCategories(music.ID),
		ReleaseAt:    int(music.PublishedAt), // Go int vs Python int (timestamp)
		IsFullLength: false,                  // The masterdata does not contain full_length info currently
	}

	// 2. 难度信息 DifficultyInfo
	diffInfo, err := b.buildDifficultyInfo(music.ID)
	if err != nil {
		return nil, err
	}

	// 3. Vocal信息
	vocalInfo, err := b.buildVocalInfo(music.ID)
	if err != nil {
		return nil, err
	}

	// 4. 路径构建
	jacketPath := b.BuildMusicJacketPath(music.AssetBundleName)

	aliases := b.buildMusicAliases(music)
	if len(aliases) == 0 {
		aliases = []string{}
	}

	// 5. 组装请求
	req := &model.MusicDetailRequest{
		Region:          region,
		MusicInfo:       musicInfo,
		Difficulty:      *diffInfo,
		Vocal:           *vocalInfo,
		MusicJacketPath: jacketPath,
		Alias:           aliases,
	}

	if event, err := b.masterdata.GetPrimaryEventByMusicID(music.ID); err == nil && event != nil {
		req.EventID = &event.ID
		req.EventBannerPath = b.buildEventBannerPath(event.AssetBundleName)
	}
	if limited := b.buildLimitedTimes(music.ID, region); len(limited) > 0 {
		req.LimitedTimes = limited
	}

	return req, nil
}

// BuildMusicChartRequest 构建谱面预览请求体
func (b *MusicBuilder) BuildMusicChartRequest(query model.MusicChartQuery, music *masterdata.Music) (*model.MusicChartRequest, error) {
	diff := strings.ToLower(strings.TrimSpace(query.Difficulty))
	if diff == "" {
		diff = "master"
	}

	playLevel := b.GetDifficultyLevel(music.ID, diff)
	if playLevel == 0 {
		return nil, fmt.Errorf("music %s does not have %s chart", music.Title, diff)
	}

	// 1. 封面路径：修正去除 _rip 后缀
	jacketPath := b.BuildMusicJacketPath(music.AssetBundleName)

	// 2. 谱面路径：修正去除 _rip，后缀改为 .txt，且移除难度子文件夹
	// 预期物理路径: music/music_score/{id_major}_{id_minor}/{diff}.txt
	susDir := fmt.Sprintf("%04d_01", music.ID)
	susFile := fmt.Sprintf("%s.txt", diff)
	susRelative := filepath.Join("music", "music_score", susDir, susFile)
	susPath := asset.ResolveAssetPath(b.assets, b.assetDir, susRelative)

	if susPath == "" {
		fmt.Printf("[WARN] Sus/Chart file missing: %s\n", susRelative)
	}

	// 3. NoteHost：目前保留路径解析
	// 预期物理路径: Z:/pjskdata/Data/lunabot_static_images/chart_asset/notes
	noteHost := "lunabot_static_images/chart_asset/notes"

	stylePath := strings.TrimSpace(query.Style)
	var styleAbs *string
	if stylePath != "" {
		abs := filepath.ToSlash(filepath.Join(b.assetDir, stylePath))
		styleAbs = &abs
	}

	return &model.MusicChartRequest{
		MusicID:    music.ID,
		Title:      music.Title,
		Artist:     b.BuildChartArtist(music),
		Difficulty: diff,
		PlayLevel:  playLevel,
		Skill:      query.Skill,
		JacketPath: asset.MakeRelative(b.assetDir, jacketPath),
		SusPath:    asset.MakeRelative(b.assetDir, susPath),
		StylePath:  styleAbs,
		NoteHost:   noteHost,
	}, nil
}

func (b *MusicBuilder) BuildChartArtist(music *masterdata.Music) string {
	composer := strings.TrimSpace(music.Composer)
	arranger := strings.TrimSpace(music.Arranger)
	switch {
	case composer == arranger:
		return composer
	case composer == "-" || strings.Contains(arranger, composer):
		return arranger
	case arranger == "-" || strings.Contains(composer, arranger):
		return composer
	case composer == "" && arranger == "":
		return "Unknown"
	default:
		return fmt.Sprintf("%s / %s", composer, arranger)
	}
}

func (b *MusicBuilder) GetDifficultyLevel(musicID int, diff string) int {
	difficulties, err := b.masterdata.GetMusicDifficulties(musicID)
	if err != nil {
		return 0
	}

	for _, d := range difficulties {
		if strings.EqualFold(d.MusicDifficulty, diff) {
			return d.PlayLevel
		}
	}
	return 0
}

// buildDifficultyInfo 构建难度信息
func (b *MusicBuilder) buildDifficultyInfo(musicID int) (*model.DifficultyInfo, error) {
	diffs, err := b.masterdata.GetMusicDifficulties(musicID)
	if err != nil {
		return &model.DifficultyInfo{
			Level:     []int{0, 0, 0, 0, 0},
			NoteCount: []int{0, 0, 0, 0, 0},
			HasAppend: false,
			Order:     []string{"easy", "normal", "hard", "expert", "master"},
		}, nil
	}

	type diffStat struct {
		level int
		notes int
	}
	diffMap := make(map[string]diffStat)
	for _, d := range diffs {
		key := strings.ToLower(strings.TrimSpace(d.MusicDifficulty))
		diffMap[key] = diffStat{level: d.PlayLevel, notes: d.TotalNoteCount}
	}

	baseOrder := []string{"easy", "normal", "hard", "expert", "master"}
	levels := make([]int, 0, len(baseOrder)+1)
	notes := make([]int, 0, len(baseOrder)+1)
	order := make([]string, 0, len(baseOrder)+1)

	for _, diff := range baseOrder {
		if stat, ok := diffMap[diff]; ok {
			levels = append(levels, stat.level)
			notes = append(notes, stat.notes)
		} else {
			levels = append(levels, 0)
			notes = append(notes, 0)
		}
		order = append(order, diff)
	}

	hasAppend := false
	if stat, ok := diffMap["append"]; ok {
		hasAppend = true
		levels = append(levels, stat.level)
		notes = append(notes, stat.notes)
		order = append(order, "append")
	}

	return &model.DifficultyInfo{
		Level:     levels,
		NoteCount: notes,
		HasAppend: hasAppend,
		Order:     order,
	}, nil
}

// buildVocalInfo 构建 Vocal 信息
func (b *MusicBuilder) buildVocalInfo(musicID int) (*model.MusicVocalInfo, error) {
	vocals, err := b.masterdata.GetMusicVocals(musicID)
	if err != nil {
		return &model.MusicVocalInfo{
			VocalInfo:   make(map[string]interface{}),
			VocalAssets: make(map[string]string),
		}, nil
	}

	infoMap := make(map[string]interface{})
	assetsMap := make(map[string]string)

	for _, v := range vocals {
		// Build character list
		var chars []map[string]string
		for _, char := range v.Characters {
			// Resolve Character Name
			var name string
			var charID int

			if char.CharacterType == "game_character" {
				charID = char.CharacterID
				if cObj, err := b.masterdata.GetCharacterByID(char.CharacterID); err == nil {
					name = cObj.FirstName + cObj.GivenName
				}
			} else {
				// Virtual Singer
				charID = char.CharacterID
				if cObj, err := b.masterdata.GetCharacterByID(char.CharacterID); err == nil {
					name = cObj.FirstName + cObj.GivenName
				} else {
					name = "VS"
				}
			}
			chars = append(chars, map[string]string{"characterName": name})

			// Populate Asset Map for Icon
			if name != "" && charID != 0 {
				iconPath := b.BuildCharacterIconPath(charID)
				assetsMap[name] = iconPath
			}
		}

		caption := normalizeVocalCaption(v.Caption, v.MusicVocalType)

		entry := map[string]interface{}{
			"caption":    caption,
			"characters": chars,
		}
		infoMap[v.AssetBundleName] = entry
	}

	return &model.MusicVocalInfo{
		VocalInfo:   infoMap,
		VocalAssets: assetsMap,
	}, nil
}

// BuildCharacterIconPath 构建角色图标路径
func (b *MusicBuilder) BuildCharacterIconPath(characterID int) string { // 2. 使用通用角色图标路径
	if nickname, ok := asset.CharacterIDToNickname[characterID]; ok {
		return asset.ResolveAssetPath(b.assets, b.assetDir, filepath.Join("chara_icon", nickname+".png"))
	}
	return asset.ResolveAssetPath(b.assets, b.assetDir, filepath.Join("chara_icon", fmt.Sprintf("chr_icon_%d.png", characterID)))
}

// buildCategories 构建分类列表
func (b *MusicBuilder) buildCategories(musicID int) []string {
	tags, _ := b.masterdata.GetMusicTags(musicID)
	return tags
}

// BuildMusicJacketPath 构建封面路径
func (b *MusicBuilder) BuildMusicJacketPath(assetName string) string {
	// 修正：不应当强制加 _rip，因为本地目录并没有这个后缀
	return filepath.ToSlash(filepath.Join(b.assetDir, "music", "jacket", assetName, assetName+".png"))
}

func (b *MusicBuilder) buildMusicAliases(music *masterdata.Music) []string {
	var aliases []string
	if music == nil {
		return aliases
	}
	if pronunciation := strings.TrimSpace(music.Pronunciation); pronunciation != "" {
		aliases = append(aliases, pronunciation)
	}
	if tags, err := b.masterdata.GetMusicTags(music.ID); err == nil {
		for _, tag := range tags {
			tag = strings.TrimSpace(tag)
			if tag == "" {
				continue
			}
			if !containsString(aliases, tag) {
				aliases = append(aliases, tag)
			}
		}
	}
	return aliases
}

func (b *MusicBuilder) buildLimitedTimes(musicID int, region string) [][2]string {
	limited := b.masterdata.GetLimitedTimeMusics(musicID)
	if len(limited) == 0 {
		return nil
	}
	loc := regionToLocation(region)
	result := make([][2]string, 0, len(limited))
	for _, item := range limited {
		start := formatTimestamp(item.StartAt, loc)
		end := formatTimestamp(item.EndAt, loc)
		result = append(result, [2]string{start, end})
	}
	return result
}

func containsString(values []string, target string) bool {
	for _, v := range values {
		if strings.EqualFold(v, target) {
			return true
		}
	}
	return false
}

func regionToLocation(region string) *time.Location {
	switch strings.ToLower(strings.TrimSpace(region)) {
	case "jp", "kr":
		return time.FixedZone("UTC+9", 9*3600)
	case "tw", "cn":
		return time.FixedZone("UTC+8", 8*3600)
	case "en":
		return time.UTC
	default:
		return time.FixedZone("UTC+9", 9*3600)
	}
}

func formatTimestamp(ts int64, loc *time.Location) string {
	if loc == nil {
		loc = time.UTC
	}
	return time.UnixMilli(ts).In(loc).Format("2006-01-02 15:04")
}

var vocalCaptionOverrides = map[string]string{
	"セカイver.":                      "Sekai",
	"セカイ ver.":                     "Sekai",
	"バーチャル・シンガーver.":               "Virtual Singer",
	"バーチャルシンガーver.":                "Virtual Singer",
	"アナザーボーカルver.":                 "Another Vocal",
	"原曲ver.":                       "Original Song",
	"原曲 ver.":                      "Original Song",
	"ストリーミングライブver.":               "Connect Live",
	"ストリーミングライブ ver.":              "Connect Live",
	"エイプリルフールver.":                 "April Fool",
	"あんさんぶるスターズ！！コラボver.":          "Ensemble Stars!! Collab",
	"「劇場版プロジェクトセカイ」ver.":           "Movie",
	"sekai ver.":                   "Sekai",
	"sekai":                        "Sekai",
	"virtual singer ver.":          "Virtual Singer",
	"virtual singer":               "Virtual Singer",
	"another vocal ver.":           "Another Vocal",
	"another vocal":                "Another Vocal",
	"original song ver.":           "Original Song",
	"original song":                "Original Song",
	"streaming live ver.":          "Connect Live",
	"streaming live":               "Connect Live",
	"instrumental ver.":            "Inst.",
	"instrumental":                 "Inst.",
	"april fool 2022 ver.":         "April Fool",
	"april_fool_2022 ver.":         "April Fool",
	"april_fool_2022":              "April Fool",
	"april fool":                   "April Fool",
	"sekai version":                "Sekai",
	"virtual singer version":       "Virtual Singer",
	"another vocal version":        "Another Vocal",
	"original song version":        "Original Song",
	"streaming live version":       "Connect Live",
	"instrumental version":         "Inst.",
	"april fool 2022 version":      "April Fool",
	"ensemble stars!! collab":      "Ensemble Stars!! Collab",
	"ensemble stars!! collab ver.": "Ensemble Stars!! Collab",
	"movie ver.":                   "Movie",
	"movie":                        "Movie",
}

var vocalTypeFallbacks = map[string]string{
	"sekai":           "Sekai",
	"virtual_singer":  "Virtual Singer",
	"original_song":   "Original Song",
	"another_vocal":   "Another Vocal",
	"streaming_live":  "Connect Live",
	"instrumental":    "Inst.",
	"april_fool_2022": "April Fool",
}

func normalizeVocalCaption(raw string, vocalType string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		trimmed = strings.TrimSpace(vocalType)
	}
	key := strings.ToLower(trimmed)
	key = strings.ReplaceAll(key, "　", " ")
	key = strings.ReplaceAll(key, "．", ".")
	key = strings.ReplaceAll(key, "version", "ver.")
	key = strings.ReplaceAll(key, "ver..", "ver.")
	key = strings.TrimSpace(key)
	for strings.Contains(key, "  ") {
		key = strings.ReplaceAll(key, "  ", " ")
	}
	if len(key) > 0 && key[len(key)-1] == ' ' {
		key = strings.TrimSpace(key)
	}
	if strings.HasSuffix(key, "ver") {
		key += "."
	}
	if resolved, ok := vocalCaptionOverrides[key]; ok {
		return resolved
	}
	if resolved, ok := vocalTypeFallbacks[strings.ToLower(vocalType)]; ok {
		return resolved
	}
	return trimmed
}

func (b *MusicBuilder) buildEventBannerPath(assetBundleName string) string {
	if assetBundleName == "" {
		return ""
	}
	candidates := []string{
		filepath.Join("home", "banner", fmt.Sprintf("%s_rip", assetBundleName), assetBundleName+".png"),
		filepath.Join("home", "banner", assetBundleName, assetBundleName+".png"),
		filepath.Join("event", assetBundleName, "banner.png"),
	}
	return asset.ResolveAssetPath(b.assets, b.assetDir, candidates...)
}
