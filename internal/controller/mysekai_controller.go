package controller

import (
	"fmt"

	"Haruki-Service-API/internal/service"
)

// MysekaiController handles mysekai module build/render requests.
type MysekaiController struct {
	drawing *service.DrawingService
}

func NewMysekaiController(drawing *service.DrawingService) *MysekaiController {
	return &MysekaiController{drawing: drawing}
}

func (c *MysekaiController) ensure() error {
	if c == nil || c.drawing == nil {
		return fmt.Errorf("mysekai controller is not initialized")
	}
	return nil
}

func (c *MysekaiController) Build(req interface{}) (interface{}, error) {
	if err := c.ensure(); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, fmt.Errorf("mysekai request is empty")
	}
	return req, nil
}

func (c *MysekaiController) RenderResource(req interface{}) ([]byte, error) {
	payload, err := c.Build(req)
	if err != nil {
		return nil, err
	}
	return c.drawing.GenerateMysekaiResource(payload)
}

func (c *MysekaiController) RenderFixtureList(req interface{}) ([]byte, error) {
	payload, err := c.Build(req)
	if err != nil {
		return nil, err
	}
	return c.drawing.GenerateMysekaiFixtureList(payload)
}

func (c *MysekaiController) RenderFixtureDetail(req interface{}) ([]byte, error) {
	payload, err := c.Build(req)
	if err != nil {
		return nil, err
	}
	return c.drawing.GenerateMysekaiFixtureDetail(payload)
}

func (c *MysekaiController) RenderDoorUpgrade(req interface{}) ([]byte, error) {
	payload, err := c.Build(req)
	if err != nil {
		return nil, err
	}
	return c.drawing.GenerateMysekaiDoorUpgrade(payload)
}

func (c *MysekaiController) RenderMusicRecord(req interface{}) ([]byte, error) {
	payload, err := c.Build(req)
	if err != nil {
		return nil, err
	}
	return c.drawing.GenerateMysekaiMusicRecord(payload)
}

func (c *MysekaiController) RenderTalkList(req interface{}) ([]byte, error) {
	payload, err := c.Build(req)
	if err != nil {
		return nil, err
	}
	return c.drawing.GenerateMysekaiTalkList(payload)
}
