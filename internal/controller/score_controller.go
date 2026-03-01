package controller

import (
	"fmt"
	"path/filepath"
	"strings"

	"Haruki-Service-API/internal/model"
	"Haruki-Service-API/internal/service"
)

// ScoreController handles score module endpoints.
type ScoreController struct {
	drawing *service.DrawingService
}

func NewScoreController(drawing *service.DrawingService) *ScoreController {
	return &ScoreController{drawing: drawing}
}

func (c *ScoreController) ensure() error {
	if c == nil || c.drawing == nil {
		return fmt.Errorf("score controller is not initialized")
	}
	return nil
}

func (c *ScoreController) BuildScoreControlRequest(req model.ScoreControlRequest) (model.ScoreControlRequest, error) {
	if err := c.ensure(); err != nil {
		return model.ScoreControlRequest{}, err
	}
	if req.MusicID <= 0 || req.TargetPoint <= 0 {
		return model.ScoreControlRequest{}, fmt.Errorf("invalid score control request")
	}
	req.MusicCoverPath = normalizeScoreCoverPath(req.MusicCoverPath)
	return req, nil
}

func (c *ScoreController) RenderScoreControl(req model.ScoreControlRequest) ([]byte, error) {
	payload, err := c.BuildScoreControlRequest(req)
	if err != nil {
		return nil, err
	}
	return c.drawing.GenerateScoreControl(payload)
}

func (c *ScoreController) BuildCustomRoomScoreRequest(req model.CustomRoomScoreRequest) (model.CustomRoomScoreRequest, error) {
	if err := c.ensure(); err != nil {
		return model.CustomRoomScoreRequest{}, err
	}
	if req.TargetPoint <= 0 || len(req.CandidatePairs) == 0 {
		return model.CustomRoomScoreRequest{}, fmt.Errorf("invalid custom-room score request")
	}
	for key, list := range req.MusicListMap {
		for i := range list {
			if raw, ok := list[i]["music_cover"].(string); ok {
				list[i]["music_cover"] = normalizeScoreCoverPath(raw)
			}
		}
		req.MusicListMap[key] = list
	}
	return req, nil
}

func (c *ScoreController) RenderCustomRoomScore(req model.CustomRoomScoreRequest) ([]byte, error) {
	payload, err := c.BuildCustomRoomScoreRequest(req)
	if err != nil {
		return nil, err
	}
	return c.drawing.GenerateCustomRoomScore(payload)
}

func (c *ScoreController) BuildMusicMetaRequest(req []model.MusicMetaRequest) ([]model.MusicMetaRequest, error) {
	if err := c.ensure(); err != nil {
		return nil, err
	}
	if len(req) == 0 {
		return nil, fmt.Errorf("music meta request is empty")
	}
	for i := range req {
		req[i].MusicCoverPath = normalizeScoreCoverPath(req[i].MusicCoverPath)
	}
	return req, nil
}

func (c *ScoreController) RenderMusicMeta(req []model.MusicMetaRequest) ([]byte, error) {
	payload, err := c.BuildMusicMetaRequest(req)
	if err != nil {
		return nil, err
	}
	return c.drawing.GenerateMusicMeta(payload)
}

func (c *ScoreController) BuildMusicBoardRequest(req model.MusicBoardRequest) (model.MusicBoardRequest, error) {
	if err := c.ensure(); err != nil {
		return model.MusicBoardRequest{}, err
	}
	if len(req.Items) == 0 {
		return model.MusicBoardRequest{}, fmt.Errorf("music board request has no items")
	}
	for i := range req.Items {
		req.Items[i].MusicCoverPath = normalizeScoreCoverPath(req.Items[i].MusicCoverPath)
	}
	return req, nil
}

func (c *ScoreController) RenderMusicBoard(req model.MusicBoardRequest) ([]byte, error) {
	payload, err := c.BuildMusicBoardRequest(req)
	if err != nil {
		return nil, err
	}
	return c.drawing.GenerateMusicBoard(payload)
}

// normalizeScoreCoverPath keeps backward compatibility for old score payload paths.
// Example:
// - jacket/jacket_s_001_rip/jacket_s_001.png -> music/jacket/jacket_s_001/jacket_s_001.png
func normalizeScoreCoverPath(raw string) string {
	path := filepath.ToSlash(strings.TrimSpace(raw))
	if path == "" {
		return path
	}
	const legacyPrefix = "jacket/"
	const modernPrefix = "music/jacket/"

	if strings.HasPrefix(path, legacyPrefix) {
		rest := strings.TrimPrefix(path, legacyPrefix)
		parts := strings.Split(rest, "/")
		if len(parts) >= 2 {
			dir := strings.TrimSuffix(parts[0], "_rip")
			file := parts[1]
			if dir != "" && file != "" {
				return modernPrefix + dir + "/" + file
			}
		}
		return modernPrefix + rest
	}
	if strings.HasPrefix(path, modernPrefix) {
		return path
	}
	return path
}
