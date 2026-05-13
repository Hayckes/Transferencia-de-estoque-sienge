package ui

import (
	"errors"
	"strings"
	"testing"

	"sienge-transfer/storage"
)

func TestRefreshHistoricoLoadsSummaries(t *testing.T) {
	historyStore := &fakeHistoryStore{summaries: []storage.HistoricoResumo{
		{DataHora: "15/07/2024 10:30:00", IDMovimento: "MOV-1", Solicitante: "Maria", ObraOrigem: "121 - A", ObraDestino: "205 - B", QuantidadeItens: 2, TotalQuantidade: 70.5},
	}}
	state := NewAppState(testConfig())
	state.HistoryStore = historyStore

	if err := RefreshHistorico(state); err != nil {
		t.Fatalf("RefreshHistorico() error = %v", err)
	}
	if !historyStore.readCalled {
		t.Fatal("ReadHistorySummary() was not called")
	}
	if len(state.Historico.Resumos) != 1 || state.Historico.Resumos[0].IDMovimento != "MOV-1" {
		t.Fatalf("Resumos = %#v, want MOV-1", state.Historico.Resumos)
	}
	if state.Historico.UltimoStatus != "Historico atualizado." {
		t.Fatalf("UltimoStatus = %q, want updated", state.Historico.UltimoStatus)
	}
}

func TestRefreshHistoricoRequiresStoreAndPreservesPreviousOnError(t *testing.T) {
	state := NewAppState(testConfig())
	err := RefreshHistorico(state)
	if err == nil || !strings.Contains(err.Error(), "nao configurado") {
		t.Fatalf("RefreshHistorico() error = %v, want not configured", err)
	}

	wantErr := errors.New("json corrompido")
	state.HistoryStore = &fakeHistoryStore{err: wantErr}
	state.Historico.Resumos = []storage.HistoricoResumo{{IDMovimento: "ANTERIOR"}}
	err = RefreshHistorico(state)
	if !errors.Is(err, wantErr) {
		t.Fatalf("RefreshHistorico() error = %v, want %v", err, wantErr)
	}
	if len(state.Historico.Resumos) != 1 || state.Historico.Resumos[0].IDMovimento != "ANTERIOR" {
		t.Fatalf("previous summaries should be preserved: %#v", state.Historico.Resumos)
	}
}

func TestOpenHistoricoExcelRecreatesAndOpensFile(t *testing.T) {
	historyStore := &fakeHistoryStore{excelPath: "C:/tmp/transferencias.xlsx"}
	opener := &fakeFileOpener{}
	state := NewAppState(testConfig())
	state.HistoryStore = historyStore
	state.FileOpener = opener

	if err := OpenHistoricoExcel(state); err != nil {
		t.Fatalf("OpenHistoricoExcel() error = %v", err)
	}
	if !historyStore.ensureExcelCalled {
		t.Fatal("EnsureExcelFromHistory() was not called")
	}
	if !opener.called || opener.path != historyStore.excelPath {
		t.Fatalf("opener = called %v path %q, want %q", opener.called, opener.path, historyStore.excelPath)
	}
}

func TestOpenHistoricoExcelDoesNotOpenWhenRecreateFails(t *testing.T) {
	wantErr := errors.New("falha ao recriar")
	historyStore := &fakeHistoryStore{ensureErr: wantErr, excelPath: "x.xlsx"}
	opener := &fakeFileOpener{}
	state := NewAppState(testConfig())
	state.HistoryStore = historyStore
	state.FileOpener = opener

	err := OpenHistoricoExcel(state)
	if !errors.Is(err, wantErr) {
		t.Fatalf("OpenHistoricoExcel() error = %v, want %v", err, wantErr)
	}
	if opener.called {
		t.Fatal("file should not be opened when EnsureExcelFromHistory fails")
	}
}

func TestOpenHistoricoExcelRequiresDependencies(t *testing.T) {
	state := NewAppState(testConfig())
	if err := OpenHistoricoExcel(state); err == nil || !strings.Contains(err.Error(), "historico nao configurado") {
		t.Fatalf("OpenHistoricoExcel() error = %v, want history store error", err)
	}

	state.HistoryStore = &fakeHistoryStore{}
	if err := OpenHistoricoExcel(state); err == nil || !strings.Contains(err.Error(), "abridor de arquivos nao configurado") {
		t.Fatalf("OpenHistoricoExcel() error = %v, want opener error", err)
	}
}

func TestHistoricoResumoRow(t *testing.T) {
	row := HistoricoResumoRow(storage.HistoricoResumo{
		DataHora:        "15/07/2024 10:30:00",
		IDMovimento:     "MOV-1",
		Solicitante:     "Maria",
		ObraOrigem:      "121 - A",
		ObraDestino:     "205 - B",
		QuantidadeItens: 2,
		TotalQuantidade: 70.5,
	})

	for _, want := range []string{"15/07/2024", "MOV-1", "Maria", "121 - A", "205 - B", "2", "70.5"} {
		if !strings.Contains(row, want) {
			t.Fatalf("HistoricoResumoRow() = %q, want containing %q", row, want)
		}
	}
}

func TestBuildHistoricoTabReturnsObject(t *testing.T) {
	state := NewAppState(testConfig())
	state.Historico.Resumos = []storage.HistoricoResumo{{IDMovimento: "MOV-1"}}
	if BuildHistoricoTab(state) == nil {
		t.Fatal("BuildHistoricoTab() returned nil")
	}
}

func TestSendTransferenciaRefreshesHistoryAfterSuccess(t *testing.T) {
	historyStore := &fakeCombinedTransferHistoryStorage{
		fakeHistoryStore: fakeHistoryStore{
			summaries: []storage.HistoricoResumo{{IDMovimento: "MOV-1"}},
			excelPath: "x.xlsx",
		},
	}
	state := validTransferStateWithItem()
	state.Transfer = &fakeTransferService{movementID: "MOV-1"}
	state.TransferStore = historyStore
	state.HistoryStore = historyStore

	if _, err := SendTransferencia(nil, state); err != nil {
		t.Fatalf("SendTransferencia() error = %v", err)
	}
	if !historyStore.readCalled {
		t.Fatal("history should be refreshed after successful transfer")
	}
	if len(state.Historico.Resumos) != 1 || state.Historico.Resumos[0].IDMovimento != "MOV-1" {
		t.Fatalf("Historico.Resumos = %#v, want MOV-1", state.Historico.Resumos)
	}
}

type fakeHistoryStore struct {
	summaries         []storage.HistoricoResumo
	err               error
	ensureErr         error
	excelPath         string
	readCalled        bool
	ensureExcelCalled bool
}

func (s *fakeHistoryStore) ReadHistorySummary() ([]storage.HistoricoResumo, error) {
	s.readCalled = true
	if s.err != nil {
		return nil, s.err
	}
	return append([]storage.HistoricoResumo(nil), s.summaries...), nil
}

func (s *fakeHistoryStore) EnsureExcelFromHistory() error {
	s.ensureExcelCalled = true
	return s.ensureErr
}

func (s *fakeHistoryStore) ExcelPath() string {
	return s.excelPath
}

type fakeFileOpener struct {
	called bool
	path   string
	err    error
}

func (o *fakeFileOpener) Open(path string) error {
	o.called = true
	o.path = path
	return o.err
}

type fakeCombinedTransferHistoryStorage struct {
	fakeTransferStorage
	fakeHistoryStore
}
