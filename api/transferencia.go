package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"sienge-transfer/models"
)

const stockTransfersEndpoint = "/stock-movements/transfer"

type TransferValidationError struct {
	Errors []string
}

func (e *TransferValidationError) Error() string {
	return "transferencia invalida: " + strings.Join(e.Errors, "; ")
}

type StockTransferPayload struct {
	SourceCostCenterID      int                        `json:"sourceCostCenterId"`
	DestinationCostCenterID int                        `json:"destinationCostCenterId"`
	SourceDepartmentID      int                        `json:"sourceDepartmentId,omitempty"`
	DestinationDepartmentID int                        `json:"destinationDepartmentId,omitempty"`
	DocumentID              string                     `json:"documentId"`
	MovementTypeID          int                        `json:"movementTypeId"`
	MovementDate            string                     `json:"movementDate"`
	Notes                   string                     `json:"notes"`
	Items                   []StockTransferItemPayload `json:"items"`
}

type StockTransferItemPayload struct {
	Source      StockTransferItemSidePayload `json:"source"`
	Destination StockTransferItemSidePayload `json:"destination"`
}

type StockTransferItemSidePayload struct {
	ResourceID             int                                  `json:"resourceId"`
	DetailID               int                                  `json:"detailId,omitempty"`
	TrademarkID            int                                  `json:"trademarkId,omitempty"`
	Quantity               float64                              `json:"quantity,omitempty"`
	UnitOfMeasure          string                               `json:"unitOfMeasure,omitempty"`
	UnitPrice              float64                              `json:"unitPrice,omitempty"`
	BuildingAppropriations []StockTransferBuildingAppropriation `json:"buildingAppropriations,omitempty"`
}

type StockTransferBuildingAppropriation struct {
	BuildingUnitID int     `json:"buildingUnitId"`
	SheetItemID    int     `json:"sheetItemId"`
	Percentage     float64 `json:"percentage"`
}

func BuildStockTransferPayload(transfer models.Transferencia) (StockTransferPayload, error) {
	if validationErrors := ValidateTransferencia(transfer); len(validationErrors) > 0 {
		return StockTransferPayload{}, &TransferValidationError{Errors: validationErrors}
	}

	items := make([]StockTransferItemPayload, 0, len(transfer.Insumos))
	for _, item := range transfer.Insumos {
		sourceAppropriations := transferPayloadAppropriations(item.ApropriacaoOrigemBuildingUnitID, item.ApropriacaoOrigemSheetItemID)
		destinationAppropriations := transferPayloadAppropriations(item.ApropriacaoDestinoBuildingUnitID, item.ApropriacaoDestinoSheetItemID)
		items = append(items, StockTransferItemPayload{
			Source: StockTransferItemSidePayload{
				ResourceID:             item.ID,
				DetailID:               item.DetalheID,
				TrademarkID:            item.MarcaID,
				Quantity:               item.Quantidade,
				UnitOfMeasure:          strings.TrimSpace(item.Unidade),
				BuildingAppropriations: sourceAppropriations,
			},
			Destination: StockTransferItemSidePayload{
				ResourceID:             item.ID,
				DetailID:               item.DetalheID,
				TrademarkID:            item.MarcaID,
				UnitPrice:              item.PrecoUnitario,
				BuildingAppropriations: destinationAppropriations,
			},
		})
	}

	return StockTransferPayload{
		SourceCostCenterID:      transfer.ObraOrigemID,
		DestinationCostCenterID: transfer.ObraDestinoID,
		SourceDepartmentID:      transfer.Insumos[0].ApropriacaoOrigemBuildingUnitID,
		DestinationDepartmentID: transfer.Insumos[0].ApropriacaoDestinoBuildingUnitID,
		DocumentID:              strings.TrimSpace(transfer.CodigoTipoDocumento),
		MovementTypeID:          transfer.CodigoTipoMovimento,
		MovementDate:            transfer.DataHora.Format("2006-01-02"),
		Notes:                   BuildTransferNote(transfer),
		Items:                   items,
	}, nil
}

func transferPayloadAppropriations(buildingUnitID int, sheetItemID int) []StockTransferBuildingAppropriation {
	if buildingUnitID <= 0 || sheetItemID <= 0 {
		return nil
	}
	return []StockTransferBuildingAppropriation{{
		BuildingUnitID: buildingUnitID,
		SheetItemID:    sheetItemID,
		Percentage:     100,
	}}
}

