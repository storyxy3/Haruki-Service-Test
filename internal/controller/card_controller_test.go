package controller

import (
	"testing"

	"Haruki-Service-API/internal/service"
	"Haruki-Service-API/pkg/masterdata"
)

func TestCardController_BuildCardListRequest(t *testing.T) {
	// Mock Repo
	repo := service.NewMasterDataService("", "JP")
	repo.SetCards([]masterdata.Card{
		{ID: 1, CharacterID: 5, AssetBundleName: "mnr1", ReleaseAt: 100},
		{ID: 2, CharacterID: 5, AssetBundleName: "mnr2", ReleaseAt: 200},
	})
	repo.SetNicknames(map[string]int{"mnr": 5})

	// Service
	parser := service.NewCardParser(map[string]int{"mnr": 5})
	cardSource := service.NewMasterDataCardSource(repo)
	eventSource := service.NewMasterDataEventSource(repo)
	searchService := service.NewCardSearchService(cardSource, parser)

	// Controller
	ctrl := NewCardController(cardSource, cardSource, eventSource, repo, nil, searchService, "http://localhost:8000", nil, nil)

	// Test
	cardIDs := []int{1, 2}
	region := "jp"
	if _, err := ctrl.RenderCardListFromIDs(cardIDs, region); err == nil {
		t.Fatalf("expected error when drawing service is nil")
	} else if err != ErrDrawingServiceUnavailable {
		t.Fatalf("unexpected error: %v", err)
	}
}
