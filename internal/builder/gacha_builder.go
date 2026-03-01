package builder

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"Haruki-Service-API/internal/model"
	"Haruki-Service-API/internal/service"
	"Haruki-Service-API/pkg/asset"
	"Haruki-Service-API/pkg/masterdata"
)

const gachaEndPaddingMillis = int64(time.Minute / time.Millisecond)

// GachaBuilder 负责卡池相关的数据组装
type GachaBuilder struct {
	source   service.GachaDataSource
	assets   *asset.AssetHelper
	assetDir string
}

// NewGachaBuilder 创建 GachaBuilder
func NewGachaBuilder(source service.GachaDataSource, assets *asset.AssetHelper, assetDir string) *GachaBuilder {
	return &GachaBuilder{
		source:   source,
		assets:   assets,
		assetDir: assetDir,
	}
}

// BuildGachaListRequest 构建卡池列表请求体
func (b *GachaBuilder) BuildGachaListRequest(query model.GachaListQuery) (*model.GachaListRequest, error) {
	page := query.Page
	if page <= 0 {
		page = 1
	}
	pageSize := query.PageSize
	if pageSize <= 0 {
		pageSize = 6
	}

	now := time.Now()
	var filtered []*masterdata.Gacha
	all := b.source.GetGachas()

	cardFilter := query.CardID
	keyword := strings.ToLower(strings.TrimSpace(query.Keyword))

	for _, g := range all {
		if query.Year > 0 {
			year := time.UnixMilli(g.StartAt).Year()
			if year != query.Year {
				continue
			}
		}
		if !query.IncludeFuture && time.UnixMilli(g.StartAt).After(now) {
			continue
		}
		if !query.IncludePast && time.UnixMilli(g.EndAt).Before(now) {
			continue
		}
		if cardFilter > 0 && !gachaContainsCard(g, cardFilter) {
			continue
		}
		if keyword != "" && !strings.Contains(strings.ToLower(g.Name), keyword) {
			continue
		}
		if query.IsRerelease && !strings.Contains(strings.ToLower(g.Name), "it's back") &&
			!strings.Contains(strings.ToLower(g.Name), "复刻") {
			continue
		}
		if query.IsRecall && !strings.Contains(strings.ToLower(g.Name), "回响") &&
			!strings.Contains(strings.ToLower(g.Name), "colorful festival") {
			continue
		}
		if query.OnlyCurrent {
			if time.UnixMilli(g.StartAt).After(now) || time.UnixMilli(g.EndAt).Before(now) {
				continue
			}
		}
		filtered = append(filtered, g)
	}

	if len(filtered) == 0 {
		return nil, fmt.Errorf("no gacha data matched filters")
	}

	sort.Slice(filtered, func(i, j int) bool {
		if filtered[i].StartAt == filtered[j].StartAt {
			return filtered[i].ID > filtered[j].ID
		}
		return filtered[i].StartAt > filtered[j].StartAt
	})

	startIndex := (page - 1) * pageSize
	if startIndex >= len(filtered) {
		startIndex = 0
		page = 1
	}
	endIndex := startIndex + pageSize
	if endIndex > len(filtered) {
		endIndex = len(filtered)
	}
	selected := filtered[startIndex:endIndex]
	region := query.Region
	if region == "" {
		region = b.source.DefaultRegion()
	}

	var briefs []model.GachaBrief
	logos := make(map[int]string)
	for _, g := range selected {
		briefs = append(briefs, model.GachaBrief{
			ID:        g.ID,
			Name:      g.Name,
			GachaType: g.GachaType,
			StartAt:   g.StartAt,
			EndAt:     g.EndAt,
			AssetName: g.AssetBundleName,
		})
		logos[g.ID] = b.buildGachaLogoPath(g)
	}

	return &model.GachaListRequest{
		Gachas:     briefs,
		PageSize:   pageSize,
		Region:     region,
		GachaLogos: logos,
		Filter: model.GachaFilter{
			Page: page,
		},
	}, nil
}

