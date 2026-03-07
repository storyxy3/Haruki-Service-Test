package service

import (
	"encoding/json"
)

func injectOmakaseMusicMeta(data []byte) []byte {
	var metas []map[string]interface{}
	if err := json.Unmarshal(data, &metas); err != nil {
		return data
	}

	omakaseMusicID := 10000
	diffs := []string{"master", "expert", "hard"}

	for _, m := range metas {
		if id, ok := m["music_id"].(float64); ok && int(id) == omakaseMusicID {
			return data
		}
	}

	// Average all master, expert, hard together
	omakaseScore := map[string]interface{}{
		"music_time":        0.0,
		"event_rate":        0.0,
		"base_score":        0.0,
		"base_score_auto":   0.0,
		"fever_score":       0.0,
		"fever_end_time":    0.0,
		"tap_count":         0.0,
		"skill_score_solo":  []float64{0, 0, 0, 0, 0, 0},
		"skill_score_auto":  []float64{0, 0, 0, 0, 0, 0},
		"skill_score_multi": []float64{0, 0, 0, 0, 0, 0},
	}
	count := 0.0

	for _, m := range metas {
		d, _ := m["difficulty"].(string)
		if d == "master" || d == "expert" || d == "hard" {
			count++
			if v, ok := m["music_time"].(float64); ok {
				omakaseScore["music_time"] = omakaseScore["music_time"].(float64) + v
			}
			if v, ok := m["event_rate"].(float64); ok {
				omakaseScore["event_rate"] = omakaseScore["event_rate"].(float64) + v
			}
			if v, ok := m["base_score"].(float64); ok {
				omakaseScore["base_score"] = omakaseScore["base_score"].(float64) + v
			}
			if v, ok := m["base_score_auto"].(float64); ok {
				omakaseScore["base_score_auto"] = omakaseScore["base_score_auto"].(float64) + v
			}
			if v, ok := m["fever_score"].(float64); ok {
				omakaseScore["fever_score"] = omakaseScore["fever_score"].(float64) + v
			}
			if v, ok := m["fever_end_time"].(float64); ok {
				omakaseScore["fever_end_time"] = omakaseScore["fever_end_time"].(float64) + v
			}
			if v, ok := m["tap_count"].(float64); ok {
				omakaseScore["tap_count"] = omakaseScore["tap_count"].(float64) + v
			}

			for _, key := range []string{"skill_score_solo", "skill_score_auto", "skill_score_multi"} {
				if arr, ok := m[key].([]interface{}); ok {
					baseArr := omakaseScore[key].([]float64)
					for i, val := range arr {
						if i < len(baseArr) {
							if v, ok := val.(float64); ok {
								baseArr[i] += v
							}
						}
					}
				}
			}
		}
	}

	if count > 0 {
		omakaseScore["music_time"] = omakaseScore["music_time"].(float64) / count
		omakaseScore["event_rate"] = float64(int(omakaseScore["event_rate"].(float64) / count))
		omakaseScore["base_score"] = omakaseScore["base_score"].(float64) / count
		omakaseScore["base_score_auto"] = omakaseScore["base_score_auto"].(float64) / count
		omakaseScore["fever_score"] = omakaseScore["fever_score"].(float64) / count
		omakaseScore["fever_end_time"] = omakaseScore["fever_end_time"].(float64) / count
		omakaseScore["tap_count"] = float64(int(omakaseScore["tap_count"].(float64) / count))

		for _, key := range []string{"skill_score_solo", "skill_score_auto", "skill_score_multi"} {
			baseArr := omakaseScore[key].([]float64)
			for i := range baseArr {
				baseArr[i] /= count
			}
		}

		// Inject for all diffs to satisfy C++ engine finding the specified diff
		for _, diff := range diffs {
			omakase := map[string]interface{}{
				"music_id":          float64(omakaseMusicID),
				"difficulty":        diff,
				"music_time":        omakaseScore["music_time"],
				"event_rate":        omakaseScore["event_rate"],
				"base_score":        omakaseScore["base_score"],
				"base_score_auto":   omakaseScore["base_score_auto"],
				"fever_score":       omakaseScore["fever_score"],
				"fever_end_time":    omakaseScore["fever_end_time"],
				"tap_count":         omakaseScore["tap_count"],
				"skill_score_solo":  omakaseScore["skill_score_solo"],
				"skill_score_auto":  omakaseScore["skill_score_auto"],
				"skill_score_multi": omakaseScore["skill_score_multi"],
			}
			metas = append(metas, omakase)
		}
	}

	newData, err := json.Marshal(metas)
	if err != nil {
		return data
	}
	return newData
}
