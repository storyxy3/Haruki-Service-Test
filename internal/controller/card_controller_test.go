package controller

import (
	"Haruki-Service-API/internal/service"
	"Haruki-Service-API/pkg/masterdata"
	"testing"
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
	searchService := service.NewCardSearchService(repo, parser)

	// Controller
	ctrl := NewCardController(repo, nil, searchService, "http://localhost:8000", nil, nil)

	// Test
	cardIDs := []int{1, 2}
	region := "jp"
	req, err := ctrl.RenderCardListFromIDs(cardIDs, region)
	if err != nil {
		t.Fatalf("RenderCardListFromIDs failed: %v", err)
	}

	if len(req) == 0 {
		t.Errorf("Expected render bytes length gt 0")
	}
}
