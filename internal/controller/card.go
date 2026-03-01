package controller

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"Haruki-Service-API/internal/builder"
	"Haruki-Service-API/internal/model"
	"Haruki-Service-API/internal/service"
	"Haruki-Service-API/pkg/asset"
	"Haruki-Service-API/pkg/masterdata"
)

var ErrDrawingServiceUnavailable = errors.New("drawing service is not configured")

// CardController 卡牌控制器
type CardController struct {
	cards         service.CardDataSource
	secondary     service.CardDataSource
	events        service.EventDataSource
	masterdata    *service.MasterDataService
	drawing       *service.DrawingService
	searchService *service.CardSearchService
	drawingURL    string // DrawingAPI 的基础 URL
	assetDir      string // 资源文件根目录 (e.g. D:\pjskdata\data)
	assets        *asset.AssetHelper
	userData      *service.UserDataService
	cardSources   map[string]service.CardDataSource
	eventSources  map[string]service.EventDataSource
	cardSearchers map[string]*service.CardSearchService
	defaultRegion string
}

func (c *CardController) requireDrawingService() error {
	if c.drawing == nil {
		return ErrDrawingServiceUnavailable
	}
	return nil
}

func (c *CardController) RegisterEventSource(src service.EventDataSource) {
	c.registerEventSource(src)
}

func (c *CardController) registerCardSource(src service.CardDataSource, searcher *service.CardSearchService) {
	if src == nil {
		return
	}
	if c.cardSources == nil {
		c.cardSources = make(map[string]service.CardDataSource)
	}
	region := c.normalizeRegion(src.DefaultRegion())
	if region == "" {
		region = c.defaultRegion
	}
	if region == "" {
		region = c.normalizeRegion("jp")
	}
	if _, exists := c.cardSources[region]; !exists {
		c.cardSources[region] = src
	}
	if searcher != nil {
		if c.cardSearchers == nil {
			c.cardSearchers = make(map[string]*service.CardSearchService)
		}
		c.cardSearchers[region] = searcher
	}
	if c.defaultRegion == "" {
		c.defaultRegion = region
	}
}

func (c *CardController) registerEventSource(src service.EventDataSource) {
	if src == nil {
		return
	}
	if c.eventSources == nil {
		c.eventSources = make(map[string]service.EventDataSource)
	}
	region := c.normalizeRegion(src.DefaultRegion())
	if region == "" {
		region = c.defaultRegion
	}
	if _, exists := c.eventSources[region]; !exists {
		c.eventSources[region] = src
	}
}

