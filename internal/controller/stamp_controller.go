package controller

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"Haruki-Service-API/internal/model"
	"Haruki-Service-API/internal/service"
	"Haruki-Service-API/pkg/asset"
)

// StampController handles stamp module render requests.
type StampController struct {
	source   service.StampDataSource
	sources  map[string]service.StampDataSource
	drawing  *service.DrawingService
	asset    *asset.AssetHelper
	assetDir string
}

func NewStampController(source service.StampDataSource, drawing *service.DrawingService, ast *asset.AssetHelper) *StampController {
	assetDir := ""
	if ast != nil {
		assetDir = ast.Primary()
	}
	ctrl := &StampController{
		source:   source,
		sources:  make(map[string]service.StampDataSource),
		drawing:  drawing,
		asset:    ast,
		assetDir: assetDir,
	}
	ctrl.registerSource(source)
	return ctrl
}

func (c *StampController) RegisterSource(src service.StampDataSource) {
	c.registerSource(src)
}

func (c *StampController) registerSource(src service.StampDataSource) {
	if src == nil {
		return
	}
	region := strings.ToLower(strings.TrimSpace(src.DefaultRegion()))
	if region == "" {
		return
	}
	c.sources[region] = src
}

func (c *StampController) sourceForRegion(region string) service.StampDataSource {
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

func (c *StampController) ensure() error {
	if c == nil || c.drawing == nil {
		return fmt.Errorf("stamp controller is not initialized")
	}
	return nil
}

func (c *StampController) RenderStampList(req model.StampListRequest) ([]byte, error) {
	if err := c.ensure(); err != nil {
		return nil, err
	}
	return c.drawing.GenerateStampList(req)
}

func (c *StampController) BuildStampListRequest(query model.StampListQuery) (model.StampListRequest, error) {
	src := c.sourceForRegion(query.Region)
	if src == nil {
		return model.StampListRequest{}, fmt.Errorf("stamp data source not configured")
	}
	stamps, err := src.GetStamps()
	if err != nil {
		return model.StampListRequest{}, err
	}
	if len(stamps) == 0 {
		return model.StampListRequest{}, fmt.Errorf("no stamp data available")
	}

	filter := make(map[int]struct{}, len(query.IDs))
	for _, id := range query.IDs {
		if id > 0 {
			filter[id] = struct{}{}
		}
	}

	items := make([]model.StampData, 0, len(stamps))
	for _, s := range stamps {
		if len(filter) > 0 {
			if _, ok := filter[s.ID]; !ok {
				continue
			}
		}
		candidates := []string{
			filepath.ToSlash(filepath.Join("stamp", s.AssetbundleName, s.AssetbundleName+".png")),
			filepath.ToSlash(filepath.Join("stamp", s.AssetbundleName+"_rip", s.AssetbundleName+".png")),
		}
		imagePath := candidates[0]
		if c.asset != nil && c.assetDir != "" {
			if resolvedAbs := c.asset.FirstExisting(candidates...); resolvedAbs != "" {
				resolved := filepath.ToSlash(resolvedAbs)
				imagePath = relativeAssetPath(c.assetDir, resolved)
			} else {
				// Cloud master data may contain stale stamp ids that have no local asset; skip them.
				continue
			}
		}
		items = append(items, model.StampData{
			ID:        s.ID,
			ImagePath: imagePath,
			TextColor: [4]int{200, 0, 0, 255},
		})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	if query.Limit > 0 && len(items) > query.Limit {
		items = items[:query.Limit]
	}
	if len(items) == 0 {
		return model.StampListRequest{}, fmt.Errorf("no stamps matched the query")
	}

	prompt := strings.TrimSpace(query.PromptMessage)
	if prompt == "" {
		prompt = "表情列表"
	}

	return model.StampListRequest{
		PromptMessage: &prompt,
		Stamps:        items,
	}, nil
}

func relativeAssetPath(base, target string) string {
	base = filepath.ToSlash(filepath.Clean(base))
	target = filepath.ToSlash(filepath.Clean(target))
	base = strings.TrimSuffix(base, "/")
	if base == "" || base == "." {
		return target
	}
	prefix := base + "/"
	if strings.HasPrefix(target, prefix) {
		return strings.TrimPrefix(target, prefix)
	}
	return target
}
