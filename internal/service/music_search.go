package service

import (
	"fmt"
	"Haruki-Service-API/pkg/masterdata"
	"strings"
)

// MusicSearchService 音乐搜索服务
type MusicSearchService struct {
	repo   *MasterDataService
	parser *MusicParser
}

// NewMusicSearchService 创建音乐搜索服务
func NewMusicSearchService(repo *MasterDataService, parser *MusicParser) *MusicSearchService {
	return &MusicSearchService{
		repo:   repo,
		parser: parser,
	}
}

// Search 音乐搜索
func (s *MusicSearchService) Search(query string) (*masterdata.Music, error) {
	// 1. Parse
	info, err := s.parser.Parse(query)
	if err != nil {
		return nil, err
	}

	// 2. Dispatch
	switch info.Type {
	case QueryTypeMusicID:
		return s.repo.GetMusicByID(info.Value)

	case QueryTypeMusicSeq:
		// Index based on Sorted Music List (by PublishedAt)
		musics := s.repo.GetMusics()
		count := len(musics)

		idx := info.Value
		if idx < 0 {
			idx = count + idx
		} else {
			// Positive index: 1-based?
			idx = idx - 1
		}

		if idx < 0 || idx >= count {
			return nil, fmt.Errorf("music index out of range: %d", info.Value)
		}
		return musics[idx], nil

	case QueryTypeMusicEvent:
		return s.repo.GetMusicByEventID(info.Value)

	case QueryTypeMusicBan:
		// Ban Event Music Search
		// Logic: Find the "Ban Event" (mnr1 -> 1st Minori Event), then get that event's music.
		// GetBanEvents returns list of events for that char.
		events := s.repo.GetBanEvents(info.BanCharID)
		if len(events) == 0 {
			return nil, fmt.Errorf("no ban events found for charID %d", info.BanCharID)
		}

		// Info.BanSeq is 1-based index into these events.
		// Events are sorted by time.
		if info.BanSeq < 1 || info.BanSeq > len(events) {
			return nil, fmt.Errorf("ban event index out of range: %d (total: %d)", info.BanSeq, len(events))
		}

		event := events[info.BanSeq-1]
		return s.repo.GetMusicByEventID(event.ID)

	case QueryTypeMusicTitle:
		// Exact Match First (ignoring case)
		musics := s.repo.GetMusics()
		lowerKey := strings.ToLower(info.Keyword)

		// 1. Title Exact Match
		for _, m := range musics {
			if strings.ToLower(m.Title) == lowerKey {
				return m, nil
			}
		}

		// 2. Title Contains Match (Simple)
		for _, m := range musics {
			if strings.Contains(strings.ToLower(m.Title), lowerKey) {
				return m, nil
			}
		}

		// 3. Pronunciation? (Not available in struct yet, assuming basic title search for now)

		return nil, fmt.Errorf("music not found by title: %s", info.Keyword)
	}

	return nil, fmt.Errorf("unknown query type: %d", info.Type)
}
