package controller

import (
	"fmt"
	"strings"

	"Haruki-Service-API/internal/builder"
	"Haruki-Service-API/internal/model"
	"Haruki-Service-API/internal/service"
	"Haruki-Service-API/pkg/asset"
)

// ProfileController handles profile rendering requests.
type ProfileController struct {
	source  service.ProfileDataSource
	sources map[string]service.ProfileDataSource
	Drawing *service.DrawingService
	Asset   *asset.AssetHelper
}

// NewProfileController creates a controller with a default profile source.
func NewProfileController(source service.ProfileDataSource, draw *service.DrawingService, ast *asset.AssetHelper) *ProfileController {
	ctrl := &ProfileController{
		source:  source,
		sources: make(map[string]service.ProfileDataSource),
		Drawing: draw,
		Asset:   ast,
	}
	ctrl.registerSource(source)
	return ctrl
}

func (c *ProfileController) RegisterSource(src service.ProfileDataSource) {
	c.registerSource(src)
}

func (c *ProfileController) registerSource(src service.ProfileDataSource) {
	if src == nil {
		return
	}
	region := strings.ToLower(strings.TrimSpace(src.DefaultRegion()))
	if region == "" {
		return
	}
	c.sources[region] = src
}

func (c *ProfileController) sourceForRegion(region string) service.ProfileDataSource {
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

// RenderProfile renders profile image.
func (c *ProfileController) RenderProfile(userID string, region string, userDataSvc *service.UserDataService) ([]byte, error) {
	req, err := c.BuildProfileRequest(userID, region, userDataSvc)
	if err != nil {
		return nil, err
	}
	return c.Drawing.GenerateProfile(req)
}

// BuildProfileRequest assembles profile payload for drawing API.
func (c *ProfileController) BuildProfileRequest(userID string, region string, userDataSvc *service.UserDataService) (model.ProfileRequest, error) {
	source := c.sourceForRegion(region)
	if source == nil {
		return model.ProfileRequest{}, fmt.Errorf("profile data source not configured")
	}
	if c.Asset == nil {
		return model.ProfileRequest{}, fmt.Errorf("asset helper not configured")
	}
	pb := builder.NewProfileBuilder(source, c.Asset, c.Asset.Primary(), userDataSvc)
	return pb.BuildProfileRequest(region)
}
