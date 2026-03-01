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

// MusicBuilder 婵犵數濮烽弫鎼佸磻閻愬搫鍨傞柛顐ｆ礀缁犳澘鈹戦悩瀹犲缂佺姵鐓￠弻娑㈠Ψ椤斿彞绮跺┑鐐村灟閸╁嫰寮崱娑欑厱閻忕偛澧介埥澶嬨亜韫囨梹鍊愭慨濠呮缁辨帒螣濞茬粯鈷栭柣搴ゎ潐濞叉粓寮繝姘畺闁跨喓濮寸粻锝夋煥閺冨倹娅曢柛妯绘倐閺岋絾鎯旈婊呅ｆ繛瀛樼矋缁捇銆侀弮鍫濈厸闁告劦浜為敍婵嬫⒑缁嬫寧婀扮紒瀣笧娴滄悂骞嶉鍓э紲?DrawingAPI 濠电姷鏁告慨鐑藉极閹间礁纾婚柣鎰惈閸ㄥ倿鏌ｉ姀鐘冲暈闁稿顑夐弻鐔兼偋閸喓鍑￠梺鎼炲妼閸婂潡寮诲☉妯兼殕闁逞屽墴瀹曟垿鎮欓崫鍕紵?Music 濠电姷鏁告慨鐑姐€傞挊澹╋綁宕ㄩ弶鎴濈€銈呯箰閻楀棝鎮為崹顐犱簻闁圭儤鍨甸弳鐐烘煟濠垫劒閭柡?JSON Payload
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

