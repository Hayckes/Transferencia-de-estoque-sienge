package models

import "testing"

func TestCalculateTransferStockSnapshot_CalculatesBalances(t *testing.T) {
	originAppropriation := 20.0
	destinationAppropriation := 7.0
	snapshot, err := CalculateTransferStockSnapshot(TransferStockSnapshotInput{
		EstoqueOrigemAntes:      50,
		EstoqueDestinoAntes:     12,
		ApropriacaoOrigemAntes:  &originAppropriation,
		ApropriacaoDestinoAntes: &destinationAppropriation,
		Quantidade:              5,
	})
	if err != nil {
		t.Fatalf("CalculateTransferStockSnapshot() error = %v", err)
	}
	if snapshot.EstoqueOrigemDepois != 45 || snapshot.EstoqueDestinoDepois != 17 || snapshot.QuantidadeEnviada != 5 || snapshot.QuantidadeRecebida != 5 {
		t.Fatalf("snapshot = %#v, want stock balances", snapshot)
	}
	if *snapshot.ApropriacaoOrigemDepois != 15 || *snapshot.ApropriacaoDestinoDepois != 12 {
		t.Fatalf("snapshot appropriations = %#v, want 15/12", snapshot)
	}
}

func TestCalculateTransferStockSnapshot_HandlesNoAppropriations(t *testing.T) {
	snapshot, err := CalculateTransferStockSnapshot(TransferStockSnapshotInput{EstoqueOrigemAntes: 10, EstoqueDestinoAntes: 1, Quantidade: 2})
	if err != nil {
		t.Fatalf("CalculateTransferStockSnapshot() error = %v", err)
	}
	if snapshot.ApropriacaoOrigemAntes != nil || snapshot.ApropriacaoDestinoDepois != nil {
		t.Fatalf("snapshot = %#v, want nil appropriation balances", snapshot)
	}
}

func TestCalculateTransferStockSnapshot_RejectsInvalidQuantity(t *testing.T) {
	if _, err := CalculateTransferStockSnapshot(TransferStockSnapshotInput{EstoqueOrigemAntes: 3, Quantidade: 5}); err == nil {
		t.Fatal("CalculateTransferStockSnapshot() error = nil, want origin stock error")
	}
	appropriation := 3.0
	if _, err := CalculateTransferStockSnapshot(TransferStockSnapshotInput{EstoqueOrigemAntes: 10, ApropriacaoOrigemAntes: &appropriation, Quantidade: 5}); err == nil {
		t.Fatal("CalculateTransferStockSnapshot() error = nil, want appropriation stock error")
	}
}
