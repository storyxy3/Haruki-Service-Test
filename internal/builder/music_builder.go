package builder

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"Haruki-Service-API/internal/model"
	"Haruki-Service-API/internal/service"
	"Haruki-Service-API/pkg/asset"
	"Haruki-Service-API/pkg/masterdata"
)

type MusicBuilder struct {
	source   service.MusicDataSource
	assets   *asset.AssetHelper
	assetDir string
	userData *service.UserDataService
}

func NewMusicBuilder(source service.MusicDataSource, a *asset.AssetHelper, d string, u *service.UserDataService) *MusicBuilder {
	return &MusicBuilder{
		source:   source,
		assets:   a,
		assetDir: d,
		userData: u,
	}
}

func (b *MusicBuilder) BuildMusicDetailRequest(music *masterdata.Music, region string) (*model.MusicDetailRequest, error) {
	regionCode := strings.ToUpper(strings.TrimSpace(region))
	if regionCode == "" {
		regionCode = "JP"
	}
	displayTitle := b.buildDisplayMusicTitle(music, regionCode)

	musicInfo := model.MusicMD{
		ID:           music.ID,
		Title:        displayTitle,
		Composer:     music.Composer,
		Lyricist:     music.Lyricist,
		Arranger:     music.Arranger,
		Categories:   b.buildCategories(music.ID),
		ReleaseAt:    music.PublishedAt,
		IsFullLength: false, // The masterdata does not contain full_length info currently
	}

	diffInfo, err := b.buildDifficultyInfo(music.ID)
	if err != nil {
		return nil, err
	}

	// 3. Vocal info
	vocalInfo, err := b.buildVocalInfo(music.ID, regionCode)
	if err != nil {
		return nil, err
	}

	// 4. build jacket path
	jacketPath := b.BuildMusicJacketPath(music.AssetBundleName)

	aliases := b.buildMusicAliases(music)
	if len(aliases) == 0 {
		aliases = []string{}
	}

	// 5. assemble request
	req := &model.MusicDetailRequest{
		Region:          regionCode,
		MusicInfo:       musicInfo,
		Difficulty:      *diffInfo,
		Vocal:           *vocalInfo,
		MusicJacketPath: jacketPath,
		Alias:           aliases,
	}

	if event, err := b.source.GetPrimaryEventByMusicID(music.ID); err == nil && event != nil {
		req.EventID = &event.ID
		req.EventBannerPath = b.buildEventBannerPath(event.AssetBundleName)
	}
	if limited := b.buildLimitedTimes(music.ID, regionCode); len(limited) > 0 {
		req.LimitedTimes = limited
	}

	return req, nil
}

