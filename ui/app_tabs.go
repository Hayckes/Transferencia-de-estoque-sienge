package ui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"sienge-transfer/models"
)

func BuildMainContent(state *AppState) fyne.CanvasObject {
	status := NewStatusView(state.Window, state.Status)

	return container.NewBorder(
		BuildTopBar(state.Config),
		status.Object(),
		nil,
		nil,
		BuildMainTabs(state),
	)
}

func BuildMainTabs(state *AppState) *container.AppTabs {
	tabs := container.NewAppTabs(
		container.NewTabItem(TabObras, BuildObrasTab(state)),
		container.NewTabItem(TabConsulta, BuildConsultaTab(state)),
		container.NewTabItem(TabTransferencia, BuildTransferenciaTab(state)),
		container.NewTabItem(TabHistorico, BuildHistoricoTab(state)),
	)
	tabs.SetTabLocation(container.TabLocationTop)
	tabs.OnSelected = func(tab *container.TabItem) {
		if tab != nil {
			state.ActiveTab = tab.Text
		}
	}

	selectMainTab(tabs, state.ActiveTab)
	return tabs
}

func selectMainTab(tabs *container.AppTabs, title string) {
	for _, item := range tabs.Items {
		if item.Text == title {
			tabs.Select(item)
			return
		}
	}
}

func BuildTopBar(cfg models.Config) fyne.CanvasObject {
	empresa := widget.NewLabel(fmt.Sprintf("Empresa: %s", cfg.Empresa.Nome))
	usuario := widget.NewLabel(fmt.Sprintf("Usuario: %s", cfg.Usuario.Nome))
	cargo := widget.NewLabel(fmt.Sprintf("Cargo: %s", cfg.Usuario.Cargo))

	return container.NewHScroll(container.NewHBox(empresa, widget.NewSeparator(), usuario, widget.NewSeparator(), cargo))
}
