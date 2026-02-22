package builder

import (
	"Haruki-Service-API/internal/model"
	"Haruki-Service-API/pkg/masterdata"
)

// buildThumbnailInfo 构建卡牌缩略图请求信息

func (b *CardBuilder) buildThumbnailInfo(card *masterdata.Card) []model.CardFullThumbnailRequest {
	requests := []model.CardFullThumbnailRequest{
		BuildCardThumbnail(b.assets, b.assetDir, card, ThumbnailOptions{AfterTraining: false}),
	}

	if card.CardRarityType == "rarity_3" || card.CardRarityType == "rarity_4" {
		requests = append(requests, BuildCardThumbnail(b.assets, b.assetDir, card, ThumbnailOptions{AfterTraining: true}))
	}

	return requests
}
