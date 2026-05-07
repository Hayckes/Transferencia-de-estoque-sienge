package ui

import (
	"context"
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"sienge-transfer/models"
	"sienge-transfer/storage"
)

const appID = "br.com.sienge-transfer.app"

const (
	TabObras         = "Obras"
	TabConsulta      = "Consulta"
	TabTransferencia = "Transferencia"
	TabHistorico     = "Historico"
)

type AppState struct {
	Config        models.Config
	Store         ConfigStore
	Stock         StockService
	CostCenters   CostCenterService
	Transfer      TransferService
	TransferStore TransferStorage
	HistoryStore  HistoryStorage
	FileOpener    FileOpener
	Status        string
	ActiveTab     string
	Obras         ObrasTabState
	Consulta      ConsultaTabState
	Transferencia TransferenciaTabState
	Historico     HistoricoTabState
	Runner        AsyncRunner
	Window        fyne.Window
	RefreshUI     func()
}

type StockService interface {
	GetStockItemsByIDs(ctx context.Context, costCenterID int, ids []int) ([]models.Insumo, error)
	GetBuildingAppropriations(ctx context.Context, costCenterID, resourceID int) ([]models.Apropriacao, error)
}

type CostCenterService interface {
	GetCostCenters(ctx context.Context, costCenterID int) ([]models.Obra, error)
}

type TransferService interface {
	CreateStockTransfer(ctx context.Context, transfer models.Transferencia) (string, error)
}

type TransferStorage interface {
	AppendHistory(transfer models.Transferencia) error
	AppendTransferToExcel(transfer models.Transferencia) error
}

type HistoryStorage interface {
	ReadHistorySummary() ([]storage.HistoricoResumo, error)
	EnsureExcelFromHistory() error
	ExcelPath() string
}

type FileOpener interface {
	Open(path string) error
}

func NewAppStateWithStore(cfg models.Config, store ConfigStore) *AppState {
	state := NewAppState(cfg)
	state.Store = store
	return state
}

func NewAppState(cfg models.Config) *AppState {
	return &AppState{
		Config:        cfg,
		Status:        "Pronto.",
		ActiveTab:     TabObras,
		Obras:         NewObrasTabState(cfg),
		Consulta:      NewConsultaTabState(),
		Transferencia: NewTransferenciaTabState(),
		Historico:     NewHistoricoTabState(),
		Runner:        NewAsyncRunner(func(fn func()) { fn() }),
	}
}

func BuildMainContent(state *AppState) fyne.CanvasObject {
	statusLabel := widget.NewLabel(state.Status)

	return container.NewBorder(
		BuildTopBar(state.Config),
		container.NewBorder(nil, nil, widget.NewLabel("Status:"), nil, statusLabel),
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

func (state *AppState) Refresh() {
	if state != nil && state.RefreshUI != nil {
		state.RefreshUI()
	}
}

func (state *AppState) RefreshTab(tab string) {
	if state != nil {
		state.ActiveTab = tab
		state.Refresh()
	}
}

func BuildTopBar(cfg models.Config) fyne.CanvasObject {
	empresa := widget.NewLabel(fmt.Sprintf("Empresa: %s", cfg.Empresa.Nome))
	usuario := widget.NewLabel(fmt.Sprintf("Usuario: %s", cfg.Usuario.Nome))
	cargo := widget.NewLabel(fmt.Sprintf("Cargo: %s", cfg.Usuario.Cargo))

	return container.NewHBox(empresa, widget.NewSeparator(), usuario, widget.NewSeparator(), cargo)
}
