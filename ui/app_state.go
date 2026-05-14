package ui

import (
	"context"
	"sync"

	"fyne.io/fyne/v2"
	"sienge-transfer/models"
	"sienge-transfer/storage"
)

const (
	TabObras         = "Obras"
	TabConsulta      = "Consulta"
	TabTransferencia = "Transferencia"
	TabEmprestimos   = "Emprestimos"
	TabHistorico     = "Historico"
)

type AppState struct {
	Config           models.Config
	Store            ConfigStore
	Stock            StockService
	CostCenters      CostCenterService
	PurchaseRequests PurchaseRequestService
	Transfer         TransferService
	TransferStore    TransferStorage
	HistoryStore     HistoryStorage
	LoanStore        LoanStorage
	FileOpener       FileOpener
	Status           string
	ActiveTab        string
	Obras            ObrasTabState
	Consulta         ConsultaTabState
	Transferencia    TransferenciaTabState
	Emprestimos      EmprestimosTabState
	Historico        HistoricoTabState
	Runner           AsyncRunner
	Window           fyne.Window
	RefreshUI        func()
	transferSubmitMu sync.Mutex
}

type StockService interface {
	GetStockItemsByIDs(ctx context.Context, costCenterID int, ids []int) ([]models.Insumo, error)
	GetBuildingAppropriations(ctx context.Context, costCenterID, resourceID int) ([]models.Apropriacao, error)
	GetStockAppropriationsWithDescriptions(ctx context.Context, costCenterID, resourceID int) ([]models.Apropriacao, error)
	GetStockAppropriationsWithDescriptionsForItem(ctx context.Context, costCenterID int, item models.Insumo) ([]models.Apropriacao, error)
}

type CostCenterService interface {
	GetCostCenters(ctx context.Context, costCenterID int) ([]models.Obra, error)
}

type TransferService interface {
	CreateStockTransfer(ctx context.Context, transfer models.Transferencia) (string, error)
}

type PurchaseRequestService interface {
	GetPurchaseRequestItems(ctx context.Context, purchaseRequestID int, buildingID int) ([]models.PurchaseRequestItem, error)
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

type LoanStorage interface {
	ListLoans() ([]models.LoanRecord, error)
	SaveAllLoans([]models.LoanRecord) error
	UpsertLoan(models.LoanRecord) error
	GetLoanByID(id string) (models.LoanRecord, error)
	UpdateLoanAfterReturn(returnTransfer models.Transferencia) error
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
		Emprestimos:   NewEmprestimosTabState(),
		Historico:     NewHistoricoTabState(),
		Runner:        NewAsyncRunner(func(fn func()) { fn() }),
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
