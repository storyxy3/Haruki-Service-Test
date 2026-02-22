package controller

import (
	"fmt"
	"Haruki-Service-API/internal/builder"
	"Haruki-Service-API/internal/model"
	"Haruki-Service-API/internal/service"
	"Haruki-Service-API/pkg/asset"
)

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
// TODO: Implement CardBox logic properly
func (c *CardController) RenderCardBox(queries []model.CardQuery) ([]byte, error) {
	return nil, fmt.Errorf("not implemented")
}
