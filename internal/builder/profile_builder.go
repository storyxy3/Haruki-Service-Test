package builder

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"Haruki-Service-API/internal/model"
	"Haruki-Service-API/internal/service"
	"Haruki-Service-API/pkg/asset"
)

// ProfileBuilder 负责构建 ProfileRequest
type ProfileBuilder struct {
	source       service.ProfileDataSource
	assets       *asset.AssetHelper
	assetDir     string
	userData     *service.UserDataService
	honorBuilder *HonorBuilder
}

func NewProfileBuilder(source service.ProfileDataSource, a *asset.AssetHelper, d string, u *service.UserDataService) *ProfileBuilder {
	return &ProfileBuilder{
		source:       source,
		assets:       a,
		assetDir:     d,
		userData:     u,
		honorBuilder: NewHonorBuilder(source, a, d),
	}
}

// BuildProfileRequest 组装完整的 ProfileRequest
func (b *ProfileBuilder) BuildProfileRequest(region string) (model.ProfileRequest, error) {
	detail := b.userData.DetailedProfile(region)
	raw := b.userData.GetRawData()
	if detail == nil || raw == nil {
		return model.ProfileRequest{}, fmt.Errorf("user data not initialized")
	}

	// Frame Paths
	framePaths, hasFrame := b.buildFramePaths(raw.UserFrames)

	// 1. Basic Profile
	basic := model.BasicProfile{
		ID:              detail.ID,
		Region:          detail.Region,
		Nickname:        detail.Nickname,
		IsHideUID:       detail.IsHideUID,
		LeaderImagePath: detail.LeaderImagePath,
		HasFrame:        hasFrame,
	}

	// 2. PCards
	pcards := b.buildPCards(raw.UserCards, raw.UserDecks, raw.UserGamedata.Deck)

	// 3. Honors (获取名片装备的三个称号)
	honors := b.buildHonors(raw)

	// 4. Music Clear Counts
	musicCounts := b.buildMusicCounts(raw.UserMusicClear, raw.UserMusicStats)

	// 5. Character Ranks
	charaRanks := b.buildCharacterRanks(raw.UserCharacters)

	// 6. Solo Live Rank (取最高分的一个)
	soloLive := b.buildSoloLive(raw.UserChallengeLiveSoloResults, raw.UserChallengeLiveSoloStages)

	// 7. Word (清理 HTML 标签)
	word := b.cleanWord(raw.UserProfile.Word)

	// 8. Chara Rank Icon Path Map
	charaIconMap := b.buildCharaIconMap()

	req := model.ProfileRequest{
		Profile:              basic,
		Rank:                 raw.UserGamedata.Rank,
		TwitterID:            raw.UserProfile.TwitterID,
		Word:                 word,
		PCards:               pcards,
		Honors:               honors,
		MusicDifficultyCount: musicCounts,
		CharacterRank:        charaRanks,
		SoloLive:             soloLive,
		UpdateTime:           detail.UpdateTime,
		LvRankBgPath:         "user/lv_rank_bg.png",
		XIconPath:            "user/icon_twitter.png",
		IconClearPath:        "icon_clear.png",
		IconFcPath:           "icon_fc.png",
		IconApPath:           "icon_ap.png",
		CharaRankIconPathMap: charaIconMap,
		FramePaths:           framePaths,
		BgSettings: &model.ProfileBgSettings{
			Alpha:    100,
			Blur:     4,
			Vertical: false,
		},
	}

	return req, nil
}

func (b *ProfileBuilder) buildFramePaths(userFrames []service.RawUserFrame) (*model.PlayerFramePaths, bool) {
	var equippedID int
	for _, f := range userFrames {
		if f.PlayerFrameAttachStatus == "equipped" {
			equippedID = f.PlayerFrameID
			break
		}
	}
	if equippedID == 0 {
		return nil, false
	}

	frame, err := b.source.GetPlayerFrameByID(equippedID)
	if err != nil {
		return nil, false
	}

	frameGroup, err := b.source.GetPlayerFrameGroupByID(frame.PlayerFrameGroupID)
	if err != nil {
		return nil, false
	}

	assetName := frameGroup.AssetbundleName
	assetPath := fmt.Sprintf("player_frame/%s/%d", assetName, equippedID)

	return &model.PlayerFramePaths{
		Base:        fmt.Sprintf("%s/horizontal/frame_base.png", assetPath),
		CenterTop:   fmt.Sprintf("%s/vertical/frame_centertop.png", assetPath),
		LeftBottom:  fmt.Sprintf("%s/vertical/frame_leftbottom.png", assetPath),
		LeftTop:     fmt.Sprintf("%s/horizontal/frame_lefttop.png", assetPath),
		RightBottom: fmt.Sprintf("%s/horizontal/frame_rightbottom.png", assetPath),
		RightTop:    fmt.Sprintf("%s/horizontal/frame_righttop.png", assetPath),
	}, true
}

