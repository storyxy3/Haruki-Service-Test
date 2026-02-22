package controller

import (
	"fmt"
	"strings"

	"Haruki-Service-API/internal/builder"
	"Haruki-Service-API/internal/model"
	"Haruki-Service-API/internal/service"
	"Haruki-Service-API/pkg/asset"
)

// EventController 负责活动绘制
type EventController struct {
	masterdata *service.MasterDataService
	drawing    *service.DrawingService
	drawingURL string
	assetDir   string
	assets     *asset.AssetHelper
}

// NewEventController 构造
func NewEventController(masterdata *service.MasterDataService, drawing *service.DrawingService, drawingURL string, assetHelper *asset.AssetHelper) *EventController {
	assetDir := ""
	if assetHelper != nil {
		assetDir = assetHelper.Primary()
	}
	return &EventController{
		masterdata: masterdata,
		drawing:    drawing,
		drawingURL: drawingURL,
		assetDir:   assetDir,
		assets:     assetHelper,
	}
}

// BuildEventDetail 构造活动详情请求
func (c *EventController) BuildEventDetail(query model.EventDetailQuery) (*model.DrawingRequest, error) {
	req, err := c.BuildEventDetailRequest(query)
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
func (c *EventController) RenderEventDetail(query model.EventDetailQuery) ([]byte, error) {
	req, err := c.BuildEventDetailRequest(query)
	if err != nil {
		return nil, err
	}
	return c.drawing.GenerateEventDetail(req)
}

// BuildEventDetailRequest 生成请求体
func (c *EventController) BuildEventDetailRequest(query model.EventDetailQuery) (*model.EventDetailRequest, error) {
	b := builder.NewEventBuilder(c.masterdata, c.assets, c.assetDir)
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
	b := builder.NewEventBuilder(c.masterdata, c.assets, c.assetDir)
	return b.BuildEventListRequest(query)
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
