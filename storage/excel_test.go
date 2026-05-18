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

	if got := mustCell(t, file, 1, 1); got != "ID de Transferencia" {
		t.Fatalf("header A1 = %q, want ID de Transferencia", got)
	}
	if got := mustCell(t, file, 2, 1); got != "1" {
		t.Fatalf("row 2 transfer ID = %q, want 1", got)
	}
	if got := mustCell(t, file, 2, 2); got != "15/07/2024 10:30:00" {
		t.Fatalf("row 2 date = %q, want 15/07/2024 10:30:00", got)
	}
	if got := mustCell(t, file, 2, 6); got != "Observacao de teste" {
		t.Fatalf("row 2 observation = %q, want Observacao de teste", got)
	}
	if got := mustCell(t, file, 2, 9); got != "A001" {
		t.Fatalf("row 2 origin appropriation = %q, want A001", got)
	}
	if got := mustCell(t, file, 2, 24); got != "3421" {
		t.Fatalf("row 2 supply ID = %q, want 3421", got)
	}
	if got := mustCell(t, file, 3, 24); got != "9876" {
		t.Fatalf("row 3 supply ID = %q, want 9876", got)
	}
	if got := mustCell(t, file, 3, 17); got != "D002" {
		t.Fatalf("row 3 destination appropriation = %q, want D002", got)
	}
	if got := mustCell(t, file, 3, 11); got != "20.5000 M3" {
		t.Fatalf("row 3 quantity = %q, want 20.5000 M3", got)
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
	if got := mustCell(t, file, 4, 1); got != "2" {
		t.Fatalf("row 4 transfer ID = %q, want 2", got)
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

	if got := mustCell(t, file, 2, 2); got != "15/07/2024 10:30:00" {
		t.Fatalf("row 2 date = %q, want 15/07/2024 10:30:00", got)
	}
}

func TestEnsureExcelFromHistoryRebuildsWhenHeadersChange(t *testing.T) {
	store := NewStore(t.TempDir())
	if err := store.WriteHistory([]models.Transferencia{testTransfer("MOV-1")}); err != nil {
		t.Fatalf("WriteHistory() error = %v", err)
	}
	if err := store.RebuildExcel([]models.Transferencia{testTransfer("MOV-1")}); err != nil {
		t.Fatalf("RebuildExcel() error = %v", err)
	}

	file := openExcel(t, store.ExcelPath())
	extraHeaderCell, err := excelize.CoordinatesToCellName(len(ExcelHeaders)+1, 1)
	if err != nil {
		t.Fatalf("CoordinatesToCellName() error = %v", err)
	}
	if err := file.SetCellValue(excelSheetName, extraHeaderCell, "Coluna Removida"); err != nil {
		t.Fatalf("SetCellValue(extra header) error = %v", err)
	}
	if err := file.Save(); err != nil {
		t.Fatalf("Save(stale excel) error = %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("Close(stale excel) error = %v", err)
	}

	if err := store.EnsureExcelFromHistory(); err != nil {
		t.Fatalf("EnsureExcelFromHistory() error = %v", err)
	}

	rebuilt := openExcel(t, store.ExcelPath())
	defer rebuilt.Close()
	rows, err := rebuilt.GetRows(excelSheetName)
	if err != nil {
		t.Fatalf("GetRows() error = %v", err)
	}
	if len(rows) == 0 || len(rows[0]) != len(ExcelHeaders) {
		t.Fatalf("header columns = %d, want %d", len(rows[0]), len(ExcelHeaders))
	}
	if got := mustCell(t, rebuilt, 2, 2); got != "15/07/2024 10:30:00" {
		t.Fatalf("row 2 date = %q, want 15/07/2024 10:30:00", got)
	}
}

func TestAppendTransferToExcelCreatesFileWhenMissing(t *testing.T) {
	store := NewStore(t.TempDir())

	if err := store.AppendTransferToExcel(testTransfer("MOV-1")); err != nil {
		t.Fatalf("AppendTransferToExcel() error = %v", err)
	}

	file := openExcel(t, store.ExcelPath())
	defer file.Close()

	if got := mustCell(t, file, 1, 1); got != "ID de Transferencia" {
		t.Fatalf("header = %q, want ID de Transferencia", got)
	}
	if got := mustCell(t, file, 2, 2); got != "15/07/2024 10:30:00" {
		t.Fatalf("row 2 date = %q, want 15/07/2024 10:30:00", got)
	}
}

func TestRebuildExcelColorsLoanStatusCells(t *testing.T) {
	store := NewStore(t.TempDir())
	transfers := []models.Transferencia{
		testTransferWithLoanStatus("MOV-1", models.LoanStatusPending),
		testTransferWithLoanStatus("MOV-2", models.LoanStatusReturned),
		testTransferWithLoanStatus("MOV-3", models.LoanStatusPartiallyReturned),
	}

	if err := store.RebuildExcel(transfers); err != nil {
		t.Fatalf("RebuildExcel() error = %v", err)
	}

	file := openExcel(t, store.ExcelPath())
	defer file.Close()

	assertCellFillColor(t, file, 2, len(ExcelHeaders), "FF0000")
	assertCellFillColor(t, file, 3, len(ExcelHeaders), "00B050")
	assertCellFillColor(t, file, 4, len(ExcelHeaders), "0070C0")
}

func TestEnsureExcelFromHistoryRebuildsWhenLoanStatusStyleMissing(t *testing.T) {
	store := NewStore(t.TempDir())
	transfer := testTransferWithLoanStatus("MOV-1", models.LoanStatusPending)
	if err := store.WriteHistory([]models.Transferencia{transfer}); err != nil {
		t.Fatalf("WriteHistory() error = %v", err)
	}
	if err := store.RebuildExcel([]models.Transferencia{transfer}); err != nil {
		t.Fatalf("RebuildExcel() error = %v", err)
	}

	file := openExcel(t, store.ExcelPath())
	cell, err := excelize.CoordinatesToCellName(len(ExcelHeaders), 2)
	if err != nil {
		t.Fatalf("CoordinatesToCellName() error = %v", err)
	}
	if err := file.SetCellStyle(excelSheetName, cell, cell, 0); err != nil {
		t.Fatalf("SetCellStyle(clear) error = %v", err)
	}
	if err := file.Save(); err != nil {
		t.Fatalf("Save(unstyled excel) error = %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("Close(unstyled excel) error = %v", err)
	}

	if err := store.EnsureExcelFromHistory(); err != nil {
		t.Fatalf("EnsureExcelFromHistory() error = %v", err)
	}

	rebuilt := openExcel(t, store.ExcelPath())
	defer rebuilt.Close()
	assertCellFillColor(t, rebuilt, 2, len(ExcelHeaders), "FF0000")
}

func TestLoanStatusExcelFillColor(t *testing.T) {
	tests := []struct {
		status string
		want   string
		ok     bool
	}{
		{status: "Pendente", want: "FF0000", ok: true},
		{status: "Devolvido", want: "00B050", ok: true},
		{status: "Parcialmente devolvido", want: "0070C0", ok: true},
		{status: "Nao se aplica", ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			got, ok := loanStatusExcelFillColor(tt.status)
			if ok != tt.ok || got != tt.want {
				t.Fatalf("loanStatusExcelFillColor(%q) = %q/%v, want %q/%v", tt.status, got, ok, tt.want, tt.ok)
			}
		})
	}
}

func TestExcelHeadersMatchExpectedColumns(t *testing.T) {
	want := []string{
		"ID de Transferencia",
		"Data/Hora",
		"Usuario",
		"Cargo",
		"Solicitante",
		"Observacao",
		"ID Obra Origem",
		"Nome Obra Origem",
		"Apropriacao Origem",
		"Quantidade Origem no Momento da Transferencia",
		"Quantidade Enviada",
		"Quantidade Origem Apos Transferencia",
		"Quantidade Apropriacao Origem no Momento da Transferencia",
		"Quantidade Apropriacao Origem Apos Transferencia",
		"ID Obra Destino",
		"Nome Obra Destino",
		"Apropriacao Destino Codigo",
		"Apropriacao Destino Descricao",
		"Quantidade Destino no Momento da Transferencia",
		"Quantidade Recebida",
		"Quantidade Destino Apos Transferencia",
		"Quantidade Apropriacao Destino no Momento da Transferencia",
		"Quantidade Apropriacao Destino Apos Transferencia",
		"Insumo ID",
		"Nome do Insumo",
		"Detalhe",
		"Marca",
		"Unidade",
		"Tipo da Transferencia",
		"Status do Emprestimo",
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

func assertCellFillColor(t *testing.T, file *excelize.File, row, column int, want string) {
	t.Helper()

	cell, err := excelize.CoordinatesToCellName(column, row)
	if err != nil {
		t.Fatalf("CoordinatesToCellName(%d, %d) error = %v", column, row, err)
	}
	styleID, err := file.GetCellStyle(excelSheetName, cell)
	if err != nil {
		t.Fatalf("GetCellStyle(%s) error = %v", cell, err)
	}
	style, err := file.GetStyle(styleID)
	if err != nil {
		t.Fatalf("GetStyle(%d) error = %v", styleID, err)
	}
	if !excelStyleHasFillColor(style, want) {
		t.Fatalf("cell %s fill = %#v, want %s", cell, style.Fill, want)
	}
}

func testTransferWithLoanStatus(id string, status models.LoanStatus) models.Transferencia {
	transfer := testTransfer(id)
	transfer.TransferKind = models.TransferKindLoan
	transfer.LoanStatus = status
	transfer.Insumos = transfer.Insumos[:1]
	return transfer
}
