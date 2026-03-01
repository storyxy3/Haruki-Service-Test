package service

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Extractor 通用特征提取器
type Extractor struct {
	nicknames map[string]int // 昵称 -> CharacterID
}

var supportedRegions = []string{"jp", "en", "cn", "tw", "kr"}

// NewExtractor 创建提取器
func NewExtractor(nicknames map[string]int) *Extractor {
	return &Extractor{
		nicknames: nicknames,
	}
}

// ExtractResult 提取结果
type ExtractResult[T any] struct {
	Value     T
	Remaining string
	Found     bool
}

// -----------------------------------------------------------------------------
// 1. 角色提取
// -----------------------------------------------------------------------------

// ExtractCharacter 提取角色 ID
// 逻辑：遍历别名库，找到匹配的昵称
func (e *Extractor) ExtractCharacter(text string) ExtractResult[int] {
	textLower := strings.ToLower(text)
	// 简单实现：遍历 map。为了避免短昵称误判（如 "an"），应该优先匹配长昵称？
	// 原版 Lunabot 逻辑比较简单，只要包含就行。
	// 但要注意 "miku" 和 "mikuday" 的区别。这里先做简单包含匹配。

	// 优化：优先匹配最长的昵称
	var bestNickname string
	var bestID int

	for nickname, id := range e.nicknames {
		if strings.Contains(textLower, nickname) {
			if len(nickname) > len(bestNickname) {
				bestNickname = nickname
				bestID = id
			}
		}
	}

	if bestNickname != "" {
		// 移除匹配到的昵称
		// 注意：这里只移除一次，且不区分大小写地替换
		re := regexp.MustCompile("(?i)" + regexp.QuoteMeta(bestNickname))
		remaining := re.ReplaceAllString(text, "")
		return ExtractResult[int]{Value: bestID, Remaining: strings.TrimSpace(remaining), Found: true}
	}

	return ExtractResult[int]{Value: 0, Remaining: text, Found: false}
}

// -----------------------------------------------------------------------------
// 2. 稀有度提取
// -----------------------------------------------------------------------------

var rarityMap = map[string]string{
	"4星": "rarity_4", "4star": "rarity_4", "四星": "rarity_4",
	"3星": "rarity_3", "3star": "rarity_3", "三星": "rarity_3",
	"2星": "rarity_2", "2star": "rarity_2", "二星": "rarity_2",
	"1星": "rarity_1", "1star": "rarity_1", "一星": "rarity_1",
	"生日": "rarity_birthday", "birthday": "rarity_birthday",
}

func (e *Extractor) ExtractRarity(text string) ExtractResult[string] {
	textLower := strings.ToLower(text)
	for k, v := range rarityMap {
		if strings.Contains(textLower, k) {
			re := regexp.MustCompile("(?i)" + regexp.QuoteMeta(k))
			remaining := re.ReplaceAllString(text, "")
			return ExtractResult[string]{Value: v, Remaining: strings.TrimSpace(remaining), Found: true}
		}
	}
	return ExtractResult[string]{Value: "", Remaining: text, Found: false}
}

func (e *Extractor) ExtractRegionPrefix(text string) ExtractResult[string] {
	trimmed := strings.TrimSpace(text)
	if !strings.HasPrefix(trimmed, "/") {
		return ExtractResult[string]{Value: "", Remaining: text, Found: false}
	}
	afterSlash := trimmed[1:]
	lower := strings.ToLower(afterSlash)
	for _, region := range supportedRegions {
		if strings.HasPrefix(lower, region) {
			nextIdx := len(region)
			if len(afterSlash) > nextIdx && isAsciiLetter(afterSlash[nextIdx]) {
				continue
			}
			remaining := "/" + strings.TrimLeftFunc(afterSlash[nextIdx:], isSpaceOrTab)
			return ExtractResult[string]{Value: region, Remaining: remaining, Found: true}
		}
	}
	return ExtractResult[string]{Value: "", Remaining: text, Found: false}
}

