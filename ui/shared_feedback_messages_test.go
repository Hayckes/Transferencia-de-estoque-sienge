package ui

import (
	"errors"
	"strings"
	"testing"
)

func TestFeedbackMessage_ForConsultaNoResults(t *testing.T) {
	if !strings.Contains(ConsultaNoResultsFeedback(), "Nenhum insumo") {
		t.Fatalf("feedback = %q", ConsultaNoResultsFeedback())
	}
}

func TestFeedbackMessage_ForTransferSuccess(t *testing.T) {
	feedback := TransferSuccessFeedback("MOV-1")
	if !strings.Contains(feedback, "Transferencia enviada com sucesso") || !strings.Contains(feedback, "MOV-1") || !strings.Contains(feedback, "Historico e Excel") {
		t.Fatalf("feedback = %q", feedback)
	}
}

func TestFeedbackMessage_ForPurchaseRequestNoStockFound(t *testing.T) {
	if !strings.Contains(PurchaseRequestNoStockFeedback(), "nenhum dos insumos possui estoque") {
		t.Fatalf("feedback = %q", PurchaseRequestNoStockFeedback())
	}
}

func TestFeedbackMessage_ForNoAppropriations(t *testing.T) {
	if !strings.Contains(NoAppropriationsFeedback(true), "origem") || !strings.Contains(NoAppropriationsFeedback(false), "destino") {
		t.Fatalf("origin/destination feedback mismatch")
	}
}

func TestTransferSuccessFeedback_LocalHistoryErrorAfterAPISuccess(t *testing.T) {
	feedback := TransferLocalHistoryErrorFeedback(errors.New("disco cheio"))
	if !strings.Contains(feedback, "enviada com sucesso no Sienge") || !strings.Contains(feedback, "erro ao salvar o historico local") {
		t.Fatalf("feedback = %q", feedback)
	}
}
