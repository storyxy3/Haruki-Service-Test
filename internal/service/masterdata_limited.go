package service

import "Haruki-Service-API/pkg/masterdata"

// loadLimitedTimeMusics 加载限时乐曲数据
func (s *MasterDataService) loadLimitedTimeMusics() error {
	return s.loadJSON("limitedTimeMusics.json", &s.limitedTimeMusics)
}

// GetLimitedTimeMusics 按乐曲ID获取限时信息
func (s *MasterDataService) GetLimitedTimeMusics(musicID int) []*masterdata.LimitedTimeMusic {
	if s == nil {
		return nil
	}
	return s.limitedTimesByMusicID[musicID]
}