func (b *ProfileBuilder) buildPCards(userCards []service.RawUserCard, decks []service.RawUserDeck, activeDeckID int) []model.CardFullThumbnailRequest {
	var activeDeck service.RawUserDeck
	found := false
	for _, d := range decks {
		if d.DeckID == activeDeckID {
			activeDeck = d
			found = true
			break
		}
	}
	if !found && len(decks) > 0 {
		activeDeck = decks[0]
	}

	// PCards 只取前五个，或者按顺序取
	// lunabot 逻辑是取 member1 到 member5
	memberIDs := []int{activeDeck.Member1, activeDeck.Member2, activeDeck.Member3, activeDeck.Member4, activeDeck.Member5}

	var results []model.CardFullThumbnailRequest
	for _, cid := range memberIDs {
		if cid == 0 {
			continue
		}
		card, err := b.source.GetCardByID(cid)
		if err != nil {
			continue
		}

		// 查找玩家持有的这张卡的状态
		var userCard *service.RawUserCard
		for i := range userCards {
			if userCards[i].CardID == cid {
				userCard = &userCards[i]
				break
			}
		}

		isAfter := service.IsAfterTraining(userCard)
		displayAfterTraining := false
		if userCard != nil {
			displayAfterTraining = strings.EqualFold(userCard.DefaultImage, "special_training")
		}
		var level *int
		if userCard != nil {
			level = &userCard.Level
		}

		results = append(results, BuildCardThumbnail(b.assets, b.assetDir, card, ThumbnailOptions{
			AfterTraining: isAfter,
			TrainedArt:    displayAfterTraining,
			IsPcard:       true,
			Level:         level,
		}))
	}
	return results
}

func (b *ProfileBuilder) buildHonors(rawData *service.RawUserData) []model.HonorRequest {
	reqs := []model.HonorRequest{}
	if rawData == nil {
		return reqs
	}

	// 尝试从 UserProfileHonors 获取已装备的称号
	var selected []service.RawUserProfileHonor
	for _, ph := range rawData.UserProfileHonors {
		if ph.HonorID > 0 || ph.HonorId2 > 0 {
			selected = append(selected, ph)
		}
	}

	// 按 Seq 排序
	sort.Slice(selected, func(i, j int) bool {
		return selected[i].Seq < selected[j].Seq
	})

	for _, ph := range selected {
		rank := 0
		// 尝试获取活动排名（为了将来可能的活动记录功能保留基础设施）
		eventID := b.source.GetEventIDByHonorID(ph.HonorID)
		if eventID > 0 {
			for _, res := range rawData.UserEventResults {
				if res.EventID == eventID {
					rank = res.Rank
					break
				}
			}
		}

		query := model.HonorQuery{
			HonorID:          ph.HonorID,
			HonorLevel:       ph.HonorLevel,
			IsMain:           ph.Seq == 1,
			Rank:             rank,
			BondsHonorWordID: ph.BondsHonorWordId,
		}

		req, err := b.honorBuilder.BuildHonorRequest(query)
		if err == nil {
			reqs = append(reqs, req)
		}
	}

	// 兜底：如果没有已装备的称号且玩家拥有称号，取前三个（常见于部分 suite dump）
	if len(reqs) == 0 && len(rawData.UserHonors) > 0 {
		count := 0
		for _, h := range rawData.UserHonors {
			if count >= 3 {
				break
			}

			rank := 0
			eventID := b.source.GetEventIDByHonorID(h.HonorID)
			if eventID > 0 {
				for _, res := range rawData.UserEventResults {
					if res.EventID == eventID {
						rank = res.Rank
						break
					}
				}
			}

			query := model.HonorQuery{
				HonorID:    h.HonorID,
				HonorLevel: h.HonorLevel,
				IsMain:     count == 0,
				Rank:       rank,
			}
			req, err := b.honorBuilder.BuildHonorRequest(query)
			if err == nil {
				reqs = append(reqs, req)
				count++
			}
		}
	}

	return reqs
}

