package main

import (
	"testing"

	"Haruki-Service-API/internal/service"
)

func TestParseEventListCommand_WLDefaults(t *testing.T) {
	parser := service.NewEventParser(map[string]int{"mnr": 5, "hrk": 6})
	query, err := parseEventListCommand("/event-list wl", parser)
	if err != nil {
		t.Fatalf("parseEventListCommand failed: %v", err)
	}
	if query.Region != "jp" {
		t.Fatalf("expected default region jp, got %s", query.Region)
	}
	if query.EventType != "world_bloom" {
		t.Fatalf("expected world_bloom type, got %s", query.EventType)
	}
	if !query.IncludePast || !query.IncludeFuture {
		t.Fatalf("expected both past/future enabled by default")
	}
}

func TestParseEventListCommand_RejectUnknownToken(t *testing.T) {
	parser := service.NewEventParser(map[string]int{"mnr": 5})
	if _, err := parseEventListCommand("/events wl foo", parser); err == nil {
		t.Fatalf("expected unknown token error")
	}
}

func TestParseEventListCommand_RejectBlendWithUnit(t *testing.T) {
	parser := service.NewEventParser(map[string]int{"mnr": 5})
	if _, err := parseEventListCommand("/events blend 25h", parser); err == nil {
		t.Fatalf("expected blend/unit conflict error")
	}
}
