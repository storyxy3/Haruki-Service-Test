package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"
)

// DrawingService 封装与 DrawingAPI 通信的 HTTP 客户端
type DrawingService struct {
	baseURL        string
	httpClient     *http.Client
	retryCount     int
	assetDirStrips []string
}

// NewDrawingService 创建 DrawingService
func NewDrawingService(baseURL string, timeout time.Duration, retryCount int, assetDirs []string) *DrawingService {
	var strips []string
	for _, dir := range assetDirs {
		clean := filepath.ToSlash(filepath.Clean(dir))
		clean = strings.TrimSuffix(clean, "/")
		if clean == "" || clean == "." {
			continue
		}
		strips = append(strips, clean)
	}
	return &DrawingService{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		retryCount:     retryCount,
		assetDirStrips: strips,
	}
}

// GenerateCardDetail 生成卡片详情
func (s *DrawingService) GenerateCardDetail(req interface{}) ([]byte, error) {
	return s.callAPI("/api/pjsk/card/detail", req)
}

// GenerateCardList 生成卡片列表
func (s *DrawingService) GenerateCardList(req interface{}) ([]byte, error) {
	return s.callAPI("/api/pjsk/card/list", req)
}

// GenerateCardBox 生成卡盒
func (s *DrawingService) GenerateCardBox(req interface{}) ([]byte, error) {
	return s.callAPI("/api/pjsk/card/box", req)
}

// GenerateMusicDetail 生成乐曲详情
func (s *DrawingService) GenerateMusicDetail(req interface{}) ([]byte, error) {
	return s.callAPI("/api/pjsk/music/detail", req)
}

// GenerateMusicBriefList 生成乐曲速览
func (s *DrawingService) GenerateMusicBriefList(req interface{}) ([]byte, error) {
	return s.callAPI("/api/pjsk/music/brief-list", req)
}

// GenerateMusicList 生成乐曲列表
func (s *DrawingService) GenerateMusicList(req interface{}) ([]byte, error) {
	return s.callAPI("/api/pjsk/music/list", req)
}

// GenerateMusicProgress 生成进度统计
func (s *DrawingService) GenerateMusicProgress(req interface{}) ([]byte, error) {
	return s.callAPI("/api/pjsk/music/progress", req)
}

// GenerateMusicRewardsDetail 生成奖励详情
func (s *DrawingService) GenerateMusicRewardsDetail(req interface{}) ([]byte, error) {
	return s.callAPI("/api/pjsk/music/rewards/detail", req)
}

// GenerateMusicRewardsBasic 生成奖励基础表
func (s *DrawingService) GenerateMusicRewardsBasic(req interface{}) ([]byte, error) {
	return s.callAPI("/api/pjsk/music/rewards/basic", req)
}

// GenerateGachaList 生成卡池列表
func (s *DrawingService) GenerateGachaList(req interface{}) ([]byte, error) {
	return s.callAPI("/api/pjsk/gacha/list", req)
}

// GenerateGachaDetail 生成卡池详情
func (s *DrawingService) GenerateGachaDetail(req interface{}) ([]byte, error) {
	return s.callAPI("/api/pjsk/gacha/detail", req)
}

// GenerateMusicChart 生成谱面预览
func (s *DrawingService) GenerateMusicChart(req interface{}) ([]byte, error) {
	return s.callAPI("/api/pjsk/chart", req)
}

// GenerateEventDetail 生成活动详情
func (s *DrawingService) GenerateEventDetail(req interface{}) ([]byte, error) {
	return s.callAPI("/api/pjsk/event/detail", req)
}

// GenerateEventList 生成活动列表
func (s *DrawingService) GenerateEventList(req interface{}) ([]byte, error) {
	return s.callAPI("/api/pjsk/event/list", req)
}

// GenerateEventRecord 生成活动记录
func (s *DrawingService) GenerateEventRecord(req interface{}) ([]byte, error) {
	return s.callAPI("/api/pjsk/event/record", req)
}

// GenerateEducationChallengeLive 绘制 challenge-live 面板
func (s *DrawingService) GenerateEducationChallengeLive(req interface{}) ([]byte, error) {
	return s.callAPI("/api/pjsk/education/challenge-live", req)
}

// GenerateEducationPowerBonus 绘制 power-bonus 面板
func (s *DrawingService) GenerateEducationPowerBonus(req interface{}) ([]byte, error) {
	return s.callAPI("/api/pjsk/education/power-bonus", req)
}

// GenerateEducationAreaItem 绘制 area-item 面板
func (s *DrawingService) GenerateEducationAreaItem(req interface{}) ([]byte, error) {
	return s.callAPI("/api/pjsk/education/area-item", req)
}

// GenerateEducationBonds 绘制 bonds 面板
func (s *DrawingService) GenerateEducationBonds(req interface{}) ([]byte, error) {
	return s.callAPI("/api/pjsk/education/bonds", req)
}

// GenerateEducationLeaderCount 绘制 leader-count 面板
func (s *DrawingService) GenerateEducationLeaderCount(req interface{}) ([]byte, error) {
	return s.callAPI("/api/pjsk/education/leader-count", req)
}

// GenerateHonor 绘制称号
func (s *DrawingService) GenerateHonor(req interface{}) ([]byte, error) {
	return s.callAPI("/api/pjsk/honor/", req)
}

// GenerateProfile 绘制玩家名片
func (s *DrawingService) GenerateProfile(req interface{}) ([]byte, error) {
	return s.callAPI("/api/pjsk/profile/", req)
}

// GenerateStampList draws stamp list panel.
func (s *DrawingService) GenerateStampList(req interface{}) ([]byte, error) {
	return s.callAPI("/api/pjsk/stamp/list", req)
}

