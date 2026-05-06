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

type AppState struct {
	Config        models.Config
	Store         ConfigStore
	Stock         StockService
	Transfer      TransferService
	TransferStore TransferStorage
	HistoryStore  HistoryStorage
	FileOpener    FileOpener
	Status        string
	Obras         ObrasTabState
	Consulta      ConsultaTabState
	Transferencia TransferenciaTabState
	Historico     HistoricoTabState
	Runner        AsyncRunner
}

type StockService interface {
	GetStockItemsByIDs(ctx context.Context, costCenterID int, ids []int) ([]models.Insumo, error)
	GetBuildingAppropriations(ctx context.Context, costCenterID, resourceID int) ([]models.Apropriacao, error)
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
