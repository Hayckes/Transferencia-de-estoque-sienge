package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"sienge-transfer/models"
)

const purchaseRequestItemsLimit = 100

func (c *Client) GetPurchaseRequestItems(ctx context.Context, purchaseRequestID int, buildingID int) ([]models.PurchaseRequestItem, error) {
	if purchaseRequestID <= 0 {
		return nil, errors.New("ID da solicitacao de compra deve ser numerico positivo")
	}
	if buildingID <= 0 {
		return nil, errors.New("ID da obra da solicitacao deve ser numerico positivo")
	}

	items := make([]models.PurchaseRequestItem, 0)
	seen := make(map[string]bool)
	for offset := 0; ; offset += purchaseRequestItemsLimit {
		path := fmt.Sprintf("/purchase-requests/all/items?purchaseRequestId=%d&buildingId=%d&limit=%d&offset=%d", purchaseRequestID, buildingID, purchaseRequestItemsLimit, offset)
		body, err := c.do(ctx, "GET", path, nil)
		if err != nil {
			return nil, err
		}

		page, err := parsePurchaseRequestItems(body, purchaseRequestID, buildingID)
		if err != nil {
			return nil, err
		}
		for _, item := range page {
			key := fmt.Sprintf("%d:%d:%d", item.ResourceID, item.DetailID, item.BrandID)
			if !seen[key] {
				items = append(items, item)
				seen[key] = true
			}
		}

		if len(page) < purchaseRequestItemsLimit {
			break
		}
	}

	return items, nil
}

func parsePurchaseRequestItems(body []byte, purchaseRequestID int, buildingID int) ([]models.PurchaseRequestItem, error) {
	objects, err := decodeObjectList(body)
	if err != nil {
		return nil, err
	}

	items := make([]models.PurchaseRequestItem, 0, len(objects))
	for _, object := range objects {
		resourceID, _ := getIntFlexible(object, "resourceId", "productId", "supplyId", "itemId", "id")
		if resourceID <= 0 {
			continue
		}
		detailID, _ := getIntFlexible(object, "detailId", "resourceDetailId")
		brandID, _ := getIntFlexible(object, "brandId", "trademarkId", "resourceBrandId")
		quantity, _ := getFloat(object, "quantity", "quantidade", "requestedQuantity", "purchaseQuantity")
		original, _ := json.Marshal(object)

		items = append(items, models.PurchaseRequestItem{
			PurchaseRequestID: purchaseRequestID,
			BuildingID:        buildingID,
			ResourceID:        resourceID,
			ResourceName:      getString(object, "resourceName", "productDescription", "supplyName", "itemName", "name", "description"),
			Detail:            getString(object, "detailDescription", "detail", "detailName", "specification"),
			DetailID:          detailID,
			Brand:             getString(object, "trademarkDescription", "brandDescription", "marca", "brand", "brandName", "trademark"),
			BrandID:           brandID,
			Unit:              getString(object, "unitSymbol", "unitOfMeasure", "unidade", "unit", "measureUnit"),
			Quantity:          quantity,
			OriginalJSON:      original,
		})
	}

	return items, nil
}

func getIntFlexible(object map[string]any, keys ...string) (int, bool) {
	if value, ok := getInt(object, keys...); ok {
		return value, true
	}
	for _, key := range keys {
		value, ok := object[key]
		if !ok {
			continue
		}
		nested, ok := value.(map[string]any)
		if !ok {
			continue
		}
		if value, ok := getInt(nested, "id", "resourceId", "supplyId"); ok {
			return value, true
		}
	}
	return 0, false
}
