package builder

import (
	"fmt"
	"path/filepath"

	"Haruki-Service-API/internal/model"
	"Haruki-Service-API/internal/service"
	"Haruki-Service-API/pkg/asset"
	"Haruki-Service-API/pkg/masterdata"
)

// CardBuilder 专门负责构建供 DrawingAPI 消费的 Card 模块 JSON Payload
type CardBuilder struct {
	masterdata    *service.MasterDataService
	assets        *asset.AssetHelper
	assetDir      string
	searchService *service.CardSearchService
	userData      *service.UserDataService
}

// NewCardBuilder 初始化新卡牌组装器
func NewCardBuilder(m *service.MasterDataService, a *asset.AssetHelper, d string, s *service.CardSearchService, u *service.UserDataService) *CardBuilder {
	return &CardBuilder{
		masterdata:    m,
		assets:        a,
		assetDir:      d,
		searchService: s,
		userData:      u,
	}
}

// BuildCardDetailRequestBody 构建 CardDetailRequest 请求体
// 这是核心的数据转换逻辑
func (b *CardBuilder) BuildCardDetailRequestBody(
	card *masterdata.Card,
	region string,
) (*model.CardDetailRequest, error) {

	// 1. 构建基础卡牌信息
	cardInfo := b.buildCardBasic(card)

	// 2. 获取活动信息
	var eventInfo *model.CardEventInfo
	var eventAttrPath, eventUnitPath, eventCharaPath string

	if event, err := b.masterdata.GetEventByCardID(card.ID); err == nil && event != nil {
		eventInfo = &model.CardEventInfo{
			EventID:         event.ID,
			EventName:       event.Name,
			StartAt:         event.StartAt,
			EndAt:           event.AggregateAt + 1000,
			EventBannerPath: b.buildEventBannerPath(event.AssetBundleName),
		}

		// 获取活动加成信息
		if bonuses, err := b.masterdata.GetEventDeckBonuses(event.ID); err == nil {
			// 1. Attribute Bonus
			for _, bonus := range bonuses {
				if bonus.CardAttr != "" {
					eventAttrPath = asset.ResolveAssetPath(b.assets, b.assetDir, filepath.Join("card", fmt.Sprintf("attr_icon_%s.png", bonus.CardAttr)))
					eventInfo.BonusAttr = bonus.CardAttr // Set BonusAttr to trigger display
					// break // REMOVED: Lunabot logic iterates all and uses the last one.
				}
			}

			// 2. Unit/Character Bonus
			// Logic: specific chara ? or unit ?

			// Check if it is a "Ban Event" (Unit Event)
			// Lunabot logic: If all bonuses with a unit ID belong to the same unit -> Ban Event.
			uniqueUnits := make(map[string]struct{})
			for _, bonus := range bonuses {
				if bonus.GameCharacterUnitID > 0 {
					if gcu, err := b.masterdata.GetGameCharacterUnit(bonus.GameCharacterUnitID); err == nil {
						uniqueUnits[gcu.Unit] = struct{}{}
					}
				}
			}

			isUnitEvent := len(uniqueUnits) == 1
			var eventUnit string
			if isUnitEvent {
				for unit := range uniqueUnits {
					eventUnit = unit // Get the single unit
				}
			}

			if isUnitEvent {
				// Set Unit Icon
				unitIconName := b.getUnitIconName(eventUnit)
				if unitIconName != "" {
					eventUnitPath = asset.ResolveAssetPath(b.assets, b.assetDir, filepath.Join("unit", unitIconName+".png"))
					eventInfo.Unit = eventUnit // Set Unit to trigger display
				}

				// Set Banner Character Icon
				// Only for Unit Events
				if bannerCID, err := b.masterdata.GetEventBannerCharacterID(event.ID); err == nil {
					eventCharaPath = b.buildCharacterIconPath(bannerCID, eventUnit)
					eventInfo.BannerCID = bannerCID
				}
			} else {
				// Mixed Event: Do not show Unit or Chara icon. Only Attribute.
				// Explicitly do nothing here. eventUnitPath and eventCharaPath default to empty.
			}
		}
	}

	// 3. 获取卡池信息
	var gachaInfo *model.CardGachaInfo
	if gacha, err := b.masterdata.GetGachaByCardID(card.ID); err == nil && gacha != nil {
		gachaInfo = &model.CardGachaInfo{
			GachaID:         gacha.ID,
			GachaName:       gacha.Name,
			StartAt:         gacha.StartAt,
			EndAt:           (gacha.EndAt/1000 + 1) * 1000, // Round up to next second (Lunabot behavior)
			GachaBannerPath: b.buildGachaBannerPath(gacha.ID),
		}
	}

	// 4. 构建完整请求
	req := &model.CardDetailRequest{
		CardInfo:  cardInfo,
		Region:    region,
		EventInfo: eventInfo,
		GachaInfo: gachaInfo,

		// 图片路径
		CardImagesPath:      b.buildCardImagePaths(card),
		CostumeImagesPath:   b.buildCostumeImagePaths(card),
		CharacterIconPath:   b.buildCharacterIconPath(card.CharacterID, cardInfo.Unit),
		UnitLogoPath:        b.buildUnitLogoPath(cardInfo.Unit),
		BackgroundImagePath: nil, // 可选

		EventAttrIconPath:  eventAttrPath,
		EventUnitIconPath:  eventUnitPath,
		EventCharaIconPath: eventCharaPath,
	}

	// DEBUG: Verify Gacha and Costume info
	fmt.Printf("[DEBUG] Builder - CardID: %d\n", card.ID)
	if gachaInfo != nil {
		fmt.Printf("[DEBUG] Builder - GachaInfo: ID=%d, Name=%s, Banner=%s\n", gachaInfo.GachaID, gachaInfo.GachaName, gachaInfo.GachaBannerPath)
	} else {
		fmt.Printf("[DEBUG] Builder - GachaInfo is NIL\n")
	}
	costumePaths := b.buildCostumeImagePaths(card)
	fmt.Printf("[DEBUG] Builder - CostumePaths: %v\n", costumePaths)

	return req, nil
}

