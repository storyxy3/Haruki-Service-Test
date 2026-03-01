package service

import (
	"context"
	"strings"
	"sync"

	"Haruki-Service-API/pkg/masterdata"

	sekai "haruki-cloud/database/sekai"
	"haruki-cloud/database/sekai/challengelivehighscorereward"
	"haruki-cloud/database/sekai/resourceboxe"
)

// CloudEducationSource implements EducationDataSource backed by Haruki-Cloud database.
type CloudEducationSource struct {
	client      *sekai.Client
	region      string
	queryRegion string

	rewardMu      sync.RWMutex
	rewardsByChar map[int][]*masterdata.ChallengeLiveHighScoreReward
	rewardsLoaded bool

	boxMu        sync.RWMutex
	boxByID      map[int]*masterdata.ResourceBox
	boxByPurpose map[string]map[int]*masterdata.ResourceBox
	boxesLoaded  bool
}

func NewCloudEducationSource(client *sekai.Client, defaultRegion string) *CloudEducationSource {
	if client == nil {
		return nil
	}
	region := strings.TrimSpace(defaultRegion)
	if region == "" {
		region = "JP"
	}
	return &CloudEducationSource{
		client:        client,
		region:        region,
		queryRegion:   strings.ToLower(region),
		rewardsByChar: make(map[int][]*masterdata.ChallengeLiveHighScoreReward),
		boxByID:       make(map[int]*masterdata.ResourceBox),
		boxByPurpose:  make(map[string]map[int]*masterdata.ResourceBox),
	}
}

func (c *CloudEducationSource) DefaultRegion() string {
	return c.region
}

func (c *CloudEducationSource) context() context.Context {
	return context.Background()
}

func (c *CloudEducationSource) GetChallengeRewardsByCharacter(charID int) []*masterdata.ChallengeLiveHighScoreReward {
	if charID <= 0 {
		return nil
	}

	c.rewardMu.RLock()
	if c.rewardsLoaded {
		out := cloneChallengeRewards(c.rewardsByChar[charID])
		c.rewardMu.RUnlock()
		return out
	}
	c.rewardMu.RUnlock()

	c.rewardMu.Lock()
	defer c.rewardMu.Unlock()
	if !c.rewardsLoaded {
		items, err := c.client.Challengelivehighscorereward.Query().
			Where(challengelivehighscorereward.ServerRegionEQ(c.queryRegion)).
			All(c.context())
		if err != nil {
			return nil
		}
		for _, item := range items {
			charID := int(item.CharacterID)
			model := &masterdata.ChallengeLiveHighScoreReward{
				ID:            int(item.GameID),
				CharacterID:   charID,
				HighScore:     int(item.HighScore),
				ResourceBoxID: int(item.ResourceBoxID),
			}
			c.rewardsByChar[charID] = append(c.rewardsByChar[charID], model)
		}
		c.rewardsLoaded = true
	}

	return cloneChallengeRewards(c.rewardsByChar[charID])
}

func (c *CloudEducationSource) GetResourceBoxByPurpose(purpose string, id int) *masterdata.ResourceBox {
	if id <= 0 {
		return nil
	}
	c.boxMu.RLock()
	if c.boxesLoaded {
		var box *masterdata.ResourceBox
		if strings.TrimSpace(purpose) == "" {
			box = c.boxByID[id]
		} else if purposeMap, ok := c.boxByPurpose[purpose]; ok {
			box = purposeMap[id]
		}
		c.boxMu.RUnlock()
		return cloneResourceBox(box)
	}
	c.boxMu.RUnlock()

	c.boxMu.Lock()
	defer c.boxMu.Unlock()
	if !c.boxesLoaded {
		items, err := c.client.Resourceboxe.Query().
			Where(resourceboxe.ServerRegionEQ(c.queryRegion)).
			All(c.context())
		if err != nil {
			return nil
		}
		for _, item := range items {
			model := &masterdata.ResourceBox{
				ResourceBoxPurpose: item.ResourceBoxPurpose,
				ID:                 int(item.GameID),
				ResourceBoxType:    item.ResourceBoxType,
				Description:        item.Description,
			}
			if len(item.Details) > 0 {
				var details []masterdata.ResourceBoxDetail
				if err := decodeFlexible(item.Details, &details); err != nil {
					continue
				}
				model.Details = details
			}
			c.boxByID[model.ID] = model
			purposeKey := model.ResourceBoxPurpose
			if _, ok := c.boxByPurpose[purposeKey]; !ok {
				c.boxByPurpose[purposeKey] = make(map[int]*masterdata.ResourceBox)
			}
			c.boxByPurpose[purposeKey][model.ID] = model
		}
		c.boxesLoaded = true
	}

	if strings.TrimSpace(purpose) == "" {
		return cloneResourceBox(c.boxByID[id])
	}
	if purposeMap, ok := c.boxByPurpose[purpose]; ok {
		return cloneResourceBox(purposeMap[id])
	}
	return nil
}

func cloneChallengeRewards(src []*masterdata.ChallengeLiveHighScoreReward) []*masterdata.ChallengeLiveHighScoreReward {
	if len(src) == 0 {
		return nil
	}
	out := make([]*masterdata.ChallengeLiveHighScoreReward, 0, len(src))
	for _, item := range src {
		if item == nil {
			continue
		}
		copy := *item
		out = append(out, &copy)
	}
	return out
}

func cloneResourceBox(src *masterdata.ResourceBox) *masterdata.ResourceBox {
	if src == nil {
		return nil
	}
	copy := *src
	if src.Details != nil {
		copy.Details = append([]masterdata.ResourceBoxDetail(nil), src.Details...)
	}
	return &copy
}
