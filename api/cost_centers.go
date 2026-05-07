package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"sienge-transfer/models"
)

var ErrCostCenterNotFound = errors.New("centro de custo nao encontrado no Sienge")

func (c *Client) GetCostCenters(ctx context.Context, costCenterID int) ([]models.Obra, error) {
	if costCenterID <= 0 {
		return nil, ErrInvalidCostCenter
	}

	body, err := c.do(ctx, http.MethodGet, fmt.Sprintf("/cost-centers/%d", costCenterID), nil)
	if err != nil {
		var apiErr *APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound {
			return nil, ErrCostCenterNotFound
		}
		return nil, err
	}

	centers, err := parseCostCenters(body, costCenterID)
	if err != nil {
		return nil, err
	}
	if len(centers) == 0 {
		return nil, ErrCostCenterNotFound
	}

	return centers, nil
}

func parseCostCenters(body []byte, fallbackID int) ([]models.Obra, error) {
	objects, err := decodeObjectList(body)
	if err != nil {
		object, objectErr := decodeObject(body)
		if objectErr != nil {
			return nil, err
		}
		objects = []map[string]any{object}
	}

	centers := make([]models.Obra, 0, len(objects))
	for _, object := range objects {
		id, ok := getInt(object, "id", "costCenterId", "costCenterCode", "code")
		if !ok || id <= 0 {
			id = fallbackID
		}
		if id <= 0 {
			return nil, errors.New("resposta de centro de custo sem ID valido")
		}

		name := getString(object, "name", "description", "costCenterName", "costCenterDescription", "descr", "title")
		if name == "" {
			return nil, errors.New("resposta de centro de custo sem nome valido")
		}

		centers = append(centers, models.Obra{ID: id, Nome: name})
	}

	return centers, nil
}

func decodeObject(body []byte) (map[string]any, error) {
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()

	var object map[string]any
	if err := decoder.Decode(&object); err != nil {
		return nil, err
	}
	if object == nil {
		return nil, errors.New("resposta da API em formato inesperado")
	}

	return object, nil
}