// buildCardBasic 构建通用基础卡牌信息
func (b *CardBuilder) buildCardBasic(card *masterdata.Card) model.CardBasic {
	cardInfo := model.CardBasic{
		CardID:          card.ID,
		CharacterID:     card.CharacterID,
		Rare:            card.CardRarityType,
		Attr:            card.Attr,
		Prefix:          card.Prefix,
		AssetBundleName: card.AssetBundleName,
		ReleaseAt:       card.ReleaseAt,
		IsAfterTraining: false,                      // 默认训练前
		ThumbnailInfo:   b.buildThumbnailInfo(card), // 构建略缩图信息
		Power:           b.calculatePower(card),     // 计算综合力
	}

	// 获取角色信息
	if char, err := b.masterdata.GetCharacterByID(card.CharacterID); err == nil && char != nil {
		cardInfo.CharacterName = char.FirstName + char.GivenName
	}
	// Unit logic
	if unit, err := b.masterdata.GetUnitByCardID(card.ID); err == nil {
		cardInfo.Unit = unit
	}

	// 获取供给类型
	cardInfo.SupplyType = b.masterdata.GetCardSupplyType(card)

	// 获取技能信息
	if skill, err := b.masterdata.GetSkillByID(card.SkillID); err == nil && skill != nil {
		cardInfo.Skill = &model.CardSkill{
			SkillID:           skill.ID,
			SkillName:         card.CardSkillName,
			SkillDetail:       b.masterdata.FormatSkillDescription(skill, card.CharacterID),
			SkillType:         skill.DescriptionSpriteName,
			SkillTypeIconPath: b.buildSkillTypeIconPath(skill.DescriptionSpriteName),
		}
	}

	// Handle Special Training Skill (Fes Cards)
	if card.SpecialTrainingSkillId > 0 {
		if spSkill, err := b.masterdata.GetSkillByID(card.SpecialTrainingSkillId); err == nil && spSkill != nil {
			cardInfo.SpecialSkillInfo = &model.CardSkill{
				SkillID:           spSkill.ID,
				SkillName:         card.SpecialTrainingSkillName,
				SkillDetail:       b.masterdata.FormatSkillDescription(spSkill, card.CharacterID),
				SkillType:         spSkill.DescriptionSpriteName,
				SkillTypeIconPath: b.buildSkillTypeIconPath(spSkill.DescriptionSpriteName),
			}
		}
	}

	return cardInfo
}

