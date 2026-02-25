package builder

import (
	"fmt"
	"strconv"
	"strings"

	"Haruki-Service-API/internal/model"
	"Haruki-Service-API/internal/service"
	"Haruki-Service-API/pkg/asset"
	"Haruki-Service-API/pkg/masterdata"
)

// HonorBuilder 负责构建 HonorRequest
type HonorBuilder struct {
	masterdata *service.MasterDataService
	assets     *asset.AssetHelper
	assetDir   string
}

func NewHonorBuilder(m *service.MasterDataService, a *asset.AssetHelper, d string) *HonorBuilder {
	return &HonorBuilder{
		masterdata: m,
		assets:     a,
		assetDir:   d,
	}
}

func (b *HonorBuilder) BuildHonorRequest(query model.HonorQuery) (model.HonorRequest, error) {
	req := model.HonorRequest{
		IsMainHonor: query.IsMain,
	}

	_, errNormal := b.masterdata.GetHonorByID(query.HonorID)
	bondsHonor, errBonds := b.masterdata.GetBondsHonorByID(query.HonorID)

	isNormal := errNormal == nil
	isBonds := errBonds == nil

	if !isNormal && !isBonds {
		return req, fmt.Errorf("honor %d not found in any masterdata table", query.HonorID)
	}

	if isNormal {
		if err := b.buildNormalHonorRequest(&req, query.HonorID, query.HonorLevel); err != nil {
			return req, err
		}
	} else if isBonds {
		if err := b.buildBondsHonorRequest(&req, bondsHonor, query.HonorLevel); err != nil {
			return req, err
		}
	}

	if query.Rank > 0 {
		rankStr := fmt.Sprintf("%d", query.Rank)
		req.FcOrApLevel = &rankStr
	}

	return req, nil
}

func (b *HonorBuilder) buildNormalHonorRequest(req *model.HonorRequest, honorID int, honorLevel int) error {
	honor, _ := b.masterdata.GetHonorByID(honorID)
	group, err := b.masterdata.GetHonorGroupByID(honor.GroupID)
	if err != nil {
		return fmt.Errorf("honor group %d not found for honor %d: %w", honor.GroupID, honorID, err)
	}

	assetName := honor.AssetbundleName
	rarity := honor.HonorRarity
	for _, lv := range honor.Levels {
		if lv.Level == honorLevel {
			if lv.AssetbundleName != "" {
				assetName = lv.AssetbundleName
			}
			if lv.HonorRarity != "" {
				rarity = lv.HonorRarity
			}
			break
		}
	}

	req.HonorLevel = &honorLevel
	htype := "normal"
	req.HonorType = &htype

	gtype := group.HonorType
	if gtype == "world_link" {
		gtype = "wl_event"
	}
	req.GroupType = &gtype
	req.HonorRarity = &rarity

	bgAssetName := assetName
	if group.BackgroundAssetbundleName != nil && *group.BackgroundAssetbundleName != "" {
		bgAssetName = *group.BackgroundAssetbundleName
	}

	ms := "sub"
	if req.IsMainHonor {
		ms = "main"
	}

	var honorImgPath string
	var rankImgPath string

	if group.BackgroundAssetbundleName != nil && *group.BackgroundAssetbundleName != "" {
		honorImgPath = fmt.Sprintf("honor/%s/degree_%s.png", *group.BackgroundAssetbundleName, ms)
	} else if gtype == "rank_match" {
		honorImgPath = fmt.Sprintf("rank_live/honor/%s/degree_%s.png", bgAssetName, ms)
	} else {
		honorImgPath = fmt.Sprintf("honor/%s/degree_%s.png", assetName, ms)
	}
	req.HonorImgPath = &honorImgPath

	// 只有活动和排位有额外的 Rank 图层
	if assetName != "" && (gtype == "event" || gtype == "wl_event" || gtype == "rank_match") {
		switch gtype {
		case "rank_match":
			tmp := fmt.Sprintf("rank_live/honor/%s/%s.png", assetName, ms)
			rankImgPath = tmp
			req.RankImgPath = &rankImgPath
		case "event", "wl_event":
			// 对于 WL，如果背景就是 assetName，则可能不需要 rank_img，或者 rank_img 是 degree_main
			// 这种情况下 lunabot 会尝试加载 rank_main.png
			rel1 := fmt.Sprintf("honor/%s/rank_%s.png", assetName, ms)
			rel2 := fmt.Sprintf("honor/%s/degree_%s.png", assetName, ms)

			if found := b.assets.FirstExisting(rel1); found != "" {
				rankImgPath = rel1
			} else if honorImgPath != rel2 { // 避免 rank_img 和 background 重复
				if found := b.assets.FirstExisting(rel2); found != "" {
					rankImgPath = rel2
				}
			}

			if rankImgPath != "" {
				req.RankImgPath = &rankImgPath
			}
		}
	}

	var frameNameStr string
	if group.FrameName != nil {
		frameNameStr = *group.FrameName
	}

	rareMap := map[string]int{"low": 1, "middle": 2, "high": 3, "highest": 4}
	r := 1
	if val, ok := rareMap[rarity]; ok {
		r = val
	}

	if frameNameStr != "" {
		framePath := fmt.Sprintf("honor_frame/%s/frame_degree_%s_%d.png", frameNameStr, string(ms[0]), r)
		req.FrameImgPath = &framePath
	} else {
		framePath := fmt.Sprintf("honor/frame_degree_%s_%d.png", string(ms[0]), r)
		req.FrameImgPath = &framePath
	}

	diffScoreMap := map[int]struct {
		diff  string
		score string
	}{
		3009: {diff: "easy", score: "fullCombo"},
		3010: {diff: "normal", score: "fullCombo"},
		3011: {diff: "hard", score: "fullCombo"},
		3012: {diff: "expert", score: "fullCombo"},
		3013: {diff: "master", score: "fullCombo"},
		3014: {diff: "master", score: "allPerfect"},
		4700: {diff: "append", score: "fullCombo"},
		4701: {diff: "append", score: "allPerfect"},
	}
	if _, ok := diffScoreMap[honorID]; ok || gtype == "event" || gtype == "wl_event" {
		if ok := diffScoreMap[honorID]; ok != (struct {
			diff  string
			score string
		}{}) {
			groupFcAp := "fc_ap"
			req.GroupType = &groupFcAp
		}

		scrollPath := fmt.Sprintf("honor/%s/scroll.png", assetName)
		if found := b.assets.FirstExisting(scrollPath); found != "" {
			req.ScrollImgPath = &scrollPath
		}

		fcapLv := strconv.Itoa(honorLevel)
		req.FcOrApLevel = &fcapLv
	}

	if gtype == "character" || gtype == "achievement" || strings.HasPrefix(*req.GroupType, "fc_ap") {
		lvImg := "honor/icon_degreeLv.png"
		lv6Img := "honor/icon_degreeLv6.png"
		req.LvImgPath = &lvImg
		req.Lv6ImgPath = &lv6Img
	}

	return nil
}

