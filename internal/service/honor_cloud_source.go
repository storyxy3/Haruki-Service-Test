package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"Haruki-Service-API/pkg/masterdata"

	sekai "haruki-cloud/database/sekai"
	"haruki-cloud/database/sekai/bondshonor"
	"haruki-cloud/database/sekai/gamecharacterunit"
	"haruki-cloud/database/sekai/honor"
	"haruki-cloud/database/sekai/honorgroup"
)

// CloudHonorSource implements HonorDataSource backed by Haruki-Cloud database.
type CloudHonorSource struct {
	client      *sekai.Client
	region      string
	queryRegion string

	honorMu    sync.RWMutex
	honorCache map[int]*masterdata.Honor

	groupMu    sync.RWMutex
	groupCache map[int]*masterdata.HonorGroup

	bondsMu    sync.RWMutex
	bondsCache map[int]*masterdata.BondsHonor

	gcuMu    sync.RWMutex
	gcuCache map[int]*masterdata.GameCharacterUnit
}

func NewCloudHonorSource(client *sekai.Client, defaultRegion string) *CloudHonorSource {
	if client == nil {
		return nil
	}
	region := strings.TrimSpace(defaultRegion)
	if region == "" {
		region = "JP"
	}
	return &CloudHonorSource{
		client:      client,
		region:      region,
		queryRegion: strings.ToLower(region),
		honorCache:  make(map[int]*masterdata.Honor),
		groupCache:  make(map[int]*masterdata.HonorGroup),
		bondsCache:  make(map[int]*masterdata.BondsHonor),
		gcuCache:    make(map[int]*masterdata.GameCharacterUnit),
	}
}

func (c *CloudHonorSource) DefaultRegion() string {
	return c.region
}

func (c *CloudHonorSource) context() context.Context {
	return context.Background()
}

func (c *CloudHonorSource) GetHonorByID(id int) (*masterdata.Honor, error) {
	if id == 0 {
		return nil, fmt.Errorf("invalid honor id")
	}
	c.honorMu.RLock()
	if cached, ok := c.honorCache[id]; ok {
		c.honorMu.RUnlock()
		return cloneHonor(cached), nil
	}
	c.honorMu.RUnlock()

	entity, err := c.client.Honor.Query().
		Where(
			honor.ServerRegionEQ(c.queryRegion),
			honor.GameIDEQ(int64(id)),
		).
		Only(c.context())
	if err != nil {
		return nil, fmt.Errorf("query honor %d failed: %w", id, err)
	}
	model, err := convertCloudHonor(entity)
	if err != nil {
		return nil, err
	}

	c.honorMu.Lock()
	c.honorCache[id] = model
	c.honorMu.Unlock()
	return cloneHonor(model), nil
}

func (c *CloudHonorSource) GetHonorGroupByID(id int) (*masterdata.HonorGroup, error) {
	if id == 0 {
		return nil, fmt.Errorf("invalid honor group id")
	}
	c.groupMu.RLock()
	if cached, ok := c.groupCache[id]; ok {
		c.groupMu.RUnlock()
		return cloneHonorGroup(cached), nil
	}
	c.groupMu.RUnlock()

	entity, err := c.client.Honorgroup.Query().
		Where(
			honorgroup.ServerRegionEQ(c.queryRegion),
			honorgroup.GameIDEQ(int64(id)),
		).
		Only(c.context())
	if err != nil {
		return nil, fmt.Errorf("query honor group %d failed: %w", id, err)
	}
	model := &masterdata.HonorGroup{
		ID:          int(entity.GameID),
		HonorType:   entity.HonorType,
		Name:        entity.Name,
		Description: "",
	}
	if value := strings.TrimSpace(entity.BackgroundAssetbundleName); value != "" {
		model.BackgroundAssetbundleName = &value
	}
	if value := strings.TrimSpace(entity.FrameName); value != "" {
		model.FrameName = &value
	}

	c.groupMu.Lock()
	c.groupCache[id] = model
	c.groupMu.Unlock()
	return cloneHonorGroup(model), nil
}

