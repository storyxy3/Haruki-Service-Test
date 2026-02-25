package controller

import (
	"errors"
	"fmt"
	"strconv"

	"Haruki-Service-API/internal/builder"
	"Haruki-Service-API/internal/model"
	"Haruki-Service-API/internal/service"
	"Haruki-Service-API/pkg/asset"
)

var ErrDrawingServiceUnavailable = errors.New("drawing service is not configured")

// CardController 卡牌控制器
type CardController struct {
	masterdata    *service.MasterDataService
	drawing       *service.DrawingService
	searchService *service.CardSearchService
	drawingURL    string // DrawingAPI 的基础 URL
	assetDir      string // 资源文件根目录 (e.g. D:\pjskdata\data)
	assets        *asset.AssetHelper
	userData      *service.UserDataService
}

func (c *CardController) requireDrawingService() error {
	if c.drawing == nil {
		return ErrDrawingServiceUnavailable
	}
	return nil
}

// NewCardController 创建卡牌控制器
func NewCardController(
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
	return &CardController{
		masterdata:    masterdata,
		drawing:       drawing,
		searchService: searchService,
		drawingURL:    drawingURL,
		assetDir:      assetDir,
		assets:        assetHelper,
		userData:      userData,
	}
}

// BuildCardDetailRequest 构建卡牌详情请求（模式 1：只返回请求数据）
// 返回 DrawingRequest，包含 URL 和请求体
func (c *CardController) BuildCardDetailRequest(query model.CardQuery) (*model.DrawingRequest, error) {
	// 1. 查询 MasterData (通过 SearchService)
	card, err := c.searchService.Search(query.Query)
	if err != nil {
		return nil, fmt.Errorf("failed to search card: %w", err)
	}

	// 2. 构建 DrawingAPI 请求
	region := query.Region
	if region == "" {
		region = c.masterdata.GetRegion()
	}
	// 提取 builder 分离重逻辑
	b := builder.NewCardBuilder(c.masterdata, c.assets, c.assetDir, c.searchService, c.userData)
	req, err := b.BuildCardDetailRequestBody(card, region)
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %w", err)
	}

	// 3. 返回 DrawingRequest（包含 URL 和 Body）
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
	b := builder.NewCardBuilder(c.masterdata, c.assets, c.assetDir, c.searchService, c.userData)
	listReq, err := b.BuildCardListRequest(cardIDs, region)
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

// RenderCardList 渲染卡牌列表（模式 2）
func (c *CardController) RenderCardList(queries []model.CardQuery) ([]byte, error) {
	if len(queries) == 0 {
		return nil, fmt.Errorf("no queries provided")
	}

	// 1. Search Logic
	// Currently we only support one query string for the list search.
	query := queries[0].Query
	cards, err := c.searchService.SearchList(query)
	if err != nil {
		return nil, fmt.Errorf("failed to search card list: %w", err)
	}

	// 2. Extract IDs
	var cardIDs []int
	for _, card := range cards {
		cardIDs = append(cardIDs, card.ID)
	}

	// 3. Delegate to RenderCardListFromIDs
	region := queries[0].Region
	if region == "" {
		region = c.masterdata.GetRegion()
	}
	return c.RenderCardListFromIDs(cardIDs, region)
}

// RenderCardBox 渲染卡牌一览（模式 2）
func (c *CardController) RenderCardBox(queries []model.CardQuery) ([]byte, error) {
	if len(queries) == 0 {
		return nil, fmt.Errorf("no card query provided")
	}

	region := queries[0].Region
	if region == "" {
		region = c.masterdata.GetRegion()
	}

	cards, err := c.searchService.SearchList(queries[0].Query)
	if err != nil {
		return nil, fmt.Errorf("failed to search card box: %w", err)
	}

	var userCards []model.UserCard
	iconPaths := make(map[string]string)
	b := builder.NewCardBuilder(c.masterdata, c.assets, c.assetDir, c.searchService, c.userData)
	// 获取用户卡牌映射以提高查找效率
	userCardMap := make(map[int]service.RawUserCard)
	if c.userData != nil && c.userData.GetRawData() != nil {
		for _, uc := range c.userData.GetRawData().UserCards {
			userCardMap[uc.CardID] = uc
		}
	}

	for _, card := range cards {
		cardBasic := b.BuildCardBasic(card)

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
	pb := builder.NewProfileBuilder(c.masterdata, c.assets, c.assetDir, c.userData)
	userInfo, _ := pb.BuildDetailedProfileCardRequest(region)

	req := model.CardBoxRequest{
		Cards:              userCards,
		Region:             region,
		UserInfo:           userInfo,
		ShowID:             false, // 隐藏按钮 ID
		ShowBox:            false,
		UseAfterTraining:   true,
		CharacterIconPaths: iconPaths,
	}

	if err := c.requireDrawingService(); err != nil {
		return nil, err
	}

	return c.drawing.GenerateCardBox(req)
}
