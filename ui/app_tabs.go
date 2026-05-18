package ui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"sienge-transfer/models"
)

type mainShell struct {
	content  fyne.CanvasObject
	topBar   *fyne.Container
	tabs     *container.AppTabs
	tabItems map[string]*container.TabItem
	status   *StatusView
}

func BuildMainContent(state *AppState) fyne.CanvasObject {
	shell := newMainShell(state)
	state.mainShell = shell
	return shell.content
}

func newMainShell(state *AppState) *mainShell {
	status := NewStatusView(state.Window, state.Status)
	topBar := container.NewStack(BuildTopBar(state.Config))
	tabs := BuildMainTabs(state)
	shell := &mainShell{
		content: container.NewBorder(
			topBar,
			status.Object(),
			nil,
			nil,
			tabs,
		),
		topBar:   topBar,
		tabs:     tabs,
		tabItems: mainTabItems(tabs),
		status:   status,
	}
	return shell
}

func (shell *mainShell) RefreshAll(state *AppState) {
	if shell == nil {
		return
	}
	for _, tab := range mainTabTitles() {
		shell.replaceTabContent(state, tab)
	}
	shell.refreshTopBar(state)
	shell.refreshStatus(state)
	selectMainTab(shell.tabs, state.ActiveTab)
	shell.content.Refresh()
}

func (shell *mainShell) RefreshTab(state *AppState, tab string) {
	if shell == nil {
		return
	}
	if tab == "" {
		tab = state.ActiveTab
	}
	shell.replaceTabContent(state, tab)
	shell.refreshStatus(state)
	selectMainTab(shell.tabs, tab)
	shell.tabs.Refresh()
}

func (shell *mainShell) replaceTabContent(state *AppState, tab string) {
	item := shell.tabItems[tab]
	if item == nil {
		return
	}
	item.Content = BuildMainTabContent(state, tab)
}

func (shell *mainShell) refreshStatus(state *AppState) {
	if shell.status != nil {
		shell.status.SetText(state.Status)
	}
}

func (shell *mainShell) refreshTopBar(state *AppState) {
	if shell.topBar == nil {
		return
	}
	shell.topBar.Objects = []fyne.CanvasObject{BuildTopBar(state.Config)}
	shell.topBar.Refresh()
}

func BuildMainTabs(state *AppState) *container.AppTabs {
	tabs := container.NewAppTabs(
		container.NewTabItem(TabObras, BuildObrasTab(state)),
		container.NewTabItem(TabConsulta, BuildConsultaTab(state)),
		container.NewTabItem(TabTransferencia, BuildTransferenciaTab(state)),
		container.NewTabItem(TabEmprestimos, BuildEmprestimosTab(state)),
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

func BuildMainTabContent(state *AppState, tab string) fyne.CanvasObject {
	switch tab {
	case TabObras:
		return BuildObrasTab(state)
	case TabConsulta:
		return BuildConsultaTab(state)
	case TabTransferencia:
		return BuildTransferenciaTab(state)
	case TabEmprestimos:
		return BuildEmprestimosTab(state)
	case TabHistorico:
		return BuildHistoricoTab(state)
	default:
		return widget.NewLabel("")
	}
}

func mainTabItems(tabs *container.AppTabs) map[string]*container.TabItem {
	items := make(map[string]*container.TabItem, len(tabs.Items))
	for _, item := range tabs.Items {
		items[item.Text] = item
	}
	return items
}

func mainTabTitles() []string {
	return []string{TabObras, TabConsulta, TabTransferencia, TabEmprestimos, TabHistorico}
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
