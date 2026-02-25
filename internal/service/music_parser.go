package service

import (
	"fmt"
	"strconv"
	"strings"
)

// MusicQueryType 音乐查询类型
type MusicQueryType int

const (
	QueryTypeMusicUnknown MusicQueryType = iota
	QueryTypeMusicID                     // 指定 ID: id123, 123
	QueryTypeMusicSeq                    // 索引: -1, -5
	QueryTypeMusicEvent                  // 活动ID: event123
	QueryTypeMusicBan                    // Ban主: ick1
	QueryTypeMusicTitle                  // 标题/关键词/别名
	QueryTypeMusicChart                  // 查谱面
)

// MusicQueryInfo 解析后的音乐查询信息
type MusicQueryInfo struct {
	Type       MusicQueryType
	Value      int    // ID, Index, EventID
	Diff       string // easy, normal, hard, expert, master, append
	Difficulty string // Alias for Diff
	MusicID    int    // Specific Music ID (if resolved)
	Keyword    string // Title, Alias
	BanCharID  int
	BanSeq     int
	Original   string
}

// MusicParser 音乐查询解析器
type MusicParser struct {
	md *MasterDataService
}

// NewMusicParser 创建解析器
func NewMusicParser(md *MasterDataService) *MusicParser {
	return &MusicParser{
		md: md,
	}
}

// Parse 解析查询字符串 (通用)
func (p *MusicParser) Parse(args string) (*MusicQueryInfo, error) {
	args = strings.TrimSpace(args)

	// Extract Difficulty first
	diff, cleanArgs := p.extractDiff(args)

	// 1. 指定 ID (id123)
	if info := p.tryParseID(cleanArgs); info != nil {
		info.Diff = diff
		info.Difficulty = diff
		info.Original = args
		return info, nil
	}

	// 2. 索引 (-1)
	if info := p.tryParseSeq(cleanArgs); info != nil {
		info.Diff = diff
		info.Difficulty = diff
		info.Original = args
		return info, nil
	}

	// 3. 活动 (event123)
	if info := p.tryParseEvent(cleanArgs); info != nil {
		info.Diff = diff
		info.Difficulty = diff
		info.Original = args
		return info, nil
	}

	// 4. Ban主 (ick1)
	if info := p.tryParseBan(cleanArgs); info != nil {
		info.Diff = diff
		info.Difficulty = diff
		info.Original = args
		return info, nil
	}

	// 5. Title/Keyword (Fallback)
	if cleanArgs != "" {
		return &MusicQueryInfo{
			Type:       QueryTypeMusicTitle,
			Keyword:    cleanArgs,
			Diff:       diff,
			Difficulty: diff,
			Original:   args,
		}, nil
	}

	return nil, fmt.Errorf("无法解析的音乐指令: %s", args)
}

// ParseDetail 解析查曲指令
func (p *MusicParser) ParseDetail(args string) (*MusicQueryInfo, error) {
	return p.Parse(args)
}

// ParseChart 解析查谱面指令
func (p *MusicParser) ParseChart(args string) (*MusicQueryInfo, error) {
	info, err := p.Parse(args)
	if err != nil {
		return nil, err
	}

	// 如果没有提取到难度，默认 master
	if info.Diff == "" {
		info.Diff = "master"
		info.Difficulty = "master"
	}

	// 如果解析出的是关键词，需要先搜索出 MusicID
	switch info.Type {
	case QueryTypeMusicTitle:
		music, err := p.md.SearchMusic(info.Keyword)
		if err != nil {
			return nil, err
		}
		info.MusicID = music.ID
	case QueryTypeMusicID:
		info.MusicID = info.Value
	default:
		// Seq, Event, Ban logic needs resolving context or complex search
		// For simplicity now, return error if not ID/Title
		return nil, fmt.Errorf("chart search only supports ID or Title for now")
	}

	info.Type = QueryTypeMusicChart
	return info, nil
}

// extractDiff 提取难度并返回剩余字符串
func (p *MusicParser) extractDiff(args string) (string, string) {
	// Standard diff names
	diffs := []string{"easy", "normal", "hard", "expert", "master", "append"}
	// Also support aliases
	aliases := map[string]string{
		"ez": "easy", "nm": "normal", "hd": "hard",
		"ex": "expert", "exp": "expert", "爷": "expert",
		"ma": "master", "mas": "master", "红": "master", "紫": "master",
		"apd": "append",
	}

	lower := strings.ToLower(args)
	parts := strings.Fields(lower)

	foundDiff := ""
	var remainingParts []string

	for _, part := range parts {
		isDiff := false
		// Check standard names
		for _, d := range diffs {
			if part == d {
				foundDiff = d
				isDiff = true
				break
			}
		}
		if !isDiff {
			// Check aliases
			if d, ok := aliases[part]; ok {
				foundDiff = d
				isDiff = true
			}
		}

		if !isDiff {
			// Original case handling logic is tricky here because we split `lower`
			// But for simple keyword search, lowercased parts joined back is acceptable.
			remainingParts = append(remainingParts, part)
		}
	}

	if foundDiff != "" {
		return foundDiff, strings.Join(remainingParts, " ")
	}

	return "", args
}

func (p *MusicParser) tryParseID(args string) *MusicQueryInfo {
	if strings.HasPrefix(args, "id") {
		num := strings.TrimPrefix(args, "id")
		if isNumeric(num) {
			id, _ := strconv.Atoi(num)
			return &MusicQueryInfo{Type: QueryTypeMusicID, Value: id}
		}
	}
	if isNumeric(args) {
		id, _ := strconv.Atoi(args)
		return &MusicQueryInfo{Type: QueryTypeMusicID, Value: id}
	}
	return nil
}

func (p *MusicParser) tryParseSeq(args string) *MusicQueryInfo {
	if strings.HasPrefix(args, "-") && isNumeric(args[1:]) {
		idx, _ := strconv.Atoi(args)
		return &MusicQueryInfo{Type: QueryTypeMusicSeq, Value: idx}
	}
	return nil
}

func (p *MusicParser) tryParseEvent(args string) *MusicQueryInfo {
	if strings.HasPrefix(args, "event") {
		num := strings.TrimPrefix(args, "event")
		if isNumeric(num) {
			id, _ := strconv.Atoi(num)
			return &MusicQueryInfo{Type: QueryTypeMusicEvent, Value: id}
		}
	}
	return nil
}

func (p *MusicParser) tryParseBan(args string) *MusicQueryInfo {
	// Simple ban logic: requires nickname map
	// Since we passed md (MasterDataService) to NewMusicParser, we can access nicknames
	// But Parser struct defined `nicknames map`?
	// Let's use `p.md.GetNicknames()` if we store md, OR use the map passed in NewMusicParser.
	// NewMusicParser signature was: func NewMusicParser(md *MasterDataService)
	// So we should use p.md.GetNicknames().

	nicks := p.md.GetNicknames()
	for nick, cid := range nicks {
		if strings.HasPrefix(args, nick) {
			suffix := strings.TrimPrefix(args, nick)
			if isNumeric(suffix) {
				seq, _ := strconv.Atoi(suffix)
				return &MusicQueryInfo{
					Type:      QueryTypeMusicBan,
					BanCharID: cid,
					BanSeq:    seq,
				}
			}
		}
	}
	return nil
}
