package models

import "testing"

func TestMatchPurchaseRequestItemToStock_MatchesProductDetailAndNullTrademark(t *testing.T) {
	results := BuildConsultaPorSolicitacaoResults([]Obra{{ID: 1, Nome: "Obra"}}, []PurchaseRequestItem{{ResourceID: 1001, DetailID: 3, BrandID: 0, Unit: "sc"}}, map[int][]Insumo{1: {{ID: 1001, DetalheID: 3, MarcaID: 0, Unidade: "kg", Quantidade: 10}}})
	if len(results) != 1 {
		t.Fatalf("len(results) = %d, want 1", len(results))
	}
}

func TestMatchPurchaseRequestItemToStock_DoesNotFailWhenUnitDiffers(t *testing.T) {
	results := BuildConsultaPorSolicitacaoResults([]Obra{{ID: 1}}, []PurchaseRequestItem{{ResourceID: 1001, DetailID: 3, Unit: "sc"}}, map[int][]Insumo{1: {{ID: 1001, DetalheID: 3, Unidade: "kg", Quantidade: 10}}})
	if len(results) != 1 {
		t.Fatalf("len(results) = %d, want 1", len(results))
	}
}

func TestBuildConsultaPorSolicitacaoResults_ReturnsNoResultsFeedbackWhenNoStockFound(t *testing.T) {
	results := BuildConsultaPorSolicitacaoResults([]Obra{{ID: 1}}, []PurchaseRequestItem{{ResourceID: 1001, DetailID: 3}}, map[int][]Insumo{1: {{ID: 1001, DetalheID: 4, Quantidade: 10}}})
	if len(results) != 0 {
		t.Fatalf("results = %#v, want none", results)
	}
}

func TestBuildConsultaPorSolicitacaoResults_ReturnsMatchedStockRows(t *testing.T) {
	results := BuildConsultaPorSolicitacaoResults([]Obra{{ID: 1}}, []PurchaseRequestItem{{ResourceID: 1001, DetailID: 3, BrandID: 5}}, map[int][]Insumo{1: {{ID: 1001, DetalheID: 3, MarcaID: 5, Quantidade: 10}}})
	if len(results) != 1 || results[0].InsumoID != 1001 {
		t.Fatalf("results = %#v, want matched row", results)
	}
}
