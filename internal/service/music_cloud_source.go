package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"

	"Haruki-Service-API/pkg/masterdata"

	sekai "haruki-cloud/database/sekai"
	"haruki-cloud/database/sekai/event"
	"haruki-cloud/database/sekai/eventmusic"
	"haruki-cloud/database/sekai/gamecharacter"
	"haruki-cloud/database/sekai/limitedtimemusic"
	"haruki-cloud/database/sekai/music"
	"haruki-cloud/database/sekai/musicdifficultie"
	"haruki-cloud/database/sekai/musictag"
	"haruki-cloud/database/sekai/musicvocal"
)

// CloudMusicSource implements MusicDataSource backed by Haruki-Cloud.
type CloudMusicSource struct {
	client      *sekai.Client
	region      string
	queryRegion string

	mu            sync.RWMutex
	musicByID     map[int]*masterdata.Music
	musicList     []*masterdata.Music
	characterByID map[int]*masterdata.Character
	localizedByID map[int][]string
}

func NewCloudMusicSource(client *sekai.Client, defaultRegion string) *CloudMusicSource {
	if client == nil {
		return nil
	}
	region := strings.TrimSpace(defaultRegion)
	if region == "" {
		region = "JP"
	}
	return &CloudMusicSource{
		client:        client,
		region:        region,
		queryRegion:   strings.ToLower(region),
		musicByID:     make(map[int]*masterdata.Music),
		characterByID: make(map[int]*masterdata.Character),
		localizedByID: make(map[int][]string),
	}
}

func (c *CloudMusicSource) DefaultRegion() string {
	return c.region
}

func (c *CloudMusicSource) context() context.Context {
	return context.Background()
}

func (c *CloudMusicSource) SearchMusic(query string) (*masterdata.Music, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, fmt.Errorf("music not found: empty query")
	}

	if id, err := strconv.Atoi(query); err == nil {
		if item, err := c.GetMusicByID(id); err == nil && item != nil {
			return item, nil
		}
	}

	musics := c.GetMusics()
	if len(musics) == 0 {
		return nil, fmt.Errorf("music not found: %s", query)
	}

	queryLower := strings.ToLower(query)
	for _, m := range musics {
		if strings.ToLower(m.Title) == queryLower {
			return cloneMusic(m), nil
		}
	}

	var best *masterdata.Music
	for _, m := range musics {
		if strings.Contains(strings.ToLower(m.Title), queryLower) {
			if best == nil || len(m.Title) < len(best.Title) {
				best = m
			}
		}
	}
	if best != nil {
		return cloneMusic(best), nil
	}

	for _, m := range musics {
		titles, err := c.GetMusicLocalizedTitles(m.ID)
		if err != nil || len(titles) == 0 {
			continue
		}
		for _, title := range titles {
			if strings.EqualFold(strings.TrimSpace(title), query) {
				return cloneMusic(m), nil
			}
		}
	}
	for _, m := range musics {
		titles, err := c.GetMusicLocalizedTitles(m.ID)
		if err != nil || len(titles) == 0 {
			continue
		}
		for _, title := range titles {
			if strings.Contains(strings.ToLower(strings.TrimSpace(title)), queryLower) {
				return cloneMusic(m), nil
			}
		}
	}
	return nil, fmt.Errorf("music not found: %s", query)
}

func (c *CloudMusicSource) GetMusicByID(id int) (*masterdata.Music, error) {
	if id <= 0 {
		return nil, fmt.Errorf("invalid music id: %d", id)
	}
	c.mu.RLock()
	if cached, ok := c.musicByID[id]; ok {
		c.mu.RUnlock()
		return cloneMusic(cached), nil
	}
	c.mu.RUnlock()

	ctx := c.context()
	entity, err := c.client.Music.Query().
		Where(
			music.ServerRegionEQ(c.queryRegion),
			music.GameIDEQ(int64(id)),
		).
		Only(ctx)
	if err != nil {
		return nil, err
	}
	model := convertMusicEntity(entity)

	c.mu.Lock()
	c.musicByID[model.ID] = model
	c.mu.Unlock()
	return cloneMusic(model), nil
}

func (c *CloudMusicSource) GetMusics() []*masterdata.Music {
	c.mu.RLock()
	if len(c.musicList) > 0 {
		cached := cloneMusicList(c.musicList)
		c.mu.RUnlock()
		return cached
	}
	c.mu.RUnlock()

	ctx := c.context()
	entities, err := c.client.Music.Query().
		Where(music.ServerRegionEQ(c.queryRegion)).
		Order(music.ByPublishedAt(), music.ByGameID()).
		All(ctx)
	if err != nil {
		return nil
	}

	list := make([]*masterdata.Music, 0, len(entities))
	byID := make(map[int]*masterdata.Music, len(entities))
	for _, entity := range entities {
		model := convertMusicEntity(entity)
		list = append(list, model)
		byID[model.ID] = model
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].PublishedAt == list[j].PublishedAt {
			return list[i].ID < list[j].ID
		}
		return list[i].PublishedAt < list[j].PublishedAt
	})

	c.mu.Lock()
	c.musicList = list
	for id, item := range byID {
		c.musicByID[id] = item
	}
	c.mu.Unlock()
	return cloneMusicList(list)
}

