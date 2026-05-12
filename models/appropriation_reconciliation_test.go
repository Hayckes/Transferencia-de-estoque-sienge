package models

import "testing"

func TestReconcileStockAndAppropriations_ReturnsOKWhenSumMatches(t *testing.T) {
	result := ReconcileStockAndAppropriations(946, []Apropriacao{{Quantidade: 900}, {Quantidade: 46}})
	if !result.OK || result.AppropriationsQuantity != 946 {
		t.Fatalf("result = %#v, want OK", result)
	}
}

func TestReconcileStockAndAppropriations_ReturnsMismatchWhenSumDiffers(t *testing.T) {
	result := ReconcileStockAndAppropriations(946, []Apropriacao{{Quantidade: 5}, {Quantidade: 2}, {Quantidade: 2}})
	if result.OK || result.AppropriationsQuantity != 9 {
		t.Fatalf("result = %#v, want mismatch", result)
	}
}
