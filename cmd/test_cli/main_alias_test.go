package main

import "testing"

func TestParseMusicListCommand_Aliases(t *testing.T) {
	query, err := parseMusicListCommand("/歌曲列表 ma 32")
	if err != nil {
		t.Fatalf("parseMusicListCommand failed: %v", err)
	}
	if query.Difficulty != "master" {
		t.Fatalf("expected master difficulty, got %s", query.Difficulty)
	}
	if query.Level != 32 {
		t.Fatalf("expected level 32, got %d", query.Level)
	}
}

func TestParseMusicProgressCommand_Aliases(t *testing.T) {
	query, err := parseMusicProgressCommand("/打歌进度 ex")
	if err != nil {
		t.Fatalf("parseMusicProgressCommand failed: %v", err)
	}
	if query.Difficulty != "expert" {
		t.Fatalf("expected expert difficulty, got %s", query.Difficulty)
	}
}

func TestParseMusicChartCommand_Aliases(t *testing.T) {
	query, err := parseMusicChartCommand("/chart 39 ma")
	if err != nil {
		t.Fatalf("parseMusicChartCommand failed: %v", err)
	}
	if query.Query != "39" {
		t.Fatalf("expected query=39, got %q", query.Query)
	}
	if query.Difficulty != "master" {
		t.Fatalf("expected master difficulty, got %s", query.Difficulty)
	}
}

func TestParseGachaAliases(t *testing.T) {
	listQuery := parseGachaListCommand("/查卡池 p2")
	if listQuery.Page != 2 {
		t.Fatalf("expected page=2, got %d", listQuery.Page)
	}
	if listQuery.Keyword != "" {
		t.Fatalf("expected empty keyword after alias strip, got %q", listQuery.Keyword)
	}

	detailQuery, err := parseGachaDetailCommand("/抽卡 17")
	if err != nil {
		t.Fatalf("parseGachaDetailCommand failed: %v", err)
	}
	if detailQuery.GachaID != 17 {
		t.Fatalf("expected gacha id 17, got %d", detailQuery.GachaID)
	}
}
