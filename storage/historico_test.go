package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"sienge-transfer/models"
)

func TestReadHistoryCreatesEmptyFileWhenMissing(t *testing.T) {
	store := NewStore(t.TempDir())

	history, err := store.ReadHistory()
	if err != nil {
		t.Fatalf("ReadHistory() error = %v", err)
	}
	if len(history) != 0 {
		t.Fatalf("len(history) = %d, want 0", len(history))
	}

	data, err := os.ReadFile(store.HistoricoPath())
	if err != nil {
		t.Fatalf("ReadFile(historico.json) error = %v", err)
	}
	if string(data) != "[]\n" {
		t.Fatalf("historico.json = %q, want []", string(data))
	}
}

func TestAppendHistoryPreservesPreviousRecords(t *testing.T) {
	store := NewStore(t.TempDir())
	first := testTransfer("MOV-1")
	second := testTransfer("MOV-2")

	if err := store.AppendHistory(first); err != nil {
		t.Fatalf("AppendHistory(first) error = %v", err)
	}
	if err := store.AppendHistory(second); err != nil {
		t.Fatalf("AppendHistory(second) error = %v", err)
	}

	history, err := store.ReadHistory()
	if err != nil {
		t.Fatalf("ReadHistory() error = %v", err)
	}
	if len(history) != 2 {
		t.Fatalf("len(history) = %d, want 2", len(history))
	}
	if history[0].IDMovimento != "MOV-1" || history[1].IDMovimento != "MOV-2" {
		t.Fatalf("history order = %#v, want MOV-1 then MOV-2", history)
	}
}

func TestWriteHistoryWithNilWritesEmptyArray(t *testing.T) {
	store := NewStore(t.TempDir())

	if err := store.WriteHistory(nil); err != nil {
		t.Fatalf("WriteHistory(nil) error = %v", err)
	}

	data, err := os.ReadFile(store.HistoricoPath())
	if err != nil {
		t.Fatalf("ReadFile(historico.json) error = %v", err)
	}
	var history []models.Transferencia
	if err := json.Unmarshal(data, &history); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if len(history) != 0 {
		t.Fatalf("len(history) = %d, want 0", len(history))
	}
}

func TestReadHistoryReturnsErrorForCorruptedJSON(t *testing.T) {
	store := NewStore(t.TempDir())
	if err := store.EnsureDir(); err != nil {
		t.Fatalf("EnsureDir() error = %v", err)
	}
	if err := os.WriteFile(store.HistoricoPath(), []byte("{"), 0o600); err != nil {
		t.Fatalf("WriteFile(historico.json) error = %v", err)
	}

	_, err := store.ReadHistory()
	if err == nil {
		t.Fatal("ReadHistory() error = nil, want error")
	}
}

func TestReadHistorySummary(t *testing.T) {
	store := NewStore(t.TempDir())
	transfer := testTransfer("MOV-1")
	if err := store.WriteHistory([]models.Transferencia{transfer}); err != nil {
		t.Fatalf("WriteHistory() error = %v", err)
	}

	summaries, err := store.ReadHistorySummary()
	if err != nil {
		t.Fatalf("ReadHistorySummary() error = %v", err)
	}
	if len(summaries) != 1 {
		t.Fatalf("len(summaries) = %d, want 1", len(summaries))
	}

	want := HistoricoResumo{
		DataHora:        "15/07/2024 10:30:00",
		IDMovimento:     "MOV-1",
		Solicitante:     "Maria Santos",
		ObraOrigem:      "121 - Residencial Novo Horizonte",
		ObraDestino:     "205 - Comercial Centro",
		QuantidadeItens: 2,
		TotalQuantidade: 70.5,
	}
	if !reflect.DeepEqual(summaries[0], want) {
		t.Fatalf("summary = %#v, want %#v", summaries[0], want)
	}
}

func TestStorePathsUseConfiguredDirectory(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)

	if got, want := store.HistoricoPath(), filepath.Join(dir, HistoricoFileName); got != want {
		t.Fatalf("HistoricoPath() = %q, want %q", got, want)
	}
	if got, want := store.ExcelPath(), filepath.Join(dir, TransferenciasExcelFileName); got != want {
		t.Fatalf("ExcelPath() = %q, want %q", got, want)
	}
}

func testTransfer(id string) models.Transferencia {
	return models.Transferencia{
		IDMovimento:         id,
		DataHora:            time.Date(2024, 7, 15, 10, 30, 0, 0, time.Local),
		Usuario:             "Joao Silva",
		Cargo:               "Engenheiro",
		Solicitante:         "Maria Santos",
		ObraOrigemID:        121,
		ObraOrigemNome:      "Residencial Novo Horizonte",
		ObraDestinoID:       205,
		ObraDestinoNome:     "Comercial Centro",
		CodigoTipoDocumento: "TR",
		CodigoTipoMovimento: 3,
		Insumos: []models.ItemTransferido{
			{ID: 3421, Nome: "Cimento", Detalhe: "CP III", Marca: "Votorantim", Apropriacao: "A001", Quantidade: 50},
			{ID: 9876, Nome: "Areia", Detalhe: "Media", Marca: "Regional", Apropriacao: "A002", Quantidade: 20.5},
		},
	}
}
