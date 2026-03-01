package service

import (
	"context"
	"fmt"
	"strings"
	"sync"

	sekai "haruki-cloud/database/sekai"
	"haruki-cloud/database/sekai/stamp"
)

type CloudStampSource struct {
	client      *sekai.Client
	region      string
	queryRegion string

	mu     sync.RWMutex
	loaded bool
	stamps []StampRecord
}

func NewCloudStampSource(client *sekai.Client, defaultRegion string) *CloudStampSource {
	if client == nil {
		return nil
	}
	region := strings.TrimSpace(defaultRegion)
	if region == "" {
		region = "JP"
	}
	return &CloudStampSource{
		client:      client,
		region:      region,
		queryRegion: strings.ToLower(region),
	}
}

func (c *CloudStampSource) DefaultRegion() string {
	return c.region
}

func (c *CloudStampSource) GetStamps() ([]StampRecord, error) {
	c.mu.RLock()
	if c.loaded {
		out := append([]StampRecord(nil), c.stamps...)
		c.mu.RUnlock()
		return out, nil
	}
	c.mu.RUnlock()

	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.loaded {
		items, err := c.client.Stamp.Query().
			Where(stamp.ServerRegionEQ(c.queryRegion)).
			All(context.Background())
		if err != nil {
			return nil, fmt.Errorf("query stamps failed: %w", err)
		}
		stamps := make([]StampRecord, 0, len(items))
		for _, item := range items {
			stamps = append(stamps, StampRecord{
				ID:              int(item.GameID),
				AssetbundleName: item.AssetbundleName,
			})
		}
		c.stamps = stamps
		c.loaded = true
	}
	return append([]StampRecord(nil), c.stamps...), nil
}
