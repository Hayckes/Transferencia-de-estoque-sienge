package ui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"sienge-transfer/models"
)

const appID = "br.com.sienge-transfer.app"

type AppState struct {
	Config        models.Config
	Status        string
	Obras         ObrasTabState
	Consulta      ConsultaTabState
	Transferencia TransferenciaTabState
	Historico     HistoricoTabState
	Runner        AsyncRunner
}

func NewAppState(cfg models.Config) *AppState {
	return &AppState{
		Config:        cfg,
		Status:        "Pronto.",
		Obras:         NewObrasTabState(cfg),
		Consulta:      NewConsultaTabState(),
		Transferencia: NewTransferenciaTabState(),
		Historico:     NewHistoricoTabState(),
		Runner:        NewAsyncRunner(func(fn func()) { fn() }),
	}
}

func BuildMainContent(state *AppState) fyne.CanvasObject {
	statusLabel := widget.NewLabel(state.Status)

	tabs := container.NewAppTabs(
		container.NewTabItem("Obras", BuildObrasTab(state)),
		container.NewTabItem("Consulta", BuildConsultaTab(state)),
		container.NewTabItem("Transferencia", BuildTransferenciaTab(state)),
		container.NewTabItem("Historico", BuildHistoricoTab(state)),
	)
	tabs.SetTabLocation(container.TabLocationTop)

	return container.NewBorder(
		BuildTopBar(state.Config),
		container.NewBorder(nil, nil, widget.NewLabel("Status:"), nil, statusLabel),
		nil,
		nil,
		tabs,
	)
}

func BuildTopBar(cfg models.Config) fyne.CanvasObject {
	empresa := widget.NewLabel(fmt.Sprintf("Empresa: %s", cfg.Empresa.Nome))
	usuario := widget.NewLabel(fmt.Sprintf("Usuario: %s", cfg.Usuario.Nome))
	cargo := widget.NewLabel(fmt.Sprintf("Cargo: %s", cfg.Usuario.Cargo))

	return container.NewHBox(empresa, widget.NewSeparator(), usuario, widget.NewSeparator(), cargo)
}
