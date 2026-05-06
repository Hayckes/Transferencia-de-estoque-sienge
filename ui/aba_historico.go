package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"sienge-transfer/storage"
)

type HistoricoTabState struct {
	Resumos []storage.HistoricoResumo
}

func NewHistoricoTabState() HistoricoTabState {
	return HistoricoTabState{}
}

func BuildHistoricoTab(state *AppState) fyne.CanvasObject {
	return container.NewVBox(
		widget.NewLabel("Historico resumido"),
		widget.NewLabel("A visualizacao resumida sera conectada ao historico.json nas proximas etapas."),
	)
}
