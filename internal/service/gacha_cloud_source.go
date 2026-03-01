package service

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"

	"Haruki-Service-API/pkg/masterdata"

	sekai "haruki-cloud/database/sekai"
	gachaent "haruki-cloud/database/sekai/gacha"
)

// CloudGachaSource implements GachaDataSource using Haruki-Cloud DB.
type CloudGachaSource struct {
	client      *sekai.Client
	region      string
	queryRegion string

	mu        sync.RWMutex
	gachaByID map[int]*masterdata.Gacha
	gachas    []*masterdata.Gacha

	cardSource *CloudCardSource
}

// NewCloudGachaSource creates a cloud-backed gacha data source.
func NewCloudGachaSource(client *sekai.Client, defaultRegion string) *CloudGachaSource {
	if client == nil {
		return nil
	}
	region := strings.TrimSpace(defaultRegion)
	if region == "" {
		region = "JP"
	}
	return &CloudGachaSource{
		client:      client,
		region:      region,
		queryRegion: strings.ToLower(region),
		gachaByID:   make(map[int]*masterdata.Gacha),
		cardSource:  NewCloudCardSource(client, region),
	}
}

func (c *CloudGachaSource) DefaultRegion() string {
	return c.region
}

func (c *CloudGachaSource) context() context.Context {
	return context.Background()
}

func (c *CloudGachaSource) GetCardByID(id int) (*masterdata.Card, error) {
	if c.cardSource == nil {
		return nil, fmt.Errorf("cloud card source is unavailable")
	}
	return c.cardSource.GetCardByID(id)
}

func (c *CloudGachaSource) GetGachaByID(id int) (*masterdata.Gacha, error) {
	if id == 0 {
		return nil, fmt.Errorf("gacha id is required")
	}
	c.mu.RLock()
	if cached, ok := c.gachaByID[id]; ok {
		c.mu.RUnlock()
		return cloneGacha(cached), nil
	}
	c.mu.RUnlock()

	entity, err := c.client.Gacha.
		Query().
		Where(
			gachaent.ServerRegionEQ(c.queryRegion),
			gachaent.GameIDEQ(int64(id)),
		).
		Only(c.context())
	if err != nil {
		return nil, fmt.Errorf("query gacha %d failed: %w", id, err)
	}
	model, err := convertGachaEntity(entity)
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	c.gachaByID[id] = model
	c.mu.Unlock()
	return cloneGacha(model), nil
}

func (c *CloudGachaSource) GetGachas() []*masterdata.Gacha {
	c.mu.RLock()
	if len(c.gachas) > 0 {
		defer c.mu.RUnlock()
		return cloneGachaList(c.gachas)
	}
	c.mu.RUnlock()

	entities, err := c.client.Gacha.
		Query().
		Where(gachaent.ServerRegionEQ(c.queryRegion)).
		All(c.context())
	if err != nil {
		return nil
	}

	items := make([]*masterdata.Gacha, 0, len(entities))
	byID := make(map[int]*masterdata.Gacha, len(entities))
	for _, entity := range entities {
		model, convErr := convertGachaEntity(entity)
		if convErr != nil {
			continue
		}
		items = append(items, model)
		byID[model.ID] = model
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].StartAt == items[j].StartAt {
			return items[i].ID > items[j].ID
		}
		return items[i].StartAt > items[j].StartAt
	})

	c.mu.Lock()
	c.gachas = items
	for id, item := range byID {
		c.gachaByID[id] = item
	}
	c.mu.Unlock()

	return cloneGachaList(items)
}

func cloneGacha(src *masterdata.Gacha) *masterdata.Gacha {
	if src == nil {
		return nil
	}
	dup := *src
	if len(src.GachaPickups) > 0 {
		dup.GachaPickups = make([]masterdata.GachaPickup, len(src.GachaPickups))
		copy(dup.GachaPickups, src.GachaPickups)
	}
	if len(src.GachaDetails) > 0 {
		dup.GachaDetails = make([]masterdata.GachaDetail, len(src.GachaDetails))
		copy(dup.GachaDetails, src.GachaDetails)
	}
	if len(src.GachaCardRarityRates) > 0 {
		dup.GachaCardRarityRates = make([]masterdata.GachaCardRarityRate, len(src.GachaCardRarityRates))
		copy(dup.GachaCardRarityRates, src.GachaCardRarityRates)
	}
	if len(src.GachaBehaviors) > 0 {
		dup.GachaBehaviors = make([]masterdata.GachaBehavior, len(src.GachaBehaviors))
		copy(dup.GachaBehaviors, src.GachaBehaviors)
	}
	return &dup
}

func cloneGachaList(items []*masterdata.Gacha) []*masterdata.Gacha {
	if len(items) == 0 {
		return nil
	}
	result := make([]*masterdata.Gacha, 0, len(items))
	for _, item := range items {
		result = append(result, cloneGacha(item))
	}
	return result
}
