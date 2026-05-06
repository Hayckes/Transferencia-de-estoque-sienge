package ui

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"sienge-transfer/models"
)

func TestTransferOriginAndDestinationValidation(t *testing.T) {
	state := NewAppState(testConfig())

	if _, err := TransferOriginID(state); !errors.Is(err, ErrObraOrigemObrigatoria) {
		t.Fatalf("TransferOriginID() error = %v, want ErrObraOrigemObrigatoria", err)
	}
	if _, err := TransferDestinationID(state); !errors.Is(err, ErrObraDestinoObrigatoria) {
		t.Fatalf("TransferDestinationID() error = %v, want ErrObraDestinoObrigatoria", err)
	}

	state.Transferencia.ObraOrigem = "121 - Residencial Novo Horizonte"
	state.Transferencia.ObraDestino = "205 - Comercial Centro"
	originID, err := TransferOriginID(state)
	if err != nil || originID != 121 {
		t.Fatalf("TransferOriginID() = %d/%v, want 121/nil", originID, err)
	}
	destinationID, err := TransferDestinationID(state)
	if err != nil || destinationID != 205 {
		t.Fatalf("TransferDestinationID() = %d/%v, want 205/nil", destinationID, err)
	}
}

func TestAddTransferInsumoAddsSingleItemWithAppropriations(t *testing.T) {
	stock := &fakeStockService{
		items: []models.Insumo{{ID: 3421, Nome: "Cimento", Detalhe: "CP III", Marca: "Votorantim", Unidade: "SC", Quantidade: 150}},
		appropriations: []models.Apropriacao{
			{Codigo: "A001", Descricao: "Fundacao", Quantidade: 40},
			{Codigo: "A002", Descricao: "Estrutura", Quantidade: 20},
		},
	}
	state := validTransferState()
	state.Stock = stock

	if err := AddTransferInsumo(context.Background(), state, 3421); err != nil {
		t.Fatalf("AddTransferInsumo() error = %v", err)
	}

	if !stock.itemsCalled || !stock.approprCalled {
		t.Fatal("stock item and appropriation calls should be executed")
	}
	if stock.costCenterID != 121 || stock.resourceID != 3421 {
		t.Fatalf("stock call ids = costCenter %d resource %d, want 121/3421", stock.costCenterID, stock.resourceID)
	}
	if len(state.Transferencia.Itens) != 1 {
		t.Fatalf("len(Itens) = %d, want 1", len(state.Transferencia.Itens))
	}
	if len(state.Transferencia.Itens[0].Insumo.Apropriacoes) != 2 {
		t.Fatalf("appropriations = %#v, want 2", state.Transferencia.Itens[0].Insumo.Apropriacoes)
	}
}

func TestAddTransferInsumoRequiresOriginBeforeCallingAPI(t *testing.T) {
	stock := &fakeStockService{}
	state := NewAppState(testConfig())
	state.Stock = stock

	err := AddTransferInsumo(context.Background(), state, 3421)
	if !errors.Is(err, ErrObraOrigemObrigatoria) {
		t.Fatalf("AddTransferInsumo() error = %v, want ErrObraOrigemObrigatoria", err)
	}
	if stock.itemsCalled {
		t.Fatal("stock service should not be called without origin work")
	}
}

func TestAddTransferInsumoHandlesNotFoundAndMultipleItems(t *testing.T) {
	state := validTransferState()
	state.Stock = &fakeStockService{}

	err := AddTransferInsumo(context.Background(), state, 3421)
	if err == nil || !strings.Contains(err.Error(), "nao encontrado") {
		t.Fatalf("AddTransferInsumo() error = %v, want not found", err)
	}

	state.Stock = &fakeStockService{items: []models.Insumo{{ID: 3421, Detalhe: "CP III"}, {ID: 3421, Detalhe: "CP II"}}}
	err = AddTransferInsumo(context.Background(), state, 3421)
	var multipleErr *MultipleInsumosError
	if !errors.As(err, &multipleErr) {
		t.Fatalf("AddTransferInsumo() error type = %T, want MultipleInsumosError", err)
	}
	if len(multipleErr.Options) != 2 {
		t.Fatalf("len(Options) = %d, want 2", len(multipleErr.Options))
	}
}

func TestSetTransferItemAppropriationUpdatesAvailableQuantity(t *testing.T) {
	state := validTransferState()
	state.Transferencia.Itens = []TransferenciaItemState{{Insumo: models.Insumo{Apropriacoes: []models.Apropriacao{{Codigo: "A001", Descricao: "Fundacao", Quantidade: 40}}}}}

	if err := SetTransferItemAppropriation(state, 0, "A001 - Fundacao"); err != nil {
		t.Fatalf("SetTransferItemAppropriation() error = %v", err)
	}
	if state.Transferencia.Itens[0].ApropriacaoSelecionada != "A001 - Fundacao" {
		t.Fatalf("ApropriacaoSelecionada = %q, want A001 - Fundacao", state.Transferencia.Itens[0].ApropriacaoSelecionada)
	}
	if state.Transferencia.Itens[0].QuantidadeDisponivel != 40 {
		t.Fatalf("QuantidadeDisponivel = %v, want 40", state.Transferencia.Itens[0].QuantidadeDisponivel)
	}
}

