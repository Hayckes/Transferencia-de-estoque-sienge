package ui

import (
	"reflect"
	"testing"

	"sienge-transfer/models"
)

func TestBuildConsultaInsumoRows_MapsFieldsHorizontally(t *testing.T) {
	rows := BuildConsultaInsumoRows([]models.ConsultaResultado{{
		ObraID:     111,
		ObraNome:   "BUILDMATE",
		InsumoID:   1001,
		InsumoNome: "Cimento",
		Detalhe:    "CPIII",
		Marca:      "Votoran",
		Unidade:    "kg",
		Quantidade: 5,
	}})
	if len(rows) != 1 {
		t.Fatalf("len(rows) = %d, want 1", len(rows))
	}
	row := rows[0]
	if row.ObraLabel != "111 - BUILDMATE" || row.ID != "1001" || row.Nome != "Cimento" || row.Detalhe != "CPIII" || row.Marca != "Votoran" || row.Unidade != "kg" || row.Quantidade != "5.0000 kg" {
		t.Fatalf("row = %#v, want mapped horizontal fields", row)
	}
}

func TestConsultaInsumoTableColumns_ReturnsExpectedOrder(t *testing.T) {
	want := []string{"Obra", "ID", "Nome", "Detalhe", "Marca", "Unidade", "Qtd. em Estoque", "Acoes"}
	if got := ConsultaInsumoTableColumns(); !reflect.DeepEqual(got, want) {
		t.Fatalf("ConsultaInsumoTableColumns() = %#v, want %#v", got, want)
	}
}

func TestBuildTransferInsumoSelectionRows_MapsFieldsHorizontally(t *testing.T) {
	rows := BuildTransferInsumoSelectionRows([]models.Insumo{{ID: 1001, Nome: "Cimento", Detalhe: "CPIII", Marca: "Votoran", Unidade: "kg", Quantidade: 5}})
	if len(rows) != 1 {
		t.Fatalf("len(rows) = %d, want 1", len(rows))
	}
	row := rows[0]
	if row.ID != "1001" || row.Nome != "Cimento" || row.Detalhe != "CPIII" || row.Marca != "Votoran" || row.Unidade != "kg" || row.Quantidade != "5.0000 kg" {
		t.Fatalf("row = %#v, want mapped horizontal fields", row)
	}
}

func TestTransferInsumoSelectionTableColumns_ReturnsExpectedOrder(t *testing.T) {
	want := []string{"ID", "Nome", "Detalhe", "Marca", "Unidade", "Qtd. em Estoque"}
	if got := TransferInsumoSelectionTableColumns(); !reflect.DeepEqual(got, want) {
		t.Fatalf("TransferInsumoSelectionTableColumns() = %#v, want %#v", got, want)
	}
}

func TestResolveSelectedInsumo_ReturnsCorrectItemByIndex(t *testing.T) {
	items := []models.Insumo{{ID: 1001, Detalhe: "CPII"}, {ID: 1001, Detalhe: "CPIII"}}
	item, err := ResolveSelectedInsumo(items, 1)
	if err != nil {
		t.Fatalf("ResolveSelectedInsumo() error = %v", err)
	}
	if item.Detalhe != "CPIII" {
		t.Fatalf("ResolveSelectedInsumo() = %#v, want second item", item)
	}
}

func TestResolveSelectedInsumo_RejectsInvalidIndex(t *testing.T) {
	items := []models.Insumo{{ID: 1001}}
	if _, err := ResolveSelectedInsumo(items, -1); err == nil {
		t.Fatal("ResolveSelectedInsumo(-1) error = nil, want error")
	}
	if _, err := ResolveSelectedInsumo(items, 1); err == nil {
		t.Fatal("ResolveSelectedInsumo(out of range) error = nil, want error")
	}
}