func isAsciiLetter(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z')
}

func isSpaceOrTab(r rune) bool {
	return r == ' ' || r == '\t'
}

// -----------------------------------------------------------------------------
// 3. 属性提取
// -----------------------------------------------------------------------------

var attrMap = map[string]string{
	"cute": "cute", "可爱": "cute", "粉": "cute",
	"cool": "cool", "帅气": "cool", "蓝": "cool",
	"pure": "pure", "纯真": "pure", "草": "pure", "绿": "pure",
	"happy": "happy", "快乐": "happy", "橙": "happy",
	"mysterious": "mysterious", "神秘": "mysterious", "紫": "mysterious",
}

func (e *Extractor) ExtractAttribute(text string) ExtractResult[string] {
	textLower := strings.ToLower(text)
	for k, v := range attrMap {
		if strings.Contains(textLower, k) {
			re := regexp.MustCompile("(?i)" + regexp.QuoteMeta(k))
			remaining := re.ReplaceAllString(text, "")
			return ExtractResult[string]{Value: v, Remaining: strings.TrimSpace(remaining), Found: true}
		}
	}
	return ExtractResult[string]{Value: "", Remaining: text, Found: false}
}

// -----------------------------------------------------------------------------
// 4. 技能提取
// -----------------------------------------------------------------------------

var skillMap = map[string]string{
	"分": "score_up", "p分": "perfect_score_up", "大分": "great_score_up", // 需细化
	"判": "judgment_accuracy_up", "判定": "judgment_accuracy_up",
	"奶": "life_recovery", "回复": "life_recovery",
}

func (e *Extractor) ExtractSkill(text string) ExtractResult[string] {
	textLower := strings.ToLower(text)
	for k, v := range skillMap {
		if strings.Contains(textLower, k) {
			re := regexp.MustCompile("(?i)" + regexp.QuoteMeta(k))
			remaining := re.ReplaceAllString(text, "")
			return ExtractResult[string]{Value: v, Remaining: strings.TrimSpace(remaining), Found: true}
		}
	}
	return ExtractResult[string]{Value: "", Remaining: text, Found: false}
}

// -----------------------------------------------------------------------------
// 5. 限定类型提取
// -----------------------------------------------------------------------------

// SupplyType 常量定义 (需与 MasterData 对齐)
const (
	SupplyNormal   = "normal"
	SupplyLimited  = "limited" // 泛指所有限定
	SupplyFes      = "festival"
	SupplyBirthday = "birthday"
)

var supplyMap = map[string]string{
	"fes": "festival", "fES": "festival",
	"限定": "limited", "limit": "limited",
	"常驻": "normal", "非限": "normal",
	"生日": "birthday",
}

func (e *Extractor) ExtractSupply(text string) ExtractResult[string] {
	textLower := strings.ToLower(text)
	for k, v := range supplyMap {
		if strings.Contains(textLower, k) {
			re := regexp.MustCompile("(?i)" + regexp.QuoteMeta(k))
			remaining := re.ReplaceAllString(text, "")
			return ExtractResult[string]{Value: v, Remaining: strings.TrimSpace(remaining), Found: true}
		}
	}
	return ExtractResult[string]{Value: "", Remaining: text, Found: false}
}

// -----------------------------------------------------------------------------
// 6. 年份提取
// -----------------------------------------------------------------------------

// -----------------------------------------------------------------------------
// 7. 通用参数提取 (Flags)
// -----------------------------------------------------------------------------

// ExtractRegion 提取区服参数 (例如: -r jp, -r en)
func (e *Extractor) ExtractRegion(text string) ExtractResult[string] {
	re := regexp.MustCompile(`(?i)-r\s+([a-zA-Z]{2})`)
	if matches := re.FindStringSubmatch(text); len(matches) > 1 {
		region := strings.ToLower(matches[1])
		remaining := re.ReplaceAllString(text, "")
		return ExtractResult[string]{Value: region, Remaining: strings.TrimSpace(remaining), Found: true}
	}
	return ExtractResult[string]{Value: "", Remaining: text, Found: false}
}

