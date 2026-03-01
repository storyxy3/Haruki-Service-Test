package service

import (
	"fmt"
	"strconv"
	"strings"
)

// EventQueryType 定义活动查询类型
type EventQueryType int

const (
	QueryTypeEventUnknown EventQueryType = iota
	QueryTypeEventID                     // 指定 ID: event123
	QueryTypeEventSeq                    // 索引: -1, 10, next, prev
	QueryTypeEventBan                    // Ban主: mnr1
	QueryTypeEventFilter                 // 筛选: 25h, wl
)

// EventFilter 活动筛选条件
type EventFilter struct {
	Unit        string // 25h, vbs, etc.
	EventType   string // marathon, cheerful_carnival, world_bloom
	Year        int    // 2024
	CharacterID int    // 筛选特定角色的活动
	Attr        string // cute, cool, etc.
}

// EventQueryInfo 解析后的活动查询信息
type EventQueryInfo struct {
	Type       EventQueryType
	EventID    int
	Index      int         // 正数或负数索引
	Keyword    string      // "next", "prev", "current"
	BanCharID  int         // Ban主角色ID
	BanSeq     int         // Ban主第几次箱活
	Filter     EventFilter // 筛选条件
	Original   string
	IsDetailed bool // 是否需要详细信息 (e.g. for single event query)
}

// EventParser 活动查询解析器
type EventParser struct {
	nicknames map[string]int
}

// CharacterIDByNickname resolves a character nickname to character id.
func (p *EventParser) CharacterIDByNickname(token string) (int, bool) {
	if p == nil {
		return 0, false
	}
	cid, ok := p.nicknames[strings.ToLower(strings.TrimSpace(token))]
	return cid, ok
}

// NewEventParser 创建解析器
func NewEventParser(nicknames map[string]int) *EventParser {
	return &EventParser{
		nicknames: nicknames,
	}
}

// Parse 解析查询字符串
func (p *EventParser) Parse(args string) (*EventQueryInfo, error) {
	args = strings.TrimSpace(args)

	// 1. 指定 ID (event123)
	if info := p.tryParseEventID(args); info != nil {
		return info, nil
	}

	// 2. Ban 主活动 (mnr1)
	if info := p.tryParseBanEvent(args); info != nil {
		return info, nil
	}

	// 3. 索引/相对位置 (-1, next)
	if info := p.tryParseEventSeq(args); info != nil {
		return info, nil
	}

	// 4. 筛选条件 (25h, wl, 2024)
	// 如果是空字符串，也可能被视为“仅列出”或“当前”，视上下文而定。
	// 这里我们假设空字符串由上层处理为默认（如当前列表）。
	if info := p.tryParseFilter(args); info != nil {
		return info, nil
	}

	return nil, fmt.Errorf("无法解析的活动指令: %s", args)
}

// tryParseEventID 解析 event123
func (p *EventParser) tryParseEventID(args string) *EventQueryInfo {
	lower := strings.ToLower(args)
	if strings.HasPrefix(lower, "event") {
		numPart := strings.TrimPrefix(lower, "event")
		id, err := strconv.Atoi(numPart)
		if err == nil {
			return &EventQueryInfo{
				Type:       QueryTypeEventID,
				EventID:    id,
				IsDetailed: true,
				Original:   args,
			}
		}
	}
	// 纯数字也可能被视为 ID? 根据 Python 逻辑，isdigit() -> ID
	if isNumeric(args) {
		id, _ := strconv.Atoi(args)
		return &EventQueryInfo{
			Type:       QueryTypeEventID,
			EventID:    id,
			IsDetailed: true,
			Original:   args,
		}
	}
	return nil
}

// tryParseBanEvent 解析 mnr1
func (p *EventParser) tryParseBanEvent(args string) *EventQueryInfo {
	lower := strings.ToLower(args)
	// 遍历所有昵称
	for nickname, cid := range p.nicknames {
		if strings.HasPrefix(lower, nickname) {
			suffix := strings.TrimPrefix(lower, nickname)
			// support "mnr1" or "mnr-1" ? Python logic: replace(nickname, "", 1) then int()
			// Check if suffix is a number
			if isNumeric(suffix) {
				seq, _ := strconv.Atoi(suffix)
				return &EventQueryInfo{
					Type:       QueryTypeEventBan,
					BanCharID:  cid,
					BanSeq:     seq,
					IsDetailed: true,
					Original:   args,
				}
			}
		}
	}
	return nil
}

