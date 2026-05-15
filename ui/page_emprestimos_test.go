package ui

import (
	"errors"
	"testing"
	"time"

	"sienge-transfer/models"
)

func TestBuildLoanTableRows_DefaultFiltersPendingAndPartial(t *testing.T) {
	rows := BuildLoanTableRows(uiTestLoans(), LoanTableFilter{ShowPending: true, ShowPartial: true})
	if len(rows) != 2 {
		t.Fatalf("len(rows) = %d, want 2", len(rows))
	}
}

func TestBuildLoanTableRows_FilterByMultipleStatuses(t *testing.T) {
	rows := BuildLoanTableRows(uiTestLoans(), LoanTableFilter{ShowPending: true, ShowReturned: true})
	if len(rows) != 2 {
		t.Fatalf("len(rows) = %d, want 2", len(rows))
	}
}

func TestBuildLoanTableRows_SearchesAnyVisibleField(t *testing.T) {
	rows := BuildLoanTableRows(uiTestLoans(), LoanTableFilter{Search: "ana", ShowPending: true, ShowPartial: true, ShowReturned: true})
	if len(rows) != 1 || rows[0].Solicitor != "Ana" {
		t.Fatalf("rows = %#v, want Ana row", rows)
	}
}

func TestBuildLoanTableRows_SearchesLoanID(t *testing.T) {
	rows := BuildLoanTableRows(uiTestLoans(), LoanTableFilter{Search: "loan-2", ShowPending: true, ShowPartial: true, ShowReturned: true})
	if len(rows) != 1 || rows[0].ID != "loan-2" {
		t.Fatalf("rows = %#v, want loan-2 row", rows)
	}
}

func TestBuildLoanTableRows_SearchesOriginWork(t *testing.T) {
	rows := BuildLoanTableRows(uiTestLoans(), LoanTableFilter{Search: "origem", ShowPending: true, ShowPartial: true, ShowReturned: true})
	if len(rows) != 3 {
		t.Fatalf("len(rows) = %d, want 3", len(rows))
	}
}

func TestLoanTableColumnsStartsWithLoanID(t *testing.T) {
	columns := LoanTableColumns()
	if len(columns) == 0 {
		t.Fatal("LoanTableColumns() returned no columns")
	}
	if columns[0] != "ID Emprestimo" {
		t.Fatalf("LoanTableColumns()[0] = %q, want ID Emprestimo", columns[0])
	}
	if columns[1] != "Obra origem" {
		t.Fatalf("LoanTableColumns()[1] = %q, want Obra origem", columns[1])
	}
}

func TestLoanTableCellValueMapsLoanIDFirst(t *testing.T) {
	row := LoanTableRowFromRecord(uiTestLoans()[0])
	if got := LoanTableCellValue(row, 0); got != row.ID {
		t.Fatalf("LoanTableCellValue(col 0) = %q, want %q", got, row.ID)
	}
}

func TestLoanTableCellValueMapsOriginWork(t *testing.T) {
	row := LoanTableRowFromRecord(uiTestLoans()[0])
	if got := LoanTableCellValue(row, 1); got != "121 - Origem" {
		t.Fatalf("LoanTableCellValue(col 1) = %q, want 121 - Origem", got)
	}
}

func TestBuildLoanTableRows_StatusColors(t *testing.T) {
	if LoanStatusColor(models.LoanStatusPending) != "#DC2626" || LoanStatusColor(models.LoanStatusPartiallyReturned) != "#2563EB" || LoanStatusColor(models.LoanStatusReturned) != "#16A34A" {
		t.Fatal("loan status colors mismatch")
	}
}

func TestBuildLoanTableRows_SortsByLoanDateDescending(t *testing.T) {
	older := uiTestLoans()[0]
	older.ID = "older"
	older.LoanDate = time.Date(2024, 1, 15, 10, 0, 0, 0, time.Local)
	newer := uiTestLoans()[1]
	newer.ID = "newer"
	newer.LoanDate = time.Date(2024, 12, 1, 10, 0, 0, 0, time.Local)

	rows := BuildLoanTableRows([]models.LoanRecord{older, newer}, LoanTableFilter{ShowPending: true, ShowPartial: true})

	if len(rows) != 2 || rows[0].ID != "newer" {
		t.Fatalf("rows = %#v, want newer loan first", rows)
	}
}

func TestBuildLoanTableRows_ReturnButtonOnlyForPendingOrPartial(t *testing.T) {
	rows := BuildLoanTableRows(uiTestLoans(), LoanTableFilter{ShowPending: true, ShowPartial: true, ShowReturned: true})
	for _, row := range rows {
		if row.Status == models.LoanStatusLabel(models.LoanStatusReturned) && row.CanReturn {
			t.Fatalf("returned row should not allow return: %#v", row)
		}
	}
}

func TestLoanReturnSelection_SelectAll(t *testing.T) {
	selection := ToggleLoanReturnSelectAll(NewLoanReturnSelectionState(uiTestLoans()[0].Items), true)
	if len(SelectedLoanReturnItems(selection)) != 1 || !selection.SelectAll {
		t.Fatalf("selection = %#v, want all selected", selection)
	}
}