func (b *ProfileBuilder) buildMusicCounts(clears []service.RawMusicClear, stats []service.RawMusicResult) []model.MusicClearCount {
	diffs := []string{"easy", "normal", "hard", "expert", "master", "append"}
	results := make([]model.MusicClearCount, 0, len(diffs))

	// 如果有 pre-aggregated 的清图记录，直接使用
	if len(clears) > 0 {
		for _, d := range diffs {
			found := false
			for _, s := range clears {
				if strings.EqualFold(s.MusicDifficultyType, d) {
					results = append(results, model.MusicClearCount{
						Difficulty: d,
						Clear:      s.LiveClear,
						FC:         s.FullCombo,
						AP:         s.AllPerfect,
					})
					found = true
					break
				}
			}
			if !found {
				results = append(results, model.MusicClearCount{Difficulty: d})
			}
		}
		return results
	}

	// 否则从 raw stats 中聚合
	for _, d := range diffs {
		count := model.MusicClearCount{Difficulty: d}
		seen := make(map[int]bool) // 防止重复计算（如 solo 和 multi 都有记录）
		for _, s := range stats {
			if strings.EqualFold(s.MusicDifficultyType, d) {
				if seen[s.MusicID] {
					continue
				}
				seen[s.MusicID] = true
				count.Clear++
				if s.FullComboFlg {
					count.FC++
				}
				if s.FullPerfectFlg {
					count.AP++
				}
			}
		}
		results = append(results, count)
	}

	return results
}

func (b *ProfileBuilder) buildCharacterRanks(ranks []service.RawUserCharacter) []model.CharacterRank {
	var results []model.CharacterRank
	for _, r := range ranks {
		results = append(results, model.CharacterRank{
			CharacterID: r.CharacterID,
			Rank:        r.CharacterRank,
		})
	}
	return results
}

func (b *ProfileBuilder) buildSoloLive(results []service.RawChallengeLiveResult, stages []service.RawChallengeLiveStage) *model.SoloLiveRank {
	if len(results) == 0 {
		return nil
	}

	// 按分数排序取最高
	sort.Slice(results, func(i, j int) bool {
		return results[i].HighScore > results[j].HighScore
	})

	top := results[0]
	rank := 1
	for _, s := range stages {
		if s.CharacterID == top.CharacterID {
			if s.Rank > rank {
				rank = s.Rank
			}
		}
	}

	return &model.SoloLiveRank{
		CharacterID: top.CharacterID,
		Score:       top.HighScore,
		Rank:        rank,
	}
}

func (b *ProfileBuilder) cleanWord(word string) string {
	// 移除 <#......> 标签
	re := regexp.MustCompile(`<#.*?>`)
	return re.ReplaceAllString(word, "")
}

func (b *ProfileBuilder) buildCharaIconMap() map[int]string {
	res := make(map[int]string)
	for id, nick := range asset.CharacterIDToNickname {
		path := "chara_rank_icon/" + nick + ".png"
		res[id] = path
	}
	return res
}

// BuildDetailedProfileCardRequest 构建简易的详细个人信息请求（用于页眉）
func (b *ProfileBuilder) BuildDetailedProfileCardRequest(region string) (*model.DetailedProfileCardRequest, error) {
	detail := b.userData.DetailedProfile(region)
	raw := b.userData.GetRawData()
	if detail == nil || raw == nil {
		return nil, fmt.Errorf("user data not initialized")
	}

	frames, hasFrame := b.buildFramePaths(raw.UserFrames)
	var framePath *string
	if hasFrame {
		// 个人页头通常只需要 Base 路径或者由 Drawing API 自行根据 ID 拼接
		// 这里传 Base 路径作为兼容
		path := frames.Base
		framePath = &path
	}

	return &model.DetailedProfileCardRequest{
		ID:              detail.ID,
		Region:          detail.Region,
		Nickname:        detail.Nickname,
		Source:          "local(haruki)",
		Mode:            "latest",
		UpdateTime:      detail.UpdateTime,
		IsHideUID:       detail.IsHideUID,
		LeaderImagePath: detail.LeaderImagePath,
		HasFrame:        hasFrame,
		FramePath:       framePath,
	}, nil
}