// BuildMusicChartRequest builds chart preview request payload.
func (b *MusicBuilder) BuildMusicChartRequest(query model.MusicChartQuery, music *masterdata.Music) (*model.MusicChartRequest, error) {
	diff := strings.ToLower(strings.TrimSpace(query.Difficulty))
	if diff == "" {
		diff = "master"
	}
	regionCode := strings.ToUpper(strings.TrimSpace(query.Region))
	if regionCode == "" {
		regionCode = "JP"
	}

	playLevel := b.GetDifficultyLevel(music.ID, diff)
	if playLevel == 0 {
		return nil, fmt.Errorf("music %s does not have %s chart", music.Title, diff)
	}

	// 4. build jacket path
	jacketPath := b.BuildMusicJacketPath(music.AssetBundleName)

	susDir := fmt.Sprintf("%04d_01", music.ID)
	susFile := fmt.Sprintf("%s.txt", diff)
	susRelative := filepath.Join("music", "music_score", susDir, susFile)
	susPath := asset.ResolveAssetPath(b.assets, b.assetDir, susRelative)

	if susPath == "" {
		fmt.Printf("[WARN] Sus/Chart file missing: %s\n", susRelative)
	}

	noteHost := "lunabot_static_images/chart_asset/notes"

	stylePath := strings.TrimSpace(query.Style)
	var styleAbs *string
	if stylePath != "" {
		abs := filepath.ToSlash(filepath.Join(b.assetDir, stylePath))
		styleAbs = &abs
	}

	return &model.MusicChartRequest{
		MusicID:    music.ID,
		Title:      b.buildDisplayMusicTitle(music, regionCode),
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

var (
	hanPattern    = regexp.MustCompile(`\p{Han}`)
	kanaPattern   = regexp.MustCompile(`[\p{Hiragana}\p{Katakana}]`)
	hangulPattern = regexp.MustCompile(`\p{Hangul}`)
	latinPattern  = regexp.MustCompile(`[A-Za-z]`)

	musicCNTitleOnce sync.Once
	musicENTitleOnce sync.Once
	musicCNTitleMap  map[string]string
	musicENTitleMap  map[string]string
)

func (b *MusicBuilder) buildDisplayMusicTitle(music *masterdata.Music, region string) string {
	base := strings.TrimSpace(music.Title)
	if base == "" {
		return music.Title
	}

	regionCode := strings.ToUpper(strings.TrimSpace(region))
	if regionCode == "" || regionCode == "JP" {
		return base
	}

	titles, err := b.source.GetMusicLocalizedTitles(music.ID)
	if err != nil || len(titles) == 0 {
		return base
	}
	alt := selectLocalizedTitle(base, regionCode, titles)
	if alt == "" {
		switch regionCode {
		case "CN", "TW":
			alt = lookupI18nMusicTitle(music.ID, "cn")
		case "EN":
			alt = lookupI18nMusicTitle(music.ID, "en")
		}
	}
	if alt == "" {
		return base
	}
	return fmt.Sprintf("%s (%s)", base, alt)
}

func selectLocalizedTitle(base string, region string, titles []string) string {
	candidates := make([]string, 0, len(titles))
	for _, title := range titles {
		trimmed := strings.TrimSpace(title)
		if trimmed == "" || strings.EqualFold(trimmed, base) {
			continue
		}
		candidates = append(candidates, trimmed)
	}
	if len(candidates) == 0 {
		return ""
	}

	prefer := strings.ToUpper(strings.TrimSpace(region))
	if prefer == "CN" || prefer == "TW" {
		for _, c := range candidates {
			if hanPattern.MatchString(c) && !kanaPattern.MatchString(c) {
				return c
			}
		}
		return ""
	}
	if prefer == "KR" {
		for _, c := range candidates {
			if hangulPattern.MatchString(c) {
				return c
			}
		}
		return ""
	}
	if prefer == "EN" {
		for _, c := range candidates {
			if latinPattern.MatchString(c) {
				return c
			}
		}
		return ""
	}
	return candidates[0]
}

func lookupI18nMusicTitle(musicID int, lang string) string {
	key := fmt.Sprintf("%d", musicID)
	switch strings.ToLower(strings.TrimSpace(lang)) {
	case "cn":
		musicCNTitleOnce.Do(func() {
			musicCNTitleMap = fetchI18nMusicTitleMap("https://i18n-json.sekai.best/zh-CN/music_titles.json")
		})
		return strings.TrimSpace(musicCNTitleMap[key])
	case "en":
		musicENTitleOnce.Do(func() {
			musicENTitleMap = fetchI18nMusicTitleMap("https://i18n-json.sekai.best/en/music_titles.json")
		})
		return strings.TrimSpace(musicENTitleMap[key])
	default:
		return ""
	}
}

func fetchI18nMusicTitleMap(url string) map[string]string {
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(url)
	if err != nil || resp == nil {
		return map[string]string{}
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return map[string]string{}
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return map[string]string{}
	}
	var data map[string]string
	if err := json.Unmarshal(body, &data); err != nil {
		return map[string]string{}
	}
	return data
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
	difficulties, err := b.source.GetMusicDifficulties(musicID)
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

// buildDifficultyInfo builds difficulty info.
func (b *MusicBuilder) buildDifficultyInfo(musicID int) (*model.DifficultyInfo, error) {
	diffs, err := b.source.GetMusicDifficulties(musicID)
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

// buildVocalInfo builds vocal information.
func (b *MusicBuilder) buildVocalInfo(musicID int, region string) (*model.MusicVocalInfo, error) {
	vocals, err := b.source.GetMusicVocals(musicID)
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
				if cObj, err := b.source.GetCharacterByID(char.CharacterID); err == nil {
					name = cObj.FirstName + cObj.GivenName
				}
			} else {
				// Virtual Singer
				charID = char.CharacterID
				if cObj, err := b.source.GetCharacterByID(char.CharacterID); err == nil {
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

		caption := normalizeVocalCaption(v.Caption, v.MusicVocalType, v.AssetBundleName, region)

		entry := map[string]interface{}{
			"caption":    caption,
			"characters": chars,
		}
		mapKey := v.AssetBundleName
		if strings.EqualFold(strings.TrimSpace(region), "JP") {
			mapKey = buildJPVocalOrderKey(v)
		}
		infoMap[mapKey] = entry
	}

	return &model.MusicVocalInfo{
		VocalInfo:   infoMap,
		VocalAssets: assetsMap,
	}, nil
}

func buildJPVocalOrderKey(v *masterdata.MusicVocal) string {
	if v == nil {
		return "90_vocal"
	}
	base := strings.TrimSpace(v.AssetBundleName)
	if base == "" {
		base = "vocal"
	}
	// Service-side ordering workaround for JP:
	// drawing sorts vocal groups by count only, so ties depend on input order.
	// Prefix keys to keep VS before Sekai in JSON object order.
	priority := 90
	assetName := strings.ToLower(strings.TrimSpace(v.AssetBundleName))
	switch {
	case strings.HasPrefix(assetName, "vs_"):
		priority = 10
	case strings.HasPrefix(assetName, "se_"):
		priority = 20
	case strings.HasPrefix(assetName, "an_"):
		priority = 30
	}
	return fmt.Sprintf("%02d_%s", priority, base)
}

func (b *MusicBuilder) BuildCharacterIconPath(characterID int) string {
	if nickname, ok := asset.CharacterIDToNickname[characterID]; ok {
		return asset.ResolveAssetPath(b.assets, b.assetDir, filepath.Join("chara_icon", nickname+".png"))
	}
	return asset.ResolveAssetPath(b.assets, b.assetDir, filepath.Join("chara_icon", fmt.Sprintf("chr_icon_%d.png", characterID)))
}

// buildCategories builds category list.
func (b *MusicBuilder) buildCategories(musicID int) []string {
	tags, _ := b.source.GetMusicTags(musicID)
	return tags
}

func (b *MusicBuilder) BuildMusicJacketPath(assetName string) string {
	return filepath.ToSlash(filepath.Join(b.assetDir, "music", "jacket", assetName, assetName+".png"))
}

func (b *MusicBuilder) buildMusicAliases(music *masterdata.Music) []string {
	var aliases []string
	if music == nil {
		return aliases
	}
	if localized, err := b.source.GetMusicLocalizedTitles(music.ID); err == nil {
		for _, title := range localized {
			title = strings.TrimSpace(title)
			if title == "" {
				continue
			}
			if !containsString(aliases, title) {
				aliases = append(aliases, title)
			}
		}
	}
	if pronunciation := strings.TrimSpace(music.Pronunciation); pronunciation != "" {
		if !containsString(aliases, pronunciation) {
			aliases = append(aliases, pronunciation)
		}
	}
	if tags, err := b.source.GetMusicTags(music.ID); err == nil {
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
	limited := b.source.GetLimitedTimeMusics(musicID)
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

var vocalLocalizationByRegion = map[string]map[string]string{
	"en": {
		"sekai":          "Sekai",
		"virtual singer": "Virtual Singer",
	},
	"jp": {
		"sekai":          "Sekai",
		"virtual singer": "Virtual Singer",
	},
	"cn": {
		"sekai":          "「世界」",
		"virtual singer": "虚拟歌手",
	},
	"tw": {
		"sekai":          "「世界」",
		"virtual singer": "虛擬歌手",
	},
	"kr": {
		"sekai":          "세카이",
		"virtual singer": "버추얼 싱어",
	},
}

func normalizeVocalCaption(raw string, vocalType string, assetBundleName string, region string) string {
	preferredFromAsset := classifyVocalByAssetBundle(assetBundleName, region)
	if preferredFromAsset != "" {
		return preferredFromAsset
	}

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
	if strings.HasSuffix(key, "ver") {
		key += "."
	}
	if resolved, ok := vocalCaptionOverrides[key]; ok {
		return localizeVocalCaption(resolved, region)
	}
	if resolved, ok := vocalTypeFallbacks[strings.ToLower(vocalType)]; ok {
		return localizeVocalCaption(resolved, region)
	}
	if strings.EqualFold(key, "virtual singer") {
		return localizeVocalCaption("Virtual Singer", region)
	}
	return trimmed
}
func classifyVocalByAssetBundle(assetBundleName string, region string) string {
	name := strings.ToLower(strings.TrimSpace(assetBundleName))
	switch {
	case strings.HasPrefix(name, "se_"):
		return localizeVocalCaption("Sekai", region)
	case strings.HasPrefix(name, "vs_"):
		return localizeVocalCaption("Virtual Singer", region)
	case strings.HasPrefix(name, "an_"):
		return localizeVocalCaption("Another Vocal", region)
	default:
		return ""
	}
}

func localizeVocalCaption(caption string, region string) string {
	base := strings.TrimSpace(caption)
	if base == "" {
		return caption
	}
	normalizedRegion := strings.ToLower(strings.TrimSpace(region))
	normalizedBase := strings.ToLower(base)
	if regionMap, ok := vocalLocalizationByRegion[normalizedRegion]; ok {
		if localized, ok := regionMap[normalizedBase]; ok {
			return localized
		}
	}
	return base
}

func (b *MusicBuilder) buildEventBannerPath(assetBundleName string) string {
	if assetBundleName == "" {
		return ""
	}
	candidates := []string{
		filepath.Join("home", "banner", assetBundleName, assetBundleName+".png"),
		filepath.Join("event", assetBundleName, "banner.png"),
	}
	return asset.ResolveAssetPath(b.assets, b.assetDir, candidates...)
}
