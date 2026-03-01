package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"Haruki-Service-API/pkg/masterdata"

	sekai "haruki-cloud/database/sekai"
	"haruki-cloud/database/sekai/event"
	"haruki-cloud/database/sekai/playerframe"
	"haruki-cloud/database/sekai/playerframegroup"
)

type profileRewardRange struct {
	EventRankingRewardDetails []profileRewardDetail `json:"eventRankingRewardDetails"`
}

type profileRewardDetail struct {
	ResourceType string `json:"resourceType"`
	ResourceID   int    `json:"resourceId"`
}

// CloudProfileSource implements ProfileDataSource backed by Haruki-Cloud database.
type CloudProfileSource struct {
	client      *sekai.Client
	region      string
	queryRegion string

	cardSource  *CloudCardSource
	honorSource *CloudHonorSource

	frameMu    sync.RWMutex
	frameCache map[int]*masterdata.PlayerFrame

	frameGroupMu    sync.RWMutex
	frameGroupCache map[int]*masterdata.PlayerFrameGroup

	eventHonorMu       sync.RWMutex
	eventByHonorID     map[int]int
	eventByHonorLoaded bool
}

func NewCloudProfileSource(client *sekai.Client, defaultRegion string) *CloudProfileSource {
	if client == nil {
		return nil
	}
	region := strings.TrimSpace(defaultRegion)
	if region == "" {
		region = "JP"
	}
	return &CloudProfileSource{
		client:          client,
		region:          region,
		queryRegion:     strings.ToLower(region),
		cardSource:      NewCloudCardSource(client, region),
		honorSource:     NewCloudHonorSource(client, region),
		frameCache:      make(map[int]*masterdata.PlayerFrame),
		frameGroupCache: make(map[int]*masterdata.PlayerFrameGroup),
		eventByHonorID:  make(map[int]int),
	}
}

func (c *CloudProfileSource) DefaultRegion() string {
	return c.region
}

func (c *CloudProfileSource) context() context.Context {
	return context.Background()
}

func (c *CloudProfileSource) GetPlayerFrameByID(id int) (*masterdata.PlayerFrame, error) {
	if id == 0 {
		return nil, fmt.Errorf("invalid player frame id")
	}
	c.frameMu.RLock()
	if cached, ok := c.frameCache[id]; ok {
		c.frameMu.RUnlock()
		copy := *cached
		return &copy, nil
	}
	c.frameMu.RUnlock()

	entity, err := c.client.Playerframe.Query().
		Where(
			playerframe.ServerRegionEQ(c.queryRegion),
			playerframe.GameIDEQ(int64(id)),
		).
		Only(c.context())
	if err != nil {
		return nil, fmt.Errorf("query player frame %d failed: %w", id, err)
	}
	model := &masterdata.PlayerFrame{
		ID:                 int(entity.GameID),
		Seq:                int(entity.Seq),
		PlayerFrameGroupID: int(entity.PlayerFrameGroupID),
		Description:        entity.Description,
		GameCharacterID:    int(entity.GameCharacterID),
	}

	c.frameMu.Lock()
	c.frameCache[id] = model
	c.frameMu.Unlock()

	copy := *model
	return &copy, nil
}

func (c *CloudProfileSource) GetPlayerFrameGroupByID(id int) (*masterdata.PlayerFrameGroup, error) {
	if id == 0 {
		return nil, fmt.Errorf("invalid player frame group id")
	}
	c.frameGroupMu.RLock()
	if cached, ok := c.frameGroupCache[id]; ok {
		c.frameGroupMu.RUnlock()
		copy := *cached
		return &copy, nil
	}
	c.frameGroupMu.RUnlock()

	entity, err := c.client.Playerframegroup.Query().
		Where(
			playerframegroup.ServerRegionEQ(c.queryRegion),
			playerframegroup.GameIDEQ(int64(id)),
		).
		Only(c.context())
	if err != nil {
		return nil, fmt.Errorf("query player frame group %d failed: %w", id, err)
	}
	model := &masterdata.PlayerFrameGroup{
		ID:              int(entity.GameID),
		Seq:             int(entity.Seq),
		Name:            entity.Name,
		AssetbundleName: entity.AssetbundleName,
	}

	c.frameGroupMu.Lock()
	c.frameGroupCache[id] = model
	c.frameGroupMu.Unlock()

	copy := *model
	return &copy, nil
}

func (c *CloudProfileSource) GetCardByID(id int) (*masterdata.Card, error) {
	if c.cardSource == nil {
		return nil, fmt.Errorf("card source not configured")
	}
	return c.cardSource.GetCardByID(id)
}

func (c *CloudProfileSource) GetEventIDByHonorID(honorID int) int {
	if honorID == 0 {
		return 0
	}

	c.eventHonorMu.RLock()
	if c.eventByHonorLoaded {
		eventID := c.eventByHonorID[honorID]
		c.eventHonorMu.RUnlock()
		return eventID
	}
	c.eventHonorMu.RUnlock()

	c.eventHonorMu.Lock()
	defer c.eventHonorMu.Unlock()
	if c.eventByHonorLoaded {
		return c.eventByHonorID[honorID]
	}

	items, err := c.client.Event.Query().
		Where(event.ServerRegionEQ(c.queryRegion)).
		All(c.context())
	if err != nil {
		return 0
	}
	for _, item := range items {
		var ranges []profileRewardRange
		if decodeErr := decodeFlexible(item.EventRankingRewardRanges, &ranges); decodeErr != nil {
			continue
		}
		eventID := int(item.GameID)
		for _, rr := range ranges {
			for _, detail := range rr.EventRankingRewardDetails {
				if strings.EqualFold(strings.TrimSpace(detail.ResourceType), "honor") && detail.ResourceID > 0 {
					c.eventByHonorID[detail.ResourceID] = eventID
				}
			}
		}
	}

	c.eventByHonorLoaded = true
	return c.eventByHonorID[honorID]
}

func (c *CloudProfileSource) GetHonorByID(id int) (*masterdata.Honor, error) {
	if c.honorSource == nil {
		return nil, fmt.Errorf("honor source not configured")
	}
	return c.honorSource.GetHonorByID(id)
}

func (c *CloudProfileSource) GetHonorGroupByID(id int) (*masterdata.HonorGroup, error) {
	if c.honorSource == nil {
		return nil, fmt.Errorf("honor source not configured")
	}
	return c.honorSource.GetHonorGroupByID(id)
}

func (c *CloudProfileSource) GetBondsHonorByID(id int) (*masterdata.BondsHonor, error) {
	if c.honorSource == nil {
		return nil, fmt.Errorf("honor source not configured")
	}
	return c.honorSource.GetBondsHonorByID(id)
}

func (c *CloudProfileSource) GetGameCharacterUnitByID(id int) (*masterdata.GameCharacterUnit, bool) {
	if c.honorSource == nil {
		return nil, false
	}
	return c.honorSource.GetGameCharacterUnitByID(id)
}

func decodeFlexible(src interface{}, target interface{}) error {
	raw, err := json.Marshal(src)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, target)
}