func (c *CardController) normalizeRegion(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func (c *CardController) resolveRegion(requested string) string {
	normalized := c.normalizeRegion(requested)
	if normalized != "" {
		return normalized
	}
	if c.defaultRegion != "" {
		return c.defaultRegion
	}
	if c.cards != nil {
		return c.normalizeRegion(c.cards.DefaultRegion())
	}
	if normalized == "" {
		return "jp"
	}
	return normalized
}

func (c *CardController) cardSourceForRegion(region string) service.CardDataSource {
	normalized := c.resolveRegion(region)
	if src, ok := c.cardSources[normalized]; ok && src != nil {
		return src
	}
	if c.cards != nil {
		return c.cards
	}
	for _, src := range c.cardSources {
		if src != nil {
			return src
		}
	}
	return nil
}

func (c *CardController) eventSourceForRegion(region string) service.EventDataSource {
	normalized := c.resolveRegion(region)
	if src, ok := c.eventSources[normalized]; ok && src != nil {
		return src
	}
	if c.events != nil {
		return c.events
	}
	for _, src := range c.eventSources {
		if src != nil {
			return src
		}
	}
	return nil
}

func (c *CardController) translationSourceForRegion(region string) service.CardDataSource {
	if c.cardSources == nil {
		return nil
	}
	if c.normalizeRegion(region) != "jp" {
		return nil
	}
	if src, ok := c.cardSources["cn"]; ok {
		return src
	}
	return nil
}

func (c *CardController) searcherForRegion(region string) *service.CardSearchService {
	normalized := c.resolveRegion(region)
	if svc, ok := c.cardSearchers[normalized]; ok && svc != nil {
		return svc
	}
	if c.searchService == nil {
		return nil
	}
	repo := c.cardSourceForRegion(normalized)
	if repo == nil {
		return c.searchService
	}
	cloned := c.searchService.CloneWithRepo(repo)
	if c.cardSearchers == nil {
		c.cardSearchers = make(map[string]*service.CardSearchService)
	}
	c.cardSearchers[normalized] = cloned
	return cloned
}

// NewCardController 创建卡牌控制器
func NewCardController(
	cards service.CardDataSource,
	secondary service.CardDataSource,
	events service.EventDataSource,
	masterdata *service.MasterDataService,
	drawing *service.DrawingService,
	searchService *service.CardSearchService,
	drawingURL string,
	assetHelper *asset.AssetHelper,
	userData *service.UserDataService,
) *CardController {
	assetDir := ""
	if assetHelper != nil {
		assetDir = assetHelper.Primary()
	}
	ctrl := &CardController{
		cards:         cards,
		secondary:     secondary,
		events:        events,
		masterdata:    masterdata,
		drawing:       drawing,
		searchService: searchService,
		drawingURL:    drawingURL,
		assetDir:      assetDir,
		assets:        assetHelper,
		userData:      userData,
		cardSources:   make(map[string]service.CardDataSource),
		eventSources:  make(map[string]service.EventDataSource),
		cardSearchers: make(map[string]*service.CardSearchService),
	}
	ctrl.defaultRegion = ctrl.normalizeRegion("")
	ctrl.registerCardSource(cards, searchService)
	if secondary != nil {
		var secondarySearcher *service.CardSearchService
		if searchService != nil {
			secondarySearcher = searchService.CloneWithRepo(secondary)
		}
		ctrl.registerCardSource(secondary, secondarySearcher)
	}
	ctrl.registerEventSource(events)
	return ctrl
}

// BuildCardDetailRequest 构建卡牌详情请求（模式 1：只返回请求数据）
// 返回 DrawingRequest，包含 URL 和请求体
func (c *CardController) BuildCardDetailRequest(query model.CardQuery) (*model.DrawingRequest, error) {
	region := c.resolveRegion(query.Region)
	searcher := c.searcherForRegion(region)
	if searcher == nil {
		return nil, fmt.Errorf("no card search service for region %s", region)
	}
	card, err := searcher.Search(query.Query)
	if err != nil {
		return nil, fmt.Errorf("failed to search card: %w", err)
	}

	primary := c.cardSourceForRegion(region)
	if primary == nil {
		return nil, fmt.Errorf("no card data source for region %s", region)
	}
	events := c.eventSourceForRegion(region)
	translation := c.translationSourceForRegion(region)
	b := builder.NewCardBuilder(primary, translation, events, c.masterdata, c.assets, c.assetDir, searcher, c.userData)
	req, err := b.BuildCardDetailRequestBody(card, region)
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %w", err)
	}

	return &model.DrawingRequest{
		URL:    c.drawingURL + "/api/pjsk/card/detail",
		Method: "POST",
		Body:   req,
	}, nil
}

// RenderCardDetail 渲染卡牌详情（模式 2：直接返回图片）
// 返回图片的二进制数据
func (c *CardController) RenderCardDetail(query model.CardQuery) ([]byte, error) {
	// 1. 构建请求
	drawingReq, err := c.BuildCardDetailRequest(query)
	if err != nil {
		return nil, err
	}

	if err := c.requireDrawingService(); err != nil {
		return nil, err
	}

	// 2. 调用 DrawingAPI
	imageData, err := c.drawing.GenerateCardDetail(drawingReq.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to generate image: %w", err)
	}

	return imageData, nil
}

