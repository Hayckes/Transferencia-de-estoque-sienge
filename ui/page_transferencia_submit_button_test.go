package ui

import (
	"testing"

	"fyne.io/fyne/v2/widget"
)

func TestBuildTransferSubmitButtonViewModel_IsSuccessStyle(t *testing.T) {
	viewModel := BuildTransferSubmitButtonViewModel()
	if viewModel.Label != "Enviar Transferencia" || viewModel.Style != ButtonStyleSuccess {
		t.Fatalf("BuildTransferSubmitButtonViewModel() = %#v, want success submit button", viewModel)
	}
}

func TestNewSuccessButtonUsesSuccessImportance(t *testing.T) {
	button := NewSuccessButton("Enviar Transferencia", nil)
	if button.Importance != widget.SuccessImportance {
		t.Fatalf("Importance = %v, want SuccessImportance", button.Importance)
	}
}