// ExtractPreview 提取预览标志 (例如: -p, --preview)
func (e *Extractor) ExtractPreview(text string) ExtractResult[bool] {
	re := regexp.MustCompile(`(?i)-p|--preview`)
	if re.MatchString(text) {
		remaining := re.ReplaceAllString(text, "")
		return ExtractResult[bool]{Value: true, Remaining: strings.TrimSpace(remaining), Found: true}
	}
	return ExtractResult[bool]{Value: false, Remaining: text, Found: false}
}

// ExtractHelp 提取帮助标志 (例如: -h, --help, 帮助)
func (e *Extractor) ExtractHelp(text string) ExtractResult[bool] {
	re := regexp.MustCompile(`(?i)-h|--help|帮助`)
	if re.MatchString(text) {
		remaining := re.ReplaceAllString(text, "")
		return ExtractResult[bool]{Value: true, Remaining: strings.TrimSpace(remaining), Found: true}
	}
	return ExtractResult[bool]{Value: false, Remaining: text, Found: false}
}

// ExtractVerbose 提取详细模式标志 (例如: -v, --verbose)
func (e *Extractor) ExtractVerbose(text string) ExtractResult[bool] {
	re := regexp.MustCompile(`(?i)-v|--verbose`)
	if re.MatchString(text) {
		remaining := re.ReplaceAllString(text, "")
		return ExtractResult[bool]{Value: true, Remaining: strings.TrimSpace(remaining), Found: true}
	}
	return ExtractResult[bool]{Value: false, Remaining: text, Found: false}
}

func (e *Extractor) ExtractYear(text string) ExtractResult[int] {
	// ... (content remains the same, just keeping consistency)
	return e.extractYearInternal(text)
}

func (e *Extractor) extractYearInternal(text string) ExtractResult[int] {
	reFull := regexp.MustCompile(`(20\d{2})年?`)
	reShort := regexp.MustCompile(`(\d{2})年`)

	if matches := reFull.FindStringSubmatch(text); len(matches) > 1 {
		year, _ := strconv.Atoi(matches[1])
		remaining := reFull.ReplaceAllString(text, "")
		return ExtractResult[int]{Value: year, Remaining: strings.TrimSpace(remaining), Found: true}
	}

	if matches := reShort.FindStringSubmatch(text); len(matches) > 1 {
		year, _ := strconv.Atoi("20" + matches[1])
		remaining := reShort.ReplaceAllString(text, "")
		return ExtractResult[int]{Value: year, Remaining: strings.TrimSpace(remaining), Found: true}
	}

	if strings.Contains(text, "去年") {
		year := time.Now().Year() - 1
		remaining := strings.Replace(text, "去年", "", 1)
		return ExtractResult[int]{Value: year, Remaining: strings.TrimSpace(remaining), Found: true}
	}

	if strings.Contains(text, "今年") {
		year := time.Now().Year()
		remaining := strings.Replace(text, "今年", "", 1)
		return ExtractResult[int]{Value: year, Remaining: strings.TrimSpace(remaining), Found: true}
	}

	return ExtractResult[int]{Value: 0, Remaining: text, Found: false}
}

// ExtractID 提取纯数字 ID
func (e *Extractor) ExtractID(text string) ExtractResult[int] {
	re := regexp.MustCompile(`^\s*(\d+)\s*$`)
	if matches := re.FindStringSubmatch(text); len(matches) > 1 {
		id, _ := strconv.Atoi(matches[1])
		return ExtractResult[int]{Value: id, Remaining: "", Found: true}
	}
	return ExtractResult[int]{Value: 0, Remaining: text, Found: false}
}
