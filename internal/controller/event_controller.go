package controller

import (
	"context"
	"fmt"
	"strings"
	"time"

	"Haruki-Service-API/internal/builder"
	"Haruki-Service-API/internal/model"
	"Haruki-Service-API/internal/service"
	"Haruki-Service-API/pkg/asset"
)

// EventController 负责活动绘制
type EventController struct {
	events         service.EventDataSource
	sources        map[string]service.EventDataSource
	defaultRegion  string
	drawing        *service.DrawingService
	drawingURL     string
	assetDir       string
	assets         *asset.AssetHelper
	cloud          *service.CloudService
}

// NewEventController 构造
func NewEventController(events service.EventDataSource, drawing *service.DrawingService, drawingURL string, assetHelper *asset.AssetHelper, cloud *service.CloudService) *EventController {
	assetDir := ""
	if assetHelper != nil {
		assetDir = assetHelper.Primary()
	}
	ctrl := &EventController{
		events:    events,
		sources:   make(map[string]service.EventDataSource),
		drawing:   drawing,
		drawingURL: drawingURL,
		assetDir:  assetDir,
		assets:    assetHelper,
		cloud:     cloud,
	}
	ctrl.defaultRegion = ctrl.normalizeRegion("jp")
	ctrl.registerSource(events)
	return ctrl
}

func (c *EventController) RegisterSource(src service.EventDataSource) {
	c.registerSource(src)
}

func (c *EventController) registerSource(src service.EventDataSource) {
	if src == nil {
		return
	}
	region := c.normalizeRegion(src.DefaultRegion())
	if region == "" {
		region = c.defaultRegion
	}
	if region == "" {
		region = "jp"
	}
	if _, exists := c.sources[region]; !exists {
		c.sources[region] = src
	}
	if c.defaultRegion == "" {
		c.defaultRegion = region
	}
	if c.events == nil {
		c.events = src
	}
}

func (c *EventController) normalizeRegion(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func (c *EventController) resolveRegion(requested string) string {
	normalized := c.normalizeRegion(requested)
	if normalized != "" {
		return normalized
	}
	if c.defaultRegion != "" {
		return c.defaultRegion
	}
	if c.events != nil {
		return c.normalizeRegion(c.events.DefaultRegion())
	}
	return "jp"
}

func (c *EventController) sourceForRegion(region string) service.EventDataSource {
	normalized := c.resolveRegion(region)
	if src, ok := c.sources[normalized]; ok && src != nil {
		return src
	}
	if c.events != nil {
		return c.events
	}
	for _, src := range c.sources {
		if src != nil {
			return src
		}
	}
	return nil
}

// BuildEventDetail 构造活动详情请求
func (c *EventController) BuildEventDetail(ctx context.Context, query model.EventDetailQuery) (*model.DrawingRequest, error) {
	req, err := c.BuildEventDetailRequest(ctx, query)
	if err != nil {
		return nil, err
	}
	return &model.DrawingRequest{
		URL:    c.drawingURL + "/api/pjsk/event/detail",
		Method: "POST",
		Body:   req,
	}, nil
}

// RenderEventDetail 渲染活动详情
func (c *EventController) RenderEventDetail(ctx context.Context, query model.EventDetailQuery) ([]byte, error) {
	req, err := c.BuildEventDetailRequest(ctx, query)
	if err != nil {
		return nil, err
	}
	return c.drawing.GenerateEventDetail(req)
}

// BuildEventDetailRequest 生成请求体
func (c *EventController) BuildEventDetailRequest(ctx context.Context, query model.EventDetailQuery) (*model.EventDetailRequest, error) {
	query, err := c.resolveEventDetailQuery(ctx, query)
	if err != nil {
		return nil, err
	}
	source := c.sourceForRegion(query.Region)
	if source == nil {
		return nil, fmt.Errorf("no event data source for region %s", query.Region)
	}
	b := builder.NewEventBuilder(source, c.assets, c.assetDir)
	return b.BuildEventDetailRequest(query)
}

// BuildEventList 构建活动列表请求
func (c *EventController) BuildEventList(query model.EventListQuery) (*model.DrawingRequest, error) {
	req, err := c.BuildEventListRequest(query)
	if err != nil {
		return nil, err
	}
	return &model.DrawingRequest{
		URL:    c.drawingURL + "/api/pjsk/event/list",
		Method: "POST",
		Body:   req,
	}, nil
}

// RenderEventList 渲染活动列表
func (c *EventController) RenderEventList(query model.EventListQuery) ([]byte, error) {
	req, err := c.BuildEventListRequest(query)
	if err != nil {
		return nil, err
	}
	return c.drawing.GenerateEventList(req)
}

// BuildEventRecord 构建活动记录请求
func (c *EventController) BuildEventRecord(req model.EventRecordRequest) (*model.DrawingRequest, error) {
	if err := c.validateEventRecordRequest(req); err != nil {
		return nil, err
	}
	return &model.DrawingRequest{
		URL:    c.drawingURL + "/api/pjsk/event/record",
		Method: "POST",
		Body:   req,
	}, nil
}

// RenderEventRecord 渲染活动记录
func (c *EventController) RenderEventRecord(req model.EventRecordRequest) ([]byte, error) {
	if err := c.validateEventRecordRequest(req); err != nil {
		return nil, err
	}
	return c.drawing.GenerateEventRecord(req)
}

// BuildEventListRequest 构建活动列表请求体
func (c *EventController) BuildEventListRequest(query model.EventListQuery) (*model.EventListRequest, error) {
	query.Region = c.resolveRegion(query.Region)
	source := c.sourceForRegion(query.Region)
	if source == nil {
		return nil, fmt.Errorf("no event data source for region %s", query.Region)
	}
	b := builder.NewEventBuilder(source, c.assets, c.assetDir)
	return b.BuildEventListRequest(query)
}

func (c *EventController) resolveEventDetailQuery(ctx context.Context, query model.EventDetailQuery) (model.EventDetailQuery, error) {
	query.Region = c.resolveRegion(query.Region)
	if !query.UseCloudDB || query.EventID != 0 {
		return query, nil
	}
	if c.cloud == nil {
		return query, fmt.Errorf("cloud database client is not configured")
	}
	eventID, err := c.cloud.GetCurrentEventID(ctx, query.Region, time.Now())
	if err != nil {
		return query, err
	}
	query.EventID = eventID
	return query, nil
}

func (c *EventController) validateEventRecordRequest(req model.EventRecordRequest) error {
	if len(req.EventInfo) == 0 && len(req.WLEventInfo) == 0 {
		return fmt.Errorf("event record requires at least one history entry")
	}
	if strings.TrimSpace(req.UserInfo.Region) == "" {
		return fmt.Errorf("user_info.region is required")
	}
	if strings.TrimSpace(req.UserInfo.Nickname) == "" {
		return fmt.Errorf("user_info.nickname is required")
	}
	if strings.TrimSpace(req.UserInfo.LeaderImagePath) == "" {
		return fmt.Errorf("user_info.leader_image_path is required")
	}
	return nil
}
