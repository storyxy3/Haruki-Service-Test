package service

import (
	"Haruki-Service-API/pkg/masterdata"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

// GetSkillByID 根据 ID 获取技能
func (s *MasterDataService) GetSkillByID(id int) (*masterdata.Skill, error) {
	if skill, ok := s.skillByID[id]; ok {
		return skill, nil
	}
	return nil, nil
}

// GetCharacterByID 根据 ID 获取角色
func (s *MasterDataService) GetCharacterByID(id int) (*masterdata.Character, error) {
	if char, ok := s.charByID[id]; ok {
		return char, nil
	}
	return nil, nil
}

// GetEventByCardID 根据卡牌 ID 获取关联的活动
func (s *MasterDataService) GetEventByCardID(cardID int) (*masterdata.Event, error) {
	for eventID, cardIDs := range s.cardsByEventID {
		for _, cid := range cardIDs {
			if cid == cardID {
				if event, ok := s.eventByID[eventID]; ok {
					return event, nil
				}
			}
		}
	}
	return nil, nil
}

// GetGachaByCardID 根据卡牌 ID 获取关联的卡池
func (s *MasterDataService) GetGachaByCardID(cardID int) (*masterdata.Gacha, error) {
	card, ok := s.cardByID[cardID]
	if !ok {
		return nil, fmt.Errorf("card not found: %d", cardID)
	}

	for i := range s.gachas {
		gacha := &s.gachas[i]
		if gacha.StartAt <= card.ReleaseAt && card.ReleaseAt <= gacha.EndAt {
			isPickup := false
			for _, pickup := range gacha.GachaPickups {
				if pickup.CardID == cardID {
					isPickup = true
					break
				}
			}
			if isPickup {
				return gacha, nil
			}
		}
	}
	return nil, fmt.Errorf("gacha not found for card: %d", cardID)
}

// GetGachaByID 根据卡池ID获取卡池
func (s *MasterDataService) GetGachaByID(id int) (*masterdata.Gacha, error) {
	if gacha, ok := s.gachaByID[id]; ok {
		return gacha, nil
	}
	return nil, fmt.Errorf("gacha not found: %d", id)
}

// GetGachas 获取所有卡池
func (s *MasterDataService) GetGachas() []*masterdata.Gacha {
	result := make([]*masterdata.Gacha, len(s.gachas))
	for i := range s.gachas {
		result[i] = &s.gachas[i]
	}
	return result
}

// GetCostume3dByCardID 根据卡牌ID获取关联的3D服装 (返回第一个)
func (s *MasterDataService) GetCostume3dByCardID(cardID int) (*masterdata.Costume3d, error) {
	costumeIDs, ok := s.costume3dByCardID[cardID]
	if !ok || len(costumeIDs) == 0 {
		return nil, nil
	}
	costume, ok := s.costume3dByID[costumeIDs[0]]
	if !ok {
		return nil, fmt.Errorf("costume not found: %d", costumeIDs[0])
	}
	return costume, nil
}

// GetCostume3dsByCardID 根据卡牌ID获取所有关联的3D服装
func (s *MasterDataService) GetCostume3dsByCardID(cardID int) ([]*masterdata.Costume3d, error) {
	costumeIDs, ok := s.costume3dByCardID[cardID]
	if !ok || len(costumeIDs) == 0 {
		return nil, nil
	}
	var costumes []*masterdata.Costume3d
	for _, id := range costumeIDs {
		if c, ok := s.costume3dByID[id]; ok {
			costumes = append(costumes, c)
		}
	}
	return costumes, nil
}

// GetCardSupplyType 获取卡牌的供给类型
func (s *MasterDataService) GetCardSupplyType(card *masterdata.Card) string {
	if card.CardRarityType == "rarity_birthday" {
		return formatSupplyType("birthday")
	}
	if card.CardSupplyID != 0 {
		if supply, ok := s.cardSupplyByID[card.CardSupplyID]; ok {
			return formatSupplyType(supply.CardSupplyType)
		}
	}
	return formatSupplyType("")
}

// GetEventBannerCharacterID 获取活动的 Banner 角色 ID
func (s *MasterDataService) GetEventBannerCharacterID(eventID int) (int, error) {
	cardIDs, ok := s.cardsByEventID[eventID]
	if !ok || len(cardIDs) == 0 {
		return 0, fmt.Errorf("no cards found for event %d", eventID)
	}

	minCardID := -1
	for _, cid := range cardIDs {
		card, ok := s.cardByID[cid]
		if !ok {
			continue
		}
		isFes := false
		if card.CardSupplyID != 0 {
			if supply, ok := s.cardSupplyByID[card.CardSupplyID]; ok {
				if supply.CardSupplyType == "colorful_festival_limited" || supply.CardSupplyType == "bloom_festival_limited" {
					isFes = true
				}
			}
		}
		if isFes {
			continue
		}
		if minCardID == -1 || cid < minCardID {
			minCardID = cid
		}
	}

	if minCardID == -1 {
		return 0, fmt.Errorf("no valid banner card found for event %d", eventID)
	}
	card, ok := s.cardByID[minCardID]
	if !ok {
		return 0, fmt.Errorf("banner card %d not found", minCardID)
	}
	return card.CharacterID, nil
}

// GetUnitByCardID 获取卡牌所属团名
func (s *MasterDataService) GetUnitByCardID(cardID int) (string, error) {
	card, ok := s.cardByID[cardID]
	if !ok {
		return "", fmt.Errorf("card not found: %d", cardID)
	}
	char, ok := s.charByID[card.CharacterID]
	if !ok {
		return "", fmt.Errorf("character not found: %d", card.CharacterID)
	}
	if char.Unit != "piapro" {
		return char.Unit, nil
	}
	if card.SupportUnit != "" && card.SupportUnit != "none" {
		return card.SupportUnit, nil
	}
	return "piapro", nil
}

// FilterCards 根据查询条件筛选卡牌
func (s *MasterDataService) FilterCards(info *CardQueryInfo) []*masterdata.Card {
	var results []*masterdata.Card
	for _, card := range s.cards {
		if info.CharacterID != 0 && card.CharacterID != info.CharacterID {
			continue
		}
		if info.Rarity != "" && card.CardRarityType != info.Rarity {
			continue
		}
		if info.Attr != "" && card.Attr != info.Attr {
			continue
		}
		if info.SkillType != "" {
			skill, err := s.GetSkillByID(card.SkillID)
			if err != nil || skill == nil {
				continue
			}
			if skill.DescriptionSpriteName != info.SkillType {
				continue
			}
		}
		if info.SupplyType != "" && !matchesSupplyFilter(info.SupplyType, s.GetCardSupplyType(&card)) {
			continue
		}
		if info.Year != 0 {
			t := time.Unix(card.ReleaseAt/1000, 0)
			if t.Year() != info.Year {
				continue
			}
		}
		results = append(results, &card)
	}
	return results
}

// GetEventDeckBonuses 获取活动的加成信息
func (s *MasterDataService) GetEventDeckBonuses(eventID int) ([]*masterdata.EventDeckBonus, error) {
	bonuses, ok := s.deckBonusesByEventID[eventID]
	if !ok {
		return nil, fmt.Errorf("no bonuses found for event %d", eventID)
	}
	return bonuses, nil
}

// GetEventCards 返回活动关联的卡牌
func (s *MasterDataService) GetEventCards(eventID int) ([]*masterdata.Card, error) {
	cardIDs, ok := s.cardsByEventID[eventID]
	if !ok || len(cardIDs) == 0 {
		return nil, fmt.Errorf("no cards found for event %d", eventID)
	}
	var cards []*masterdata.Card
	for _, cid := range cardIDs {
		if card, ok := s.cardByID[cid]; ok {
			cards = append(cards, card)
		}
	}
	if len(cards) == 0 {
		return nil, fmt.Errorf("no cards found for event %d", eventID)
	}
	return cards, nil
}

// GetGameCharacterUnit 根据 ID 获取角色组合信息
func (s *MasterDataService) GetGameCharacterUnit(id int) (*masterdata.GameCharacterUnit, error) {
	unit, ok := s.gameCharUnitByID[id]
	if !ok {
		return nil, fmt.Errorf("game character unit not found: %d", id)
	}
	return unit, nil
}

// GetEventByID 根据 ID 获取活动
func (s *MasterDataService) GetEventByID(id int) (*masterdata.Event, error) {
	event, ok := s.eventByID[id]
	if !ok {
		return nil, fmt.Errorf("event not found: %d", id)
	}
	return event, nil
}

// GetEvents 获取所有活动 (按开始时间排序)
func (s *MasterDataService) GetEvents() []*masterdata.Event {
	result := make([]*masterdata.Event, len(s.events))
	for i := range s.events {
		result[i] = &s.events[i]
	}
	return result
}

// GetBanEvents 获取某角色的所有箱活
func (s *MasterDataService) GetBanEvents(charID int) []*masterdata.Event {
	var banEvents []*masterdata.Event
	for _, event := range s.events {
		if event.EventType != "marathon" && event.EventType != "cheerful_carnival" {
			continue
		}
		cardIDs := s.cardsByEventID[event.ID]
		if len(cardIDs) == 0 {
			continue
		}
		minCardID := -1
		for _, cid := range cardIDs {
			card, ok := s.cardByID[cid]
			if !ok {
				continue
			}
			isFes := false
			if card.CardSupplyID != 0 {
				if supply, ok := s.cardSupplyByID[card.CardSupplyID]; ok {
					if supply.CardSupplyType == "colorful_festival_limited" || supply.CardSupplyType == "bloom_festival_limited" {
						isFes = true
					}
				}
			}
			if isFes {
				continue
			}
			if minCardID == -1 || cid < minCardID {
				minCardID = cid
			}
		}
		if minCardID != -1 {
			if card, ok := s.cardByID[minCardID]; ok && card.CharacterID == charID {
				banEvents = append(banEvents, &event)
			}
		}
	}
	return banEvents
}

// GetWorldBloomChapters 返回 WL 章节
func (s *MasterDataService) GetWorldBloomChapters(eventID int) []*masterdata.WorldBloom {
	if chapters, ok := s.worldBloomsByEventID[eventID]; ok {
		return chapters
	}
	return nil
}

// GetHonorByID 获取称号
func (s *MasterDataService) GetHonorByID(id int) (*masterdata.Honor, error) {
	if honor, ok := s.honorByID[id]; ok {
		return honor, nil
	}
	return nil, fmt.Errorf("honor not found for id %d", id)
}

// GetHonorGroupByID 获取称号组
func (s *MasterDataService) GetHonorGroupByID(id int) (*masterdata.HonorGroup, error) {
	if group, ok := s.honorGroupByID[id]; ok {
		return group, nil
	}
	return nil, fmt.Errorf("honor group not found for id %d", id)
}

// FilterEvents 根据条件筛选活动
func (s *MasterDataService) FilterEvents(filter EventFilter) []*masterdata.Event {
	var result []*masterdata.Event
	for i := range s.events {
		event := &s.events[i]
		if filter.EventType != "" && event.EventType != filter.EventType {
			continue
		}
		if filter.Year != 0 {
			t := time.Unix(event.StartAt/1000, 0)
			if t.Year() != filter.Year {
				continue
			}
		}
		if filter.Unit != "" || filter.Attr != "" {
			bonuses := s.deckBonusesByEventID[event.ID]
			matched := false
			for _, b := range bonuses {
				matchUnit := true
				matchAttr := true
				if filter.Attr != "" && b.CardAttr != filter.Attr {
					matchAttr = false
				}
				if filter.Unit != "" {
					if gcu, ok := s.gameCharUnitByID[b.GameCharacterUnitID]; ok {
						if gcu.Unit != filter.Unit {
							matchUnit = false
						}
					} else {
						matchUnit = false
					}
				}
				if matchUnit && matchAttr {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}
		if filter.CharacterID != 0 {
			hasCard := false
			cardIDs := s.cardsByEventID[event.ID]
			for _, cid := range cardIDs {
				if c, ok := s.cardByID[cid]; ok && c.CharacterID == filter.CharacterID {
					hasCard = true
					break
				}
			}
			if !hasCard {
				continue
			}
		}
		result = append(result, event)
	}
	return result
}

// GetMusicDifficulties 根据音乐 ID 获取所有难度信息
func (s *MasterDataService) GetMusicDifficulties(musicID int) ([]*masterdata.MusicDifficulty, error) {
	diffs, ok := s.difficultiesByMusicID[musicID]
	if !ok {
		return nil, fmt.Errorf("no difficulties found for music %d", musicID)
	}
	return diffs, nil
}

// GetMusicVocals 根据音乐 ID 获取所有 Vocal 信息
func (s *MasterDataService) GetMusicVocals(musicID int) ([]*masterdata.MusicVocal, error) {
	vocals, ok := s.vocalsByMusicID[musicID]
	if !ok {
		return nil, fmt.Errorf("no vocals found for music %d", musicID)
	}
	return vocals, nil
}

// GetMusicTags 根据音乐 ID 获取所有标签
func (s *MasterDataService) GetMusicTags(musicID int) ([]string, error) {
	tags, ok := s.tagsByMusicID[musicID]
	if !ok {
		return []string{}, nil
	}
	return tags, nil
}

// SearchMusic 搜索音乐 (支持 ID 或标题模糊搜索)
func (s *MasterDataService) SearchMusic(query string) (*masterdata.Music, error) {
	if _, err := strconv.Atoi(query); err == nil {
		id, _ := strconv.Atoi(query)
		if m, ok := s.musicByID[id]; ok {
			return m, nil
		}
	}
	queryLower := strings.ToLower(query)
	var bestMatch *masterdata.Music
	for i := range s.musics {
		m := &s.musics[i]
		if strings.ToLower(m.Title) == queryLower {
			return m, nil
		}
	}
	for i := range s.musics {
		m := &s.musics[i]
		if strings.Contains(strings.ToLower(m.Title), queryLower) {
			if bestMatch == nil || len(m.Title) < len(bestMatch.Title) {
				bestMatch = m
			}
		}
	}
	if bestMatch != nil {
		return bestMatch, nil
	}
	return nil, fmt.Errorf("music not found: %s", query)
}

// GetBondsHonorByID 获取羁绊称号
func (s *MasterDataService) GetBondsHonorByID(id int) (*masterdata.BondsHonor, error) {
	if bonds, ok := s.bondsHonorByID[id]; ok {
		return bonds, nil
	}
	return nil, fmt.Errorf("bonds honor not found for id %d", id)
}

// GetGameCharacterUnitByID 获取角色分组信息
func (s *MasterDataService) GetGameCharacterUnitByID(id int) (*masterdata.GameCharacterUnit, bool) {
	unit, ok := s.gameCharUnitByID[id]
	return unit, ok
}

// GetMusicByID 根据 ID 获取音乐
func (s *MasterDataService) GetMusicByID(id int) (*masterdata.Music, error) {
	if m, ok := s.musicByID[id]; ok {
		return m, nil
	}
	return nil, fmt.Errorf("music not found: %d", id)
}

// GetMusicByEventID 获取活动的"书下曲"
func (s *MasterDataService) GetMusicByEventID(eventID int) (*masterdata.Music, error) {
	event, ok := s.eventByID[eventID]
	if !ok {
		return nil, fmt.Errorf("event not found: %d", eventID)
	}
	for i := range s.musics {
		m := &s.musics[i]
		if m.PublishedAt == event.StartAt {
			return m, nil
		}
	}
	return nil, fmt.Errorf("no music found for event %d", eventID)
}

// GetCardByID 根据 ID 获取卡牌
func (s *MasterDataService) GetCardByID(id int) (*masterdata.Card, error) {
	card, ok := s.cardByID[id]
	if !ok {
		return nil, fmt.Errorf("card not found: %d", id)
	}
	return card, nil
}

// GetEventsByMusicID 返回关联该乐曲的活动列表（按开始时间升序）
func (s *MasterDataService) GetEventsByMusicID(musicID int) ([]*masterdata.Event, error) {
	eventIDs, ok := s.eventIDsByMusicID[musicID]
	if !ok || len(eventIDs) == 0 {
		return nil, fmt.Errorf("no events found for music %d", musicID)
	}

	events := make([]*masterdata.Event, 0, len(eventIDs))
	for _, id := range eventIDs {
		if ev, ok := s.eventByID[id]; ok {
			events = append(events, ev)
		}
	}
	if len(events) == 0 {
		return nil, fmt.Errorf("no events found for music %d", musicID)
	}
	sort.Slice(events, func(i, j int) bool {
		return events[i].StartAt < events[j].StartAt
	})
	return events, nil
}

// GetPrimaryEventByMusicID 返回书下曲所属的首个活动
func (s *MasterDataService) GetPrimaryEventByMusicID(musicID int) (*masterdata.Event, error) {
	events, err := s.GetEventsByMusicID(musicID)
	if err != nil {
		return nil, err
	}
	return events[0], nil
}

// GetChallengeRewardsByCharacter 返回指定角色的 challenge live 奖励阈值
func (s *MasterDataService) GetChallengeRewardsByCharacter(charID int) []*masterdata.ChallengeLiveHighScoreReward {
	if s == nil {
		return nil
	}
	return s.challengeRewardsByCID[charID]
}

// GetResourceBox 根据 ID 获取资源箱
func (s *MasterDataService) GetResourceBox(id int) *masterdata.ResourceBox {
	if s == nil {
		return nil
	}
	return s.resourceBoxByID[id]
}

// GetResourceBoxByPurpose 根据purpose+ID查找资源箱
func (s *MasterDataService) GetResourceBoxByPurpose(purpose string, id int) *masterdata.ResourceBox {
	if s == nil {
		return nil
	}
	if purpose == "" {
		return s.resourceBoxByID[id]
	}
	purposeMap, ok := s.resourceBoxesByPurpose[purpose]
	if !ok {
		return nil
	}
	return purposeMap[id]
}

// GetCardByCharacterAndSeq 根据角色和序号查找卡牌
func (s *MasterDataService) GetCardByCharacterAndSeq(charID, seq int) (*masterdata.Card, error) {
	var charCards []*masterdata.Card
	for i := range s.cards {
		if s.cards[i].CharacterID == charID {
			charCards = append(charCards, &s.cards[i])
		}
	}

	if len(charCards) == 0 {
		return nil, fmt.Errorf("no cards found for character %d", charID)
	}

	sort.Slice(charCards, func(i, j int) bool {
		return charCards[i].ReleaseAt < charCards[j].ReleaseAt
	})

	if seq < 0 {
		index := len(charCards) + seq
		if index < 0 || index >= len(charCards) {
			return nil, fmt.Errorf("card sequence out of range: %d (total: %d)", seq, len(charCards))
		}
		return charCards[index], nil
	}

	if seq < 1 || seq > len(charCards) {
		return nil, fmt.Errorf("card sequence out of range: %d (total: %d)", seq, len(charCards))
	}

	return charCards[seq-1], nil
}

// GetMusics 获取所有音乐 (按 PublishedAt 排序)
func (s *MasterDataService) GetMusics() []*masterdata.Music {
	result := make([]*masterdata.Music, len(s.musics))
	for i := range s.musics {
		result[i] = &s.musics[i]
	}
	return result
}

// GetEventIDByHonorID 根据称号 ID 获取关联的活动 ID
func (s *MasterDataService) GetEventIDByHonorID(honorID int) int {
	if eventID, ok := s.eventIDByHonorID[honorID]; ok {
		return eventID
	}
	return 0
}
