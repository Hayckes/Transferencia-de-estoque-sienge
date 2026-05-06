package models

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestConfigJSONRoundTrip(t *testing.T) {
	original := Config{
		Usuario: Usuario{Nome: "Joao Silva", Cargo: "Engenheiro"},
		Empresa: Empresa{
			Nome:         "Construtora XYZ",
			Subdominio:   "construtoraxyz",
			APIUsuario:   "joao.silva",
			APISenha:     "senha-cifrada",
			SenhaCifrada: true,
		},
		Obras: []Obra{{ID: 121, Nome: "Residencial Novo Horizonte"}},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var decoded Config
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if !reflect.DeepEqual(original, decoded) {
		t.Fatalf("decoded config differs\nwant: %#v\n got: %#v", original, decoded)
	}
}

func TestTransferenciaJSONRoundTrip(t *testing.T) {
	dataHora := time.Date(2024, 7, 15, 10, 30, 0, 0, time.UTC)
	original := Transferencia{
		IDMovimento:         "7842",
		DataHora:            dataHora,
		Usuario:             "Joao Silva",
		Cargo:               "Engenheiro",
		Solicitante:         "Maria Santos",
		ObraOrigemID:        121,
		ObraOrigemNome:      "Residencial Novo Horizonte",
		ObraDestinoID:       205,
		ObraDestinoNome:     "Comercial Centro",
		CodigoTipoDocumento: "TR",
		CodigoTipoMovimento: 3,
		Insumos: []ItemTransferido{{
			ID:                   3421,
			Nome:                 "Cimento",
			Detalhe:              "CP III",
			Marca:                "Votorantim",
			Apropriacao:          "A001",
			ApropriacaoDescricao: "Fundacao",
			Quantidade:           50,
		}},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	if !strings.Contains(string(data), `"id_movimento":"7842"`) {
		t.Fatalf("expected id_movimento JSON field, got %s", string(data))
	}

	var decoded Transferencia
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if !reflect.DeepEqual(original, decoded) {
		t.Fatalf("decoded transfer differs\nwant: %#v\n got: %#v", original, decoded)
	}
}

func TestObraLabel(t *testing.T) {
	obra := Obra{ID: 121, Nome: "Residencial Novo Horizonte"}

	if got, want := obra.Label(), "121 - Residencial Novo Horizonte"; got != want {
		t.Fatalf("Obra.Label() = %q, want %q", got, want)
	}
}

func TestParseInsumoIDs(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []int
	}{
		{name: "comma separated", input: "3421,9876", want: []int{3421, 9876}},
		{name: "space separated", input: "3421 9876", want: []int{3421, 9876}},
		{name: "mixed separators", input: "3421, 9876 111", want: []int{3421, 9876, 111}},
		{name: "deduplicated", input: "3421,3421 9876", want: []int{3421, 9876}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseInsumoIDs(tt.input)
			if err != nil {
				t.Fatalf("ParseInsumoIDs() error = %v", err)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("ParseInsumoIDs() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestParseInsumoIDsRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr error
	}{
		{name: "empty", input: "", wantErr: ErrIDsInsumoObrigatorios},
		{name: "spaces", input: "   ", wantErr: ErrIDsInsumoObrigatorios},
		{name: "text", input: "3421 abc", wantErr: nil},
		{name: "zero", input: "0", wantErr: nil},
		{name: "negative", input: "-3", wantErr: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseInsumoIDs(tt.input)
			if err == nil {
				t.Fatal("ParseInsumoIDs() error = nil, want error")
			}

			if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
				t.Fatalf("ParseInsumoIDs() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestFormatQuantidade(t *testing.T) {
	tests := []struct {
		name    string
		value   float64
		unidade string
		want    string
	}{
		{name: "with unit", value: 150, unidade: "SC", want: "150 SC"},
		{name: "decimal", value: 10.5, unidade: "KG", want: "10.5 KG"},
		{name: "without unit", value: 80, unidade: "", want: "80"},
		{name: "trim unit", value: 3, unidade: " UN ", want: "3 UN"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FormatQuantidade(tt.value, tt.unidade); got != tt.want {
				t.Fatalf("FormatQuantidade() = %q, want %q", got, tt.want)
			}
		})
	}
}
