package controller

import (
	"fmt"

	"Haruki-Service-API/internal/model"
	"Haruki-Service-API/internal/service"
)

// MiscController handles misc module render requests.
type MiscController struct {
	drawing *service.DrawingService
}

func NewMiscController(drawing *service.DrawingService) *MiscController {
	return &MiscController{drawing: drawing}
}

func (c *MiscController) ensure() error {
	if c == nil || c.drawing == nil {
		return fmt.Errorf("misc controller is not initialized")
	}
	return nil
}

func (c *MiscController) RenderCharaBirthday(req model.CharaBirthdayRequest) ([]byte, error) {
	if err := c.ensure(); err != nil {
		return nil, err
	}
	return c.drawing.GenerateCharaBirthday(req)
}

func (c *MiscController) BuildCharaBirthdayRequest(req model.CharaBirthdayRequest) (model.CharaBirthdayRequest, error) {
	if err := c.ensure(); err != nil {
		return model.CharaBirthdayRequest{}, err
	}
	if req.CID <= 0 || req.Month <= 0 || req.Day <= 0 {
		return model.CharaBirthdayRequest{}, fmt.Errorf("invalid birthday request")
	}
	if len(req.Cards) == 0 {
		return model.CharaBirthdayRequest{}, fmt.Errorf("birthday cards are required")
	}
	return req, nil
}
