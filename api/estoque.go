package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"

	"sienge-transfer/models"
)

var ErrInvalidCostCenter = errors.New("ID da obra/centro de custo deve ser numerico positivo")

type BuildingAppropriationQuery struct {
	CostCenterID int
	ResourceID   int
	DetailID     *int
	TrademarkID  *int
}

type StockItemKey struct {
	CostCenterID int
	ResourceID   int
	DetailID     int
	TrademarkID  int
}

func (c *Client) GetStockItems(ctx context.Context, costCenterID int) ([]models.Insumo, error) {
	if costCenterID <= 0 {
		return nil, ErrInvalidCostCenter
	}

	body, err := c.do(ctx, "GET", fmt.Sprintf("/stock-inventories/%d/items", costCenterID), nil)
	if err != nil {
		return nil, err
	}

	return parseStockItems(body)
}

func (c *Client) GetStockItemsByIDs(ctx context.Context, costCenterID int, ids []int) ([]models.Insumo, error) {
	if len(ids) == 0 {
		return nil, models.ErrIDsInsumoObrigatorios
	}

	wanted := make(map[int]bool, len(ids))
	for _, id := range ids {
		if id <= 0 {
			return nil, errors.New("IDs de insumo devem conter apenas numeros positivos")
		}
		wanted[id] = true
	}

	items, err := c.GetStockItems(ctx, costCenterID)
	if err != nil {
		return nil, err
	}

	filtered := make([]models.Insumo, 0, len(items))
	for _, item := range items {
		if wanted[item.ID] {
			filtered = append(filtered, item)
		}
	}

	return filtered, nil
}

func (c *Client) GetBuildingAppropriations(ctx context.Context, costCenterID, resourceID int) ([]models.Apropriacao, error) {
	return c.GetBuildingAppropriationsByQuery(ctx, BuildingAppropriationQuery{CostCenterID: costCenterID, ResourceID: resourceID})
}

func (c *Client) GetBuildingAppropriationsByQuery(ctx context.Context, query BuildingAppropriationQuery) ([]models.Apropriacao, error) {
	if query.CostCenterID <= 0 {
		return nil, ErrInvalidCostCenter
	}
	if query.ResourceID <= 0 {
		return nil, errors.New("ID do insumo deve ser numerico positivo")
	}

	body, err := c.do(ctx, "GET", buildBuildingAppropriationPath(query), nil)
	if err != nil {
		return nil, err
	}

	return parseAppropriations(body)
}

func (c *Client) GetStockAppropriationsWithDescriptions(ctx context.Context, costCenterID, resourceID int) ([]models.Apropriacao, error) {
	return c.GetStockAppropriationsWithDescriptionsByQuery(ctx, BuildingAppropriationQuery{CostCenterID: costCenterID, ResourceID: resourceID})
}

func (c *Client) GetStockAppropriationsWithDescriptionsForItem(ctx context.Context, costCenterID int, item models.Insumo) ([]models.Apropriacao, error) {
	return c.GetStockAppropriationsWithDescriptionsByQuery(ctx, BuildingAppropriationQuery{
		CostCenterID: costCenterID,
		ResourceID:   item.ID,
		DetailID:     positiveIntPtr(item.DetalheID),
		TrademarkID:  positiveIntPtr(item.MarcaID),
	})
}

func (c *Client) GetStockAppropriationsWithDescriptionsByQuery(ctx context.Context, query BuildingAppropriationQuery) ([]models.Apropriacao, error) {
	appropriations, err := c.GetBuildingAppropriationsByQuery(ctx, query)
	if err != nil || len(appropriations) == 0 {
		return appropriations, err
	}

	debugAppropriationQuery(query, appropriations)
	sheetItemsByUnit := make(map[int][]costEstimationSheetItem)
	for index := range appropriations {
		appropriation := &appropriations[index]
		if appropriation.BuildingUnitID <= 0 || appropriation.SheetItemID <= 0 {
			continue
		}

		items, ok := sheetItemsByUnit[appropriation.BuildingUnitID]
		if !ok {
			body, err := c.do(ctx, "GET", fmt.Sprintf("/building-cost-estimations/%d/sheets/%d/items", query.CostCenterID, appropriation.BuildingUnitID), nil)
			if err != nil {
				return nil, err
			}
			items, err = parseCostEstimationSheetItems(body)
			if err != nil {
				return nil, err
			}
			sheetItemsByUnit[appropriation.BuildingUnitID] = items
		}

		if item, ok := findMatchingSheetItem(*appropriation, items); ok && strings.TrimSpace(item.Description) != "" {
			appropriation.Descricao = item.Description
			if strings.TrimSpace(appropriation.Referencia) == "" {
				appropriation.Referencia = item.Reference
			}
		}
	}

	return appropriations, nil
}

