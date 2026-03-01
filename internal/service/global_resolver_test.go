package service

import "testing"

func TestGlobalResolver_LunabotAliasRouting(t *testing.T) {
	r := NewGlobalCommandResolver(map[string]int{"mnr": 5})

	cases := []struct {
		cmd    string
		module TargetModule
		mode   string
	}{
		{"/卡牌一览 mnr", ModuleCard, "card-box"},
		{"/卡牌列表 mnr", ModuleCard, "card-list"},
		{"/查卡牌 mnr", ModuleCard, "card-list"},
		{"/pjsk card mnr", ModuleCard, "card-list"},
		{"/pjsk box mnr", ModuleCard, "card-box"},
		{"/歌曲一览 ma 32", ModuleMusic, "music-list"},
		{"/乐曲一览 ma 32", ModuleMusic, "music-list"},
		{"/查乐曲 ma 32", ModuleMusic, "music-list"},
		{"/打歌进度 ma", ModuleMusic, "music-progress"},
		{"/pjsk进度 ma", ModuleMusic, "music-progress"},
		{"/查谱图 1 ma", ModuleMusic, "music-chart"},
		{"/查音乐 1", ModuleMusic, "music-detail"},
		{"/查询乐曲 1", ModuleMusic, "music-detail"},
		{"/活动一览 wl", ModuleEvent, "event-list"},
		{"/查活动列表 wl", ModuleEvent, "event-list"},
		{"/查活动 17", ModuleEvent, "event-detail"},
		{"/活动组卡", ModuleDeck, "deck-event"},
		{"/组卡 17", ModuleDeck, "deck-event"},
		{"/挑战组卡 miku", ModuleDeck, "deck-challenge"},
		{"/最强卡组", ModuleDeck, "deck-no-event"},
		{"/控分组卡 120", ModuleDeck, "deck-bonus"},
		{"/ms组卡", ModuleDeck, "deck-mysekai"},
		{"/个人信息", ModuleProfile, "profile"},
		{"/名片", ModuleProfile, "profile"},
		{"/pjsk profile", ModuleProfile, "profile"},
		{"/卡池一览", ModuleGacha, "gacha"},
		{"/抽卡 17", ModuleGacha, "gacha"},
		{"/pjsk gacha 17", ModuleGacha, "gacha"},
		{"/sk-line", ModuleSK, "sk-line"},
		{"/mysekai-resource", ModuleMysekai, "mysekai-resource"},
	}

	for _, tc := range cases {
		res, err := r.Resolve(tc.cmd)
		if err != nil {
			t.Fatalf("Resolve(%q) error: %v", tc.cmd, err)
		}
		if res.Module != tc.module || res.Mode != tc.mode {
			t.Fatalf("Resolve(%q) => module=%v mode=%s, want module=%v mode=%s", tc.cmd, res.Module, res.Mode, tc.module, tc.mode)
		}
	}
}
