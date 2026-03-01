package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"Haruki-Service-API/internal/apiutils"
	"Haruki-Service-API/internal/config"

	_ "github.com/lib/pq"
	"haruki-cloud/database/sekai/event"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg, err := config.Load("configs.yaml")
	if err != nil {
		logger.Error("Failed to load config", "error", err)
		os.Exit(1)
	}

	clients, err := apiutils.InitCloudClients(cfg.HarukiCloud, logger)
	if err != nil {
		logger.Error("Failed to initialize cloud clients", "error", err)
		os.Exit(1)
	}
	if clients.Sekai == nil {
		logger.Warn("Sekai client not configured; skipping probe")
		return
	}
	defer clients.Close()

	ctx := context.Background()
	count, err := clients.Sekai.Event.Query().Count(ctx)
	if err != nil {
		logger.Error("Failed to query events", "error", err)
		os.Exit(1)
	}
	fmt.Printf("Sekai event table contains %d rows\n", count)

	events, err := clients.Sekai.Event.Query().
		Order(event.ByID()).
		Limit(5).
		All(ctx)
	if err != nil {
		logger.Error("Failed to list sample events", "error", err)
		os.Exit(1)
	}
	for _, evt := range events {
		fmt.Printf("row_id=%d game_id=%d region=%s type=%s name=%s start=%d\n",
			evt.ID, evt.GameID, evt.ServerRegion, evt.EventType, evt.Name, evt.StartAt)
	}
}
