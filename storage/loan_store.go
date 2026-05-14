package storage

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"sienge-transfer/models"
)

const LoansFileName = "emprestimos.json"

func (s Store) LoansPath() string {
	return filepath.Join(s.Dir, LoansFileName)
}

func (s Store) ListLoans() ([]models.LoanRecord, error) {
	data, err := os.ReadFile(s.LoansPath())
	if errors.Is(err, os.ErrNotExist) {
		return []models.LoanRecord{}, nil
	}
	if err != nil {
		return nil, err
	}

	var loans []models.LoanRecord
	if err := json.Unmarshal(data, &loans); err != nil {
		return nil, err
	}
	if loans == nil {
		return []models.LoanRecord{}, nil
	}
	for index := range loans {
		loans[index].Recalculate()
	}
	return loans, nil
}

func (s Store) SaveAllLoans(loans []models.LoanRecord) error {
	if err := s.EnsureDir(); err != nil {
		return err
	}
	if loans == nil {
		loans = []models.LoanRecord{}
	}
	for index := range loans {
		loans[index].Recalculate()
	}
	data, err := json.MarshalIndent(loans, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return writeFileAtomically(s.LoansPath(), data, 0o600)
}

func (s Store) UpsertLoan(loan models.LoanRecord) error {
	loans, err := s.ListLoans()
	if err != nil {
		return err
	}
	loan.Recalculate()
	for index := range loans {
		if loans[index].ID == loan.ID {
			loans[index] = loan
			return s.SaveAllLoans(loans)
		}
	}
	loans = append(loans, loan)
	return s.SaveAllLoans(loans)
}

func (s Store) GetLoanByID(id string) (models.LoanRecord, error) {
	id = strings.TrimSpace(id)
	loans, err := s.ListLoans()
	if err != nil {
		return models.LoanRecord{}, err
	}
	for _, loan := range loans {
		if loan.ID == id {
			return loan, nil
		}
	}
	return models.LoanRecord{}, models.ErrLoanNotFound
}

func (s Store) UpdateLoanAfterReturn(returnTransfer models.Transferencia) error {
	if strings.TrimSpace(returnTransfer.LinkedLoanID) == "" {
		return nil
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
