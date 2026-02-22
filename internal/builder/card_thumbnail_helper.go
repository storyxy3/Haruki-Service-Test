package builder

import (
	"fmt"
	"path/filepath"

	"Haruki-Service-API/internal/model"
	"Haruki-Service-API/pkg/asset"
	"Haruki-Service-API/pkg/masterdata"
)

// ThumbnailOptions allows callers to tweak the generated CardFullThumbnailRequest.
type ThumbnailOptions struct {
	AfterTraining    bool
	ThumbnailPath    string
	RareImgPath      string
	TrainRank        *int
	TrainRankImgPath *string
	Level            *int
	BirthdayIconPath *string
	CustomText       *string
	CardLevel        interface{}
	IsPcard          bool
}

// BuildCardThumbnail 根据选项独立构建一个卡牌的缩略图对象 (特训前或特训后)
func BuildCardThumbnail(assets *asset.AssetHelper, assetDir string, card *masterdata.Card, opts ThumbnailOptions) model.CardFullThumbnailRequest {
	thumbPath := opts.ThumbnailPath
	if thumbPath == "" {
		fileSuffix := "_normal.png"
		if opts.AfterTraining {
			fileSuffix = "_after_training.png"
		}
		memberFile := "card_normal.png"
		if opts.AfterTraining {
			memberFile = "card_after_training.png"
		}
		thumbPath = asset.ResolveAssetPath(assets, assetDir,
			filepath.Join("thumbnail", "chara", card.AssetBundleName+fileSuffix),
			filepath.Join("thumbnail", "chara_rip", card.AssetBundleName+fileSuffix),
			filepath.Join("character", "member", card.AssetBundleName, memberFile),
		)
	} else {
		thumbPath = filepath.ToSlash(thumbPath)
	}

	rareImg := opts.RareImgPath
	if rareImg == "" {
		fileName := "rare_star_normal.png"
		if opts.AfterTraining {
			fileName = "rare_star_after_training.png"
		}
		rareImg = asset.ResolveAssetPath(assets, assetDir,
			filepath.Join("card", fileName),
		)
	} else {
		rareImg = filepath.ToSlash(rareImg)
	}

	isAfter := opts.AfterTraining

	birthdayIcon := opts.BirthdayIconPath
	if birthdayIcon == nil && card.CardRarityType == "rarity_birthday" {
		path := asset.ResolveAssetPath(assets, assetDir, filepath.Join("card", "rare_birthday.png"))
		birthdayIcon = &path
	}

	trainRank := opts.TrainRank
	if trainRank == nil {
		defaultRank := 0
		trainRank = &defaultRank
	}
	framePath := asset.ResolveAssetPath(assets, assetDir,
		filepath.Join("card", fmt.Sprintf("frame_%s.png", card.CardRarityType)),
	)
	attrPath := asset.ResolveAssetPath(assets, assetDir,
		filepath.Join("card", fmt.Sprintf("attr_%s.png", card.Attr)),
	)

	return model.CardFullThumbnailRequest{
		CardID:            card.ID,
		CardThumbnailPath: thumbPath,
		Rare:              card.CardRarityType,
		FrameImgPath:      framePath,
		AttrImgPath:       attrPath,
		RareImgPath:       rareImg,
		TrainRank:         trainRank,
		TrainRankImgPath:  opts.TrainRankImgPath,
		Level:             opts.Level,
		BirthdayIconPath:  birthdayIcon,
		IsAfterTraining:   &isAfter,
		CustomText:        opts.CustomText,
		CardLevel:         opts.CardLevel,
		IsPcard:           opts.IsPcard,
	}
}