// RenderCardListFromIDs 根据 ID 列表渲染卡牌列表
func (c *CardController) RenderCardListFromIDs(cardIDs []int, region string) ([]byte, error) {
	effectiveRegion := c.resolveRegion(region)
	primary := c.cardSourceForRegion(effectiveRegion)
	if primary == nil {
		return nil, fmt.Errorf("failed to build list request: no card source for region %s", effectiveRegion)
	}
	translation := c.translationSourceForRegion(effectiveRegion)
	events := c.eventSourceForRegion(effectiveRegion)
	searcher := c.searcherForRegion(effectiveRegion)
	b := builder.NewCardBuilder(primary, translation, events, c.masterdata, c.assets, c.assetDir, searcher, c.userData)
	listReq, err := b.BuildCardListRequest(cardIDs, effectiveRegion)
	if err != nil {
		return nil, fmt.Errorf("failed to build list request: %w", err)
	}

	if err := c.requireDrawingService(); err != nil {
		return nil, err
	}

	imageData, err := c.drawing.GenerateCardList(listReq)
	if err != nil {
		return nil, fmt.Errorf("failed to generate list image: %w", err)
	}

	return imageData, nil
}

// BuildCardListRequestFromIDs builds card list payload from card IDs.
func (c *CardController) BuildCardListRequestFromIDs(cardIDs []int, region string) (*model.CardListRequest, error) {
	effectiveRegion := c.resolveRegion(region)
	primary := c.cardSourceForRegion(effectiveRegion)
	if primary == nil {
		return nil, fmt.Errorf("failed to build list request: no card source for region %s", effectiveRegion)
	}
	translation := c.translationSourceForRegion(effectiveRegion)
	events := c.eventSourceForRegion(effectiveRegion)
	searcher := c.searcherForRegion(effectiveRegion)
	b := builder.NewCardBuilder(primary, translation, events, c.masterdata, c.assets, c.assetDir, searcher, c.userData)
	listReq, err := b.BuildCardListRequest(cardIDs, effectiveRegion)
	if err != nil {
		return nil, fmt.Errorf("failed to build list request: %w", err)
	}

	return listReq, nil
}

// BuildCardListRequest builds card list payload from command queries.
func (c *CardController) BuildCardListRequest(queries []model.CardQuery) (*model.CardListRequest, error) {
	if len(queries) == 0 {
		return nil, fmt.Errorf("no queries provided")
	}

	queryText := queries[0].Query
	region := c.resolveRegion(queries[0].Region)
	searcher := c.searcherForRegion(region)
	if searcher == nil {
		return nil, fmt.Errorf("no card search service for region %s", region)
	}
	cards, err := searcher.SearchList(queryText)
	if err != nil {
		return nil, fmt.Errorf("failed to search card list: %w", err)
	}

	cardIDs := make([]int, 0, len(cards))
	for _, card := range cards {
		cardIDs = append(cardIDs, card.ID)
	}

	return c.BuildCardListRequestFromIDs(cardIDs, region)
}

// RenderCardList 渲染卡牌列表（模式 2）
func (c *CardController) RenderCardList(queries []model.CardQuery) ([]byte, error) {
	if len(queries) == 0 {
		return nil, fmt.Errorf("no queries provided")
	}

	queryText := queries[0].Query
	region := c.resolveRegion(queries[0].Region)
	searcher := c.searcherForRegion(region)
	if searcher == nil {
		return nil, fmt.Errorf("no card search service for region %s", region)
	}
	cards, err := searcher.SearchList(queryText)
	if err != nil {
		return nil, fmt.Errorf("failed to search card list: %w", err)
	}

	var cardIDs []int
	for _, card := range cards {
		cardIDs = append(cardIDs, card.ID)
	}

	return c.RenderCardListFromIDs(cardIDs, region)
}