func ValidateTransferencia(transfer models.Transferencia) []string {
	var validationErrors []string

	if transfer.ObraOrigemID <= 0 {
		validationErrors = append(validationErrors, "obra de origem obrigatoria")
	}
	if transfer.ObraDestinoID <= 0 {
		validationErrors = append(validationErrors, "obra de destino obrigatoria")
	}
	if transfer.ObraOrigemID > 0 && transfer.ObraDestinoID > 0 && transfer.ObraOrigemID == transfer.ObraDestinoID {
		validationErrors = append(validationErrors, "obra de origem deve ser diferente da obra de destino")
	}
	if strings.TrimSpace(transfer.Solicitante) == "" {
		validationErrors = append(validationErrors, "solicitante obrigatorio")
	}
	if strings.TrimSpace(transfer.CodigoTipoDocumento) == "" {
		validationErrors = append(validationErrors, "codigo do tipo de documento obrigatorio")
	}
	if transfer.CodigoTipoMovimento <= 0 {
		validationErrors = append(validationErrors, "codigo do tipo de movimento deve ser numerico positivo")
	}
	if transfer.DataHora.IsZero() {
		validationErrors = append(validationErrors, "data e hora da transferencia obrigatoria")
	}
	if len(transfer.Insumos) == 0 {
		validationErrors = append(validationErrors, "adicione pelo menos um insumo")
	}

	for index, item := range transfer.Insumos {
		prefix := fmt.Sprintf("insumo %d", index+1)
		if item.ID <= 0 {
			validationErrors = append(validationErrors, prefix+": ID do insumo deve ser numerico positivo")
		}
		if item.ApropriacaoOrigemObrigatoria && strings.TrimSpace(item.Apropriacao) == "" {
			validationErrors = append(validationErrors, prefix+": apropriacao de origem obrigatoria")
		}
		if item.ApropriacaoDestinoObrigatoria && strings.TrimSpace(item.ApropriacaoDestino) == "" {
			validationErrors = append(validationErrors, prefix+": apropriacao de destino obrigatoria")
		}
		if item.ApropriacaoOrigemObrigatoria && (item.ApropriacaoOrigemBuildingUnitID <= 0 || item.ApropriacaoOrigemSheetItemID <= 0) {
			validationErrors = append(validationErrors, prefix+": identificadores da apropriacao de origem obrigatorios")
		}
		if item.ApropriacaoDestinoObrigatoria && (item.ApropriacaoDestinoBuildingUnitID <= 0 || item.ApropriacaoDestinoSheetItemID <= 0) {
			validationErrors = append(validationErrors, prefix+": identificadores da apropriacao de destino obrigatorios")
		}
		if strings.TrimSpace(item.Unidade) == "" {
			validationErrors = append(validationErrors, prefix+": unidade de medida obrigatoria")
		}
		if item.PrecoUnitario <= 0 {
			validationErrors = append(validationErrors, prefix+": preco unitario obrigatorio")
		}
		if item.Quantidade <= 0 {
			validationErrors = append(validationErrors, prefix+": quantidade deve ser maior que zero")
		}
		if item.QuantidadeDisponivel > 0 && item.Quantidade > item.QuantidadeDisponivel {
			validationErrors = append(validationErrors, prefix+": quantidade a transferir maior que a disponivel")
		}
	}

	return validationErrors
}

func BuildTransferNote(transfer models.Transferencia) string {
	parts := []string{
		fmt.Sprintf("Transferencia realizada por %s (%s).", strings.TrimSpace(transfer.Usuario), strings.TrimSpace(transfer.Cargo)),
		fmt.Sprintf("Solicitante: %s.", strings.TrimSpace(transfer.Solicitante)),
		fmt.Sprintf("Data/hora: %s.", transfer.DataHora.Format("02/01/2006 15:04:05")),
		fmt.Sprintf("Origem: %d - %s.", transfer.ObraOrigemID, strings.TrimSpace(transfer.ObraOrigemNome)),
		fmt.Sprintf("Destino: %d - %s.", transfer.ObraDestinoID, strings.TrimSpace(transfer.ObraDestinoNome)),
	}
	if observation := strings.TrimSpace(transfer.Observacao); observation != "" {
		parts = append(parts, fmt.Sprintf("Observacao: %s.", observation))
	}

	itemParts := make([]string, 0, len(transfer.Insumos))
	for _, item := range transfer.Insumos {
		itemParts = append(itemParts, fmt.Sprintf(
			"%d - %s %s - %s | apropriacao origem %s | apropriacao destino %s | quantidade %s",
			item.ID,
			strings.TrimSpace(item.Nome),
			strings.TrimSpace(item.Detalhe),
			strings.TrimSpace(item.Marca),
			formatAppropriationText(item.Apropriacao, item.ApropriacaoDescricao),
			formatAppropriationText(item.ApropriacaoDestino, item.ApropriacaoDestinoDescricao),
			models.FormatQuantidade(item.Quantidade, ""),
		))
	}
	if len(itemParts) > 0 {
		parts = append(parts, "Insumos: "+strings.Join(itemParts, "; ")+".")
	}

	return strings.Join(parts, " ")
}

func formatAppropriationText(code, description string) string {
	if strings.TrimSpace(description) == "" {
		return strings.TrimSpace(code)
	}
	return strings.TrimSpace(code) + " - " + strings.TrimSpace(description)
}

func (c *Client) CreateStockTransfer(ctx context.Context, transfer models.Transferencia) (string, error) {
	payload, err := BuildStockTransferPayload(transfer)
	if err != nil {
		return "", err
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	resp, err := c.doResponse(ctx, http.MethodPost, stockTransfersEndpoint, body)
	if err != nil {
		return "", err
	}

	return ExtractMovementID(&http.Response{Header: resp.Header}, resp.Body), nil
}

func ExtractMovementID(resp *http.Response, body []byte) string {
	if len(body) > 0 {
		var data map[string]any
		if json.Unmarshal(body, &data) == nil {
			for _, key := range []string{"id", "movementId", "stockMovementId", "documentNumber", "movementNumber"} {
				if value, ok := data[key]; ok && value != nil {
					text := strings.TrimSpace(fmt.Sprint(value))
					if text != "" {
						return text
					}
				}
			}
		}
	}

	if resp != nil {
		if location := strings.TrimSpace(resp.Header.Get("Location")); location != "" {
			parts := strings.Split(strings.TrimRight(location, "/"), "/")
			return parts[len(parts)-1]
		}
	}

	return ""
}

func IsTransferValidationError(err error) bool {
	var validationError *TransferValidationError
	return errors.As(err, &validationError)
}

func NewTransferenciaBase() models.Transferencia {
	return models.Transferencia{
		CodigoTipoDocumento: "TR",
		CodigoTipoMovimento: 3,
		DataHora:            time.Now(),
	}
}
