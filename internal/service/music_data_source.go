package service

import "Haruki-Service-API/pkg/masterdata"
import "strings"

// MusicDataSource 抽象音乐模块读取能力，支持 MasterData 与 Cloud 切换。
type MusicDataSource interface {
	DefaultRegion() string
	SearchMusic(query string) (*masterdata.Music, error)
	GetMusicByID(id int) (*masterdata.Music, error)
	GetMusics() []*masterdata.Music
	GetMusicLocalizedTitles(musicID int) ([]string, error)
	GetMusicDifficulties(musicID int) ([]*masterdata.MusicDifficulty, error)
	GetMusicVocals(musicID int) ([]*masterdata.MusicVocal, error)
	GetMusicTags(musicID int) ([]string, error)
	GetCharacterByID(id int) (*masterdata.Character, error)
	GetPrimaryEventByMusicID(musicID int) (*masterdata.Event, error)
	GetLimitedTimeMusics(musicID int) []*masterdata.LimitedTimeMusic
}

// MasterDataMusicSource 适配 MasterDataService 到 MusicDataSource。
type MasterDataMusicSource struct {
	svc *MasterDataService
}

func NewMasterDataMusicSource(svc *MasterDataService) *MasterDataMusicSource {
	return &MasterDataMusicSource{svc: svc}
}

func (m *MasterDataMusicSource) DefaultRegion() string {
	return m.svc.GetRegion()
}

func (m *MasterDataMusicSource) SearchMusic(query string) (*masterdata.Music, error) {
	return m.svc.SearchMusic(query)
}

func (m *MasterDataMusicSource) GetMusicByID(id int) (*masterdata.Music, error) {
	return m.svc.GetMusicByID(id)
}

func (m *MasterDataMusicSource) GetMusics() []*masterdata.Music {
	return m.svc.GetMusics()
}

func (m *MasterDataMusicSource) GetMusicLocalizedTitles(musicID int) ([]string, error) {
	music, err := m.svc.GetMusicByID(musicID)
	if err != nil || music == nil {
		return nil, err
	}
	titles := make([]string, 0, 2)
	if strings.TrimSpace(music.Title) != "" {
		titles = append(titles, music.Title)
	}
	if strings.TrimSpace(music.Pronunciation) != "" {
		titles = append(titles, music.Pronunciation)
	}
	return titles, nil
}

func (m *MasterDataMusicSource) GetMusicDifficulties(musicID int) ([]*masterdata.MusicDifficulty, error) {
	return m.svc.GetMusicDifficulties(musicID)
}

func (m *MasterDataMusicSource) GetMusicVocals(musicID int) ([]*masterdata.MusicVocal, error) {
	return m.svc.GetMusicVocals(musicID)
}

func (m *MasterDataMusicSource) GetMusicTags(musicID int) ([]string, error) {
	return m.svc.GetMusicTags(musicID)
}

func (m *MasterDataMusicSource) GetCharacterByID(id int) (*masterdata.Character, error) {
	return m.svc.GetCharacterByID(id)
}

func (m *MasterDataMusicSource) GetPrimaryEventByMusicID(musicID int) (*masterdata.Event, error) {
	return m.svc.GetPrimaryEventByMusicID(musicID)
}

func (m *MasterDataMusicSource) GetLimitedTimeMusics(musicID int) []*masterdata.LimitedTimeMusic {
	return m.svc.GetLimitedTimeMusics(musicID)
}
