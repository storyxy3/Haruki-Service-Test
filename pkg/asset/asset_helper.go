package asset

import (
	"os"
	"path/filepath"
	"strings"
)

// AssetHelper 提供统一的资源路径拼接与存在性检查。
type AssetHelper struct {
	roots []string
}

// NewAssetHelper 创建资源解析器，primary 位于首位，legacy 为兜底目录。
func NewAssetHelper(primary string, legacy []string) *AssetHelper {
	var roots []string
	seen := make(map[string]struct{})
	appendRoot := func(path string) {
		clean := strings.TrimSpace(path)
		if clean == "" {
			return
		}
		clean = filepath.ToSlash(filepath.Clean(clean))
		if clean == "." {
			return
		}
		if _, ok := seen[clean]; ok {
			return
		}
		seen[clean] = struct{}{}
		roots = append(roots, clean)
	}
	appendRoot(primary)
	for _, dir := range legacy {
		appendRoot(dir)
	}
	if len(roots) == 0 {
		roots = []string{"."}
	}
	return &AssetHelper{roots: roots}
}

// Roots 返回所有资源根目录（拷贝，用于其它组件如 DrawingService）。
func (h *AssetHelper) Roots() []string {
	out := make([]string, len(h.roots))
	copy(out, h.roots)
	return out
}

// Primary 返回主目录。
func (h *AssetHelper) Primary() string {
	if len(h.roots) == 0 {
		return ""
	}
	return h.roots[0]
}

// Join 按主目录拼接路径（不检查存在性）。
func (h *AssetHelper) Join(parts ...string) string {
	if len(h.roots) == 0 {
		return ""
	}
	all := append([]string{h.Primary()}, parts...)
	return filepath.ToSlash(filepath.Join(all...))
}

// FirstExisting 在 roots 中寻找第一个存在的相对路径。
func (h *AssetHelper) FirstExisting(relPaths ...string) string {
	for _, rel := range relPaths {
		if strings.TrimSpace(rel) == "" {
			continue
		}
		for _, root := range h.roots {
			candidate := filepath.Join(root, rel)
			if _, err := os.Stat(candidate); err == nil {
				return filepath.ToSlash(candidate)
			}
		}
	}
	return ""
}

// ResolveAssetPath 从候选相对路径中寻找实际存在的资源，若找不到则退回主目录 + 第一个路径。
func ResolveAssetPath(helper *AssetHelper, assetDir string, relPaths ...string) string {
	if len(relPaths) == 0 {
		return ""
	}
	if helper != nil {
		if resolved := helper.FirstExisting(relPaths...); resolved != "" {
			return filepath.ToSlash(resolved)
		}
	}
	base := assetDir
	if base == "" && helper != nil {
		base = helper.Primary()
	}
	if base == "" {
		return filepath.ToSlash(relPaths[0])
	}
	return filepath.ToSlash(filepath.Join(base, relPaths[0]))
}

// CharacterIDToNickname 是用于图像提取时的特制代称字典
// 值对应 Z:\pjskdata\Data\chara_icon\ 目录下的实际文件名（不含 .png）
var CharacterIDToNickname = map[int]string{
	1:  "ick",   // 星乃一歌 Ichika
	2:  "saki",  // 天马咲希 Saki
	3:  "hnm",   // 望月穗波 Honami
	4:  "shiho", // 日野森志步 Shiho
	5:  "mnr",   // 花里实乃里 Minori
	6:  "hrk",   // 桐谷遥 Haruka
	7:  "airi",  // 桃井爱莉 Airi
	8:  "szk",   // 日野森雫 Shizuku
	9:  "khn",   // 小豆泽小春 Kohane
	10: "an",    // 白石杏 An
	11: "akt",   // 东云彰人 Akito
	12: "toya",  // 青柳冬弥 Toya
	13: "tks",   // 天马司 Tsukasa
	14: "emu",   // 凤笑梦 Emu
	15: "nene",  // 草薙宁宁 Nene
	16: "rui",   // 神代类 Rui
	17: "knd",   // 宵崎奏 Kanade
	18: "mfy",   // 朝比奈真冬 Mafuyu
	19: "ena",   // 东云绘名 Ena
	20: "mzk",   // 晓山瑞希 Mizuki
	21: "miku",  // 初音ミク Miku
	22: "rin",   // 镜音铃 Rin
	23: "len",   // 镜音连 Len
	24: "luka",  // 巡音流歌 Luka
	25: "meiko", // MEIKO
	26: "kaito", // KAITO
}

// MakeRelative 尝试将绝对路径转换为相对于 base 的路径。
// 如果 target 不在 base 下，则原样返回 target。
func MakeRelative(base, target string) string {
	if base == "" || target == "" {
		return target
	}
	cleanBase := filepath.ToSlash(filepath.Clean(base))
	cleanTarget := filepath.ToSlash(filepath.Clean(target))

	if strings.HasPrefix(cleanTarget, cleanBase) {
		rel := strings.TrimPrefix(cleanTarget, cleanBase)
		rel = strings.TrimPrefix(rel, "/")
		return rel
	}
	return cleanTarget
}