func (b *HonorBuilder) buildBondsHonorRequest(req *model.HonorRequest, honor *masterdata.BondsHonor, honorLevel int) error {
	htype := "bonds"
	req.HonorType = &htype
	rarity := honor.HonorRarity
	req.HonorRarity = &rarity
	req.HonorLevel = &honorLevel

	ms := "sub"
	if req.IsMainHonor {
		ms = "main"
	}

	cuid1 := honor.GameCharacterUnitId1
	cuid2 := honor.GameCharacterUnitId2

	suffix := "_sub"
	if req.IsMainHonor {
		suffix = ""
	}

	var cid1, cid2 int
	if unit1, ok := b.masterdata.GetGameCharacterUnitByID(cuid1); ok {
		cid1 = unit1.GameCharacterID
	}
	if unit2, ok := b.masterdata.GetGameCharacterUnitByID(cuid2); ok {
		cid2 = unit2.GameCharacterID
	}

	bgPath1 := fmt.Sprintf("honor/bonds/%d%s.png", cid1, suffix)
	bgPath2 := fmt.Sprintf("honor/bonds/%d%s.png", cid2, suffix)
	req.BondsBgPath = &bgPath1
	req.BondsBgPath2 = &bgPath2

	charaPath1 := fmt.Sprintf("bonds_honor/character/chr_sd_%02d_01.png", cuid1)
	charaPath2 := fmt.Sprintf("bonds_honor/character/chr_sd_%02d_01.png", cuid2)
	req.CharaIconPath = &charaPath1
	req.CharaIconPath2 = &charaPath2

	cuid1Str := strconv.Itoa(cuid1)
	cuid2Str := strconv.Itoa(cuid2)
	req.CharaID = &cuid1Str
	req.CharaID2 = &cuid2Str

	maskPath := fmt.Sprintf("honor/mask_degree_%s.png", ms)
	req.MaskImgPath = &maskPath

	rareMap := map[string]int{"low": 1, "middle": 2, "high": 3, "highest": 4}
	r := 1
	if val, ok := rareMap[rarity]; ok {
		r = val
	}
	framePath := fmt.Sprintf("honor/frame_degree_%s_%d.png", string(ms[0]), r)
	req.FrameImgPath = &framePath

	if req.IsMainHonor {
		hwid := honor.ID
		var wordbundlename string
		if hwid%100 < 50 {
			wordbundlename = fmt.Sprintf("honorname_%02d%02d_%02d_01", cid1, cid2, hwid%100)
		} else {
			wordbundlename = fmt.Sprintf("honorname_%02d%02d_default_%02d%02d_01", cid1, cid2, cuid1, cuid2)
		}
		wordPath := fmt.Sprintf("bonds_honor/word/%s.png", wordbundlename)
		req.WordImgPath = &wordPath
	}

	lvImg := "honor/icon_degreeLv.png"
	lv6Img := "honor/icon_degreeLv6.png"
	req.LvImgPath = &lvImg
	req.Lv6ImgPath = &lv6Img

	return nil
}
