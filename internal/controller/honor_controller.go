package controller

import (
	"fmt"
	"strings"

	"Haruki-Service-API/internal/builder"
	"Haruki-Service-API/internal/model"
	"Haruki-Service-API/internal/service"
	"Haruki-Service-API/pkg/asset"
)

// HonorController handles honor rendering requests.
type HonorController struct {
	source  service.HonorDataSource
	sources map[string]service.HonorDataSource
	Drawing *service.DrawingService
	Asset   *asset.AssetHelper
}

// NewHonorController creates a controller with default source.
func NewHonorController(source service.HonorDataSource, draw *service.DrawingService, ast *asset.AssetHelper) *HonorController {
	ctrl := &HonorController{
		source:  source,
		sources: make(map[string]service.HonorDataSource),
		Drawing: draw,
		Asset:   ast,
	}
	ctrl.registerSource(source)
	return ctrl
}

func (c *HonorController) RegisterSource(src service.HonorDataSource) {
	c.registerSource(src)
}

func (c *HonorController) registerSource(src service.HonorDataSource) {
	if src == nil {
		return
	}
	region := strings.ToLower(strings.TrimSpace(src.DefaultRegion()))
	if region == "" {
		return
	}
	c.sources[region] = src
}

func (c *HonorController) sourceForRegion(region string) service.HonorDataSource {
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

// BuildHonorRequest builds drawing payload from honor query.
func (c *HonorController) BuildHonorRequest(query model.HonorQuery) (model.HonorRequest, error) {
	source := c.sourceForRegion(query.Region)
	if source == nil {
		return model.HonorRequest{}, fmt.Errorf("honor data source not configured")
	}
	assetDir := ""
	if c.Asset != nil {
		assetDir = c.Asset.Primary()
	}
	return builder.NewHonorBuilder(source, c.Asset, assetDir).BuildHonorRequest(query)
}

func (c *HonorController) RenderHonorImage(req model.HonorRequest) ([]byte, error) {
	data, err := c.Drawing.GenerateHonor(req)
	if err != nil {
		return nil, fmt.Errorf("failed to generate honor image: %w", err)
	}
	return data, nil
}
