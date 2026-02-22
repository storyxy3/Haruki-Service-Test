package controller

import (
	"Haruki-Service-API/internal/builder"
	"Haruki-Service-API/internal/model"
	"Haruki-Service-API/internal/service"
	"Haruki-Service-API/pkg/asset"
)

// GachaController 负责卡池相关的构建与渲染
type GachaController struct {
	masterdata *service.MasterDataService
	drawing    *service.DrawingService
	drawingURL string
	assetDir   string
	assets     *asset.AssetHelper
}

// NewGachaController 创建 GachaController
func NewGachaController(
	masterdata *service.MasterDataService,
	drawing *service.DrawingService,
	drawingURL string,
	assetHelper *asset.AssetHelper,
) *GachaController {
	assetDir := ""
	if assetHelper != nil {
		assetDir = assetHelper.Primary()
	}
	return &GachaController{
		masterdata: masterdata,
		drawing:    drawing,
		drawingURL: drawingURL,
		assetDir:   assetDir,
		assets:     assetHelper,
	}
}

// RenderGachaList 渲染卡池列表
func (c *GachaController) RenderGachaList(query model.GachaListQuery) ([]byte, error) {
	req, err := c.BuildGachaListRequest(query)
	if err != nil {
		return nil, err
	}
	return c.drawing.GenerateGachaList(req)
}

// BuildGachaList 构建卡池列表 DrawingRequest
func (c *GachaController) BuildGachaList(query model.GachaListQuery) (*model.DrawingRequest, error) {
	req, err := c.BuildGachaListRequest(query)
	if err != nil {
		return nil, err
	}
	return &model.DrawingRequest{
		URL:    c.drawingURL + "/api/pjsk/gacha/list",
		Method: "POST",
		Body:   req,
	}, nil
}

// BuildGachaListRequest 构建卡池列表请求体
func (c *GachaController) BuildGachaListRequest(query model.GachaListQuery) (*model.GachaListRequest, error) {
	b := builder.NewGachaBuilder(c.masterdata, c.assets, c.assetDir)
	return b.BuildGachaListRequest(query)
}

// RenderGachaDetail 渲染卡池详情
func (c *GachaController) RenderGachaDetail(query model.GachaDetailQuery) ([]byte, error) {
	req, err := c.BuildGachaDetailRequest(query)
	if err != nil {
		return nil, err
	}
	return c.drawing.GenerateGachaDetail(req)
}

// BuildGachaDetail 构建卡池详情 DrawingRequest
func (c *GachaController) BuildGachaDetail(query model.GachaDetailQuery) (*model.DrawingRequest, error) {
	req, err := c.BuildGachaDetailRequest(query)
	if err != nil {
		return nil, err
	}
	return &model.DrawingRequest{
		URL:    c.drawingURL + "/api/pjsk/gacha/detail",
		Method: "POST",
		Body:   req,
	}, nil
}

// BuildGachaDetailRequest 构建卡池详情请求体
func (c *GachaController) BuildGachaDetailRequest(query model.GachaDetailQuery) (*model.GachaDetailRequest, error) {
	b := builder.NewGachaBuilder(c.masterdata, c.assets, c.assetDir)
	return b.BuildGachaDetailRequest(query)
}
