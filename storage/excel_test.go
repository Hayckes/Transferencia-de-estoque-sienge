package storage

import (
	"os"
	"testing"

	"github.com/xuri/excelize/v2"

	"sienge-transfer/models"
)

func TestRebuildExcelCreatesFileWithHeadersAndRows(t *testing.T) {
	store := NewStore(t.TempDir())
	transfer := testTransfer("MOV-1")

	if err := store.RebuildExcel([]models.Transferencia{transfer}); err != nil {
		t.Fatalf("RebuildExcel() error = %v", err)
	}

	file := openExcel(t, store.ExcelPath())
	defer file.Close()

	if got := mustCell(t, file, 1, 1); got != "ID Movimento Sienge" {
		t.Fatalf("header A1 = %q, want ID Movimento Sienge", got)
	}
	if got := mustCell(t, file, 2, 1); got != "MOV-1" {
		t.Fatalf("row 2 movement = %q, want MOV-1", got)
	}
	if got := mustCell(t, file, 2, 8); got != "3421" {
		t.Fatalf("row 2 supply ID = %q, want 3421", got)
	}
	if got := mustCell(t, file, 3, 8); got != "9876" {
		t.Fatalf("row 3 supply ID = %q, want 9876", got)
	}
	if got := mustCell(t, file, 3, 13); got != "20.5" {
		t.Fatalf("row 3 quantity = %q, want 20.5", got)
	}
}

func TestAppendTransferToExcelAddsRowsToExistingFile(t *testing.T) {
	store := NewStore(t.TempDir())
	if err := store.RebuildExcel([]models.Transferencia{testTransfer("MOV-1")}); err != nil {
		t.Fatalf("RebuildExcel() error = %v", err)
	}

	if err := store.AppendTransferToExcel(testTransfer("MOV-2")); err != nil {
		t.Fatalf("AppendTransferToExcel() error = %v", err)
	}

	file := openExcel(t, store.ExcelPath())
	defer file.Close()

	rows, err := file.GetRows(excelSheetName)
	if err != nil {
		t.Fatalf("GetRows() error = %v", err)
	}
	if len(rows) != 5 {
		t.Fatalf("len(rows) = %d, want 5", len(rows))
	}
	if got := mustCell(t, file, 4, 1); got != "MOV-2" {
		t.Fatalf("row 4 movement = %q, want MOV-2", got)
	}
}

func TestEnsureExcelFromHistoryRecreatesDeletedExcel(t *testing.T) {
	store := NewStore(t.TempDir())
	if err := store.WriteHistory([]models.Transferencia{testTransfer("MOV-1")}); err != nil {
		t.Fatalf("WriteHistory() error = %v", err)
	}
	if err := os.Remove(store.ExcelPath()); err != nil && !os.IsNotExist(err) {
		t.Fatalf("Remove(excel) error = %v", err)
	}

	if err := store.EnsureExcelFromHistory(); err != nil {
		t.Fatalf("EnsureExcelFromHistory() error = %v", err)
	}

	file := openExcel(t, store.ExcelPath())
	defer file.Close()

	if got := mustCell(t, file, 2, 1); got != "MOV-1" {
		t.Fatalf("row 2 movement = %q, want MOV-1", got)
	}
}

func TestAppendTransferToExcelCreatesFileWhenMissing(t *testing.T) {
	store := NewStore(t.TempDir())

	if err := store.AppendTransferToExcel(testTransfer("MOV-1")); err != nil {
		t.Fatalf("AppendTransferToExcel() error = %v", err)
	}

	file := openExcel(t, store.ExcelPath())
	defer file.Close()

	if got := mustCell(t, file, 1, 1); got != "ID Movimento Sienge" {
		t.Fatalf("header = %q, want ID Movimento Sienge", got)
	}
	if got := mustCell(t, file, 2, 1); got != "MOV-1" {
		t.Fatalf("row 2 movement = %q, want MOV-1", got)
	}
}

func TestExcelHeadersMatchExpectedColumns(t *testing.T) {
	want := []string{
		"ID Movimento Sienge",
		"Data e Hora",
		"Usuario",
		"Cargo",
		"Solicitante",
		"Obra Origem",
		"Obra Destino",
		"ID Insumo",
		"Nome Insumo",
		"Detalhe",
		"Marca",
		"Apropriacao",
		"Quantidade",
	}

	if len(ExcelHeaders) != len(want) {
		t.Fatalf("len(ExcelHeaders) = %d, want %d", len(ExcelHeaders), len(want))
	}
	for index := range want {
		if ExcelHeaders[index] != want[index] {
			t.Fatalf("ExcelHeaders[%d] = %q, want %q", index, ExcelHeaders[index], want[index])
		}
	}
}

func openExcel(t *testing.T, path string) *excelize.File {
	t.Helper()

	file, err := excelize.OpenFile(path)
	if err != nil {
		t.Fatalf("OpenFile(%q) error = %v", path, err)
	}

	return file
}

func mustCell(t *testing.T, file *excelize.File, row, column int) string {
	t.Helper()

	value, err := readExcelCell(file, row, column)
	if err != nil {
		t.Fatalf("readExcelCell(%d, %d) error = %v", row, column, err)
	}

	return value
}
