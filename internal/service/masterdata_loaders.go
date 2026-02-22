package service

// loadCards 加载卡牌数据
func (s *MasterDataService) loadCards() error {
	return s.loadJSON("cards.json", &s.cards)
}

// loadCharacters 加载角色数据
func (s *MasterDataService) loadCharacters() error {
	return s.loadJSON("gameCharacters.json", &s.characters)
}

// loadSkills 加载技能数据
func (s *MasterDataService) loadSkills() error {
	return s.loadJSON("skills.json", &s.skills)
}

// loadMusics 加载音乐数据
func (s *MasterDataService) loadMusics() error {
	return s.loadJSON("musics.json", &s.musics)
}

// loadMusicDifficulties 加载音乐难度数据
func (s *MasterDataService) loadMusicDifficulties() error {
	return s.loadJSON("musicDifficulties.json", &s.musicDifficulties)
}

// loadMusicVocals 加载音乐 Vocal 数据
func (s *MasterDataService) loadMusicVocals() error {
	return s.loadJSON("musicVocals.json", &s.musicVocals)
}

// loadMusicTags 加载音乐标签数据
func (s *MasterDataService) loadMusicTags() error {
	return s.loadJSON("musicTags.json", &s.musicTags)
}

// loadChallengeLiveRewards 加载 challenge live 奖励阈值
func (s *MasterDataService) loadChallengeLiveRewards() error {
	return s.loadJSON("challengeLiveHighScoreRewards.json", &s.challengeRewards)
}

// loadResourceBoxes 加载资源箱
func (s *MasterDataService) loadResourceBoxes() error {
	return s.loadJSON("resourceBoxes.json", &s.resourceBoxes)
}

// loadEvents 加载活动数据
func (s *MasterDataService) loadEvents() error {
	return s.loadJSON("events.json", &s.events)
}

// loadGachas 加载卡池数据
func (s *MasterDataService) loadGachas() error {
	return s.loadJSON("gachas.json", &s.gachas)
}

// loadEventCards 加载活动卡牌数据
func (s *MasterDataService) loadEventCards() error {
	return s.loadJSON("eventCards.json", &s.eventCards)
}

// loadEventDeckBonuses 加载活动加成数据
func (s *MasterDataService) loadEventDeckBonuses() error {
	return s.loadJSON("eventDeckBonuses.json", &s.eventDeckBonuses)
}

// loadGameCharacterUnits 加载角色组合数据
func (s *MasterDataService) loadGameCharacterUnits() error {
	return s.loadJSON("gameCharacterUnits.json", &s.gameCharacterUnits)
}

// loadCostume3ds 加载 3D 服装数据
func (s *MasterDataService) loadCostume3ds() error {
	return s.loadJSON("costume3ds.json", &s.costume3ds)
}

// loadCardCostume3ds 加载卡牌 3D 服装关联数据
func (s *MasterDataService) loadCardCostume3ds() error {
	return s.loadJSON("cardCostume3ds.json", &s.cardCostume3ds)
}

// loadCardSupplies 加载卡牌供给类型数据
func (s *MasterDataService) loadCardSupplies() error {
	return s.loadJSON("cardSupplies.json", &s.cardSupplies)
}

// loadWorldBlooms 加载 WL 章节
func (s *MasterDataService) loadWorldBlooms() error {
	return s.loadJSON("worldBlooms.json", &s.worldBlooms)
}

// loadEventMusics 加载活动-乐曲关联
func (s *MasterDataService) loadEventMusics() error {
	return s.loadJSON("eventMusics.json", &s.eventMusics)
}
