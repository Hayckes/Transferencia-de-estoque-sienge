package ui

import "strings"

type FeedbackKind string

const (
	FeedbackSuccess FeedbackKind = "success"
	FeedbackError   FeedbackKind = "error"
	FeedbackInfo    FeedbackKind = "info"
	FeedbackWarning FeedbackKind = "warning"
	FeedbackEmpty   FeedbackKind = "empty"
)

type UserFeedback struct {
	Kind    FeedbackKind
	Message string
	Details string
}

func (feedback UserFeedback) Text() string {
	if strings.TrimSpace(feedback.Details) == "" {
		return feedback.Message
	}
	return feedback.Message + " " + feedback.Details
}

func TransferSuccessFeedback(movementID string) string {
	if strings.TrimSpace(movementID) == "" {
		return "Transferencia enviada com sucesso. Historico e Excel atualizados."
	}
	return "Transferencia enviada com sucesso. Movimento gerado: " + strings.TrimSpace(movementID) + ". Historico e Excel atualizados."
}

func TransferLocalHistoryErrorFeedback(err error) string {
	if err == nil {
		return ""
	}
	return "Transferencia enviada com sucesso no Sienge, mas houve erro ao salvar o historico local: " + err.Error()
}

func ConsultaNoResultsFeedback() string {
	return "Nenhum insumo encontrado para os filtros informados."
}

func PurchaseRequestNoStockFeedback() string {
	return "A solicitacao foi encontrada, mas nenhum dos insumos possui estoque nas obras selecionadas. Verifique se as obras selecionadas estao corretas ou se os itens da solicitacao possuem detalhe/marca compativeis com o estoque."
}

func PurchaseRequestNoItemsFeedback(purchaseRequestID, buildingID string) string {
	return "Nenhum item encontrado para a solicitacao " + strings.TrimSpace(purchaseRequestID) + " na obra " + strings.TrimSpace(buildingID) + "."
}

func NoAppropriationsFeedback(origin bool) string {
	if origin {
		return "Este insumo nao possui apropriacao na obra de origem. A transferencia sera feita sem apropriacao de origem, se o Sienge permitir."
	}
	return "Este insumo nao possui apropriacao na obra de destino. Se o Sienge exigir apropriacao de destino, a transferencia sera bloqueada."
}
