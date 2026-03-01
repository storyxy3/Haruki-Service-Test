package service

import (
	"fmt"
	"Haruki-Service-API/pkg/masterdata"
)

// CardSearchService 负责卡牌搜索逻辑 (Orchestrator)
type CardSearchService struct {
	repo   CardDataSource // Data Access Layer
	parser *CardParser    // Parsing Logic
}

// NewCardSearchService 创建卡牌搜索服务
func NewCardSearchService(repo CardDataSource, parser *CardParser) *CardSearchService {
	return &CardSearchService{
		repo:   repo,
		parser: parser,
	}
}

func (s *CardSearchService) CloneWithRepo(repo CardDataSource) *CardSearchService {
	if s == nil {
		return nil
	}
	return &CardSearchService{
		repo:   repo,
		parser: s.parser,
	}
}

// Search 根据查询字符串搜索卡牌
func (s *CardSearchService) Search(query string) (*masterdata.Card, error) {
	fmt.Printf("[DEBUG] CardSearchService: Searching with Query: %s\n", query)

	// 1. Parsing (Command Understanding)
	info, err := s.parser.Parse(query)
	if err == nil && info != nil {
		switch info.Type {
		case QueryTypeID:
			fmt.Printf("[DEBUG] Executing ID Search: %d\n", info.Value)
			// 2. Database Execution (Data Access)
			card, err := s.repo.GetCardByID(info.Value)
			if err != nil {
				return nil, err
			}
			fmt.Printf("[DEBUG] Found Card by ID: %d (%s)\n", card.ID, card.AssetBundleName)
			return card, nil

		case QueryTypeSeq:
			fmt.Printf("[DEBUG] Executing Seq Search: CharID=%d, Seq=%d\n", info.CharacterID, info.Sequence)
			// 2. Database Execution
			return s.repo.GetCardByCharacterAndSeq(info.CharacterID, info.Sequence)

		case QueryTypeFilter:
			fmt.Printf("[DEBUG] Executing Filter Search: CharID=%d, Rarity=%s, Attr=%s\n", info.CharacterID, info.Rarity, info.Attr)
			// 2. Database Execution
			filtered, ferr := s.repo.FilterCards(info)
			if ferr != nil {
				return nil, ferr
			}
			if len(filtered) == 0 {
				return nil, fmt.Errorf("card not found (filter): %s", query)
			}
			// Return the latest one
			card := filtered[len(filtered)-1]
			fmt.Printf("[DEBUG] Found Card by Filter: %d (%s)\n", card.ID, card.AssetBundleName)
			return card, nil
		}
	}

	return nil, fmt.Errorf("无法解析的指令: %s", query)
}

// SearchList 根据查询字符串搜索卡牌列表
func (s *CardSearchService) SearchList(query string) ([]*masterdata.Card, error) {
	fmt.Printf("[DEBUG] CardSearchService: Searching List with Query: %s\n", query)

	// 1. Parsing
	info, err := s.parser.Parse(query)
	if err == nil && info != nil {
		switch info.Type {
		case QueryTypeFilter:
			fmt.Printf("[DEBUG] Executing Filter List Search: CharID=%d, Rarity=%s, Attr=%s\n", info.CharacterID, info.Rarity, info.Attr)
			// Return ALL matched cards
			filtered, ferr := s.repo.FilterCards(info)
			if ferr != nil {
				return nil, ferr
			}
			if len(filtered) == 0 {
				return nil, fmt.Errorf("No cards found for filter: %s", query)
			}
			fmt.Printf("[DEBUG] Found %d cards by Filter\n", len(filtered))
			return filtered, nil

		case QueryTypeID:
			// ID search returns a list of 1
			card, err := s.repo.GetCardByID(info.Value)
			if err != nil {
				return nil, err
			}
			return []*masterdata.Card{card}, nil

		case QueryTypeSeq:
			// Seq search returns a list of 1
			card, err := s.repo.GetCardByCharacterAndSeq(info.CharacterID, info.Sequence)
			if err != nil {
				return nil, err
			}
			return []*masterdata.Card{card}, nil
		}
	}

	return nil, fmt.Errorf("无法解析的列表查询指令: %s", query)
}
