package controller

import (
	"fmt"

	"Haruki-Service-API/internal/service"
)

// SkController handles sk module build/render requests.
type SkController struct {
	drawing *service.DrawingService
}

func NewSkController(drawing *service.DrawingService) *SkController {
	return &SkController{drawing: drawing}
}

func (c *SkController) ensure() error {
	if c == nil || c.drawing == nil {
		return fmt.Errorf("sk controller is not initialized")
	}
	return nil
}

func (c *SkController) Build(req map[string]interface{}) (map[string]interface{}, error) {
	if err := c.ensure(); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, fmt.Errorf("sk request is empty")
	}
	return req, nil
}

func (c *SkController) RenderLine(req map[string]interface{}) ([]byte, error) {
	payload, err := c.Build(req)
	if err != nil {
		return nil, err
	}
	return c.drawing.GenerateSKLine(payload)
}

func (c *SkController) RenderQuery(req map[string]interface{}) ([]byte, error) {
	payload, err := c.Build(req)
	if err != nil {
		return nil, err
	}
	return c.drawing.GenerateSKQuery(payload)
}

func (c *SkController) RenderCheckRoom(req map[string]interface{}) ([]byte, error) {
	payload, err := c.Build(req)
	if err != nil {
		return nil, err
	}
	return c.drawing.GenerateSKCheckRoom(payload)
}

func (c *SkController) RenderSpeed(req map[string]interface{}) ([]byte, error) {
	payload, err := c.Build(req)
	if err != nil {
		return nil, err
	}
	return c.drawing.GenerateSKSpeed(payload)
}

func (c *SkController) RenderPlayerTrace(req map[string]interface{}) ([]byte, error) {
	payload, err := c.Build(req)
	if err != nil {
		return nil, err
	}
	return c.drawing.GenerateSKPlayerTrace(payload)
}

func (c *SkController) RenderRankTrace(req map[string]interface{}) ([]byte, error) {
	payload, err := c.Build(req)
	if err != nil {
		return nil, err
	}
	return c.drawing.GenerateSKRankTrace(payload)
}

func (c *SkController) RenderWinrate(req map[string]interface{}) ([]byte, error) {
	payload, err := c.Build(req)
	if err != nil {
		return nil, err
	}
	return c.drawing.GenerateSKWinrate(payload)
}