func TestLoanReturnSelection_UnselectOneUnchecksSelectAll(t *testing.T) {
	loan := uiTestLoans()[0]
	loan.Items = append(loan.Items, models.LoanItem{ResourceID: 9999, LoanedQuantity: 1})
	selection := ToggleLoanReturnSelectAll(NewLoanReturnSelectionState(loan.Items), true)
	selection = ToggleLoanReturnItem(selection, 0, false)
	if selection.SelectAll {
		t.Fatal("SelectAll = true, want false after unselecting one item")
	}
}

func TestPrepareTransferReturnFromLoanPrefillsTransferData(t *testing.T) {
	state := NewAppState(testConfig())
	loan := uiTestLoans()[0]
	PrepareTransferReturnFromLoan(state, loan, loan.PendingItems())
	if state.Transferencia.TransferKind != models.TransferKindReturn || state.Transferencia.SelectedLoanID != loan.ID {
		t.Fatalf("transfer loan data = %#v, want linked return", state.Transferencia)
	}
	if state.Transferencia.ObraOrigem == "" || state.Transferencia.ObraDestino == "" || len(state.Transferencia.Itens) != 1 {
		t.Fatalf("transfer prefill = %#v, want works and item", state.Transferencia)
	}
}

func TestLoanReturnOptionLabelIncludesLoanID(t *testing.T) {
	loan := uiTestLoans()[0]
	label := LoanReturnOptionLabel(loan)
	want := "loan-1 | 205 - Destino | Maria | 15/07/2024"
	if label != want {
		t.Fatalf("LoanReturnOptionLabel() = %q, want %q", label, want)
	}
}

func TestLoanByReturnOptionLabelResolvesLabelWithLoanID(t *testing.T) {
	loans := uiTestLoans()
	loan, ok := LoanByReturnOptionLabel(loans, LoanReturnOptionLabel(loans[1]))
	if !ok || loan.ID != "loan-2" {
		t.Fatalf("LoanByReturnOptionLabel() = %#v/%v, want loan-2/true", loan, ok)
	}
}

func TestBuildEmprestimosTabDoesNotLoadLoans(t *testing.T) {
	state := NewAppState(testConfig())
	store := &countingLoanStore{fakeLoanStore: fakeLoanStore{err: errors.New("falha ao carregar emprestimos")}}
	state.LoanStore = store

	if BuildEmprestimosTab(state) == nil {
		t.Fatal("BuildEmprestimosTab() returned nil")
	}
	if store.listCalls != 0 {
		t.Fatalf("ListLoans calls = %d, want 0", store.listCalls)
	}
}

func TestRefreshReturnLoansForTransferReturnsListError(t *testing.T) {
	state := NewAppState(testConfig())
	state.LoanStore = &fakeLoanStore{err: errors.New("falha ao carregar emprestimos")}

	if err := RefreshReturnLoansForTransfer(state); err == nil {
		t.Fatal("RefreshReturnLoansForTransfer() error = nil, want list error")
	}
}

func TestMarkLoanAsReturnedManuallyPersistsReturnedLoan(t *testing.T) {
	state := NewAppState(testConfig())
	loanStore := &fakeLoanStore{loans: []models.LoanRecord{uiTestLoans()[0]}}
	state.LoanStore = loanStore
	returnedAt := time.Date(2024, 7, 20, 10, 0, 0, 0, time.Local)

	if err := MarkLoanAsReturnedManually(state, "loan-1", returnedAt); err != nil {
		t.Fatalf("MarkLoanAsReturnedManually() error = %v", err)
	}
	updated := loanStore.loans[0]
	if updated.Status != models.LoanStatusReturned || updated.Items[0].ReturnedQuantity != updated.Items[0].LoanedQuantity {
		t.Fatalf("updated loan = %#v, want returned", updated)
	}
	if updated.LastReturnDate == nil || !updated.LastReturnDate.Equal(returnedAt) {
		t.Fatalf("LastReturnDate = %v, want %v", updated.LastReturnDate, returnedAt)
	}
	if len(state.Emprestimos.Loans) != 1 || state.Emprestimos.Loans[0].Status != models.LoanStatusReturned {
		t.Fatalf("state loans = %#v, want refreshed returned loan", state.Emprestimos.Loans)
	}
}

type countingLoanStore struct {
	fakeLoanStore
	listCalls int
}

func (s *countingLoanStore) ListLoans() ([]models.LoanRecord, error) {
	s.listCalls++
	return s.fakeLoanStore.ListLoans()
}

func uiTestLoans() []models.LoanRecord {
	makeLoan := func(id string, solicitor string, status models.LoanStatus, returned float64) models.LoanRecord {
		loan := models.LoanRecord{ID: id, LoanDate: time.Date(2024, 7, 15, 10, 0, 0, 0, time.Local), DestinationWorkID: 205, DestinationWorkName: "Destino", OriginWorkID: 121, OriginWorkName: "Origem", Solicitor: solicitor, Type: models.TransferKindLoan, Items: []models.LoanItem{{ResourceID: 3421, ResourceName: "Cimento", Unit: "SC", LoanedQuantity: 10, ReturnedQuantity: returned}}}
		loan.Recalculate()
		loan.Status = status
		return loan
	}
	return []models.LoanRecord{
		makeLoan("loan-1", "Maria", models.LoanStatusPending, 0),
		makeLoan("loan-2", "Ana", models.LoanStatusPartiallyReturned, 5),
		makeLoan("loan-3", "Carlos", models.LoanStatusReturned, 10),
	}
}