func buildBuildingAppropriationPath(query BuildingAppropriationQuery) string {
	path := fmt.Sprintf("/stock-inventories/%d/items/%d/building-appropriation", query.CostCenterID, query.ResourceID)
	params := url.Values{}
	params.Set("offset", "0")
	params.Set("limit", "100")
	if query.DetailID != nil {
		params.Set("detailId", strconv.Itoa(*query.DetailID))
	}
	if query.TrademarkID != nil {
		params.Set("trademarkId", strconv.Itoa(*query.TrademarkID))
	}
	return path + "?" + params.Encode()
}

func StockItemCacheKey(costCenterID int, item models.Insumo) StockItemKey {
	return StockItemKey{CostCenterID: costCenterID, ResourceID: item.ID, DetailID: item.DetalheID, TrademarkID: item.MarcaID}
}

func positiveIntPtr(value int) *int {
	if value <= 0 {
		return nil
	}
	return &value
}

func debugAppropriationQuery(query BuildingAppropriationQuery, appropriations []models.Apropriacao) {
	if strings.ToLower(strings.TrimSpace(os.Getenv("SIENGE_TRANSFER_DEBUG_APPROPRIATIONS"))) != "1" {
		return
	}
	sum := 0.0
	for _, appropriation := range appropriations {
		sum += appropriation.Quantidade
	}
	fmt.Printf("DEBUG Apropriacoes: url=%s sumAppropriations=%.4f count=%d\n", buildBuildingAppropriationPath(query), sum, len(appropriations))
}

type costEstimationSheetItem struct {
	ID          int
	Reference   string
	Description string
}

func parseCostEstimationSheetItems(body []byte) ([]costEstimationSheetItem, error) {
	objects, err := decodeObjectList(body)
	if err != nil {
		return nil, err
	}

	items := make([]costEstimationSheetItem, 0, len(objects))
	for _, object := range objects {
		id, _ := getInt(object, "id", "itemId", "sheetItemId")
		items = append(items, costEstimationSheetItem{
			ID:          id,
			Reference:   getString(object, "reference", "code", "itemReference", "costEstimationItemReference"),
			Description: getString(object, "description", "name", "itemDescription"),
		})
	}

	return items, nil
}

func findMatchingSheetItem(appropriation models.Apropriacao, items []costEstimationSheetItem) (costEstimationSheetItem, bool) {
	for _, item := range items {
		if item.ID != appropriation.SheetItemID {
			continue
		}
		if appropriation.Referencia != "" && item.Reference != "" && appropriation.Referencia != item.Reference {
			continue
		}
		return item, true
	}
	for _, item := range items {
		if item.ID == appropriation.SheetItemID {
			return item, true
		}
	}
	return costEstimationSheetItem{}, false
}

func parseStockItems(body []byte) ([]models.Insumo, error) {
	objects, err := decodeObjectList(body)
	if err != nil {
		return nil, err
	}

	items := make([]models.Insumo, 0, len(objects))
	for _, object := range objects {
		id, ok := getInt(object, "resourceId", "supplyId", "id")
		if !ok || id <= 0 {
			return nil, errors.New("resposta de estoque sem ID de insumo valido")
		}

		quantity, _ := getFloat(object, "quantity", "availableQuantity", "balance", "stockQuantity")
		detailID, _ := getInt(object, "detailId", "resourceDetailId")
		trademarkID, _ := getInt(object, "trademarkId", "brandId", "resourceTrademarkId", "resourceBrandId")
		averagePrice, _ := getFloat(object, "averagePrice", "unitPrice")
		original, _ := json.Marshal(object)

		items = append(items, models.Insumo{
			ID:           id,
			Nome:         getString(object, "resourceName", "supplyName", "name", "description"),
			Detalhe:      getString(object, "detailDescription", "detail", "detailName", "specification"),
			DetalheID:    detailID,
			Marca:        getString(object, "trademarkDescription", "brand", "brandName", "trademark"),
			MarcaID:      trademarkID,
			Unidade:      getString(object, "unitOfMeasure", "unit", "measureUnit"),
			Quantidade:   quantity,
			PrecoMedio:   averagePrice,
			OriginalJSON: string(original),
		})
	}

	return items, nil
}