func TestRemoveTransferItem(t *testing.T) {
	state := validTransferState()
	state.Transferencia.Itens = []TransferenciaItemState{{Insumo: models.Insumo{ID: 1}}, {Insumo: models.Insumo{ID: 2}}}

	if err := RemoveTransferItem(state, 0); err != nil {
		t.Fatalf("RemoveTransferItem() error = %v", err)
	}
	if len(state.Transferencia.Itens) != 1 || state.Transferencia.Itens[0].Insumo.ID != 2 {
		t.Fatalf("Itens = %#v, want only ID 2", state.Transferencia.Itens)
	}
	if err := RemoveTransferItem(state, 9); err == nil {
		t.Fatal("RemoveTransferItem() error = nil, want error")
	}
}

func TestBuildTransferenciaFromState(t *testing.T) {
	state := validTransferStateWithItem()

	transfer, err := BuildTransferenciaFromState(state)
	if err != nil {
		t.Fatalf("BuildTransferenciaFromState() error = %v", err)
	}
	if transfer.Usuario != "Joao Silva" || transfer.Cargo != "Engenheiro" || transfer.Solicitante != "Maria Santos" {
		t.Fatalf("transfer user/requester = %#v", transfer)
	}
	if transfer.ObraOrigemID != 121 || transfer.ObraDestinoID != 205 {
		t.Fatalf("transfer buildings = %d/%d, want 121/205", transfer.ObraOrigemID, transfer.ObraDestinoID)
	}
	if transfer.CodigoTipoDocumento != "TR" || transfer.CodigoTipoMovimento != 3 {
		t.Fatalf("transfer codes = %s/%d, want TR/3", transfer.CodigoTipoDocumento, transfer.CodigoTipoMovimento)
	}
	if len(transfer.Insumos) != 1 {
		t.Fatalf("len(Insumos) = %d, want 1", len(transfer.Insumos))
	}
	if transfer.Insumos[0].Apropriacao != "A001" || transfer.Insumos[0].ApropriacaoDescricao != "Fundacao" || transfer.Insumos[0].Quantidade != 10.5 {
		t.Fatalf("transfer item = %#v, want parsed appropriation and quantity", transfer.Insumos[0])
	}
}

func TestBuildTransferenciaFromStateRejectsInvalidFields(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*AppState)
		want   string
	}{
		{name: "same works", mutate: func(s *AppState) { s.Transferencia.ObraDestino = s.Transferencia.ObraOrigem }, want: "diferente"},
		{name: "missing requester", mutate: func(s *AppState) { s.Transferencia.Solicitante = "" }, want: "solicitante"},
		{name: "invalid movement", mutate: func(s *AppState) { s.Transferencia.CodigoMovimento = "abc" }, want: "movimento"},
		{name: "missing item", mutate: func(s *AppState) { s.Transferencia.Itens = nil }, want: "adicione pelo menos um insumo"},
		{name: "missing appropriation", mutate: func(s *AppState) { s.Transferencia.Itens[0].ApropriacaoSelecionada = "" }, want: "apropriacao"},
		{name: "quantity greater than available", mutate: func(s *AppState) { s.Transferencia.Itens[0].QuantidadeTransferir = "50" }, want: "maior que a disponivel"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := validTransferStateWithItem()
			tt.mutate(state)
			_, err := BuildTransferenciaFromState(state)
			if err == nil {
				t.Fatal("BuildTransferenciaFromState() error = nil, want error")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %q, want containing %q", err.Error(), tt.want)
			}
		})
	}
}

func TestSendTransferenciaSavesHistoryAndExcelOnlyAfterAPISuccess(t *testing.T) {
	transferService := &fakeTransferService{movementID: "MOV-1"}
	transferStore := &fakeTransferStorage{}
	state := validTransferStateWithItem()
	state.Transfer = transferService
	state.TransferStore = transferStore

	movementID, err := SendTransferencia(context.Background(), state)
	if err != nil {
		t.Fatalf("SendTransferencia() error = %v", err)
	}
	if movementID != "MOV-1" {
		t.Fatalf("movementID = %q, want MOV-1", movementID)
	}
	if !transferService.called || !transferStore.historySaved || !transferStore.excelSaved {
		t.Fatalf("called/saved = %v/%v/%v, want all true", transferService.called, transferStore.historySaved, transferStore.excelSaved)
	}
	if transferStore.historyTransfer.IDMovimento != "MOV-1" || transferStore.excelTransfer.IDMovimento != "MOV-1" {
		t.Fatalf("saved transfer IDs = %q/%q, want MOV-1", transferStore.historyTransfer.IDMovimento, transferStore.excelTransfer.IDMovimento)
	}
	if len(state.Transferencia.Itens) != 0 || state.Transferencia.CodigoMovimento != "3" {
		t.Fatalf("transfer state should be reset after success: %#v", state.Transferencia)
	}
}

