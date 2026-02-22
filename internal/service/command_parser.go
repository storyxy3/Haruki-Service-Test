package service

import (
	"fmt"
	"strconv"
	"strings"
)

// CommandType 定义指令类型
type CommandType int

const (
	CmdTypeUnknown CommandType = iota

	// 查询类 (SK)
	CmdTypeEventQuerySelf      // 查自己 (sk)
	CmdTypeEventQueryAt        // 查别人 (sk @123)
	CmdTypeEventQueryUID       // 查指定UID (sk 350...)
	CmdTypeEventQueryRank      // 查指定排名 (sk 100)
	CmdTypeEventQueryRankRange // 查排名范围 (sk 100-200)
	CmdTypeEventQueryMultiRank // 查多个排名 (sk 1 2 3)

	// 操作类 (Bind)
	CmdTypeBind   // 绑定 (bind 350...)
	CmdTypeUnbind // 解绑 (unbind)
)

// EventCommand 是提供给数据库开发者的接口结构体
// 数据库开发者拿到这个结构体后，根据 Type 字段决定执行什么 SQL
type EventCommand struct {
	Type      CommandType
	TargetID  string // QQ ID (@12345) 或 Game UID (350...)
	Param1    int    // Rank Start, or Single Rank
	Param2    int    // Rank End
	MultiArgs []int  // Multiple Ranks
	Original  string // 原始指令
}

// CommandParser 负责解析数据库相关指令
type CommandParser struct{}

func NewCommandParser() *CommandParser {
	return &CommandParser{}
}

// Parse 解析指令字符串
func (p *CommandParser) Parse(args string) (*EventCommand, error) {
	args = strings.TrimSpace(args)
	cmd := &EventCommand{Original: args}

	// 1. 处理空指令 -> 查自己
	if args == "" {
		cmd.Type = CmdTypeEventQuerySelf
		return cmd, nil
	}

	// 2. 处理 Bind/Unbind (简单的关键词匹配)
	if strings.HasPrefix(args, "bind ") {
		cmd.Type = CmdTypeBind
		cmd.TargetID = strings.TrimSpace(strings.TrimPrefix(args, "bind "))
		// 简单的 UID 校验 (根据 profile.py: 13-20 digits)
		if !isNumeric(cmd.TargetID) || len(cmd.TargetID) < 10 {
			return nil, fmt.Errorf("无效的游戏ID: %s", cmd.TargetID)
		}
		return cmd, nil
	}
	if args == "unbind" {
		cmd.Type = CmdTypeUnbind
		return cmd, nil
	}

	// 3. 处理 SK 查询指令
	// 3.1 处理 @ (Lunabot 格式: [CQ:at,qq=123]) 或者纯数字QQ号
	// 这里假设传入的是已经提取过的纯文本或者模拟格式，具体取决于 Bot 框架。
	// 假设输入的 args 已经是纯文本参数。

	// 如果包含 @ (模拟)
	if strings.HasPrefix(args, "@") {
		cmd.Type = CmdTypeEventQueryAt
		cmd.TargetID = strings.TrimPrefix(args, "@")
		if !isNumeric(cmd.TargetID) {
			return nil, fmt.Errorf("无效的用户ID: %s", cmd.TargetID)
		}
		return cmd, nil
	}

	// 3.2 处理 Range (100-200)
	if strings.Contains(args, "-") && !strings.HasPrefix(args, "-") { // 排除负数 (mnr-1 是 Parser 的事，这里假设 SK 指令不混用)
		parts := strings.Split(args, "-")
		if len(parts) == 2 {
			start, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
			end, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))
			if err1 == nil && err2 == nil {
				if start > end {
					return nil, fmt.Errorf("起始排名不能大于结束排名")
				}
				cmd.Type = CmdTypeEventQueryRankRange
				cmd.Param1 = start
				cmd.Param2 = end
				return cmd, nil
			}
		}
	}

	// 3.3 处理多个排名 (100 200)
	fields := strings.Fields(args)
	if len(fields) > 1 {
		var ranks []int
		for _, f := range fields {
			if r, err := strconv.Atoi(f); err == nil {
				ranks = append(ranks, r)
			} else {
				// 混入了非数字，可能是不支持的格式
				return nil, fmt.Errorf("无法解析的排名参数: %s", f)
			}
		}
		cmd.Type = CmdTypeEventQueryMultiRank
		cmd.MultiArgs = ranks
		return cmd, nil
	}

	// 3.4 处理单个数字 (可能是 Rank 或 UID)
	if isNumeric(args) {
		val, _ := strconv.Atoi(args) // int range limit?
		// Python logic doesn't explicitly distinguish by value size for "sk <num>",
		// but typically huge numbers are UIDs.
		// profile.py: validate_uid 13-20 digits.
		if len(args) >= 10 {
			cmd.Type = CmdTypeEventQueryUID
			cmd.TargetID = args
		} else {
			cmd.Type = CmdTypeEventQueryRank
			cmd.Param1 = val
		}
		return cmd, nil
	}

	return nil, fmt.Errorf("无法识别的指令格式: %s", args)
}