func (c *CloudHonorSource) GetBondsHonorByID(id int) (*masterdata.BondsHonor, error) {
	if id == 0 {
		return nil, fmt.Errorf("invalid bonds honor id")
	}
	c.bondsMu.RLock()
	if cached, ok := c.bondsCache[id]; ok {
		c.bondsMu.RUnlock()
		return cloneBondsHonor(cached), nil
	}
	c.bondsMu.RUnlock()

	entity, err := c.client.Bondshonor.Query().
		Where(
			bondshonor.ServerRegionEQ(c.queryRegion),
			bondshonor.GameIDEQ(int64(id)),
		).
		Only(c.context())
	if err != nil {
		return nil, fmt.Errorf("query bonds honor %d failed: %w", id, err)
	}
	model := &masterdata.BondsHonor{
		ID:                   int(entity.GameID),
		GameCharacterUnitId1: int(entity.GameCharacterUnitId1),
		GameCharacterUnitId2: int(entity.GameCharacterUnitId2),
		HonorRarity:          entity.HonorRarity,
		Name:                 entity.Name,
		Description:          entity.Description,
		BondsGroupId:         int(entity.BondsGroupID),
	}

	c.bondsMu.Lock()
	c.bondsCache[id] = model
	c.bondsMu.Unlock()
	return cloneBondsHonor(model), nil
}

func (c *CloudHonorSource) GetGameCharacterUnitByID(id int) (*masterdata.GameCharacterUnit, bool) {
	if id == 0 {
		return nil, false
	}
	c.gcuMu.RLock()
	if cached, ok := c.gcuCache[id]; ok {
		c.gcuMu.RUnlock()
		return cloneGameCharacterUnit(cached), true
	}
	c.gcuMu.RUnlock()

	entity, err := c.client.Gamecharacterunit.Query().
		Where(
			gamecharacterunit.ServerRegionEQ(c.queryRegion),
			gamecharacterunit.GameIDEQ(int64(id)),
		).
		Only(c.context())
	if err != nil {
		return nil, false
	}
	model := &masterdata.GameCharacterUnit{
		ID:              int(entity.GameID),
		GameCharacterID: int(entity.GameCharacterID),
		Unit:            entity.Unit,
		ColorCode:       entity.ColorCode,
	}

	c.gcuMu.Lock()
	c.gcuCache[id] = model
	c.gcuMu.Unlock()
	return cloneGameCharacterUnit(model), true
}

func convertCloudHonor(entity *sekai.Honor) (*masterdata.Honor, error) {
	model := &masterdata.Honor{
		ID:              int(entity.GameID),
		GroupID:         int(entity.GroupID),
		HonorRarity:     entity.HonorRarity,
		Name:            entity.Name,
		Description:     "",
		AssetbundleName: entity.AssetbundleName,
	}
	if len(entity.Levels) > 0 {
		raw, err := json.Marshal(entity.Levels)
		if err != nil {
			return nil, fmt.Errorf("marshal honor levels failed: %w", err)
		}
		if err := json.Unmarshal(raw, &model.Levels); err != nil {
			return nil, fmt.Errorf("unmarshal honor levels failed: %w", err)
		}
	}
	return model, nil
}

func cloneHonor(src *masterdata.Honor) *masterdata.Honor {
	if src == nil {
		return nil
	}
	copy := *src
	if src.Levels != nil {
		copy.Levels = append([]masterdata.HonorLevel(nil), src.Levels...)
	}
	return &copy
}

func cloneHonorGroup(src *masterdata.HonorGroup) *masterdata.HonorGroup {
	if src == nil {
		return nil
	}
	copy := *src
	if src.BackgroundAssetbundleName != nil {
		value := *src.BackgroundAssetbundleName
		copy.BackgroundAssetbundleName = &value
	}
	if src.FrameName != nil {
		value := *src.FrameName
		copy.FrameName = &value
	}
	return &copy
}

func cloneBondsHonor(src *masterdata.BondsHonor) *masterdata.BondsHonor {
	if src == nil {
		return nil
	}
	copy := *src
	return &copy
}

func cloneGameCharacterUnit(src *masterdata.GameCharacterUnit) *masterdata.GameCharacterUnit {
	if src == nil {
		return nil
	}
	copy := *src
	return &copy
}