// GenerateCharaBirthday draws character birthday panel.
func (s *DrawingService) GenerateCharaBirthday(req interface{}) ([]byte, error) {
	return s.callAPI("/api/pjsk/misc/chara-birthday", req)
}

// GenerateScoreControl draws score-control panel.
func (s *DrawingService) GenerateScoreControl(req interface{}) ([]byte, error) {
	return s.callAPI("/api/pjsk/score/control", req)
}

// GenerateCustomRoomScore draws custom-room score panel.
func (s *DrawingService) GenerateCustomRoomScore(req interface{}) ([]byte, error) {
	return s.callAPI("/api/pjsk/score/custom-room", req)
}

// GenerateMusicMeta draws music-meta panel.
func (s *DrawingService) GenerateMusicMeta(req interface{}) ([]byte, error) {
	return s.callAPI("/api/pjsk/score/music-meta", req)
}

// GenerateMusicBoard draws music-board panel.
func (s *DrawingService) GenerateMusicBoard(req interface{}) ([]byte, error) {
	return s.callAPI("/api/pjsk/score/music-board", req)
}

// GenerateDeckRecommend draws deck recommendation panel.
func (s *DrawingService) GenerateDeckRecommend(req interface{}) ([]byte, error) {
	return s.callAPI("/api/pjsk/deck/recommend", req)
}

// GenerateSKLine draws sk line panel.
func (s *DrawingService) GenerateSKLine(req interface{}) ([]byte, error) {
	return s.callAPI("/api/pjsk/sk/line", req)
}

// GenerateSKQuery draws sk query panel.
func (s *DrawingService) GenerateSKQuery(req interface{}) ([]byte, error) {
	return s.callAPI("/api/pjsk/sk/query", req)
}

// GenerateSKCheckRoom draws sk check-room panel.
func (s *DrawingService) GenerateSKCheckRoom(req interface{}) ([]byte, error) {
	return s.callAPI("/api/pjsk/sk/check-room", req)
}

// GenerateSKSpeed draws sk speed panel.
func (s *DrawingService) GenerateSKSpeed(req interface{}) ([]byte, error) {
	return s.callAPI("/api/pjsk/sk/speed", req)
}

// GenerateSKPlayerTrace draws sk player-trace panel.
func (s *DrawingService) GenerateSKPlayerTrace(req interface{}) ([]byte, error) {
	return s.callAPI("/api/pjsk/sk/player-trace", req)
}

// GenerateSKRankTrace draws sk rank-trace panel.
func (s *DrawingService) GenerateSKRankTrace(req interface{}) ([]byte, error) {
	return s.callAPI("/api/pjsk/sk/rank-trace", req)
}

// GenerateSKWinrate draws sk winrate panel.
func (s *DrawingService) GenerateSKWinrate(req interface{}) ([]byte, error) {
	return s.callAPI("/api/pjsk/sk/winrate", req)
}

// callAPI 统一的 HTTP 调用封装
// GenerateMysekaiResource draws mysekai resource panel.
func (s *DrawingService) GenerateMysekaiResource(req interface{}) ([]byte, error) {
	return s.callAPI("/api/pjsk/mysekai/resource", req)
}

// GenerateMysekaiFixtureList draws mysekai fixture-list panel.
func (s *DrawingService) GenerateMysekaiFixtureList(req interface{}) ([]byte, error) {
	return s.callAPI("/api/pjsk/mysekai/fixture-list", req)
}

// GenerateMysekaiFixtureDetail draws mysekai fixture-detail panel.
func (s *DrawingService) GenerateMysekaiFixtureDetail(req interface{}) ([]byte, error) {
	return s.callAPI("/api/pjsk/mysekai/fixture-detail", req)
}

// GenerateMysekaiDoorUpgrade draws mysekai door-upgrade panel.
func (s *DrawingService) GenerateMysekaiDoorUpgrade(req interface{}) ([]byte, error) {
	return s.callAPI("/api/pjsk/mysekai/door-upgrade", req)
}

// GenerateMysekaiMusicRecord draws mysekai music-record panel.
func (s *DrawingService) GenerateMysekaiMusicRecord(req interface{}) ([]byte, error) {
	return s.callAPI("/api/pjsk/mysekai/music-record", req)
}

// GenerateMysekaiTalkList draws mysekai talk-list panel.
func (s *DrawingService) GenerateMysekaiTalkList(req interface{}) ([]byte, error) {
	return s.callAPI("/api/pjsk/mysekai/talk-list", req)
}

func (s *DrawingService) callAPI(endpoint string, reqBody interface{}) ([]byte, error) {
	url := s.baseURL + endpoint

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	jsonData = s.stripAssetPrefix(jsonData)

	fmt.Printf("[DEBUG] DrawingAPI Request Payload (%s): %s\n", endpoint, string(jsonData))

	var lastErr error
	for i := 0; i <= s.retryCount; i++ {
		if i > 0 {
			time.Sleep(time.Second * time.Duration(i))
		}

		resp, err := s.httpClient.Post(url, "application/json", bytes.NewBuffer(jsonData))
		if err != nil {
			lastErr = err
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			lastErr = fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
			continue
		}

		data, err := io.ReadAll(resp.Body)
		if err != nil {
			lastErr = err
			continue
		}

		return data, nil
	}

	return nil, fmt.Errorf("failed after %d retries: %w", s.retryCount, lastErr)
}

func (s *DrawingService) stripAssetPrefix(payload []byte) []byte {
	if len(s.assetDirStrips) == 0 {
		return payload
	}
	result := payload
	for _, prefix := range s.assetDirStrips {
		if prefix == "" {
			continue
		}
		repl := prefix
		if !strings.HasSuffix(repl, "/") {
			repl += "/"
		}
		result = bytes.ReplaceAll(result, []byte(repl), []byte(""))
	}
	return result
}
