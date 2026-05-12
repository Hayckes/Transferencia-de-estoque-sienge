package models

import "testing"

func TestCalculateTransferBalances_UsesAppropriationStockWhenSelected(t *testing.T) {
	origin := 20.0
	destination := 7.0
	balances, err := CalculateTransferBalances(TransferBalanceInput{OriginTotalStock: 50, DestinationTotalStock: 12, OriginAppropriationStock: &origin, DestinationAppropriationStock: &destination, QuantityToTransfer: 5})
	if err != nil {
		t.Fatalf("CalculateTransferBalances() error = %v", err)
	}
	if balances.OriginCurrentStock != 20 || balances.OriginAfterTransfer != 15 || balances.DestinationCurrentStock != 7 || balances.DestinationAfterTransfer != 12 {
		t.Fatalf("balances = %#v, want appropriation balances", balances)
	}
}

func TestCalculateTransferBalances_UsesTotalStockWhenNoAppropriation(t *testing.T) {
	balances, err := CalculateTransferBalances(TransferBalanceInput{OriginTotalStock: 50, DestinationTotalStock: 12, QuantityToTransfer: 5})
	if err != nil {
		t.Fatalf("CalculateTransferBalances() error = %v", err)
	}
	if balances.OriginCurrentStock != 50 || balances.OriginAfterTransfer != 45 || balances.DestinationAfterTransfer != 17 {
		t.Fatalf("balances = %#v, want total stock balances", balances)
	}
}

func TestCalculateTransferBalances_RejectsNegativeOriginAfterTransfer(t *testing.T) {
	origin := 3.0
	if _, err := CalculateTransferBalances(TransferBalanceInput{OriginTotalStock: 50, OriginAppropriationStock: &origin, QuantityToTransfer: 5}); err == nil {
		t.Fatal("CalculateTransferBalances() error = nil, want negative origin error")
	}
}

func TestCalculateTransferBalances_UpdatesAfterQuantityChange(t *testing.T) {
	first, _ := CalculateTransferBalances(TransferBalanceInput{OriginTotalStock: 10, DestinationTotalStock: 1, QuantityToTransfer: 2})
	second, _ := CalculateTransferBalances(TransferBalanceInput{OriginTotalStock: 10, DestinationTotalStock: 1, QuantityToTransfer: 3})
	if first.OriginAfterTransfer == second.OriginAfterTransfer || second.OriginAfterTransfer != 7 {
		t.Fatalf("first/second = %#v/%#v, want updated balance", first, second)
	}
}