// BuildCardListRequest 构建卡牌列表请求
func (b *CardBuilder) BuildCardListRequest(cardIDs []int, region string) (*model.CardListRequest, error) {
	var cards []model.CardBasic

	for _, id := range cardIDs {
		card, err := b.masterdata.GetCardByID(id)
		if err != nil {
			// 如果卡牌不存在，跳过或报错？这里选择跳过
			fmt.Printf("[WARN] Card ID %d not found\n", id)
			continue
		}

		// 基础卡牌信息
		// 注意：DrawingAPI 已修改为会绘制 ThumbnailInfo 中的所有图片
		// 因此不需要拆分特训前/后为两个对象，只需传一个包含完整 ThumbnailInfo 的对象即可
		baseCard := b.buildCardBasic(card)

		// 适配 DrawingAPI 列表逻辑：将 "常驻" 转换为 "normal" 以避免框体标黄
		if baseCard.SupplyType == "常驻" {
			baseCard.SupplyType = "normal"
		}

		cards = append(cards, baseCard)
	}

	if len(cards) == 0 {
		return nil, fmt.Errorf("no valid cards found from provided IDs")
	}

	req := &model.CardListRequest{
		Cards:               cards,
		Region:              region,
		UserInfo:            b.getDetailedProfile(region),
		BackgroundImagePath: nil, // 默认背景
	}

	return req, nil
}

// calculatePower 计算卡牌综合力
// 根据 cardParameters 计算 power1, power2, power3 和总综合力
func (b *CardBuilder) calculatePower(card *masterdata.Card) *model.CardPower {
	var power1, power2, power3 int

	// 从 cardParameters 中提取最大值
	for _, param := range card.CardParameters {
		switch param.CardParameterType {
		case "param1":
			if param.Power > power1 {
				power1 = param.Power
			}
		case "param2":
			if param.Power > power2 {
				power2 = param.Power
			}
		case "param3":
			if param.Power > power3 {
				power3 = param.Power
			}
		}
	}

	// 计算总综合力
	power1 += card.SpecialTrainingPower1BonusFixed
	power2 += card.SpecialTrainingPower2BonusFixed
	power3 += card.SpecialTrainingPower3BonusFixed
	powerTotal := power1 + power2 + power3

	return &model.CardPower{
		Power1:     power1,
		Power2:     power2,
		Power3:     power3,
		PowerTotal: powerTotal,
	}
}

func (b *CardBuilder) getDetailedProfile(region string) *model.DetailedProfileCardRequest {
	if b.userData != nil {
		if profile := b.userData.DetailedProfile(region); profile != nil {
			return profile
		}
	}
	return nil
}

// buildCardImagePaths 构建卡牌图片路径
// 根据实际资源文件结构
func (b *CardBuilder) buildCardImagePaths(card *masterdata.Card) []string {
	// 实际路径格式:
	// character/member/{assetBundleName}/{assetBundleName}_rip/card_normal.png
	// 或者直接 character/member/{assetBundleName}_rip/card_normal.png
	// 实际路径格式验证: D:\pjskdata\data\character\member\res005_no047\card_normal.png (No _rip suffix)
	basePath := fmt.Sprintf("%s/character/member/%s", b.assetDir, card.AssetBundleName)

	paths := []string{
		fmt.Sprintf("%s/card_normal.png", basePath),
	}

	// 只有 3 星和 4 星卡牌有特训后图片
	if card.CardRarityType == "rarity_3" || card.CardRarityType == "rarity_4" {
		paths = append(paths, fmt.Sprintf("%s/card_after_training.png", basePath))
	}

	return paths
}

