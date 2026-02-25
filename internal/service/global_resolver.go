package service

import (
	"fmt"
	"regexp"
	"strings"
)

// TargetModule 指令目标模块
type TargetModule int

const (
	ModuleUnknown TargetModule = iota
	ModuleCard                 // 查卡/查牌
	ModuleMusic                // 查曲
	ModuleEvent                // 活动
	ModuleProfile              // 个人信息/SK
	ModuleHelp                 // 帮助
)

// ResolvedCommand 解析后的全局指令
type ResolvedCommand struct {
	Module    TargetModule
	Mode      string // 具体子模式 (例如 card-detail, card-list)
	Query     string // 剩余的查询文本
	Region    string // 强制指定的区服 (jp, en, cn)
	IsHelp    bool
	IsVerbose bool
	IsPreview bool
}

// GlobalCommandResolver 统一指令解析器
type GlobalCommandResolver struct {
	extractor *Extractor
	routes    []route
}

type route struct {
	pattern *regexp.Regexp
	module  TargetModule
	mode    string
}

func NewGlobalCommandResolver(nicknames map[string]int) *GlobalCommandResolver {
	r := &GlobalCommandResolver{
		extractor: NewExtractor(nicknames),
	}
	r.initRoutes()
	return r
}

func (r *GlobalCommandResolver) initRoutes() {
	// 定义正则路由表 (兼容大部分 lunabot 常用前缀)
	r.routes = []route{
		// 1. 卡牌类
		{regexp.MustCompile(`(?i)^/(卡面|详情|card-detail)\s*(.*)`), ModuleCard, "card-detail"},
		{regexp.MustCompile(`(?i)^/(查卡|查牌|卡片|card)\s*(.*)`), ModuleCard, "card-box"}, // 默认 Box，后面分支判断 ID
		{regexp.MustCompile(`(?i)^/(查框|box|card-box)\s*(.*)`), ModuleCard, "card-box"},
		{regexp.MustCompile(`(?i)^/(列表|卡池列表|gacha-list)\s*(.*)`), ModuleCard, "gacha-list"},
		// 2. 乐曲类
		{regexp.MustCompile(`(?i)^/(查曲|查歌|乐曲|music)\s*(.*)`), ModuleMusic, "music-detail"},
		{regexp.MustCompile(`(?i)^/(谱面预览|chart)\s*(.*)`), ModuleMusic, "music-chart"},
		// 3. 活动类
		{regexp.MustCompile(`(?i)^/(活动|event)\s*(.*)`), ModuleEvent, "event-detail"},
		{regexp.MustCompile(`(?i)^/(活动列表|event-list)\s*(.*)`), ModuleEvent, "event-list"},
		// 4. 个人信息/排名类 (SK)
		{regexp.MustCompile(`(?i)^/sk\s*(.*)`), ModuleProfile, "profile"},
		{regexp.MustCompile(`(?i)^/(个人中心|profile)\s*(.*)`), ModuleProfile, "profile"},
	}
}

// Resolve 解析原始字符串
func (r *GlobalCommandResolver) Resolve(input string) (*ResolvedCommand, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return &ResolvedCommand{Module: ModuleHelp, IsHelp: true}, nil
	}

	res := &ResolvedCommand{}

	// 1. 提取全局控制 Flag
	// 顺序：Region -> Verbose -> Preview -> Help
	regRes := r.extractor.ExtractRegion(input)
	res.Region = regRes.Value
	input = regRes.Remaining

	verbRes := r.extractor.ExtractVerbose(input)
	res.IsVerbose = verbRes.Value
	input = verbRes.Remaining

	preRes := r.extractor.ExtractPreview(input)
	res.IsPreview = preRes.Value
	input = preRes.Remaining

	helpRes := r.extractor.ExtractHelp(input)
	res.IsHelp = helpRes.Value
	input = helpRes.Remaining

	if res.IsHelp {
		res.Module = ModuleHelp
		return res, nil
	}

	// 2. 路由匹配
	for _, rt := range r.routes {
		if matches := rt.pattern.FindStringSubmatch(input); len(matches) > 1 {
			res.Module = rt.module
			res.Mode = rt.mode
			// 如果正则有第二个捕获组，说明是参数部分
			query := ""
			if len(matches) > 2 {
				query = strings.TrimSpace(matches[2])
			}
			res.Query = query

			// 智能区分逻辑：如果是卡牌模块且处于默认 Box 模式，但参数为纯数字，则切换为 Detail
			if res.Module == ModuleCard && res.Mode == "card-box" {
				idRes := r.extractor.ExtractID(query)
				if idRes.Found {
					res.Mode = "card-detail"
				}
			}

			return res, nil
		}
	}

	// 3. 兜底处理：如果没有任何前缀但包含特定关键词，或者纯数字
	// 这里可以添加更复杂的模糊识别，目前先报错
	return nil, fmt.Errorf("无法识别指令格式，请发送 /help 查看说明")
}
