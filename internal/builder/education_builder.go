package builder

import (
	"fmt"
	"path/filepath"
	"strings"

	"Haruki-Service-API/internal/model"
	"Haruki-Service-API/internal/service"
	"Haruki-Service-API/pkg/asset"
)

type EducationBuilder struct {
	masterdata *service.MasterDataService
	userData   *service.UserDataService
	assets     *asset.AssetHelper
	assetDir   string
}

func NewEducationBuilder(masterdata *service.MasterDataService, userData *service.UserDataService, assets *asset.AssetHelper, assetDir string) *EducationBuilder {
	return &EducationBuilder{
		masterdata: masterdata,
		userData:   userData,
		assets:     assets,
		assetDir:   assetDir,
	}
}

// BuildChallengeLiveRequest assembles challenge-live detail payload from user data.
func (b *EducationBuilder) BuildChallengeLiveRequest(region string) (*model.ChallengeLiveDetailsRequest, error) {
	if b.userData == nil {
		return nil, fmt.Errorf("user data is not configured, please provide suite dump")
	}
	challenge := b.userData.ChallengeLive()
	if challenge == nil {
		return nil, fmt.Errorf("user data missing challenge live entries")
	}
	profile := b.userData.DetailedProfile(region)
	if profile == nil {
		return nil, fmt.Errorf("user data missing base profile information")
	}
	scoreByCID := make(map[int]int, len(challenge.Results))
	for _, res := range challenge.Results {
		scoreByCID[res.CharacterID] = res.HighScore
	}
	rankByCID := make(map[int]int, len(challenge.Stages))
	for _, st := range challenge.Stages {
		if st.Rank > rankByCID[st.CharacterID] {
			rankByCID[st.CharacterID] = st.Rank
		}
	}
	claimed := make(map[int]struct{}, len(challenge.Rewards))
	for _, r := range challenge.Rewards {
		claimed[r.RewardID] = struct{}{}
	}
	maxScore := 0
	var challenges []model.CharacterChallengeInfo
	for cid := 1; cid <= 26; cid++ {
		score := scoreByCID[cid]
		if score > maxScore {
			maxScore = score
		}
		rank := rankByCID[cid]
		jewel, shard := b.pickChallengeRewards(cid, claimed)
		challenges = append(challenges, model.CharacterChallengeInfo{
			CharaID:       cid,
			Rank:          rank,
			Score:         score,
			Jewel:         jewel,
			Shard:         shard,
			CharaIconPath: b.relativeAsset(b.characterIconPath(cid)),
		})
	}
	displayMax := b.estimateChallengeMaxScore()
	if maxScore > displayMax {
		displayMax = maxScore
	}
	req := &model.ChallengeLiveDetailsRequest{
		Profile:             *profile,
		CharacterChallenges: challenges,
		MaxScore:            displayMax,
	}
	if icon := b.findStaticIcon("jewel.png"); icon != "" {
		req.JewelIconPath = &icon
	}
	if icon := b.findStaticIcon("shard.png"); icon != "" {
		req.ShardIconPath = &icon
	}
	return req, nil
}

func (b *EducationBuilder) pickChallengeRewards(charID int, claimed map[int]struct{}) (int, int) {
	rewards := b.masterdata.GetChallengeRewardsByCharacter(charID)
	if len(rewards) == 0 {
		return 0, 0
	}
	jewelTotal := 0
	shardTotal := 0
	for _, reward := range rewards {
		if _, ok := claimed[reward.ID]; ok {
			continue
		}
		box := b.masterdata.GetResourceBoxByPurpose("challenge_live_high_score", reward.ResourceBoxID)
		if box == nil {
			continue
		}
		for _, detail := range box.Details {
			switch strings.ToLower(detail.ResourceType) {
			case "jewel":
				jewelTotal += detail.ResourceQuantity
			case "material":
				if detail.ResourceID == 15 {
					shardTotal += detail.ResourceQuantity
				}
			}
		}
	}
	return jewelTotal, shardTotal
}

func (b *EducationBuilder) estimateChallengeMaxScore() int {
	maxScore := 0
	for cid := 1; cid <= 26; cid++ {
		for _, reward := range b.masterdata.GetChallengeRewardsByCharacter(cid) {
			if reward.HighScore > maxScore {
				maxScore = reward.HighScore
			}
		}
	}
	if maxScore < 3_000_000 {
		maxScore = 3_000_000
	}
	return maxScore
}

func (b *EducationBuilder) findStaticIcon(filename string) string {
	candidates := []string{
		filepath.Join("lunabot_static_images", filename),
		filename,
	}
	if resolved := asset.ResolveAssetPath(b.assets, b.assetDir, candidates...); resolved != "" {
		return b.relativeAsset(resolved)
	}
	return ""
}

func (b *EducationBuilder) relativeAsset(absPath string) string {
	normBase := filepath.ToSlash(filepath.Clean(b.assetDir))
	normPath := filepath.ToSlash(filepath.Clean(absPath))
	normBase = strings.TrimSuffix(normBase, "/")
	if normBase == "" || normBase == "." {
		return normPath
	}
	prefix := normBase + "/"
	if strings.HasPrefix(normPath, prefix) {
		return strings.TrimPrefix(normPath, prefix)
	}
	return normPath
}

func (b *EducationBuilder) characterIconPath(charID int) string {
	if nickname, ok := asset.CharacterIDToNickname[charID]; ok {
		return asset.ResolveAssetPath(b.assets, b.assetDir, filepath.Join("chara_icon", nickname+".png"))
	}
	return asset.ResolveAssetPath(b.assets, b.assetDir, filepath.Join("chara_icon", fmt.Sprintf("chr_icon_%d.png", charID)))
}