// buildCostumeImagePaths 构建服装图片路径
func (b *CardBuilder) buildCostumeImagePaths(card *masterdata.Card) []string {
	costumes := []string{}
	// Use the plural method to get all costumes
	costume3ds, err := b.masterdata.GetCostume3dsByCardID(card.ID)
	if err != nil || len(costume3ds) == 0 {
		return costumes
	}

	for _, costume := range costume3ds {
		// Costume path: thumbnail/costume_rip/{assetBundleName}.png
		// Manual verification needed. Based on card_190_info.md:
		// D:\pjskdata\data\thumbnail\costume_rip\cos0080_unique_head.png
		path := filepath.ToSlash(filepath.Join(b.assetDir, "thumbnail", "costume_rip", costume.AssetBundleName+".png"))
		costumes = append(costumes, path)
	}

	return costumes
}

// buildCharacterIconPath 构建角色图标路径
func (b *CardBuilder) buildCharacterIconPath(characterID int, unit string) string {
	// 1. 初音未来 (ID=21) 优先使用团体特定图标
	if characterID == 21 && unit != "" && unit != "piapro" {
		return asset.ResolveAssetPath(b.assets, b.assetDir, fmt.Sprintf("chara_icon/miku_%s.png", unit))
	}

	// 2. 使用通用角色图标路径
	if nickname, ok := asset.CharacterIDToNickname[characterID]; ok {
		return asset.ResolveAssetPath(b.assets, b.assetDir, filepath.Join("chara_icon", nickname+".png"))
	}
	return asset.ResolveAssetPath(b.assets, b.assetDir, filepath.Join("chara_icon", fmt.Sprintf("chr_icon_%d.png", characterID)))
}

// buildUnitLogoPath 构建团队 logo 路径
func (b *CardBuilder) buildUnitLogoPath(unit string) string {
	if unit == "" {
		return ""
	}
	// 实际路径格式: logo_{unit}.png (在 data 根目录)
	return asset.ResolveAssetPath(b.assets, b.assetDir, fmt.Sprintf("logo_%s.png", unit))
}

// buildSkillTypeIconPath 构建技能类型图标路径
func (b *CardBuilder) buildSkillTypeIconPath(skillType string) string {
	if skillType == "" {
		return ""
	}
	// 实际路径格式: skill/skill_{skillType}.png
	return asset.ResolveAssetPath(b.assets, b.assetDir, fmt.Sprintf("skill/skill_%s.png", skillType))
}

// buildEventBannerPath 构建活动 banner 路径
func (b *CardBuilder) buildEventBannerPath(assetBundleName string) string {
	if assetBundleName == "" {
		return ""
	}
	candidates := []string{
		filepath.Join("home", "banner", fmt.Sprintf("%s_rip", assetBundleName), assetBundleName+".png"),
		filepath.Join("home", "banner", assetBundleName, assetBundleName+".png"),
		filepath.Join("event", assetBundleName, "banner.png"),
	}
	return asset.ResolveAssetPath(b.assets, b.assetDir, candidates...)
}

// buildGachaBannerPath 构建卡池 banner 路径
func (b *CardBuilder) buildGachaBannerPath(gachaID int) string {
	if gachaID == 0 {
		return ""
	}
	candidates := []string{
		filepath.Join("home", "banner", fmt.Sprintf("banner_gacha%d", gachaID), fmt.Sprintf("banner_gacha%d.png", gachaID)),
		filepath.Join("home", "banner", fmt.Sprintf("banner_gacha%d_rip", gachaID), fmt.Sprintf("banner_gacha%d.png", gachaID)),
		filepath.Join("gacha", fmt.Sprintf("banner_gacha%d.png", gachaID)),
	}
	return asset.ResolveAssetPath(b.assets, b.assetDir, candidates...)
}

// getUnitIconName 获取团图标文件名
func (b *CardBuilder) getUnitIconName(unit string) string {
	return asset.UnitIconFilename(unit)
}