// BuildMusicDetailRequest 闂傚倸鍊搁崐椋庣矆娓氣偓楠炴牠顢曢敂缁樻櫈闂佸憡渚楅崹顏堝磻閹炬剚娼╅柣鎾抽椤偆绱撴担浠嬪摵闁圭懓娲悰顔嘉熼懖鈺冿紲濠殿喗顨呭Λ娆撍夋繝鍥ㄢ拻濞达絼璀﹂悞鍓х磼缂佹ê濮嶆鐐诧躬楠炲洭鏌囬敂鑺ユ珦闂備焦鎮堕崕娲礈濮樿泛纾婚柨婵嗩槹閻撴洟鏌曟竟顖氬暙楠炲鈹戦敍鍕闁革綇缍佸璇测槈濞嗘劕鍔呴梺鎸庣☉鐎氼噣顢欐繝鍥ㄢ拺鐟滅増甯╁Λ鎴濃攽閻愨晛浜鹃梻浣告惈鐞氼偊宕濇惔銊ョ劦妞ゆ帒锕︾粔鐢告煕閹炬潙鍝虹€?
func (b *MusicBuilder) BuildMusicDetailRequest(music *masterdata.Music, region string) (*model.MusicDetailRequest, error) {
	regionCode := strings.ToUpper(strings.TrimSpace(region))
	if regionCode == "" {
		regionCode = "JP"
	}
	displayTitle := b.buildDisplayMusicTitle(music, regionCode)
	// 1. 闂傚倸鍊搁崐鐑芥嚄閸撲焦鍏滈柛顐ｆ礀閻ら箖鏌ｉ幇顓犮偞闁哄绉归弻銊モ攽閸♀晜肖闂侀€炲苯鍘哥紒鑸佃壘閻ｇ兘濡搁埡濠冩櫓闂備焦顑欓崹閬嶅极閻㈠憡鈷掑ù锝堫潐閻忛亶鏌￠崨顔炬创鐎规洦鍨堕崺锟犲礃閳哄﹥缍?MusicMD
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

	// 2. 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸缁犵娀鐓崶銊р槈缁炬崘娉曢埀顒冾潐濞叉牕煤閻樿纾婚柟鎹愬煐閸忔粓鏌涘☉鍗炴灓鐟滄妸鍛＝濞达絽鎽滈悞绋棵瑰鍐煟鐎规洘妞藉畷鐔碱敍濮橀硸妲伴柣鐔哥矊缁绘帞鍒掗崼銉ョ劦?DifficultyInfo
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

	// 2. 闂傚倸鍊峰ù鍥х暦閸偅鍙忛柟缁㈠櫘閺佸嫬顭跨捄渚剳妞も晜鐓￠幃瑙勬姜閹峰矈鍔呴梺缁樺笂缁瑩寮婚悢灏佹灁闁割煈鍠楅悘宥夋倵鐟欏嫭绀€妞わ缚鍗虫俊鐢稿礋椤栨艾鍞ㄩ梺闈涱煭婵″洤鈻撻妶鍥╃＝濞达絿鎳撴慨澶愭煕鐎ｃ劌鈧繂顕ｆ繝姘╅柕澶堝灪椤秴鈹戦悙鍙夘棡閻㈩垱甯¤棢婵犻潧顑嗛埛鎴︽偣閸ワ絺鍋撻搹顐や憾婵＄偑鍊戦崝宀勬偋韫囨拹锝夊箛閺夎法顔掗柣鐘叉穿鐏忔瑩宕濋敃鈧—鍐Χ閸℃鐟愰梺鐓庡暱閻栧ジ宕?.txt闂傚倸鍊搁崐鐑芥倿閿旈敮鍋撶粭娑樻噽閻瑩鏌熺€电浠ч梻鍕閺岋繝宕橀妸銉㈠亾閼姐倗涓嶉柡鍐ㄧ墛閻撴稓鈧箍鍎辨鍛婄閻愮儤鐓熸い鎾跺枑鐏忣參鏌嶇憴鍕伌闁诡喒鏅涢悾鐑藉炊閼稿灚顔愰梻鍌欒兌缁垳鎹㈤崟顖氱獥闁哄诞灞剧稁婵犵數濮甸懝鎯ь啅濠靛洢浜滈柡鍐ㄥ€哥敮璺衡攽椤斿搫鐏查柡宀嬬稻閹棃鏁愰崱妤佺暚缂備礁澧介搹搴ㄥ矗閸愵喖鏄ラ柍褜鍓氶妵鍕箳瀹ュ牆鍘￠梺鎰佸灡濞茬喖寮诲☉銏犵睄闁逞屽墴閹囨偐瀹割喖娈ㄩ梺褰掓？缁€浣虹矆閸愵喗鐓冮柛婵嗗閺嬨倖绻涢弶鎴濐伃闁哄矉缍侀幃鈺呮濞戞ê顬嗗┑鐐茬摠缁繑鎱ㄩ妶鍥╃焿?	// 婵犵數濮烽。钘壩ｉ崨鏉戠；闁告侗鍙庨悢鍡樹繆椤栨氨姣為柛瀣尭椤繈鎮℃惔锛勵啇闂備焦瀵уú蹇涘垂娴犲鏋侀柟鍓х帛閸嬫劙鏌涢幇顖氱处缂傚啯娲熷缁樻媴缁嬫寧姣愰梺鍦拡閸嬪﹤鐣烽鐑嗘晬闁绘劘灏欓敍娆撴⒑缂佹ê濮夐柛搴涘€濆畷鎴︽晲閸涱偀鍋撻幒鎴僵闁绘挸娴锋禒鈺呮⒑鐠囪尙绠冲┑顕€顥撳Σ? music/music_score/{id_major}_{id_minor}/{diff}.txt
	susDir := fmt.Sprintf("%04d_01", music.ID)
	susFile := fmt.Sprintf("%s.txt", diff)
	susRelative := filepath.Join("music", "music_score", susDir, susFile)
	susPath := asset.ResolveAssetPath(b.assets, b.assetDir, susRelative)

	if susPath == "" {
		fmt.Printf("[WARN] Sus/Chart file missing: %s\n", susRelative)
	}

	// 3. NoteHost闂傚倸鍊搁崐鐑芥倿閿旈敮鍋撶粭娑樻噽閻瑩鏌熼悜妯虹劸婵炲皷鏅犻弻鏇熺箾閸喖澹勯梺缁樺灦閿氭い鏇憾閺屸剝寰勭€ｎ亞浠煎銈忓瘜閸欏啴骞冨Δ鍛祦闁割煈鍠栨慨搴ㄦ煟鎼淬垹鍤柛鐘虫皑閸掓帞绱掑Ο纰辨祫闁诲函缍嗛崑鍡涘储娴犲鈷戦柟绋挎捣缁犳挻淇婇锝囨创妤犵偛锕畷鍗炩槈濞嗘垵骞楁俊鐐€栭幐楣冨窗鎼淬劌绀夐柕鍫濇娴滄粓鏌￠崶顭嬵亪宕濋敂閿亾鐟欏嫭绀堥柛鐘崇墵閵嗕礁鈻庨幒婵囆梺姹囧焺閸亪鍩€椤掍礁澧柛?	// 婵犵數濮烽。钘壩ｉ崨鏉戠；闁告侗鍙庨悢鍡樹繆椤栨氨姣為柛瀣尭椤繈鎮℃惔锛勵啇闂備焦瀵уú蹇涘垂娴犲鏋侀柟鍓х帛閸嬫劙鏌涢幇顖氱处缂傚啯娲熷缁樻媴缁嬫寧姣愰梺鍦拡閸嬪﹤鐣烽鐑嗘晬闁绘劘灏欓敍娆撴⒑缂佹ê濮夐柛搴涘€濆畷鎴︽晲閸涱偀鍋撻幒鎴僵闁绘挸娴锋禒鈺呮⒑鐠囪尙绠冲┑顕€顥撳Σ? Z:/pjskdata/Data/lunabot_static_images/chart_asset/notes
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

// BuildCharacterIconPath 闂傚倸鍊搁崐椋庣矆娓氣偓楠炴牠顢曢敂缁樻櫈闂佸憡渚楅崹顏堝磻閹炬剚娼╅柣鎾抽椤偆绱撴担浠嬪摵闁圭懓娲悰顔碱潨閳ь剟骞婂鍫燁棃婵炴垶鐟ч埀顒夊弮濮婄粯鎷呴崨濠呯闂佺绨洪崐鏇犲弲闂佸搫绋侀崢鏃€寰勬惔锝囨澑濠电偞鍨堕悷褔宕㈤柆宥嗏拺闂傚牊鍐荤槐锟犳煕閹板苯鍠氬Λ婊堟⒒閸屾瑨鍏岀紒顕呭灦閹囶敇閻樿尙绛忛梺绉嗗嫷娈旂紒鐘崇叀閺屾盯寮撮妸銉т紘闂佽桨绀佺粔鐢稿箞閵娿儙鐔稿緞缁嬫寧鍎撻梻?
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

// BuildMusicJacketPath 闂傚倸鍊搁崐椋庣矆娓氣偓楠炴牠顢曢敂缁樻櫈闂佸憡渚楅崹顏堝磻閹炬剚娼╅柣鎾抽椤偆绱撴担浠嬪摵闁圭懓娲悰顔碱潨閳ь剙顕ｉ悽鍛婂仺闂傚牊绋戦埛鏇熺節閻㈤潧浠﹂柟绋款煼瀹曟椽宕橀鑲╋紱闂佽鍎抽顓犵不妤ｅ啯鐓曢煫鍥ㄦ礀鐢埖銇勯妷銉剶闁哄瞼鍠栭幃鐑藉级濞嗗彞绱旀俊鐐€戦崹铏圭矙閹达讣缍栨繝闈涙处缂嶅洭鏌嶉崫鍕殭缂?
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
	"闂傚倸鍊搁崐椋庢濮橆剦鐒界憸鏃堝箖瑜庣换婵嬪炊閵娿儱澹嬮柣鐔哥矌婢ф鏁幒鏂款棜濠电姵纰嶉悡娆撴煟閹寸儑渚涙繛鍫熸瀹曞爼寮堕幋顓熷瘜闂侀潧鐗嗙换妯侯瀶椤斿墽纾奸柡鍐ㄥ€婚敍?": "Sekai",
	"闂傚倸鍊搁崐椋庢濮橆剦鐒界憸鏃堝箖瑜庣换婵嬪炊閵娿儱澹嬮柣鐔哥矌婢ф鏁幒鏂款棜濠电姵纰嶉悡娆撴煟閹寸儑渚涙繛鍫熸瀹?ver.":                         "Sekai",
	"闂傚倸鍊搁崐椋庢濮橆剦鐒界憸鏃囨婵炴潙鍚嬪娆戠矆婢舵劖鐓欓弶鍫濆⒔閻ｈ京绱掗埀顒勫磼濞戞绠氶梺闈涚墕閹冲繘宕宠ぐ鎺戣埞妞ゆ牗绋撶粻楣冨级閸繂鈷旂紒澶樺枟缁绘稒鎷呴崘鍙夋悙缂佲偓婢跺鍙忔俊鐐额嚙娴滈箖姊洪崫鍕拱婵炲弶锚椤曘儵宕熼瀣枑濞碱亪骞嶉钘夌樆闂傚倸鍊烽懗鍫曗€﹂崼銏″床闁硅揪绠戠粻鏌ユ煕閵夛絽濡虹紒璇叉閺屾盯顢曢敐鍡欘槬闂佸憡鍨规慨鐢稿Φ閸曨垰绠涢柍杞拌兌娴犲ジ鏌ｉ姀鈺佺仭闁圭懓娲ら～蹇撁洪鍜佹濠电偞鍨堕懝楣冨煕閸儲鈷戠紒瀣硶缁犳煡鏌ｉ悢鍙夋珔闁伙絽鍢茶灃闁逞屽墴閿濈偠绠涘☉娆忎虎濡?": "Virtual Singer",
	"闂傚倸鍊搁崐椋庢濮橆剦鐒界憸鏃囨婵炴潙鍚嬪娆戠矆婢舵劖鐓欓弶鍫濆⒔閻ｈ京绱掗埀顒勫磼濞戞绠氶梺闈涚墕閹冲繘宕宠ぐ鎺戣埞妞ゆ牗绋撶粻楣冨级閸繂鈷旂紒澶樺枟缁绘稒鎷呴崘鍙夋悙缂佲偓婢跺鍙忔俊鐐额嚙娴滈箖姊洪崫鍕拱婵炲弶锚椤曘儵宕熼瀣枑濞碱亪骞嶉钘夌樆闂傚倸鍊烽懗鍫曗€﹂崼銉晞闁瑰濮靛畷鍙夌節闂堟稒鐭楃紒璇叉閺岋綁骞嬮敐鍛呮捇鏌涙繝鍕幋闁哄矉绠戣灒濞撴凹鍨卞暩濠电姰鍨奸～澶娒洪悢濂夋綎婵炲樊浜濋ˉ鍫熺箾閹达綁鍝哄Δ鏃傜磽娴ｇ懓鍔ゅ褌绮欓獮鎰版儑?":                          "Virtual Singer",
	"闂傚倸鍊搁崐椋庢濮橆剦鐒界憸鏃堝箖瑜嶉…銊╁礃閸撗冨Ш闂備胶顫嬮崟鍨暥缂佺偓鍎冲锟犲蓟閿熺姴鐐婇柕澶堝劤娴犲ジ鏌涢妷锔藉唉婵﹦绮换婵囨償閳ユ剚娼介柣鐔哥矋濠㈡﹢宕弶鎴犳殾闁靛繈鍊曠粻顕€鏌ら幁鎺戝姢闁告﹩浜濈换婵嬪閿濆棛銆愬銈嗗灥閹冲酣鍩㈤幘缁樺亹缂備焦顭囬崢閬嶆椤愩垺澶勬俊顐㈢箰閳绘挸顭ㄩ崘锝嗘杸闂佺偨鍎辩壕顓㈠春閿濆棎浜滈柕蹇婂墲缁€瀣煙椤旇娅婃鐐存崌楠炴帡骞婇崜浣插亾椤＄幎.":                                                           "Another Vocal",
	"闂傚倸鍊搁崐椋庣矆娓氣偓楠炲鏁撻悩顐熷亾閿曞倸鐐婃い鎺嗗亾缂佹劖顨婇弻鐔兼焽閿曗偓閸旀帡鏌ц箛鎾磋础闁告宀搁弻鏇㈠幢濡や焦娅?": "Original Song",
	"闂傚倸鍊搁崐椋庣矆娓氣偓楠炲鏁撻悩顐熷亾閿曞倸鐐婃い鎺嗗亾缂佹劖顨婇弻鐔兼焽閿曗偓閸?ver.":                  "Original Song",
	"闂傚倸鍊搁崐椋庢濮橆剦鐒界憸鏃堝箖瑜斿畷鍗炩枎閹存粎鐟濋柣搴＄畭閸庨亶藝椤栨粌顥氶柛褎顨嗛悡娆撴煙鐟欏嫬濮﹂柛銈嗙懇瀹曨剟顢涘鍏煎瘜闂侀潧鐗嗛幊鎰不娴煎瓨鐓曞┑鐘插暞閸婃劗鈧娲忛崹浠嬬嵁鎼淬劍鍤嶉柕澹啫濡囨繝鐢靛О閸ㄧ厧鈻斿☉銏″殣妞ゆ牜鍋涢崙鐘绘煕瀹€鈧崑鐐烘偂閺囥垻鍙撻柛銉ｅ妽鐏忣參鏌嶈閸撴氨鈧瑳鍕攳濠电姴娲ら柋鍥煛閸モ晛鏋庢繛鍫熺箓椤啴濡堕崱妤€娼戦梺绋款儐閹瑰洭寮诲☉銏犵睄闁规儳澧庨弳銈夋⒑閸濆嫭婀版繛鑼枎閻ｇ兘鎮℃惔妯活潔濠碘槅鍨伴崲鍙夌妤ｅ啯鐓欓弶鍫濆⒔閻ｈ京鐥悙顒€顕滈柕鍥у閺佹劙宕卞▎妯圭椽r.":                                                                                                                                    "Connect Live",
	"闂傚倸鍊搁崐椋庢濮橆剦鐒界憸鏃堝箖瑜斿畷鍗炩枎閹存粎鐟濋柣搴＄畭閸庨亶藝椤栨粌顥氶柛褎顨嗛悡娆撴煙鐟欏嫬濮﹂柛銈嗙懇瀹曨剟顢涘鍏煎瘜闂侀潧鐗嗛幊鎰不娴煎瓨鐓曞┑鐘插暞閸婃劗鈧娲忛崹浠嬬嵁鎼淬劍鍤嶉柕澹啫濡囨繝鐢靛О閸ㄧ厧鈻斿☉銏″殣妞ゆ牜鍋涢崙鐘绘煕瀹€鈧崑鐐烘偂閺囥垻鍙撻柛銉ｅ妽鐏忣參鏌嶈閸撴氨鈧瑳鍕攳濠电姴娲ら柋鍥煛閸モ晛鏋庢繛鍫熺箓椤啴濡堕崱妤€娼戦梺绋款儐閹瑰洭寮诲☉銏犵睄闁规儳澧庨弳銈夋⒑閸濆嫭婀版繛鑼枎閻ｇ兘鎮℃惔妯活潔濠碘槅鍨伴崲鍙夌妤ｅ啯鐓欓弶鍫濆⒔閻ｈ京鐥?ver.":                                                                                                                                                    "Connect Live",
	"闂傚倸鍊搁崐椋庢濮橆剦鐒界憸鏃堝箖瑜斿畷鐔碱敍濮橀硸妲峰┑鐘垫暩婵潙煤閵堝洤顥氬┑鐘崇閻撴洟鏌嶉埡浣告殶闁宠棄顦靛畷顒勵敍濮橈絾鏂€闂佹寧绋戠€氼剚绂掕閺屾稑螣閻樺弶鎼愮紒鈧径灞惧枑闁哄啫鐗嗛拑鐔兼煛閸モ晛鏋庣紒鍓佸仦娣囧﹪顢涘┑鍥モ偓鍐磼閵婏附灏︽慨濠呮缁瑩骞愭惔銏犻棷濠电姷鏁搁崑鎰板磻閹惧绡€缁炬澘顦辩弧鈧銈冨劜閹搁箖宕氶幒妤婃晣闁靛繒濮崇槐鍫曟⒑閸涘﹥澶勯柛瀣閳er.":                                                                                                                                                                                                           "April Fool",
	"闂傚倸鍊搁崐椋庢濮橆剦鐒界憸宥囧弲闂侀潧鐗嗛ˇ顖滃瑜版帗鐓曟い鎰剁稻缁€鈧紒鐐劤缂嶅﹤顫忓ú顏嶆晢闁逞屽墰缁梻鈧稒锕╅弨浼存⒒閸屾瑧顦﹂柟纰卞亜鐓ら柨鏇炲€归弲顏堟⒒娓氣偓閳ь剛鍋涢懟顖涙櫠閺夋嚚鐟邦煥閸℃鍘繛锝呮搐閿曨亪骞冩禒瀣紶闁靛鍎辩徊楣冩⒒娴ｅ憡璐＄紒顕呭灠椤斿繑绻濆顒傦紱闂佺懓澧界划顖炲磻閹扮増鍋ｉ柛銉戝啯顥濋悷婊冮叄閵嗗啴濡烽埡鍌氣偓鐑芥煠绾板崬鍘搁柧蹇撳缁绘稓鈧數顭堥埢鍫㈢磼閻樺磭澧电€殿喖顭峰鎾偄妞嬪海鐛柣搴″帨閸嬫挸鈹戦悩鎻掆偓鍛婁繆閹绢喗鈷掑ù锝勮閻掔偓銇勯幋婵嗘殻鐎规洘鍔曢埞鎴犫偓锝庝簼閻庮剟鎮楅獮鍨姎婵☆偅鐩幃鐑芥偡閹佃櫕鏂€闂佺粯蓱瑜板啴顢旈锔界叆婵炴垶鐟ч惌鎺撴叏婵犲啯銇濇俊顐㈠暙閳藉顫濋钘夊綑闂傚倷鑳堕…鍫ヮ敄閸愵喖绀夐柟瀛樼箘閺嗭箓鏌￠崶銉ョ仾闁稿鍔庣槐鎺斺偓锝庡弾閸わ箓鏌曟繝蹇擃洭缂佺娀绠栭弻鐔煎垂椤斿吋娈插┑顔兼贡閳?": "Ensemble Stars!! Collab",
	"闂傚倸鍊搁崐椋庢濮橆剦鐒界憸宥堢亱濠德板€曢幊蹇涘吹閹邦厹浜滈柡宥冨妿閳笺儳绱掔拠鍙夘棦闁哄矉缍佹慨鈧柕蹇婃櫇閸旀悂姊绘担绋胯埞婵炲樊鍙冮獮鍐ㄎ旈崨顔间缓濠电偛鐗愬▔鏇㈠礉缁嬫娓婚柕鍫濈箳閸掍即鏌曢崼鐔稿€愮€殿喖顭峰鎾偄妞嬪海鐛梺璇插嚱缁插宕濈€ｎ剙鍨濆┑鐘崇閳锋垹绱掔€ｎ亝鍋ユい搴㈩殘缁辨挸顓奸崪鍐╂暰婵烇絽娲ら敃顏勭暦婵傜鍗抽柣鎰問閸熷洭姊绘担铏瑰笡闁告梹锚椤曪綁骞橀崜浣猴紴闂佸搫娲㈤崹娲煕閹达附鈷掗柛顐ゅ枔閵嗘帒顭胯娴滃爼寮诲☉鈶┾偓锕傚箣濠靛棌鎷柣搴ゎ潐濞叉ê煤閻旇偐宓佹俊顖氬悑鐎氭岸鏌涘▎蹇撯偓鎴﹀礋椤掆偓瀵潡姊洪懡銈呮灁濠⒀勵殜楠炴牠骞囬鐐垫嚀椤劑鍩€椤掑倸鍨濇繛鍡樻尭閽冪喖鏌￠崶銉ョ仼闂佸崬娲︾换娑㈠箣閻愬啯宀稿畷鎴﹀箻鐠囪尙顓洪梺鎸庢磵閸嬫捇鏌ｉ幒鎴犱粵闁靛洤瀚伴獮鎺楀幢濡炴儳顥氶梻鍌欑濠€閬嶆惞鎼粹垾锝嗗垔?":                    "Movie",
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
