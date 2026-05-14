package api

import "testing"

func TestParsePurchaseRequestItems_AcceptsResultadosWrapper(t *testing.T) {
	items, err := parsePurchaseRequestItems([]byte(`{"resultados":[{"productId":1001,"productDescription":"Cimento"}]}`), 2339, 111)
	if err != nil {
		t.Fatalf("parsePurchaseRequestItems() error = %v", err)
	}
	if len(items) != 1 || items[0].ResourceID != 1001 {
		t.Fatalf("items = %#v, want parsed resultados", items)
	}
}

func TestParsePurchaseRequestItems_MapsProductFields(t *testing.T) {
	items, err := parsePurchaseRequestItems([]byte(`{"resultados":[{"purchaseRequestId":2339,"itemNumber":1,"productId":1001,"productDescription":"Cimento","detailId":3,"detailDescription":"CPIII","trademarkId":456,"trademarkDescription":"Votoran","quantity":12,"unitSymbol":"sc"}]}`), 2339, 111)
	if err != nil {
		t.Fatalf("parsePurchaseRequestItems() error = %v", err)
	}
	item := items[0]
	if item.ResourceID != 1001 || item.ResourceName != "Cimento" || item.DetailID != 3 || item.Detail != "CPIII" || item.BrandID != 456 || item.Brand != "Votoran" || item.Unit != "sc" || item.Quantity != 12 {
		t.Fatalf("item = %#v, want product fields mapped", item)
	}
}

func TestParsePurchaseRequestItems_HandlesNullTrademarkID(t *testing.T) {
	items, err := parsePurchaseRequestItems([]byte(`{"resultados":[{"productId":1001,"detailId":3,"trademarkId":null,"quantity":12}]}`), 2339, 111)
	if err != nil {
		t.Fatalf("parsePurchaseRequestItems() error = %v", err)
	}
	if items[0].BrandID != 0 {
		t.Fatalf("BrandID = %d, want zero for null", items[0].BrandID)
	}
}

func TestParsePurchaseRequestItems_AcceptsPortugueseTranslatedFieldsIfPresent(t *testing.T) {
	items, err := parsePurchaseRequestItems([]byte(`{"resultados":[{"productId":1001,"quantidade":7,"unidade":"kg"}]}`), 2339, 111)
	if err != nil {
		t.Fatalf("parsePurchaseRequestItems() error = %v", err)
	}
	if items[0].Quantity != 7 || items[0].Unit != "kg" {
		t.Fatalf("item = %#v, want translated fields", items[0])
	}
}
