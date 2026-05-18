package models

import (
	"errors"
	"strconv"
	"testing"
	"time"
)

func TestLoanItem_PendingQuantity(t *testing.T) {
	item := LoanItem{LoanedQuantity: 10, ReturnedQuantity: 4}
	if got := item.PendingQuantity(); got != 6 {
		t.Fatalf("PendingQuantity() = %v, want 6", got)
	}
}

func TestLoanRecord_CalculatesTotals(t *testing.T) {
	loan := testLoanRecord()
	loan.Recalculate()
	if loan.TotalLoanedQuantity != 30 || loan.TotalReturnedQuantity != 5 || loan.ItemCount != 2 {
		t.Fatalf("loan totals = %#v, want 30/5/2", loan)
	}
}

func TestLoanRecord_StatusPendingWhenNoReturn(t *testing.T) {
	loan := testLoanRecord()
	loan.Items[0].ReturnedQuantity = 0
	loan.Recalculate()
	if loan.Status != LoanStatusPending {
		t.Fatalf("Status = %q, want pending", loan.Status)
	}
}

func TestLoanRecord_StatusPartiallyReturned(t *testing.T) {
	loan := testLoanRecord()
	loan.Recalculate()
	if loan.Status != LoanStatusPartiallyReturned {
		t.Fatalf("Status = %q, want partially_returned", loan.Status)
	}
}

func TestLoanRecord_StatusReturned(t *testing.T) {
	loan := testLoanRecord()
	loan.Items[0].ReturnedQuantity = 10
	loan.Items[1].ReturnedQuantity = 20
	loan.Recalculate()
	if loan.Status != LoanStatusReturned {
		t.Fatalf("Status = %q, want returned", loan.Status)
	}
}

func TestMarkLoanReturnedManuallyMarksAllPendingItems(t *testing.T) {
	loan := testLoanRecord()
	returnedAt := time.Date(2024, 7, 20, 10, 0, 0, 0, time.Local)

	updated := MarkLoanReturnedManually(loan, returnedAt)

	if updated.Status != LoanStatusReturned || updated.TotalReturnedQuantity != updated.TotalLoanedQuantity {
		t.Fatalf("updated loan = %#v, want returned with full returned quantity", updated)
	}
	for _, item := range updated.Items {
		if item.ReturnedQuantity != item.LoanedQuantity {
			t.Fatalf("item = %#v, want returned quantity equal loaned quantity", item)
		}
	}
	if updated.LastReturnDate == nil || !updated.LastReturnDate.Equal(returnedAt) {
		t.Fatalf("LastReturnDate = %v, want %v", updated.LastReturnDate, returnedAt)
	}
}

func TestMarkLoanReturnedManuallyDoesNotMutateInput(t *testing.T) {
	loan := testLoanRecord()
	originalReturnedQuantity := loan.Items[1].ReturnedQuantity

	_ = MarkLoanReturnedManually(loan, time.Now())

	if loan.Items[1].ReturnedQuantity != originalReturnedQuantity || loan.Status != LoanStatusPartiallyReturned {
		t.Fatalf("input loan mutated: %#v", loan)
	}
}

func TestCreateLoanRecordFromTransfer_CreatesPendingLoan(t *testing.T) {
	transfer := testLoanTransfer()
	loan := CreateLoanRecordFromTransfer(transfer, 1)
	if loan.Status != LoanStatusPending || loan.Type != TransferKindLoan || loan.OriginalMovementID != "MOV-1" {
		t.Fatalf("loan = %#v, want pending loan from transfer", loan)
	}
}

func TestCreateLoanRecordFromTransfer_CopiesWorkAndSolicitorData(t *testing.T) {
	loan := CreateLoanRecordFromTransfer(testLoanTransfer(), 1)
	if loan.OriginWorkID != 121 || loan.DestinationWorkID != 205 || loan.Solicitor != "Maria" || loan.User != "Joao" {
		t.Fatalf("loan = %#v, want copied transfer data", loan)
	}
}

func TestBuildLoanIDUsesTimestampAndSequence(t *testing.T) {
	transfer := testLoanTransfer()
	want := "EM-" + strconv.FormatInt(transfer.DataHora.UnixNano(), 10) + "-2"
	if got := BuildLoanID(transfer, 2); got != want {
		t.Fatalf("BuildLoanID() = %q, want %q", got, want)
	}
}

func TestBuildLoanIDDefaultsInvalidSequenceToOne(t *testing.T) {
	transfer := testLoanTransfer()
	want := "EM-" + strconv.FormatInt(transfer.DataHora.UnixNano(), 10) + "-1"
	if got := BuildLoanID(transfer, 0); got != want {
		t.Fatalf("BuildLoanID() = %q, want %q", got, want)
	}
}

func TestNextLoanSequenceUsesGlobalEMSuffix(t *testing.T) {
	loans := []LoanRecord{
		{ID: "EM-100-1"},
		{ID: "loan-100-sem-movimento"},
		{ID: "EM-200-3"},
		{ID: "EM-invalid-9"},
	}
	if got := NextLoanSequence(loans); got != 4 {
		t.Fatalf("NextLoanSequence() = %d, want 4", got)
	}
}