func (c *CloudMusicSource) GetMusicDifficulties(musicID int) ([]*masterdata.MusicDifficulty, error) {
	ctx := c.context()
	items, err := c.client.Musicdifficultie.Query().
		Where(
			musicdifficultie.ServerRegionEQ(c.queryRegion),
			musicdifficultie.MusicIDEQ(int64(musicID)),
		).
		Order(musicdifficultie.ByID()).
		All(ctx)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("no difficulties found for music %d", musicID)
	}
	result := make([]*masterdata.MusicDifficulty, 0, len(items))
	for _, item := range items {
		result = append(result, &masterdata.MusicDifficulty{
			ID:              int(item.GameID),
			MusicID:         int(item.MusicID),
			MusicDifficulty: item.MusicDifficulty,
			PlayLevel:       int(item.PlayLevel),
			TotalNoteCount:  int(item.TotalNoteCount),
		})
	}
	return result, nil
}

func (c *CloudMusicSource) GetMusicLocalizedTitles(musicID int) ([]string, error) {
	if musicID <= 0 {
		return nil, fmt.Errorf("invalid music id: %d", musicID)
	}
	c.mu.RLock()
	if titles, ok := c.localizedByID[musicID]; ok {
		c.mu.RUnlock()
		return append([]string(nil), titles...), nil
	}
	c.mu.RUnlock()

	ctx := c.context()
	items, err := c.client.Music.Query().
		Where(music.GameIDEQ(int64(musicID))).
		All(ctx)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return []string{}, nil
	}

	unique := make(map[string]struct{}, len(items)*2)
	titles := make([]string, 0, len(items)*2)
	appendTitle := func(raw string) {
		title := strings.TrimSpace(raw)
		if title == "" {
			return
		}
		key := strings.ToLower(title)
		if _, exists := unique[key]; exists {
			return
		}
		unique[key] = struct{}{}
		titles = append(titles, title)
	}
	for _, item := range items {
		appendTitle(item.Title)
		appendTitle(item.Pronunciation)
	}

	c.mu.Lock()
	c.localizedByID[musicID] = append([]string(nil), titles...)
	c.mu.Unlock()
	return titles, nil
}

func (c *CloudMusicSource) GetMusicVocals(musicID int) ([]*masterdata.MusicVocal, error) {
	ctx := c.context()
	items, err := c.client.Musicvocal.Query().
		Where(
			musicvocal.ServerRegionEQ(c.queryRegion),
			musicvocal.MusicIDEQ(int64(musicID)),
		).
		Order(musicvocal.BySeq()).
		All(ctx)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("no vocals found for music %d", musicID)
	}
	result := make([]*masterdata.MusicVocal, 0, len(items))
	for _, item := range items {
		result = append(result, &masterdata.MusicVocal{
			ID:              int(item.GameID),
			MusicID:         int(item.MusicID),
			MusicVocalType:  item.MusicVocalType,
			Caption:         item.Caption,
			Characters:      parseMusicVocalCharacters(item.Characters, int(item.GameID), int(item.MusicID)),
			AssetBundleName: item.AssetbundleName,
		})
	}
	return result, nil
}

func (c *CloudMusicSource) GetMusicTags(musicID int) ([]string, error) {
	ctx := c.context()
	items, err := c.client.Musictag.Query().
		Where(
			musictag.ServerRegionEQ(c.queryRegion),
			musictag.MusicIDEQ(int64(musicID)),
		).
		Order(musictag.BySeq()).
		All(ctx)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return []string{}, nil
	}
	result := make([]string, 0, len(items))
	for _, item := range items {
		if strings.TrimSpace(item.MusicTag) == "" {
			continue
		}
		result = append(result, item.MusicTag)
	}
	return result, nil
}

func (c *CloudMusicSource) GetCharacterByID(id int) (*masterdata.Character, error) {
	if id <= 0 {
		return nil, fmt.Errorf("invalid character id: %d", id)
	}
	c.mu.RLock()
	if cached, ok := c.characterByID[id]; ok {
		c.mu.RUnlock()
		copy := *cached
		return &copy, nil
	}
	c.mu.RUnlock()

	ctx := c.context()
	entity, err := c.client.Gamecharacter.Query().
		Where(
			gamecharacter.ServerRegionEQ(c.queryRegion),
			gamecharacter.GameIDEQ(int64(id)),
		).
		Only(ctx)
	if err != nil {
		entity, err = c.client.Gamecharacter.Query().
			Where(
				gamecharacter.ServerRegionEQ(c.queryRegion),
				gamecharacter.IDEQ(id),
			).
			Only(ctx)
		if err != nil {
			return nil, err
		}
	}
	model := &masterdata.Character{
		ID:        int(entity.GameID),
		FirstName: entity.FirstName,
		GivenName: entity.GivenName,
		Unit:      entity.Unit,
	}
	c.mu.Lock()
	c.characterByID[id] = model
	c.mu.Unlock()
	copy := *model
	return &copy, nil
}