func TestSendTransferenciaDoesNotSaveWhenAPIFails(t *testing.T) {
	wantErr := errors.New("api falhou")
	transferStore := &fakeTransferStorage{}
	state := validTransferStateWithItem()
	state.Transfer = &fakeTransferService{err: wantErr}
	state.TransferStore = transferStore

	_, err := SendTransferencia(context.Background(), state)
	if !errors.Is(err, wantErr) {
		t.Fatalf("SendTransferencia() error = %v, want %v", err, wantErr)
	}
	if transferStore.historySaved || transferStore.excelSaved {
		t.Fatal("storage should not be updated when API fails")
	}
}

func TestParseQuantidadeTransferirAcceptsCommaAndDot(t *testing.T) {
	tests := map[string]float64{"10,5": 10.5, "10.5": 10.5, "3": 3}
	for input, want := range tests {
		got, err := ParseQuantidadeTransferir(input)
		if err != nil {
			t.Fatalf("ParseQuantidadeTransferir(%q) error = %v", input, err)
		}
		if got != want {
			t.Fatalf("ParseQuantidadeTransferir(%q) = %v, want %v", input, got, want)
		}
	}

	if _, err := ParseQuantidadeTransferir("abc"); err == nil {
		t.Fatal("ParseQuantidadeTransferir(abc) error = nil, want error")
	}
}

func TestAppropriationHelpers(t *testing.T) {
	appropriations := []models.Apropriacao{{Codigo: "A001", Descricao: "Fundacao"}, {Codigo: "A002"}}
	want := []string{"A001 - Fundacao", "A002"}
	if got := AppropriationLabels(appropriations); !reflect.DeepEqual(got, want) {
		t.Fatalf("AppropriationLabels() = %#v, want %#v", got, want)
	}
	code, description := SplitAppropriationLabel("A001 - Fundacao")
	if code != "A001" || description != "Fundacao" {
		t.Fatalf("SplitAppropriationLabel() = %q/%q, want A001/Fundacao", code, description)
	}
}

func TestClearTransferenciaResetsDefaults(t *testing.T) {
	state := validTransferStateWithItem()
	ClearTransferencia(state)
	if state.Transferencia.CodigoDocumento != "TR" || state.Transferencia.CodigoMovimento != "3" || len(state.Transferencia.Itens) != 0 {
		t.Fatalf("Transferencia after clear = %#v, want defaults", state.Transferencia)
	}
}

func TestBuildTransferenciaTabReturnsObject(t *testing.T) {
	state := NewAppState(testConfig())
	if BuildTransferenciaTab(state) == nil {
		t.Fatal("BuildTransferenciaTab() returned nil")
	}
}

func validTransferState() *AppState {
	state := NewAppState(testConfig())
	state.Transferencia.ObraOrigem = "121 - Residencial Novo Horizonte"
	state.Transferencia.ObraDestino = "205 - Comercial Centro"
	state.Transferencia.Solicitante = "Maria Santos"
	state.Transferencia.CodigoMovimento = "3"
	return state
}

func validTransferStateWithItem() *AppState {
	state := validTransferState()
	state.Transferencia.Itens = []TransferenciaItemState{{
		Insumo:                 models.Insumo{ID: 3421, Nome: "Cimento", Detalhe: "CP III", Marca: "Votorantim", Unidade: "SC"},
		ApropriacaoSelecionada: "A001 - Fundacao",
		QuantidadeDisponivel:   20,
		QuantidadeTransferir:   "10,5",
	}}
	return state
}

type fakeTransferService struct {
	called     bool
	movementID string
	err        error
	transfer   models.Transferencia
}

func (s *fakeTransferService) CreateStockTransfer(ctx context.Context, transfer models.Transferencia) (string, error) {
	s.called = true
	s.transfer = transfer
	if s.err != nil {
		return "", s.err
	}
	return s.movementID, nil
}

type fakeTransferStorage struct {
	historySaved    bool
	excelSaved      bool
	historyTransfer models.Transferencia
	excelTransfer   models.Transferencia
	historyErr      error
	excelErr        error
}

func (s *fakeTransferStorage) AppendHistory(transfer models.Transferencia) error {
	if s.historyErr != nil {
		return s.historyErr
	}
	s.historySaved = true
	s.historyTransfer = transfer
	return nil
}

func (s *fakeTransferStorage) AppendTransferToExcel(transfer models.Transferencia) error {
	if s.excelErr != nil {
		return s.excelErr
	}
	s.excelSaved = true
	s.excelTransfer = transfer
	return nil
}
