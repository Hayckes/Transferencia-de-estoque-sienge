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

const stockTransfersEndpoint = "/stock-transfers"

type TransferValidationError struct {
	Errors []string
}

func (e *TransferValidationError) Error() string {
	return "transferencia invalida: " + strings.Join(e.Errors, "; ")
}

type StockTransferPayload struct {
	OriginBuildingID      int                        `json:"originBuildingId"`
	DestinationBuildingID int                        `json:"destinationBuildingId"`
	DocumentTypeCode      string                     `json:"documentTypeCode"`
	MovementTypeCode      int                        `json:"movementTypeCode"`
	TransferDate          string                     `json:"transferDate"`
	Note                  string                     `json:"note"`
	Items                 []StockTransferItemPayload `json:"items"`
}

type StockTransferItemPayload struct {
	SupplyID          int     `json:"supplyId"`
	Detail            string  `json:"detail"`
	Brand             string  `json:"brand"`
	AppropriationCode string  `json:"appropriationCode"`
	Quantity          float64 `json:"quantity"`
}

func BuildStockTransferPayload(transfer models.Transferencia) (StockTransferPayload, error) {
	if validationErrors := ValidateTransferencia(transfer); len(validationErrors) > 0 {
		return StockTransferPayload{}, &TransferValidationError{Errors: validationErrors}
	}

	items := make([]StockTransferItemPayload, 0, len(transfer.Insumos))
	for _, item := range transfer.Insumos {
		items = append(items, StockTransferItemPayload{
			SupplyID:          item.ID,
			Detail:            strings.TrimSpace(item.Detalhe),
			Brand:             strings.TrimSpace(item.Marca),
			AppropriationCode: strings.TrimSpace(item.Apropriacao),
			Quantity:          item.Quantidade,
		})
	}

	return StockTransferPayload{
		OriginBuildingID:      transfer.ObraOrigemID,
		DestinationBuildingID: transfer.ObraDestinoID,
		DocumentTypeCode:      strings.TrimSpace(transfer.CodigoTipoDocumento),
		MovementTypeCode:      transfer.CodigoTipoMovimento,
		TransferDate:          transfer.DataHora.Format("2006-01-02T15:04:05"),
		Note:                  BuildTransferNote(transfer),
		Items:                 items,
	}, nil
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
		if strings.TrimSpace(item.Apropriacao) == "" {
			validationErrors = append(validationErrors, prefix+": apropriacao de origem obrigatoria")
		}
		if strings.TrimSpace(item.ApropriacaoDestino) == "" {
			validationErrors = append(validationErrors, prefix+": apropriacao de destino obrigatoria")
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
			strings.TrimSpace(item.Apropriacao),
			strings.TrimSpace(item.ApropriacaoDestino),
			models.FormatQuantidade(item.Quantidade, ""),
		))
	}
	if len(itemParts) > 0 {
		parts = append(parts, "Insumos: "+strings.Join(itemParts, "; ")+".")
	}

	return strings.Join(parts, " ")
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