func parseAppropriations(body []byte) ([]models.Apropriacao, error) {
	objects, err := decodeObjectList(body)
	if err != nil {
		return nil, err
	}

	appropriations := make([]models.Apropriacao, 0, len(objects))
	for _, object := range objects {
		quantity, _ := getFloat(object, "quantity", "availableQuantity", "balance", "stockQuantity")
		buildingUnitID, _ := getInt(object, "buildingUnitId", "buildingUnitID", "unitId")
		sheetItemID, _ := getInt(object, "sheetItemId", "sheetItemID", "itemId")
		blocked, _ := getBool(object, "blocked", "locked", "isBlocked", "isLocked", "bloqueado", "blockedForAppropriation", "blockedForAppropriations", "budgetItemBlocked", "isBudgetItemBlocked")
		reference := getString(object, "costEstimationItemReference", "reference")
		code := getString(object, "appropriationCode", "buildingAppropriationCode", "costEstimationItemReference", "code", "id")
		if code == "" && sheetItemID > 0 {
			code = strconv.Itoa(sheetItemID)
		}
		description := getString(object, "appropriationDescription", "buildingAppropriationDescription", "description", "name")
		if description == "" {
			description = reference
		}
		appropriations = append(appropriations, models.Apropriacao{
			Codigo:         code,
			Descricao:      description,
			Referencia:     reference,
			BuildingUnitID: buildingUnitID,
			SheetItemID:    sheetItemID,
			Quantidade:     quantity,
			Bloqueado:      blocked,
		})
	}

	return appropriations, nil
}

func decodeObjectList(body []byte) ([]map[string]any, error) {
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()

	var data any
	if err := decoder.Decode(&data); err != nil {
		return nil, err
	}

	var rawItems []any
	switch typed := data.(type) {
	case []any:
		rawItems = typed
	case map[string]any:
		for _, key := range []string{"results", "resultados", "items", "data"} {
			if items, ok := typed[key].([]any); ok {
				rawItems = items
				break
			}
		}
		if rawItems == nil {
			return nil, errors.New("resposta da API sem lista de resultados")
		}
	default:
		return nil, errors.New("resposta da API em formato inesperado")
	}

	objects := make([]map[string]any, 0, len(rawItems))
	for _, item := range rawItems {
		object, ok := item.(map[string]any)
		if !ok {
			return nil, errors.New("item da resposta da API em formato inesperado")
		}
		objects = append(objects, object)
	}

	return objects, nil
}

func getString(object map[string]any, keys ...string) string {
	for _, key := range keys {
		value, ok := object[key]
		if !ok || value == nil {
			continue
		}

		if nested, ok := value.(map[string]any); ok {
			if text := getString(nested, "description", "name", "code", "id"); text != "" {
				return text
			}
		}

		text := valueToString(value)
		if text != "" {
			return text
		}
	}

	return ""
}

func getInt(object map[string]any, keys ...string) (int, bool) {
	for _, key := range keys {
		value, ok := object[key]
		if !ok || value == nil {
			continue
		}

		switch typed := value.(type) {
		case json.Number:
			parsed, err := typed.Int64()
			if err == nil {
				return int(parsed), true
			}
		case float64:
			return int(typed), true
		case string:
			parsed, err := strconv.Atoi(strings.TrimSpace(typed))
			if err == nil {
				return parsed, true
			}
		}
	}

	return 0, false
}

func getFloat(object map[string]any, keys ...string) (float64, bool) {
	for _, key := range keys {
		value, ok := object[key]
		if !ok || value == nil {
			continue
		}

		switch typed := value.(type) {
		case json.Number:
			parsed, err := typed.Float64()
			if err == nil {
				return parsed, true
			}
		case float64:
			return typed, true
		case string:
			parsed, err := strconv.ParseFloat(strings.ReplaceAll(strings.TrimSpace(typed), ",", "."), 64)
			if err == nil {
				return parsed, true
			}
		}
	}

	return 0, false
}

func getBool(object map[string]any, keys ...string) (bool, bool) {
	for _, key := range keys {
		value, ok := object[key]
		if !ok || value == nil {
			continue
		}

		switch typed := value.(type) {
		case bool:
			return typed, true
		case string:
			parsed, err := strconv.ParseBool(strings.TrimSpace(typed))
			if err == nil {
				return parsed, true
			}
		case json.Number:
			parsed, err := typed.Int64()
			if err == nil {
				return parsed != 0, true
			}
		case float64:
			return typed != 0, true
		}
	}

	return false, false
}

func valueToString(value any) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case json.Number:
		return typed.String()
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64)
	default:
		return ""
	}
}
