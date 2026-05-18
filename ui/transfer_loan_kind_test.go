package ui

import (
	"errors"
	"strings"
	"testing"

	"sienge-transfer/models"
)

func TestTransferKind_DefaultsToNotApplicable(t *testing.T) {
	state := NewAppState(testConfig())
	transfer, err := BuildTransferenciaFromState(validTransferStateWithItem())
	if err != nil {
		t.Fatalf("BuildTransferenciaFromState() error = %v", err)
	}
	if state.Transferencia.TransferKind != models.TransferKindNotApplicable || transfer.TransferKind != models.TransferKindNotApplicable {
		t.Fatalf("transfer kind state/transfer = %q/%q, want not_applicable", state.Transferencia.TransferKind, transfer.TransferKind)
	}
}

func TestTransferKind_LoanCreatesLoanAfterSuccess(t *testing.T) {
	state := validTransferStateWithItem()
	state.Transferencia.TransferKind = models.TransferKindLoan
	state.Transfer = &fakeTransferService{movementID: "MOV-LOAN"}
	state.TransferStore = &fakeTransferStorage{}
	loanStore := &fakeLoanStore{}
	state.LoanStore = loanStore

	movementID, err := SendTransferencia(nil, state)
	if err != nil {
		t.Fatalf("SendTransferencia() error = %v", err)
	}
	if movementID != "MOV-LOAN" || len(loanStore.loans) != 1 || loanStore.loans[0].Status != models.LoanStatusPending {
		t.Fatalf("movement/loans = %q/%#v, want pending loan", movementID, loanStore.loans)
	}
	if !strings.HasPrefix(loanStore.loans[0].ID, "EM-") || !strings.HasSuffix(loanStore.loans[0].ID, "-1") {
		t.Fatalf("loan ID = %q, want EM timestamp sequence", loanStore.loans[0].ID)
	}
}

func TestTransferKind_LoanIncrementsGlobalLoanSequence(t *testing.T) {
	state := validTransferStateWithItem()
	state.Transferencia.TransferKind = models.TransferKindLoan
	state.Transfer = &fakeTransferService{movementID: "MOV-LOAN"}
	state.TransferStore = &fakeTransferStorage{}
	loanStore := &fakeLoanStore{loans: []models.LoanRecord{{ID: "EM-100-1"}, {ID: "EM-200-3"}, {ID: "loan-100-sem-movimento"}}}
	state.LoanStore = loanStore

	if _, err := SendTransferencia(nil, state); err != nil {
		t.Fatalf("SendTransferencia() error = %v", err)
	}
	if len(loanStore.loans) != 4 || !strings.HasSuffix(loanStore.loans[3].ID, "-4") {
		t.Fatalf("loans = %#v, want new loan with sequence 4", loanStore.loans)
	}
}

func TestTransferKind_NotApplicableDoesNotCreateLoan(t *testing.T) {
	state := validTransferStateWithItem()
	state.Transfer = &fakeTransferService{movementID: "MOV-1"}
	state.TransferStore = &fakeTransferStorage{}
	loanStore := &fakeLoanStore{}
	state.LoanStore = loanStore

	if _, err := SendTransferencia(nil, state); err != nil {
		t.Fatalf("SendTransferencia() error = %v", err)
	}
	if len(loanStore.loans) != 0 {
		t.Fatalf("loans = %#v, want none", loanStore.loans)
	}
}

func TestSendTransferenciaSavesHistoryAndExcelWhenLoanStoreFailsAfterAPISuccess(t *testing.T) {
	state := validTransferStateWithItem()
	state.Transferencia.TransferKind = models.TransferKindLoan
	state.Transfer = &fakeTransferService{movementID: "MOV-LOAN"}
	transferStore := &fakeTransferStorage{}
	state.TransferStore = transferStore
	state.LoanStore = &fakeLoanStore{err: errors.New("loan store falhou")}

	_, err := SendTransferencia(nil, state)

	if err == nil || !strings.Contains(err.Error(), "atualizar emprestimos") {
		t.Fatalf("SendTransferencia() error = %v, want loan update warning", err)
	}
	if !transferStore.historySaved || !transferStore.excelSaved {
		t.Fatalf("history/excel saved = %v/%v, want both saved", transferStore.historySaved, transferStore.excelSaved)
	}
	if transferStore.historyTransfer.LinkedLoanID == "" || transferStore.historyTransfer.LoanStatus != models.LoanStatusPending {
		t.Fatalf("history transfer loan fields = %#v, want loan ID and pending status", transferStore.historyTransfer)
	}
}

