package service

import (
	"fmt"
	"strconv"
	"strings"
)

// QueryType 定义查询类型
type QueryType int

const (
	QueryTypeUnknown QueryType = iota
	QueryTypeID                // 纯数字 ID
	QueryTypeSeq               // 昵称+序号 (mnr-1)
	QueryTypeFilter            // 筛选 (属性、技能等)
)

// CardQueryInfo 解析后的查询信息
type CardQueryInfo struct {
	Type        QueryType
	Value       int    // ID
	Sequence    int    // Sequence (negative)
	CharacterID int    // Character ID for filter/seq
	Rarity      string // Star count (rarity_3, rarity_4) - changed to string
	Attr        string // Attribute (cute, cool, etc)
	SkillType   string // Skill type (score_up, etc)
	SupplyType  string // Supply type (normal, limited, festival)
	Year        int    // Release year
	Original    string
}

// CardParser 卡牌查询解析器
type CardParser struct {
	extractor *Extractor
}

// NewCardParser 创建解析器
func NewCardParser(nicknames map[string]int) *CardParser {
	return &CardParser{
		extractor: NewExtractor(nicknames),
	}
}

// Parse 解析查询字符串
func (p *CardParser) Parse(args string) (*CardQueryInfo, error) {
	args = strings.TrimSpace(args)

	// 1. 优先检查昵称 + 倒数序号 (mnr-1)
	if info := p.tryParseNicknameSeq(args); info != nil {
		return info, nil
	}

	// 2. 检查纯数字 ID (190)
	if info := p.tryParseID(args); info != nil {
		return info, nil
	}

	// 3. 尝试解析为筛选条件 (Filter)
	if info := p.tryParseFilter(args); info != nil {
		return info, nil
	}

	// 4. 无法解析
	return nil, fmt.Errorf("无法解析的指令: %s", args)
}

// tryParseNicknameSeq 尝试解析昵称+序号格式 (例如: mnr-1)
func (p *CardParser) tryParseNicknameSeq(args string) *CardQueryInfo {
	// 使用 Extractor 提取角色
	res := p.extractor.ExtractCharacter(args)
	if !res.Found {
		return nil
	}

	// 剩余部分必须是负数序号 (Lunabot逻辑)
	// 例如 "mnr-1" -> res.Value=CID, res.Remaining="-1"
	remaining := strings.TrimSpace(res.Remaining)
	if strings.HasPrefix(remaining, "-") {
		numPart := remaining[1:]
		if isNumeric(numPart) {
			seq, _ := strconv.Atoi(remaining)
			return &CardQueryInfo{
				Type:        QueryTypeSeq,
				Sequence:    seq,
				CharacterID: res.Value,
				Original:    args,
			}
		}
	}
	return nil
}

// tryParseID 尝试解析 ID 格式 (例如: 190)
func (p *CardParser) tryParseID(args string) *CardQueryInfo {
	if isNumeric(args) {
		id, err := strconv.Atoi(args)
		if err == nil {
			return &CardQueryInfo{
				Type:     QueryTypeID,
				Value:    id,
				Original: args,
			}
		}
	}
	return nil
}

// tryParseFilter 尝试解析筛选条件 (例如: mnr 4star 分)
func (p *CardParser) tryParseFilter(args string) *CardQueryInfo {
	currentArgs := args
	info := &CardQueryInfo{
		Type:     QueryTypeFilter,
		Original: args,
	}
	matched := false

	// 1. 提取角色
	if res := p.extractor.ExtractCharacter(currentArgs); res.Found {
		info.CharacterID = res.Value
		currentArgs = res.Remaining
		matched = true
	}

	// 2. 提取稀有度
	if res := p.extractor.ExtractRarity(currentArgs); res.Found {
		info.Rarity = res.Value
		currentArgs = res.Remaining
		matched = true
	}

	// 3. 提取属性
	if res := p.extractor.ExtractAttribute(currentArgs); res.Found {
		info.Attr = res.Value
		currentArgs = res.Remaining
		matched = true
	}

	// 4. 提取技能
	if res := p.extractor.ExtractSkill(currentArgs); res.Found {
		info.SkillType = res.Value
		currentArgs = res.Remaining
		matched = true
	}

	// 5. 提取限定类型
	if res := p.extractor.ExtractSupply(currentArgs); res.Found {
		info.SupplyType = res.Value
		currentArgs = res.Remaining
		matched = true
	}

	// 6. 提取年份
	if res := p.extractor.ExtractYear(currentArgs); res.Found {
		info.Year = res.Value
		currentArgs = res.Remaining
		matched = true
	}

	if matched {
		return info
	}
	return nil
}
