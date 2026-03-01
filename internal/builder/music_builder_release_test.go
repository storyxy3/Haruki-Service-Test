package builder

import (
	"testing"

	"Haruki-Service-API/pkg/masterdata"
)

type stubMusicSource struct{}

func (s *stubMusicSource) DefaultRegion() string { return "jp" }
func (s *stubMusicSource) SearchMusic(query string) (*masterdata.Music, error) {
	return nil, nil
}
func (s *stubMusicSource) GetMusicByID(id int) (*masterdata.Music, error) { return nil, nil }
func (s *stubMusicSource) GetMusics() []*masterdata.Music                 { return nil }
func (s *stubMusicSource) GetMusicLocalizedTitles(musicID int) ([]string, error) {
	return nil, nil
}
func (s *stubMusicSource) GetMusicDifficulties(musicID int) ([]*masterdata.MusicDifficulty, error) {
	return nil, nil
}
func (s *stubMusicSource) GetMusicVocals(musicID int) ([]*masterdata.MusicVocal, error) {
	return nil, nil
}
func (s *stubMusicSource) GetMusicTags(musicID int) ([]string, error) { return nil, nil }
func (s *stubMusicSource) GetCharacterByID(id int) (*masterdata.Character, error) {
	return nil, nil
}
func (s *stubMusicSource) GetPrimaryEventByMusicID(musicID int) (*masterdata.Event, error) {
	return nil, nil
}
func (s *stubMusicSource) GetLimitedTimeMusics(musicID int) []*masterdata.LimitedTimeMusic {
	return nil
}

func TestBuildMusicDetailRequest_ReleaseAtInt64(t *testing.T) {
	b := NewMusicBuilder(&stubMusicSource{}, nil, "", nil)
	music := &masterdata.Music{
		ID:          1,
		Title:       "test",
		PublishedAt: int64(1735689600000), // 2025-01-01 UTC in ms; exceeds int32
	}
	req, err := b.BuildMusicDetailRequest(music, "jp")
	if err != nil {
		t.Fatalf("BuildMusicDetailRequest failed: %v", err)
	}
	if req.MusicInfo.ReleaseAt != music.PublishedAt {
		t.Fatalf("release_at mismatch: got %d want %d", req.MusicInfo.ReleaseAt, music.PublishedAt)
	}
}
