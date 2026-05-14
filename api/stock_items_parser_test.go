package api

import "testing"

func TestParseStockItem_FillsDetailAndTrademarkIDs(t *testing.T) {
	items, err := parseStockItems([]byte(`{"results":[{"resourceId":1001,"resourceName":"Cimento","detailId":123,"detailDescription":"CPIII","trademarkId":456,"trademarkDescription":"Votoran","unitOfMeasure":"kg","quantity":946.0}]}`))
	if err != nil {
		t.Fatalf("parseStockItems() error = %v", err)
	}
	item := items[0]
	if item.ID != 1001 || item.DetalheID != 123 || item.Detalhe != "CPIII" || item.MarcaID != 456 || item.Marca != "Votoran" || item.Quantidade != 946 {
		t.Fatalf("item = %#v, want detail/trademark IDs", item)
	}
}

func TestParseStockItem_HandlesMissingTrademark(t *testing.T) {
	items, err := parseStockItems([]byte(`{"results":[{"resourceId":1001,"resourceName":"Cimento","resourceDetailId":123,"detailDescription":"CPIII","unitOfMeasure":"kg","quantity":754.0}]}`))
	if err != nil {
		t.Fatalf("parseStockItems() error = %v", err)
	}
	item := items[0]
	if item.Marca != "" || item.MarcaID != 0 || item.DetalheID != 123 {
		t.Fatalf("item = %#v, want missing trademark as zero/empty", item)
	}
}
