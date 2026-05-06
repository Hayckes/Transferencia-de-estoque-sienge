package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"sienge-transfer/models"
)

type ConsultaTabState struct {
	ObraSelecionada string
	InsumoIDsInput  string
	Observacao      string
	Resultados      []models.Insumo
}

func NewConsultaTabState() ConsultaTabState {
	return ConsultaTabState{}
}

func BuildConsultaTab(state *AppState) fyne.CanvasObject {
	return container.NewVBox(
		widget.NewLabel("Consulta de estoque"),
		widget.NewLabel("Os filtros e resultados desta aba permanecem em memoria enquanto o app estiver aberto."),
	)
}