func TestSendTransferenciaSavesHistoryAndExcelWhenReturnLoanUpdateFailsAfterAPISuccess(t *testing.T) {
	state := validTransferStateWithItem()
	state.Transferencia.TransferKind = models.TransferKindReturn
	state.Transferencia.SelectedLoanID = "loan-1"
	state.Transfer = &fakeTransferService{movementID: "MOV-RETURN"}
	transferStore := &fakeTransferStorage{}
	state.TransferStore = transferStore
	loan := models.LoanRecord{ID: "loan-1", Items: []models.LoanItem{{ResourceID: 3421, LoanedQuantity: 20}}}
	loan.Recalculate()
	state.LoanStore = &fakeLoanStore{loans: []models.LoanRecord{loan}, err: errors.New("loan store falhou")}

	_, err := SendTransferencia(nil, state)

	if err == nil || !strings.Contains(err.Error(), "atualizar emprestimos") {
		t.Fatalf("SendTransferencia() error = %v, want loan update warning", err)
	}
	if !transferStore.historySaved || !transferStore.excelSaved {
		t.Fatalf("history/excel saved = %v/%v, want both saved", transferStore.historySaved, transferStore.excelSaved)
	}
	if transferStore.historyTransfer.LoanStatus != models.LoanStatusPartiallyReturned {
		t.Fatalf("history transfer loan status = %q, want partially returned", transferStore.historyTransfer.LoanStatus)
	}
}

func TestReturnTransferCannotExceedPendingLoanQuantity(t *testing.T) {
	state := validTransferStateWithItem()
	state.Transferencia.TransferKind = models.TransferKindReturn
	state.Transferencia.SelectedLoanID = "loan-1"
	state.Transferencia.Itens[0].QuantidadeTransferir = "11,0000"
	loan := models.LoanRecord{ID: "loan-1", Items: []models.LoanItem{{ResourceID: 3421, DetailID: intPtr(10), BrandID: intPtr(5), LoanedQuantity: 10}}}
	loan.Recalculate()
	state.LoanStore = &fakeLoanStore{loans: []models.LoanRecord{loan}}

	transfer, err := BuildTransferenciaFromState(state)
	if err != nil {
		t.Fatalf("BuildTransferenciaFromState() error = %v", err)
	}
	if err := ValidateLoanReturnBeforeSend(state, transfer); err == nil {
		t.Fatal("ValidateLoanReturnBeforeSend() error = nil, want excess return error")
	}
}

type fakeLoanStore struct {
	loans []models.LoanRecord
	err   error
}

func (s *fakeLoanStore) ListLoans() ([]models.LoanRecord, error) {
	if s.err != nil {
		return nil, s.err
	}
	return append([]models.LoanRecord(nil), s.loans...), nil
}

func (s *fakeLoanStore) SaveAllLoans(loans []models.LoanRecord) error {
	s.loans = append([]models.LoanRecord(nil), loans...)
	return s.err
}

func (s *fakeLoanStore) UpsertLoan(loan models.LoanRecord) error {
	if s.err != nil {
		return s.err
	}
	for index := range s.loans {
		if s.loans[index].ID == loan.ID {
			s.loans[index] = loan
			return nil
		}
	}
	s.loans = append(s.loans, loan)
	return nil
}

func (s *fakeLoanStore) GetLoanByID(id string) (models.LoanRecord, error) {
	for _, loan := range s.loans {
		if loan.ID == id {
			return loan, nil
		}
	}
	return models.LoanRecord{}, models.ErrLoanNotFound
}

func (s *fakeLoanStore) UpdateLoanAfterReturn(returnTransfer models.Transferencia) error {
	if s.err != nil {
		return s.err
	}
	loan, err := s.GetLoanByID(returnTransfer.LinkedLoanID)
	if err != nil {
		return err
	}
	updated, err := models.ApplyReturnToLoan(loan, returnTransfer)
	if err != nil {
		return err
	}
	return s.UpsertLoan(updated)
}

func intPtr(value int) *int { return &value }

var _ LoanStorage = (*fakeLoanStore)(nil)