// tryParseEventSeq 解析 -1, +1, next, prev
func (p *EventParser) tryParseEventSeq(args string) *EventQueryInfo {
	lower := strings.ToLower(args)
	if lower == "next" || lower == "下期" || lower == "下" {
		return &EventQueryInfo{Type: QueryTypeEventSeq, Keyword: "next", IsDetailed: true, Original: args}
	}
	if lower == "prev" || lower == "perv" || lower == "上期" || lower == "上" { // perv is a common typo
		return &EventQueryInfo{Type: QueryTypeEventSeq, Keyword: "prev", IsDetailed: true, Original: args}
	}
	if lower == "current" || lower == "curr" || lower == "当期" || lower == "今" {
		return &EventQueryInfo{Type: QueryTypeEventSeq, Keyword: "current", IsDetailed: true, Original: args}
	}

	// Negative index: -1, -2
	if strings.HasPrefix(args, "-") && isNumeric(args[1:]) {
		idx, _ := strconv.Atoi(args)
		return &EventQueryInfo{
			Type:       QueryTypeEventSeq,
			Index:      idx,
			IsDetailed: true,
			Original:   args,
		}
	}
	return nil
}

// tryParseFilter 解析筛选条件 (25h, wl, 2024 ...)
// 支持多个条件组合，空格分隔? Python 是空格分隔
func (p *EventParser) tryParseFilter(args string) *EventQueryInfo {
	parts := strings.Fields(strings.ToLower(args))
	if len(parts) == 0 {
		// Empty string -> show current list? Or handle specially.
		// For now return nil if empty, let caller handle empty args as "Show List"?
		// Actually if args is empty, Parser usually returns error or special "Empty" type.
		// Let's assume empty args handled before Parse, or Parse returns nil.
		return nil
	}

	filter := EventFilter{}
	matched := false

	// Helper map for units
	units := map[string]string{
		"l/n": "light_sound", "ln": "light_sound", "leoneed": "light_sound",
		"mmj": "idol", "moremorejump": "idol",
		"vbs": "street", "vividbadsquad": "street",
		"ws": "theme_park", "wxs": "theme_park", "wonderlands": "theme_park",
		"25h": "school_refusal", "niigo": "school_refusal", "25": "school_refusal",
		"vs": "piapro", "virtualsinger": "piapro",
		"mix": "blend", "混合": "blend",
	}

	// Helper map for event types
	types := map[string]string{
		"marathon": "marathon", "普活": "marathon", "马拉松": "marathon",
		"cheerful": "cheerful_carnival", "5v5": "cheerful_carnival", "carnival": "cheerful_carnival",
		"wl": "world_bloom", "worldlink": "world_bloom", "world": "world_bloom",
	}

	for _, part := range parts {
		partMatched := false

		// 1. Year (2020-2030, or 24年)
		if strings.HasSuffix(part, "年") {
			yStr := strings.TrimSuffix(part, "年")
			if isNumeric(yStr) {
				y, _ := strconv.Atoi(yStr)
				if y < 100 {
					y += 2000 // 24 -> 2024
				}
				filter.Year = y
				matched = true
				partMatched = true
			}
		} else if isNumeric(part) {
			y, _ := strconv.Atoi(part)
			if y > 2019 && y < 2030 {
				filter.Year = y
				matched = true
				partMatched = true
			}
		}

		if partMatched {
			continue
		}

		// 2. Unit
		if u, ok := units[part]; ok {
			filter.Unit = u
			matched = true
			continue
		}

		// 3. Type
		if t, ok := types[part]; ok {
			filter.EventType = t
			matched = true
			continue
		}

		// 4. Character
		for nick, cid := range p.nicknames {
			if part == nick {
				filter.CharacterID = cid
				matched = true
				partMatched = true
				break
			}
		}
		if partMatched {
			continue
		}

		// 5. Attribute
		attrAliases := map[string]string{
			"cute": "cute", "可爱": "cute", "粉": "cute",
			"cool": "cool", "帅气": "cool", "蓝": "cool",
			"pure": "pure", "纯真": "pure", "草": "pure", "绿": "pure",
			"happy": "happy", "快乐": "happy", "橙": "happy",
			"mysterious": "mysterious", "神秘": "mysterious", "紫": "mysterious",
		}
		for alias, attr := range attrAliases {
			if part == alias {
				filter.Attr = attr
				matched = true
				partMatched = true
				break
			}
		}
		if !partMatched {
			return nil // unrecognized token in filter list
		}
	}

	if matched {
		return &EventQueryInfo{
			Type:     QueryTypeEventFilter,
			Filter:   filter,
			Original: args,
		}
	}

	return nil
}

// isNumeric regex check or helper (reused from utils or redefined for regex specific needs)
// Assuming isNumeric exists in utils.go or define local one
// The one in utils.go uses Atoi check.
