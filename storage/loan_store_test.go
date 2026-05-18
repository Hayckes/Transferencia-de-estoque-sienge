package storage

import (
	"errors"
	"testing"

	"sienge-transfer/models"
)

func TestLoanStore_ReturnsEmptyWhenFileDoesNotExist(t *testing.T) {
	loans, err := NewStore(t.TempDir()).ListLoans()
	if err != nil {
		t.Fatalf("ListLoans() error = %v", err)
	}
	if len(loans) != 0 {
		t.Fatalf("len(loans) = %d, want 0", len(loans))
	}
}

func TestLoanStore_UpsertCreatesLoan(t *testing.T) {
	store := NewStore(t.TempDir())
	loan := storageTestLoan()
	if err := store.UpsertLoan(loan); err != nil {
		t.Fatalf("UpsertLoan() error = %v", err)
	}
	loans, err := store.ListLoans()
	if err != nil {
		t.Fatalf("ListLoans() error = %v", err)
	}
	if len(loans) != 1 || loans[0].ID != loan.ID {
		t.Fatalf("loans = %#v, want created loan", loans)
	}
}

func TestLoanStore_UpsertUpdatesLoan(t *testing.T) {
	store := NewStore(t.TempDir())
	loan := storageTestLoan()
	if err := store.UpsertLoan(loan); err != nil {
		t.Fatalf("UpsertLoan(create) error = %v", err)
	}
	loan.Solicitor = "Novo"
	if err := store.UpsertLoan(loan); err != nil {
		t.Fatalf("UpsertLoan(update) error = %v", err)
	}
	loaded, err := store.GetLoanByID(loan.ID)
	if err != nil {
		t.Fatalf("GetLoanByID() error = %v", err)
	}
	if loaded.Solicitor != "Novo" {
		t.Fatalf("Solicitor = %q, want Novo", loaded.Solicitor)
	}
}

func TestLoanStore_GetByIDReturnsNotFound(t *testing.T) {
	_, err := NewStore(t.TempDir()).GetLoanByID("missing")
	if !errors.Is(err, models.ErrLoanNotFound) {
		t.Fatalf("GetLoanByID() error = %v, want ErrLoanNotFound", err)
	}
}

func TestLoanStore_UpdateAfterReturn(t *testing.T) {
	store := NewStore(t.TempDir())
	loan := storageTestLoan()
	if err := store.UpsertLoan(loan); err != nil {
		t.Fatalf("UpsertLoan() error = %v", err)
	}
	ret := models.Transferencia{LinkedLoanID: loan.ID, TransferKind: models.TransferKindReturn, Insumos: []models.ItemTransferido{{ID: 3421, Quantidade: 4}}}
	if err := store.UpdateLoanAfterReturn(ret); err != nil {
		t.Fatalf("UpdateLoanAfterReturn() error = %v", err)
	}
	updated, err := store.GetLoanByID(loan.ID)
	if err != nil {
		t.Fatalf("GetLoanByID() error = %v", err)
	}
	if updated.TotalReturnedQuantity != 4 {
		t.Fatalf("TotalReturnedQuantity = %v, want 4", updated.TotalReturnedQuantity)
	}
}

func storageTestLoan() models.LoanRecord {
	loan := models.LoanRecord{ID: "loan-1", Solicitor: "Maria", Type: models.TransferKindLoan, Items: []models.LoanItem{{ResourceID: 3421, LoanedQuantity: 10}}}
	loan.Recalculate()
	return loan
}
