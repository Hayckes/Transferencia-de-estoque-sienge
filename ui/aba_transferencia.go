package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"sienge-transfer/models"
)

type TransferenciaTabState struct {
	ObraOrigem      string
	ObraDestino     string
	Solicitante     string
	CodigoDocumento string
	CodigoMovimento string
	InsumoIDInput   string
	Itens           []models.ItemTransferido
}

func NewTransferenciaTabState() TransferenciaTabState {
	return TransferenciaTabState{
		CodigoDocumento: "TR",
		CodigoMovimento: "3",
	}
}

func BuildTransferenciaTab(state *AppState) fyne.CanvasObject {
	return container.NewVBox(
		widget.NewLabel("Transferencia de insumos"),
		widget.NewLabel("O formulario desta aba sera implementado nas proximas etapas."),
	)
}