// RenderCardBox 渲染卡牌一览（模式 2）
func (c *CardController) BuildCardBoxRequest(queries []model.CardQuery) (model.CardBoxRequest, error) {
	if len(queries) == 0 {
		return model.CardBoxRequest{}, fmt.Errorf("no card query provided")
	}

	region := c.resolveRegion(queries[0].Region)

	searcher := c.searcherForRegion(region)
	if searcher == nil {
		return model.CardBoxRequest{}, fmt.Errorf("no card search service for region %s", region)
	}

	primary := c.cardSourceForRegion(region)
	if primary == nil {
		return model.CardBoxRequest{}, fmt.Errorf("no card data source for region %s", region)
	}

	queryText := strings.TrimSpace(queries[0].Query)
	var cards []*masterdata.Card
	var err error
	if queryText == "" {
		// Lunabot parity: empty box query means "all released cards" (exclude leaks).
		cards, err = primary.FilterCards(&service.CardQueryInfo{})
		if err != nil {
			return model.CardBoxRequest{}, fmt.Errorf("failed to load cards for card box: %w", err)
		}
		now := time.Now().UnixMilli()
		filtered := cards[:0]
		for _, card := range cards {
			if card == nil {
				continue
			}
			if card.ReleaseAt > now {
				continue
			}
			filtered = append(filtered, card)
		}
		cards = filtered
		if len(cards) == 0 {
			return model.CardBoxRequest{}, fmt.Errorf("no released cards found for region %s", region)
		}
	} else {
		cards, err = searcher.SearchList(queryText)
		if err != nil {
			return model.CardBoxRequest{}, fmt.Errorf("failed to search card box: %w", err)
		}
	}

	var userCards []model.UserCard
	iconPaths := make(map[string]string)
	translation := c.translationSourceForRegion(region)
	events := c.eventSourceForRegion(region)
	b := builder.NewCardBuilder(primary, translation, events, c.masterdata, c.assets, c.assetDir, searcher, c.userData)
	// 获取用户卡牌映射以提高查找效率
	userCardMap := make(map[int]service.RawUserCard)
	if c.userData != nil && c.userData.GetRawData() != nil {
		for _, uc := range c.userData.GetRawData().UserCards {
			userCardMap[uc.CardID] = uc
		}
	}

	for _, card := range cards {
		cardBasic := b.BuildCardBasic(card, region)

		// 默认状态
		hasCard := false

		// 如果有用户数据，则进行匹配
		if uc, exists := userCardMap[card.ID]; exists {
			hasCard = true

			// 根据用户数据决定是否展示特训后卡面
			if uc.SpecialTrainingStatus == "done" || uc.DefaultImage == "special_training" {
				cardBasic.IsAfterTraining = true
			} else {
				cardBasic.IsAfterTraining = false
			}

			// 设置实际等级和特训等级
			for i := range cardBasic.ThumbnailInfo {
				cardBasic.ThumbnailInfo[i].IsPcard = true

				level := uc.Level
				cardBasic.ThumbnailInfo[i].Level = &level

				rank := uc.MasterRank
				cardBasic.ThumbnailInfo[i].TrainRank = &rank

				// 设置特训等级图片路径
				rankPath := asset.ResolveAssetPath(c.assets, c.assetDir, fmt.Sprintf("card/train_rank_%d.png", rank))
				cardBasic.ThumbnailInfo[i].TrainRankImgPath = &rankPath
			}
		} else {
			// 如果没有拥有此卡，且没有全局用户上下文，则不显示 pcard (即不显示等级条)
			for i := range cardBasic.ThumbnailInfo {
				cardBasic.ThumbnailInfo[i].IsPcard = false
			}

			// 查全图时，3/4星默认显示特训后
			if len(cardBasic.ThumbnailInfo) > 1 {
				cardBasic.IsAfterTraining = true
			}
		}

		userCards = append(userCards, model.UserCard{
			Card:    cardBasic,
			HasCard: hasCard,
		})

		// Setup Icon Paths Map for the Drawing API Backend (CardBoxRequest)
		charStr := strconv.Itoa(card.CharacterID)
		if _, exists := iconPaths[charStr]; !exists && card.CharacterID > 0 {
			iconPaths[charStr] = b.BuildCharacterIconPath(card.CharacterID, cardBasic.Unit)
		}
	}

	// 获取用户信息用于页眉
	pb := builder.NewProfileBuilder(service.NewMasterDataProfileSource(c.masterdata), c.assets, c.assetDir, c.userData)
	userInfo, _ := pb.BuildDetailedProfileCardRequest(region)

	return model.CardBoxRequest{
		Cards:              userCards,
		Region:             region,
		UserInfo:           userInfo,
		ShowID:             false, // 隐藏按钮 ID
		ShowBox:            false,
		UseAfterTraining:   true,
		CharacterIconPaths: iconPaths,
	}, nil
}

func (c *CardController) RenderCardBox(queries []model.CardQuery) ([]byte, error) {
	req, err := c.BuildCardBoxRequest(queries)
	if err != nil {
		return nil, err
	}

	if err := c.requireDrawingService(); err != nil {
		return nil, err
	}

	return c.drawing.GenerateCardBox(req)
}
