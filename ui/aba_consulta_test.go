package ui

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"sienge-transfer/models"
)

func TestObraLabelsAndIDFromLabel(t *testing.T) {
	cfg := testConfig()
	labels := ObraLabels(cfg.Obras)
	want := []string{"121 - Residencial Novo Horizonte", "205 - Comercial Centro"}
	if !reflect.DeepEqual(labels, want) {
		t.Fatalf("ObraLabels() = %#v, want %#v", labels, want)
	}

	id, ok := ObraIDFromLabel(cfg.Obras, "121 - Residencial Novo Horizonte")
	if !ok || id != 121 {
		t.Fatalf("ObraIDFromLabel() = %d/%v, want 121/true", id, ok)
	}
	if _, ok := ObraIDFromLabel(cfg.Obras, "999 - Inexistente"); ok {
		t.Fatal("ObraIDFromLabel() ok = true, want false")
	}
}

func TestConsultaTabStateDoesNotKeepObservationField(t *testing.T) {
	stateType := reflect.TypeOf(ConsultaTabState{})
	if _, ok := stateType.FieldByName("Observacao"); ok {
		t.Fatal("ConsultaTabState should not keep an observation field")
	}
	if _, ok := stateType.FieldByName("DetalheAberto"); ok {
		t.Fatal("ConsultaTabState should not keep a details modal state")
	}
}

func TestValidateConsultaInput(t *testing.T) {
	state := NewAppState(testConfig())
	state.Consulta.ObraSelecionada = "121 - Residencial Novo Horizonte"
	state.Consulta.InsumoIDsInput = "3421, 9876 3421"

	obraID, ids, err := ValidateConsultaInput(state)
	if err != nil {
		t.Fatalf("ValidateConsultaInput() error = %v", err)
	}
	if obraID != 121 {
		t.Fatalf("obraID = %d, want 121", obraID)
	}
	wantIDs := []int{3421, 9876}
	if !reflect.DeepEqual(ids, wantIDs) {
		t.Fatalf("ids = %#v, want %#v", ids, wantIDs)
	}
}

func TestValidateConsultaInputRejectsMissingWorkAndIDs(t *testing.T) {
	state := NewAppState(testConfig())
	state.Consulta.InsumoIDsInput = "3421"

	_, _, err := ValidateConsultaInput(state)
	if !errors.Is(err, ErrObraConsultaObrigatoria) {
		t.Fatalf("ValidateConsultaInput() error = %v, want ErrObraConsultaObrigatoria", err)
	}

	state.Consulta.ObraSelecionada = "121 - Residencial Novo Horizonte"
	state.Consulta.InsumoIDsInput = ""
	_, _, err = ValidateConsultaInput(state)
	if !errors.Is(err, models.ErrIDsInsumoObrigatorios) {
		t.Fatalf("ValidateConsultaInput() error = %v, want ErrIDsInsumoObrigatorios", err)
	}
}

func TestRunConsultaCallsStockService(t *testing.T) {
	stock := &fakeStockService{items: []models.Insumo{
		{ID: 3421, Nome: "Cimento", Detalhe: "CP III", Marca: "Votorantim", Quantidade: 150, Unidade: "SC"},
		{ID: 3421, Nome: "Cimento", Detalhe: "CP II", Marca: "Intercement", Quantidade: 80, Unidade: "SC"},
	}}
	state := NewAppState(testConfig())
	state.Stock = stock
	state.Consulta.ObraSelecionada = "121 - Residencial Novo Horizonte"
	state.Consulta.InsumoIDsInput = "3421"

	if err := RunConsulta(context.Background(), state); err != nil {
		t.Fatalf("RunConsulta() error = %v", err)
	}

	if !stock.itemsCalled {
		t.Fatal("stock service was not called")
	}
	if stock.costCenterID != 121 {
		t.Fatalf("costCenterID = %d, want 121", stock.costCenterID)
	}
	if !reflect.DeepEqual(stock.ids, []int{3421}) {
		t.Fatalf("ids = %#v, want [3421]", stock.ids)
	}
	if len(state.Consulta.Resultados) != 2 {
		t.Fatalf("len(Resultados) = %d, want 2", len(state.Consulta.Resultados))
	}
}

