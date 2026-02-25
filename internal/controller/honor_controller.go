package controller

import (
	"Haruki-Service-API/internal/builder"
	"Haruki-Service-API/internal/model"
	"Haruki-Service-API/internal/service"
	"Haruki-Service-API/pkg/asset"
	"fmt"
)

// HonorController 称号控制器
type HonorController struct {
	MasterData   *service.MasterDataService
	Drawing      *service.DrawingService
	Asset        *asset.AssetHelper
	HonorBuilder *builder.HonorBuilder
}

// NewHonorController 创建称号控制器
func NewHonorController(md *service.MasterDataService, draw *service.DrawingService, ast *asset.AssetHelper) *HonorController {
	return &HonorController{
		MasterData:   md,
		Drawing:      draw,
		Asset:        ast,
		HonorBuilder: builder.NewHonorBuilder(md, ast, ast.Primary()),
	}
}

// BuildHonorRequest 组装基础的称号请求用于绘图
func (c *HonorController) BuildHonorRequest(query model.HonorQuery) (model.HonorRequest, error) {
	return c.HonorBuilder.BuildHonorRequest(query)
}

func (c *HonorController) RenderHonorImage(req model.HonorRequest) ([]byte, error) {
	// 直接将 req 转发给 DrawingService
	data, err := c.Drawing.GenerateHonor(req)
	if err != nil {
		return nil, fmt.Errorf("failed to generate honor image: %w", err)
	}
	return data, nil
}