// BuildGachaDetailRequest 构建卡池详情请求体
func (b *GachaBuilder) BuildGachaDetailRequest(query model.GachaDetailQuery) (*model.GachaDetailRequest, error) {
	if query.GachaID == 0 {
		return nil, fmt.Errorf("gacha id is required")
	}
	gacha, err := b.source.GetGachaByID(query.GachaID)
	if err != nil {
		return nil, err
	}

	region := query.Region
	if region == "" {
		region = b.source.DefaultRegion()
	}

	rarityCounts := map[string]int{
		"rarity_1":        0,
		"rarity_2":        0,
		"rarity_3":        0,
		"rarity_4":        0,
		"rarity_birthday": 0,
	}
	cardWeight := make(map[int]float64)
	cardRarity := make(map[int]string)
	cardCache := make(map[int]*masterdata.Card)
	rarityWeights := make(map[string]float64)
	pickupSet := make(map[int]struct{})
	pickupOrder := make([]int, 0, len(gacha.GachaPickups))
	for _, p := range gacha.GachaPickups {
		if _, exists := pickupSet[p.CardID]; exists {
			continue
		}
		pickupSet[p.CardID] = struct{}{}
		pickupOrder = append(pickupOrder, p.CardID)
	}

	var guaranteedType string
	for _, behavior := range gacha.GachaBehaviors {
		switch strings.ToLower(behavior.GachaBehaviorType) {
		case "over_rarity_4_once":
			guaranteedType = "rarity_4"
		case "over_rarity_3_once":
			if guaranteedType != "rarity_4" {
				guaranteedType = "rarity_3"
			}
		}
	}

	for _, detail := range gacha.GachaDetails {
		card, err := b.source.GetCardByID(detail.CardID)
		if err != nil {
			continue
		}
		cardCache[card.ID] = card
		rarity := strings.ToLower(card.CardRarityType)
		cardRarity[card.ID] = rarity
		rarityCounts[rarity]++
		cardWeight[detail.CardID] += float64(detail.Weight)
		rarityWeights[rarity] += float64(detail.Weight)
	}
	if len(pickupOrder) == 0 {
		for id := range pickupSet {
			pickupOrder = append(pickupOrder, id)
		}
	}

	rarityRateFraction := map[string]float64{}

	var pickupCards []model.GachaCardWeight
	for _, cardID := range pickupOrder {
		card := cardCache[cardID]
		if card == nil {
			var err error
			card, err = b.source.GetCardByID(cardID)
			if err != nil {
				continue
			}
			cardCache[cardID] = card
			cardRarity[cardID] = strings.ToLower(card.CardRarityType)
		}
		thumb := b.buildGachaThumbnail(card)
		pickupCards = append(pickupCards, model.GachaCardWeight{
			ID:               card.ID,
			Rarity:           card.CardRarityType,
			IsPickup:         true,
			ThumbnailRequest: thumb,
		})
	}

	computeCardRate := func(cardID int) float64 {
		rarity := cardRarity[cardID]
		if rarity == "" {
			return 0
		}
		total := rarityWeights[rarity]
		if total <= 0 {
			return 0
		}
		base := rarityRateFraction[rarity]
		if base == 0 {
			return 0
		}
		return (cardWeight[cardID] / total) * base
	}

	weightInfo := model.GachaWeight{
		GuaranteedRates: map[string]float64{},
	}
	for _, rate := range gacha.GachaCardRarityRates {
		rarity := strings.ToLower(rate.CardRarityType)
		if strings.EqualFold(rate.LotteryType, "normal") {
			fraction := rate.Rate / 100.0
			switch rarity {
			case "rarity_1":
				weightInfo.Rarity1Rate = fraction
			case "rarity_2":
				weightInfo.Rarity2Rate = fraction
			case "rarity_3":
				weightInfo.Rarity3Rate = fraction
			case "rarity_4":
				weightInfo.Rarity4Rate = fraction
			case "rarity_birthday":
				weightInfo.RarityBirthdayRate = fraction
			}
			rarityRateFraction[rarity] = fraction
		}
	}

	if guaranteedType != "" {
		guaranteedRates := map[string]float64{
			"rarity_1":        0,
			"rarity_2":        0,
			"rarity_3":        0,
			"rarity_4":        0,
			"rarity_birthday": 0,
		}
		for rarity, fraction := range rarityRateFraction {
			guaranteedRates[rarity] = fraction
		}
		if guaranteedType == "rarity_4" || guaranteedType == "rarity_3" {
			guaranteedRates[guaranteedType] += guaranteedRates["rarity_2"]
			guaranteedRates["rarity_2"] = 0
		}
		if guaranteedType == "rarity_4" {
			guaranteedRates[guaranteedType] += guaranteedRates["rarity_3"]
			guaranteedRates["rarity_3"] = 0
		}
		weightInfo.GuaranteedRates = guaranteedRates
	}

	for idx, card := range pickupCards {
		pickupCards[idx].Rate = computeCardRate(card.ID)
	}

	var bannerPath *string
	if path := b.buildGachaBannerPath(gacha); path != "" {
		bannerPath = &path
	}
	var logoPath *string
	if logo := b.buildGachaLogoPath(gacha); logo != "" {
		logoPath = &logo
	}

	var ceilPath *string
	if gacha.GachaCeilItemID != nil && *gacha.GachaCeilItemID != 0 {
		if path := b.buildCeilItemIconPath(*gacha.GachaCeilItemID); path != "" {
			ceilPath = &path
		}
	}

	info := model.GachaInfo{
		ID:                gacha.ID,
		Name:              gacha.Name,
		GachaType:         gacha.GachaType,
		Summary:           gacha.GachaInformation.Summary,
		Desc:              gacha.GachaInformation.Description,
		StartAt:           gacha.StartAt,
		EndAt:             gacha.EndAt + gachaEndPaddingMillis,
		AssetName:         gacha.AssetBundleName,
		CeilItemImgPath:   ceilPath,
		Behaviors:         b.convertBehaviors(gacha),
		Rarity1Count:      rarityCounts["rarity_1"],
		Rarity2Count:      rarityCounts["rarity_2"],
		Rarity3Count:      rarityCounts["rarity_3"],
		Rarity4Count:      rarityCounts["rarity_4"],
		RarityBirthdayCnt: rarityCounts["rarity_birthday"],
		PickupCount:       len(pickupSet),
	}

	return &model.GachaDetailRequest{
		Gacha:         info,
		WeightInfo:    weightInfo,
		PickupCards:   pickupCards,
		LogoImgPath:   logoPath,
		BannerImgPath: bannerPath,
		BgImgPath:     nil, // 硬编码为 nil 是原定目标
		Region:        region,
	}, nil
}

