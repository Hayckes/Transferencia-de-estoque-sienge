package models

import "testing"

func TestResolveObrasParaConsulta_ReturnsAllOrSelected(t *testing.T) {
	obras := []Obra{{ID: 1, Nome: "Obra 1"}, {ID: 2, Nome: "Obra 2"}}
	all, err := ResolveObrasParaConsulta(obras, nil, true)
	if err != nil || len(all) != 2 {
		t.Fatalf("ResolveObrasParaConsulta(all) = %#v/%v, want all", all, err)
	}
	selected, err := ResolveObrasParaConsulta(obras, []Obra{{ID: 2}}, false)
	if err != nil || len(selected) != 1 || selected[0].ID != 2 {
		t.Fatalf("ResolveObrasParaConsulta(selected) = %#v/%v, want obra 2", selected, err)
	}
	if _, err := ResolveObrasParaConsulta(obras, nil, false); err == nil {
		t.Fatal("ResolveObrasParaConsulta() error = nil, want no selection error")
	}
}

func TestBuildConsultaPorSolicitacaoResults_ReturnsOnlyItemsWithStock(t *testing.T) {
	obras := []Obra{{ID: 102, Nome: "Obra 102"}, {ID: 103, Nome: "Obra 103"}}
	requestItems := []PurchaseRequestItem{{ResourceID: 1001, DetailID: 5, BrandID: 8}}
	stockByWork := map[int][]Insumo{
		102: {{ID: 1001, Nome: "Cimento", DetalheID: 5, MarcaID: 8, Quantidade: 50}},
		103: {{ID: 1001, Nome: "Cimento", DetalheID: 5, MarcaID: 8, Quantidade: 0}, {ID: 9999, Quantidade: 10}},
	}

	results := BuildConsultaPorSolicitacaoResults(obras, requestItems, stockByWork)
	if len(results) != 1 || results[0].ObraID != 102 || results[0].InsumoID != 1001 {
		t.Fatalf("results = %#v, want only item with stock from request", results)
	}
}
