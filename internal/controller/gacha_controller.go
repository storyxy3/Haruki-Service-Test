package controller

import (
	"fmt"
	"strings"

	"Haruki-Service-API/internal/builder"
	"Haruki-Service-API/internal/model"
	"Haruki-Service-API/internal/service"
	"Haruki-Service-API/pkg/asset"
)

// GachaController 负责卡池相关的构建与渲染
type GachaController struct {
	source        service.GachaDataSource
	sources       map[string]service.GachaDataSource
	defaultRegion string
	drawing       *service.DrawingService
	drawingURL    string
	assetDir      string
	assets        *asset.AssetHelper
}

// NewGachaController 创建 GachaController
func NewGachaController(
	source service.GachaDataSource,
	drawing *service.DrawingService,
	drawingURL string,
	assetHelper *asset.AssetHelper,
) *GachaController {
	assetDir := ""
	if assetHelper != nil {
		assetDir = assetHelper.Primary()
	}
	ctrl := &GachaController{
		source:     source,
		sources:    make(map[string]service.GachaDataSource),
		drawing:    drawing,
		drawingURL: drawingURL,
		assetDir:   assetDir,
		assets:     assetHelper,
	}
	ctrl.defaultRegion = ctrl.normalizeRegion("")
	ctrl.registerSource(source)
	return ctrl
}

func (c *GachaController) RegisterSource(src service.GachaDataSource) {
	c.registerSource(src)
}

func (c *GachaController) registerSource(src service.GachaDataSource) {
	if src == nil {
		return
	}
	region := c.normalizeRegion(src.DefaultRegion())
	if region == "" {
		region = c.defaultRegion
	}
	if region == "" {
		region = c.normalizeRegion("jp")
	}
	if _, exists := c.sources[region]; !exists {
		c.sources[region] = src
	}
	if c.defaultRegion == "" {
		c.defaultRegion = region
	}
	if c.source == nil {
		c.source = src
	}
}

func (c *GachaController) normalizeRegion(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func (c *GachaController) resolveRegion(requested string) string {
	normalized := c.normalizeRegion(requested)
	if normalized != "" {
		return normalized
	}
	if c.defaultRegion != "" {
		return c.defaultRegion
	}
	if c.source != nil {
		return c.normalizeRegion(c.source.DefaultRegion())
	}
	return "jp"
}

func (c *GachaController) sourceForRegion(region string) service.GachaDataSource {
	normalized := c.resolveRegion(region)
	if src, ok := c.sources[normalized]; ok && src != nil {
		return src
	}
	if c.source != nil {
		return c.source
	}
	for _, src := range c.sources {
		if src != nil {
			return src
		}
	}
	return nil
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
	effectiveRegion := c.resolveRegion(query.Region)
	source := c.sourceForRegion(effectiveRegion)
	if source == nil {
		return nil, fmt.Errorf("no gacha data source for region %s", effectiveRegion)
	}
	query.Region = effectiveRegion
	b := builder.NewGachaBuilder(source, c.assets, c.assetDir)
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
	effectiveRegion := c.resolveRegion(query.Region)
	source := c.sourceForRegion(effectiveRegion)
	if source == nil {
		return nil, fmt.Errorf("no gacha data source for region %s", effectiveRegion)
	}
	query.Region = effectiveRegion
	b := builder.NewGachaBuilder(source, c.assets, c.assetDir)
	return b.BuildGachaDetailRequest(query)
}