func (b *GachaBuilder) buildGachaLogoPath(gacha *masterdata.Gacha) string {
	var candidates []string
	if gacha != nil {
		if assetName := strings.TrimSpace(gacha.AssetBundleName); assetName != "" {
			// 优先匹配具体文件夹下的 logo
			candidates = append(candidates, filepath.Join("gacha", assetName, "logo", "logo.png"))

			// 尝试匹配通用 logo 目录下的前缀文件
			candidates = append(candidates, filepath.Join("logo", assetName+".png"))

			if digits := extractNumericToken(assetName); digits != "" {
				// 尝试匹配 id 风格的 logo
				candidates = append(candidates, filepath.Join("logo", fmt.Sprintf("banner_logo%s.png", digits)))
			}
		}

		// 通过 ID 匹配 ab_ 风格的文件夹
		idStr := fmt.Sprintf("%d", gacha.ID)
		candidates = append(candidates, filepath.Join("gacha", "ab_gacha_"+idStr, "logo", "logo.png"))

		if gacha.Seq != 0 {
			candidates = append(candidates, filepath.Join("logo", fmt.Sprintf("banner_logo%d.png", gacha.Seq)))
		}
		candidates = append(candidates, filepath.Join("logo", fmt.Sprintf("banner_logo%d.png", gacha.ID)))
	}

	for _, rel := range candidates {
		if strings.TrimSpace(rel) == "" {
			continue
		}
		if b.assetDir != "" {
			if _, err := os.Stat(filepath.Join(b.assetDir, rel)); err == nil {
				return strings.ReplaceAll(filepath.ToSlash(rel), "\\", "/")
			}
		}
	}
	return ""
}

