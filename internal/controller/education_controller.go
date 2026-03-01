package controller

import (
	"fmt"
	"strings"

	"Haruki-Service-API/internal/builder"
	"Haruki-Service-API/internal/model"
	"Haruki-Service-API/internal/service"
	"Haruki-Service-API/pkg/asset"
)

// EducationController handles DrawingAPI education endpoints.
type EducationController struct {
	drawing  *service.DrawingService
	source   service.EducationDataSource
	sources  map[string]service.EducationDataSource
	userData *service.UserDataService
	assetDir string
	assets   *asset.AssetHelper
}

// NewEducationController creates a new controller instance.
func NewEducationController(source service.EducationDataSource, drawing *service.DrawingService, assetHelper *asset.AssetHelper, userData *service.UserDataService) *EducationController {
	assetDir := ""
	if assetHelper != nil {
		assetDir = assetHelper.Primary()
	}
	ctrl := &EducationController{
		drawing:  drawing,
		source:   source,
		sources:  make(map[string]service.EducationDataSource),
		userData: userData,
		assetDir: assetDir,
		assets:   assetHelper,
	}
	ctrl.registerSource(source)
	return ctrl
}

func (c *EducationController) RegisterSource(src service.EducationDataSource) {
	c.registerSource(src)
}

func (c *EducationController) registerSource(src service.EducationDataSource) {
	if src == nil {
		return
	}
	region := strings.ToLower(strings.TrimSpace(src.DefaultRegion()))
	if region == "" {
		return
	}
	c.sources[region] = src
}

func (c *EducationController) sourceForRegion(region string) service.EducationDataSource {
	normalized := strings.ToLower(strings.TrimSpace(region))
	if normalized == "" {
		if c.source != nil {
			return c.source
		}
		for _, src := range c.sources {
			return src
		}
		return nil
	}
	if src, ok := c.sources[normalized]; ok {
		return src
	}
	if c.source != nil {
		return c.source
	}
	return nil
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
	source := c.sourceForRegion(region)
	if source == nil {
		return nil, fmt.Errorf("education data source not configured")
	}
	b := builder.NewEducationBuilder(source, c.userData, c.assets, c.assetDir)
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

// BuildPowerBonusRequest validates and returns power-bonus payload.
func (c *EducationController) BuildPowerBonusRequest(req model.PowerBonusDetailRequest) (model.PowerBonusDetailRequest, error) {
	if err := c.ensure(); err != nil {
		return model.PowerBonusDetailRequest{}, err
	}
	return req, nil
}

// RenderAreaItemMaterials calls /education/area-item.
func (c *EducationController) RenderAreaItemMaterials(req model.AreaItemUpgradeMaterialsRequest) ([]byte, error) {
	if err := c.ensure(); err != nil {
		return nil, err
	}
	return c.drawing.GenerateEducationAreaItem(req)
}

// BuildAreaItemMaterialsRequest validates and returns area-item payload.
func (c *EducationController) BuildAreaItemMaterialsRequest(req model.AreaItemUpgradeMaterialsRequest) (model.AreaItemUpgradeMaterialsRequest, error) {
	if err := c.ensure(); err != nil {
		return model.AreaItemUpgradeMaterialsRequest{}, err
	}
	return req, nil
}

// RenderBonds calls /education/bonds.
func (c *EducationController) RenderBonds(req model.BondsRequest) ([]byte, error) {
	if err := c.ensure(); err != nil {
		return nil, err
	}
	return c.drawing.GenerateEducationBonds(req)
}

// BuildBondsRequest validates and returns bonds payload.
func (c *EducationController) BuildBondsRequest(req model.BondsRequest) (model.BondsRequest, error) {
	if err := c.ensure(); err != nil {
		return model.BondsRequest{}, err
	}
	return req, nil
}

// RenderLeaderCount calls /education/leader-count.
func (c *EducationController) RenderLeaderCount(req model.LeaderCountRequest) ([]byte, error) {
	if err := c.ensure(); err != nil {
		return nil, err
	}
	return c.drawing.GenerateEducationLeaderCount(req)
}

// BuildLeaderCountRequest validates and returns leader-count payload.
func (c *EducationController) BuildLeaderCountRequest(req model.LeaderCountRequest) (model.LeaderCountRequest, error) {
	if err := c.ensure(); err != nil {
		return model.LeaderCountRequest{}, err
	}
	return req, nil
}