func TestRunConsultaDoesNotCallAPIWhenIDsAreEmpty(t *testing.T) {
	stock := &fakeStockService{}
	state := NewAppState(testConfig())
	state.Stock = stock
	state.Consulta.ObraSelecionada = "121 - Residencial Novo Horizonte"

	err := RunConsulta(context.Background(), state)
	if !errors.Is(err, models.ErrIDsInsumoObrigatorios) {
		t.Fatalf("RunConsulta() error = %v, want ErrIDsInsumoObrigatorios", err)
	}
	if stock.itemsCalled {
		t.Fatal("stock service should not be called when IDs are empty")
	}
}

func TestRunConsultaReturnsServiceErrorWithoutClearingPreviousResults(t *testing.T) {
	wantErr := errors.New("api fora")
	state := NewAppState(testConfig())
	state.Stock = &fakeStockService{err: wantErr}
	state.Consulta.ObraSelecionada = "121 - Residencial Novo Horizonte"
	state.Consulta.InsumoIDsInput = "3421"
	state.Consulta.Resultados = []models.Insumo{{ID: 1, Nome: "Anterior"}}

	err := RunConsulta(context.Background(), state)
	if !errors.Is(err, wantErr) {
		t.Fatalf("RunConsulta() error = %v, want %v", err, wantErr)
	}
	if len(state.Consulta.Resultados) != 1 || state.Consulta.Resultados[0].Nome != "Anterior" {
		t.Fatalf("previous results were cleared: %#v", state.Consulta.Resultados)
	}
}

func TestClearConsultaResetsOnlyConsultaState(t *testing.T) {
	state := NewAppState(testConfig())
	state.Consulta.ObraSelecionada = "121 - Residencial Novo Horizonte"
	state.Consulta.InsumoIDsInput = "3421"
	state.Consulta.Resultados = []models.Insumo{{ID: 3421}}

	ClearConsulta(state)

	if state.Consulta.ObraSelecionada != "" || state.Consulta.InsumoIDsInput != "" || len(state.Consulta.Resultados) != 0 {
		t.Fatalf("Consulta was not cleared: %#v", state.Consulta)
	}
	if state.Config.Usuario.Nome == "" {
		t.Fatal("ClearConsulta() should not clear app config")
	}
}

func TestConsultaResultRow(t *testing.T) {
	item := models.Insumo{
		ID:         3421,
		Nome:       "Cimento",
		Detalhe:    "CP III",
		Marca:      "Votorantim",
		Quantidade: 150,
		Unidade:    "SC",
		Apropriacoes: []models.Apropriacao{
			{Codigo: "A001", Descricao: "Fundacao", Quantidade: 40},
		},
	}

	row := ConsultaResultRow(item)
	for _, want := range []string{"3421", "Cimento", "CP III", "Votorantim", "150.0000 SC"} {
		if !strings.Contains(row, want) {
			t.Fatalf("ConsultaResultRow() = %q, want containing %q", row, want)
		}
	}
}

func TestBuildConsultaTabReturnsObject(t *testing.T) {
	state := NewAppState(testConfig())
	if BuildConsultaTab(state) == nil {
		t.Fatal("BuildConsultaTab() returned nil")
	}
}

type fakeStockService struct {
	items                      []models.Insumo
	appropriations             []models.Apropriacao
	appropriationsByCostCenter map[int][]models.Apropriacao
	err                        error
	itemsCalled                bool
	approprCalled              bool
	costCenterID               int
	approprCostCenterIDs       []int
	resourceID                 int
	ids                        []int
}

func (s *fakeStockService) GetStockItemsByIDs(ctx context.Context, costCenterID int, ids []int) ([]models.Insumo, error) {
	s.itemsCalled = true
	s.costCenterID = costCenterID
	s.ids = append([]int(nil), ids...)
	if s.err != nil {
		return nil, s.err
	}

	return append([]models.Insumo(nil), s.items...), nil
}

func (s *fakeStockService) GetBuildingAppropriations(ctx context.Context, costCenterID, resourceID int) ([]models.Apropriacao, error) {
	s.approprCalled = true
	s.costCenterID = costCenterID
	s.approprCostCenterIDs = append(s.approprCostCenterIDs, costCenterID)
	s.resourceID = resourceID
	if s.err != nil {
		return nil, s.err
	}
	if s.appropriationsByCostCenter != nil {
		return append([]models.Apropriacao(nil), s.appropriationsByCostCenter[costCenterID]...), nil
	}

	return append([]models.Apropriacao(nil), s.appropriations...), nil
}
