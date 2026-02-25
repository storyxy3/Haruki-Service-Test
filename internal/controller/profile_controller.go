package controller

import (
	"Haruki-Service-API/internal/builder"
	"Haruki-Service-API/internal/model"
	"Haruki-Service-API/internal/service"
	"Haruki-Service-API/pkg/asset"
)

// ProfileController 玩家名片控制器
type ProfileController struct {
	MasterData *service.MasterDataService
	Drawing    *service.DrawingService
	Asset      *asset.AssetHelper
}

// NewProfileController 创建玩家名片控制器
func NewProfileController(md *service.MasterDataService, draw *service.DrawingService, ast *asset.AssetHelper) *ProfileController {
	return &ProfileController{
		MasterData: md,
		Drawing:    draw,
		Asset:      ast,
	}
}

// RenderProfile 绘制玩家名片图片
func (c *ProfileController) RenderProfile(userID string, region string, userDataSvc *service.UserDataService) ([]byte, error) {
	req, err := c.BuildProfileRequest(userID, region, userDataSvc)
	if err != nil {
		return nil, err
	}
	return c.Drawing.GenerateProfile(req)
}

// BuildProfileRequest 组装 DrawingAPI 所需的名片请求
func (c *ProfileController) BuildProfileRequest(userID string, region string, userDataSvc *service.UserDataService) (model.ProfileRequest, error) {
	pb := builder.NewProfileBuilder(c.MasterData, c.Asset, c.Asset.Primary(), userDataSvc)
	return pb.BuildProfileRequest(region)
}
