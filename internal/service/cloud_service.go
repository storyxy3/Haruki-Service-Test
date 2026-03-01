package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"entgo.io/ent/dialect/sql"

	sekai "haruki-cloud/database/sekai"
	"haruki-cloud/database/sekai/event"
)

var ErrCloudClientUnavailable = errors.New("haruki cloud database client is not configured")

// CloudService wraps Haruki-Cloud ent clients for runtime queries.
type CloudService struct {
	sekai *sekai.Client
}

// NewCloudService builds a CloudService. Returns nil when sekaiClient is nil.
func NewCloudService(sekaiClient *sekai.Client) *CloudService {
	if sekaiClient == nil {
		return nil
	}
	return &CloudService{sekai: sekaiClient}
}

// GetCurrentEventID returns the game_id of the currently running event for the region.
func (c *CloudService) GetCurrentEventID(ctx context.Context, region string, now time.Time) (int, error) {
	evt, err := c.GetCurrentEvent(ctx, region, now)
	if err != nil {
		return 0, err
	}
	if evt.GameID == 0 {
		return 0, fmt.Errorf("current event missing game_id")
	}
	return int(evt.GameID), nil
}

// GetCurrentEvent queries the Sekai database for the event active at the provided time.
func (c *CloudService) GetCurrentEvent(ctx context.Context, region string, now time.Time) (*sekai.Event, error) {
	if c == nil || c.sekai == nil {
		return nil, ErrCloudClientUnavailable
	}
	region = normalizeRegion(region)
	currentMillis := now.UnixMilli()
	evt, err := c.sekai.Event.
		Query().
		Where(
			event.ServerRegionEQ(region),
			event.StartAtLTE(currentMillis),
			event.ClosedAtGTE(currentMillis),
		).
		Order(event.ByStartAt(sql.OrderDesc())).
		First(ctx)
	if err != nil {
		return nil, fmt.Errorf("query current event failed: %w", err)
	}
	return evt, nil
}

func normalizeRegion(region string) string {
	region = strings.TrimSpace(region)
	if region == "" {
		region = "JP"
	}
	return strings.ToUpper(region)
}
