package service

import (
	"fmt"
	"Haruki-Service-API/pkg/masterdata"
	"regexp"
	"strconv"
	"strings"
)

// FormatSkillDescription 格式化技能描述
// 替换 {{id;op}} 占位符为实际数值
// cardCharID: 卡牌对应的角色ID，用于解析 {{0;c}}
func (s *MasterDataService) FormatSkillDescription(skill *masterdata.Skill, cardCharID int) string {
	desc := skill.Description
	re := regexp.MustCompile(`\{\{(.*?)\}\}`)

	return re.ReplaceAllStringFunc(desc, func(match string) string {
		// match: {{32;v}}
		content := match[2 : len(match)-2] // 32;v
		parts := strings.Split(content, ";")
		if len(parts) != 2 {
			return match
		}

		idStr := parts[0]
		op := parts[1]

		ids := []int{}
		for _, sub := range strings.Split(idStr, ",") {
			if id, err := strconv.Atoi(sub); err == nil {
				ids = append(ids, id)
			}
		}

		if len(ids) == 0 {
			return match
		}

		// Helper to get diverse values
		getValues := func(eff *masterdata.SkillEffect) []int {
			if len(eff.SkillEffectDetails) == 0 {
				return []int{0}
			}
			var vals []int
			for _, d := range eff.SkillEffectDetails {
				vals = append(vals, d.ActivateEffectValue)
			}
			return vals
		}

		// Helper to format values
		formatValues := func(vals []int) string {
			if len(vals) == 0 {
				return ""
			}
			allSame := true
			first := vals[0]
			for _, v := range vals {
				if v != first {
					allSame = false
					break
				}
			}
			if allSame {
				return fmt.Sprintf("%d", first)
			}
			strs := []string{}
			used := make(map[string]bool)
			for _, v := range vals {
				str := fmt.Sprintf("%d", v)
				if !used[str] {
					strs = append(strs, str)
					used[str] = true
				}
			}
			return strings.Join(strs, "/")
		}

		// Special case for 'c' (Character Name), no effect lookup needed
		if op == "c" {
			if char, ok := s.charByID[cardCharID]; ok {
				return char.FirstName + char.GivenName
			}
			return "???"
		}

		// Find effects
		var effects []*masterdata.SkillEffect
		for _, id := range ids {
			for i := range skill.SkillEffects {
				if skill.SkillEffects[i].ID == id {
					effects = append(effects, &skill.SkillEffects[i])
					break
				}
			}
		}
		if len(effects) != len(ids) {
			return "?"
		}

		// Single ID Logic
		if len(ids) == 1 {
			eff := effects[0]
			switch op {
			case "d": // Duration
				if len(eff.SkillEffectDetails) > 0 {
					return fmt.Sprintf("%.1f", float64(eff.SkillEffectDetails[0].ActivateEffectDuration))
				}
				return "0.0"
			case "v": // Value
				return formatValues(getValues(eff))
			case "e": // Enhance
				return fmt.Sprintf("%d", eff.SkillEnhance.ActivateEffectValue)
			case "m": // Full Unit Enhance
				enhance := eff.SkillEnhance.ActivateEffectValue
				vals := getValues(eff)
				for i := range vals {
					vals[i] += enhance * 5
				}
				return formatValues(vals)
			case "c": // Character Name
				if char, ok := s.charByID[cardCharID]; ok {
					return char.FirstName + char.GivenName
				}
				return "???"
			}
		} else if len(ids) == 2 {
			e1, e2 := effects[0], effects[1]
			vals1 := getValues(e1)
			vals2 := getValues(e2)

			switch op {
			case "v": // Sum
				var sumVals []int
				for i := 0; i < len(vals1) && i < len(vals2); i++ {
					sumVals = append(sumVals, vals1[i]+vals2[i])
				}
				return formatValues(sumVals)
			case "r": // Character Rank Bonus
				return "..." // Dynamic (Show ... for static view)
			case "s": // Total Rank Bonus
				return "..." // Dynamic (Show ... for static view)
			case "u": // Full Unit Max Enhance
				var sumVals []int
				for i := 0; i < len(vals1) && i < len(vals2); i++ {
					sumVals = append(sumVals, vals1[i]+vals2[i])
				}
				return formatValues(sumVals)
			case "o": // Full Unit Max Enhance + Normal Bonus
				getBestVals := func(e *masterdata.SkillEffect, baseVals []int) []int {
					if len(e.SkillEffectDetails) == 0 {
						return []int{0}
					}
					var res []int
					for i, d := range e.SkillEffectDetails {
						if d.ActivateEffectValue2 != nil {
							res = append(res, *d.ActivateEffectValue2)
						} else {
							if i < len(baseVals) {
								res = append(res, baseVals[i])
							} else {
								res = append(res, 0)
							}
						}
					}
					return res
				}

				v2_1 := getBestVals(e1, vals1)
				v2_2 := getBestVals(e2, vals2)

				var sumVals []int
				for i := 0; i < len(v2_1) && i < len(v2_2); i++ {
					sumVals = append(sumVals, v2_1[i]+v2_2[i])
				}
				return formatValues(sumVals)
			}
		}

		return match
	})
}