func (c *CloudMusicSource) GetPrimaryEventByMusicID(musicID int) (*masterdata.Event, error) {
	ctx := c.context()
	links, err := c.client.Eventmusic.Query().
		Where(
			eventmusic.ServerRegionEQ(c.queryRegion),
			eventmusic.MusicIDEQ(int64(musicID)),
		).
		Order(eventmusic.BySeq()).
		All(ctx)
	if err != nil {
		return nil, err
	}
	if len(links) == 0 {
		return nil, fmt.Errorf("no events found for music %d", musicID)
	}

	eventIDs := make([]int64, 0, len(links))
	seen := make(map[int64]struct{}, len(links))
	for _, link := range links {
		if _, ok := seen[link.EventID]; ok {
			continue
		}
		seen[link.EventID] = struct{}{}
		eventIDs = append(eventIDs, link.EventID)
	}

	items, err := c.client.Event.Query().
		Where(
			event.ServerRegionEQ(c.queryRegion),
			event.GameIDIn(eventIDs...),
		).
		Order(event.ByStartAt()).
		All(ctx)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("no events found for music %d", musicID)
	}
	return cloneEvent(convertEventEntity(items[0])), nil
}

func (c *CloudMusicSource) GetLimitedTimeMusics(musicID int) []*masterdata.LimitedTimeMusic {
	ctx := c.context()
	items, err := c.client.Limitedtimemusic.Query().
		Where(
			limitedtimemusic.ServerRegionEQ(c.queryRegion),
			limitedtimemusic.MusicIDEQ(int64(musicID)),
		).
		Order(limitedtimemusic.ByStartAt()).
		All(ctx)
	if err != nil || len(items) == 0 {
		return nil
	}
	result := make([]*masterdata.LimitedTimeMusic, 0, len(items))
	for _, item := range items {
		result = append(result, &masterdata.LimitedTimeMusic{
			ID:      int(item.GameID),
			MusicID: int(item.MusicID),
			StartAt: item.StartAt,
			EndAt:   item.EndAt,
		})
	}
	return result
}

func convertMusicEntity(entity *sekai.Music) *masterdata.Music {
	return &masterdata.Music{
		ID:                 int(entity.GameID),
		Seq:                int(entity.Seq),
		ReleaseConditionId: int(entity.ReleaseConditionID),
		Categories:         toStringSlice(entity.Categories),
		Title:              entity.Title,
		Pronunciation:      entity.Pronunciation,
		Lyricist:           entity.Lyricist,
		Composer:           entity.Composer,
		Arranger:           entity.Arranger,
		DancerCount:        int(entity.DancerCount),
		SelfDancerCount:    int(entity.SelfDancerPosition),
		AssetBundleName:    entity.AssetbundleName,
		PublishedAt:        entity.PublishedAt,
		DigitizedAt:        entity.ReleasedAt,
	}
}

func parseMusicVocalCharacters(raw []interface{}, vocalID, musicID int) []masterdata.MusicVocalCharacter {
	result := make([]masterdata.MusicVocalCharacter, 0, len(raw))
	for idx, item := range raw {
		entry, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		charType, _ := entry["characterType"].(string)
		charID, ok := interfaceToInt(entry["characterId"])
		if !ok {
			continue
		}
		result = append(result, masterdata.MusicVocalCharacter{
			ID:            idx + 1,
			MusicID:       musicID,
			MusicVocalID:  vocalID,
			CharacterType: charType,
			CharacterID:   charID,
		})
	}
	return result
}

func toStringSlice(values []interface{}) []string {
	result := make([]string, 0, len(values))
	for _, item := range values {
		if s, ok := item.(string); ok && strings.TrimSpace(s) != "" {
			result = append(result, s)
		}
	}
	return result
}

func interfaceToInt(value interface{}) (int, bool) {
	switch v := value.(type) {
	case int:
		return v, true
	case int32:
		return int(v), true
	case int64:
		return int(v), true
	case float64:
		return int(v), true
	case json.Number:
		n, err := v.Int64()
		if err != nil {
			return 0, false
		}
		return int(n), true
	case string:
		n, err := strconv.Atoi(strings.TrimSpace(v))
		if err != nil {
			return 0, false
		}
		return n, true
	default:
		return 0, false
	}
}

func cloneMusic(src *masterdata.Music) *masterdata.Music {
	if src == nil {
		return nil
	}
	copy := *src
	if src.Categories != nil {
		copy.Categories = append([]string(nil), src.Categories...)
	}
	return &copy
}

func cloneMusicList(items []*masterdata.Music) []*masterdata.Music {
	result := make([]*masterdata.Music, 0, len(items))
	for _, item := range items {
		result = append(result, cloneMusic(item))
	}
	return result
}