func (b *GachaBuilder) buildGachaBannerPath(gacha *masterdata.Gacha) string {
	if gacha == nil {
		return ""
	}
	id := gacha.ID
	idStr := strconv.Itoa(id)

	candidates := []string{
		// 优先从 ID 对应的文件夹找
		filepath.Join("home", "banner", "banner_gacha"+idStr, "banner_gacha"+idStr+".png"),

		// 配合 ab_ 前缀文件夹
		filepath.Join("gacha", "ab_gacha_"+idStr, "screen", "texture", "bg_gacha"+idStr+".png"),

		// 配合 AssetBundleName
		filepath.Join("home", "banner", gacha.AssetBundleName, gacha.AssetBundleName+".png"),
		filepath.Join("gacha", gacha.AssetBundleName+".png"),

		// 兜底到 gacha 根目录下的简单命名
		filepath.Join("gacha", "banner_gacha"+idStr+".png"),
	}
	for _, rel := range candidates {
		if strings.TrimSpace(rel) == "" {
			continue
		}
		if b.assetDir != "" {
			if _, err := os.Stat(filepath.Join(b.assetDir, rel)); err == nil {
				return strings.ReplaceAll(filepath.ToSlash(rel), "\\", "/")
			}
		}
	}
	return ""
}

func (b *GachaBuilder) buildGachaThumbnail(card *masterdata.Card) model.CardFullThumbnailRequest {
	return BuildCardThumbnail(b.assets, b.assetDir, card, ThumbnailOptions{AfterTraining: false})
}

func gachaContainsCard(gacha *masterdata.Gacha, cardID int) bool {
	for _, detail := range gacha.GachaDetails {
		if detail.CardID == cardID {
			return true
		}
	}
	for _, pickup := range gacha.GachaPickups {
		if pickup.CardID == cardID {
			return true
		}
	}
	return false
}

func (b *GachaBuilder) convertBehaviors(gacha *masterdata.Gacha) []model.GachaBehavior {
	var behaviors []model.GachaBehavior
	jewelIcon := asset.ResolveAssetPath(b.assets, b.assetDir, "jewel.png")
	for _, bb := range gacha.GachaBehaviors {
		var costType *string
		var costQty *int
		var costIcon *string
		if bb.CostResourceType != "" {
			ct := bb.CostResourceType
			costType = &ct
			lowerType := strings.ToLower(ct)
			if strings.Contains(lowerType, "jewel") && jewelIcon != "" {
				icon := jewelIcon
				costIcon = &icon
			}
		}
		if bb.CostResourceQuantity != 0 {
			q := bb.CostResourceQuantity
			costQty = &q
		}
		behaviors = append(behaviors, model.GachaBehavior{
			Type:         bb.GachaBehaviorType,
			SpinCount:    bb.SpinCount,
			CostType:     costType,
			CostIconPath: costIcon,
			CostQuantity: costQty,
			ExecuteLimit: bb.ExecuteLimit,
			ColorfulPass: strings.EqualFold(bb.GachaSpinnableType, "colorful_pass"),
		})
	}
	return behaviors
}

func (b *GachaBuilder) buildCeilItemIconPath(_ int) string {
	return asset.ResolveAssetPath(b.assets, b.assetDir, "ceil_item.png")
}

var numericTokenPattern = regexp.MustCompile(`(\d+)`)

func extractNumericToken(assetName string) string {
	matches := numericTokenPattern.FindAllString(assetName, -1)
	if len(matches) == 0 {
		return ""
	}
	token := matches[len(matches)-1]
	value, err := strconv.Atoi(token)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%d", value)
}