func TestApplyReturnToLoan_UpdatesReturnedQuantity(t *testing.T) {
	loan := testLoanRecord()
	loan.Items[0].ReturnedQuantity = 0
	updated, err := ApplyReturnToLoan(loan, testReturnTransfer(4))
	if err != nil {
		t.Fatalf("ApplyReturnToLoan() error = %v", err)
	}
	if updated.Items[0].ReturnedQuantity != 4 {
		t.Fatalf("ReturnedQuantity = %v, want 4", updated.Items[0].ReturnedQuantity)
	}
}

func TestApplyReturnToLoan_DoesNotMutateInput(t *testing.T) {
	loan := testLoanRecord()
	loan.Items[0].ReturnedQuantity = 0
	_, err := ApplyReturnToLoan(loan, testReturnTransfer(4))
	if err != nil {
		t.Fatalf("ApplyReturnToLoan() error = %v", err)
	}
	if loan.Items[0].ReturnedQuantity != 0 {
		t.Fatalf("input ReturnedQuantity = %v, want unchanged", loan.Items[0].ReturnedQuantity)
	}
}

func TestApplyReturnToLoan_UpdatesStatusToReturned(t *testing.T) {
	loan := testLoanRecord()
	loan.Items = loan.Items[:1]
	loan.Items[0].ReturnedQuantity = 0
	updated, err := ApplyReturnToLoan(loan, testReturnTransfer(10))
	if err != nil {
		t.Fatalf("ApplyReturnToLoan() error = %v", err)
	}
	if updated.Status != LoanStatusReturned {
		t.Fatalf("Status = %q, want returned", updated.Status)
	}
}

func TestApplyReturnToLoan_RejectsQuantityAbovePending(t *testing.T) {
	loan := testLoanRecord()
	loan.Items[0].ReturnedQuantity = 0
	_, err := ApplyReturnToLoan(loan, testReturnTransfer(11))
	if err == nil {
		t.Fatal("ApplyReturnToLoan() error = nil, want error")
	}
}

func TestApplyReturnToLoan_RejectsZeroQuantity(t *testing.T) {
	loan := testLoanRecord()
	loan.Items[0].ReturnedQuantity = 0
	_, err := ApplyReturnToLoan(loan, testReturnTransfer(0))
	if err == nil {
		t.Fatal("ApplyReturnToLoan() error = nil, want zero quantity error")
	}
}

func TestValidateReturnAgainstLoan_IgnoresManualReturn(t *testing.T) {
	transfer := testReturnTransfer(999)
	transfer.LinkedLoanID = ""
	if err := ValidateReturnAgainstLoan(testLoanRecord(), transfer); err != nil {
		t.Fatalf("ValidateReturnAgainstLoan() error = %v, want nil for manual return", err)
	}
}

func TestErrLoanNotFoundExists(t *testing.T) {
	if !errors.Is(ErrLoanNotFound, ErrLoanNotFound) {
		t.Fatal("ErrLoanNotFound should be comparable with errors.Is")
	}
}

func testLoanRecord() LoanRecord {
	detailID := 10
	brandID := 5
	loan := LoanRecord{
		ID:                  "loan-1",
		OriginalMovementID:  "MOV-1",
		LoanDate:            time.Date(2024, 7, 15, 10, 0, 0, 0, time.Local),
		OriginWorkID:        121,
		OriginWorkName:      "Origem",
		DestinationWorkID:   205,
		DestinationWorkName: "Destino",
		Solicitor:           "Maria",
		User:                "Joao",
		Type:                TransferKindLoan,
		Items: []LoanItem{
			{ResourceID: 3421, ResourceName: "Cimento", DetailID: &detailID, BrandID: &brandID, Unit: "SC", LoanedQuantity: 10, ReturnedQuantity: 5},
			{ResourceID: 9876, ResourceName: "Areia", Unit: "M3", LoanedQuantity: 20},
		},
	}
	loan.Recalculate()
	return loan
}

func testLoanTransfer() Transferencia {
	return Transferencia{
		IDMovimento:         "MOV-1",
		DataHora:            time.Date(2024, 7, 15, 10, 0, 0, 0, time.Local),
		Usuario:             "Joao",
		Cargo:               "Engenheiro",
		Solicitante:         "Maria",
		ObraOrigemID:        121,
		ObraOrigemNome:      "Origem",
		ObraDestinoID:       205,
		ObraDestinoNome:     "Destino",
		TransferKind:        TransferKindLoan,
		CodigoTipoDocumento: "TR",
		CodigoTipoMovimento: 3,
		Insumos:             []ItemTransferido{{ID: 3421, Nome: "Cimento", DetalheID: 10, MarcaID: 5, Unidade: "SC", Quantidade: 10, PrecoUnitario: 1}},
	}
}

func testReturnTransfer(quantity float64) Transferencia {
	return Transferencia{
		IDMovimento:  "RET-1",
		DataHora:     time.Date(2024, 7, 16, 10, 0, 0, 0, time.Local),
		TransferKind: TransferKindReturn,
		LinkedLoanID: "loan-1",
		Insumos:      []ItemTransferido{{ID: 3421, DetalheID: 10, MarcaID: 5, Quantidade: quantity}},
	}
}
