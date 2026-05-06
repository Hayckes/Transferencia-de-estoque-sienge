package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"sienge-transfer/models"
)

var ErrInvalidCostCenter = errors.New("ID da obra/centro de custo deve ser numerico positivo")

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
	if costCenterID <= 0 {
		return nil, ErrInvalidCostCenter
	}
	if resourceID <= 0 {
		return nil, errors.New("ID do insumo deve ser numerico positivo")
	}

	body, err := c.do(ctx, "GET", fmt.Sprintf("/stock-inventories/%d/items/%d/building-appropriation", costCenterID, resourceID), nil)
	if err != nil {
		return nil, err
	}

	return parseAppropriations(body)
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
		original, _ := json.Marshal(object)

		items = append(items, models.Insumo{
			ID:           id,
			Nome:         getString(object, "resourceName", "supplyName", "name", "description"),
			Detalhe:      getString(object, "detailDescription", "detail", "detailName", "specification"),
			Marca:        getString(object, "trademarkDescription", "brand", "brandName", "trademark"),
			Unidade:      getString(object, "unitOfMeasure", "unit", "measureUnit"),
			Quantidade:   quantity,
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
		appropriations = append(appropriations, models.Apropriacao{
			Codigo:     getString(object, "appropriationCode", "buildingAppropriationCode", "code", "id"),
			Descricao:  getString(object, "appropriationDescription", "buildingAppropriationDescription", "description", "name"),
			Quantidade: quantity,
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
		for _, key := range []string{"results", "items", "data"} {
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
