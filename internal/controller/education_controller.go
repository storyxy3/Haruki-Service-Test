package controller

import (
	"fmt"

	"Haruki-Service-API/internal/builder"
	"Haruki-Service-API/internal/model"
	"Haruki-Service-API/internal/service"
	"Haruki-Service-API/pkg/asset"
)

// EducationController handles DrawingAPI education endpoints.
type EducationController struct {
	drawing    *service.DrawingService
	masterdata *service.MasterDataService
	userData   *service.UserDataService
	assetDir   string
	assets     *asset.AssetHelper
}

// NewEducationController creates a new controller instance.
func NewEducationController(masterdata *service.MasterDataService, drawing *service.DrawingService, assetHelper *asset.AssetHelper, userData *service.UserDataService) *EducationController {
	assetDir := ""
	if assetHelper != nil {
		assetDir = assetHelper.Primary()
	}
	return &EducationController{
		drawing:    drawing,
		masterdata: masterdata,
		userData:   userData,
		assetDir:   assetDir,
		assets:     assetHelper,
	}
}

func (c *EducationController) ensure() error {
	if c == nil || c.drawing == nil {
		return fmt.Errorf("education controller is not initialized")
	}
	return nil
}

// BuildChallengeLiveRequest assembles challenge-live detail payload from user data.
func (c *EducationController) BuildChallengeLiveRequest(region string) (*model.ChallengeLiveDetailsRequest, error) {
	if err := c.ensure(); err != nil {
		return nil, err
	}
	b := builder.NewEducationBuilder(c.masterdata, c.userData, c.assets, c.assetDir)
	return b.BuildChallengeLiveRequest(region)
}

// RenderChallengeLiveDetailFromUser builds payload from user data then renders.
func (c *EducationController) RenderChallengeLiveDetailFromUser(region string) ([]byte, error) {
	req, err := c.BuildChallengeLiveRequest(region)
	if err != nil {
		return nil, err
	}
	return c.drawing.GenerateEducationChallengeLive(req)
}

// RenderChallengeLiveDetail calls /education/challenge-live.
func (c *EducationController) RenderChallengeLiveDetail(req model.ChallengeLiveDetailsRequest) ([]byte, error) {
	if err := c.ensure(); err != nil {
		return nil, err
	}
	return c.drawing.GenerateEducationChallengeLive(req)
}

// RenderPowerBonusDetail calls /education/power-bonus.
func (c *EducationController) RenderPowerBonusDetail(req model.PowerBonusDetailRequest) ([]byte, error) {
	if err := c.ensure(); err != nil {
		return nil, err
	}
	return c.drawing.GenerateEducationPowerBonus(req)
}

// RenderAreaItemMaterials calls /education/area-item.
func (c *EducationController) RenderAreaItemMaterials(req model.AreaItemUpgradeMaterialsRequest) ([]byte, error) {
	if err := c.ensure(); err != nil {
		return nil, err
	}
	return c.drawing.GenerateEducationAreaItem(req)
}

// RenderBonds calls /education/bonds.
func (c *EducationController) RenderBonds(req model.BondsRequest) ([]byte, error) {
	if err := c.ensure(); err != nil {
		return nil, err
	}
	return c.drawing.GenerateEducationBonds(req)
}

// RenderLeaderCount calls /education/leader-count.
func (c *EducationController) RenderLeaderCount(req model.LeaderCountRequest) ([]byte, error) {
	if err := c.ensure(); err != nil {
		return nil, err
	}
	return c.drawing.GenerateEducationLeaderCount(req)
}
