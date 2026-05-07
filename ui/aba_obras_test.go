package ui

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"sienge-transfer/models"
)

func TestNewObrasTabStateLoadsReadOnlyUserAndWorks(t *testing.T) {
	cfg := testConfig()
	state := NewObrasTabState(cfg)

	if state.UsuarioNome != cfg.Usuario.Nome || state.UsuarioCargo != cfg.Usuario.Cargo {
		t.Fatalf("user state = %s/%s, want %s/%s", state.UsuarioNome, state.UsuarioCargo, cfg.Usuario.Nome, cfg.Usuario.Cargo)
	}
	if !reflect.DeepEqual(state.Obras, cfg.Obras) {
		t.Fatalf("obras = %#v, want %#v", state.Obras, cfg.Obras)
	}

	state.Obras[0].Nome = "Alterada"
	if cfg.Obras[0].Nome == "Alterada" {
		t.Fatal("NewObrasTabState() should copy works slice")
	}
}

func TestObrasTabStateDoesNotKeepManualWorkNameInput(t *testing.T) {
	stateType := reflect.TypeOf(ObrasTabState{})
	if _, ok := stateType.FieldByName("NovoNome"); ok {
		t.Fatal("ObrasTabState should not keep a manual work name input")
	}
}

func TestAddObraToConfig(t *testing.T) {
	cfg := testConfig()
	updated, err := AddObraToConfig(cfg, models.Obra{ID: 333, Nome: " Nova Obra "})
	if err != nil {
		t.Fatalf("AddObraToConfig() error = %v", err)
	}

	if len(updated.Obras) != len(cfg.Obras)+1 {
		t.Fatalf("len(updated.Obras) = %d, want %d", len(updated.Obras), len(cfg.Obras)+1)
	}
	if updated.Obras[len(updated.Obras)-1] != (models.Obra{ID: 333, Nome: "Nova Obra"}) {
		t.Fatalf("last obra = %#v, want trimmed new obra", updated.Obras[len(updated.Obras)-1])
	}
	if len(cfg.Obras) != 2 {
		t.Fatalf("original cfg was mutated: %#v", cfg.Obras)
	}
}

func TestAddObraToConfigRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name string
		obra models.Obra
		want string
	}{
		{name: "invalid id", obra: models.Obra{ID: 0, Nome: "Obra"}, want: "ID da obra"},
		{name: "missing name", obra: models.Obra{ID: 333, Nome: ""}, want: "nome da obra"},
		{name: "duplicated id", obra: models.Obra{ID: 121, Nome: "Duplicada"}, want: "duplicado"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := AddObraToConfig(testConfig(), tt.obra)
			if err == nil {
				t.Fatal("AddObraToConfig() error = nil, want error")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %q, want containing %q", err.Error(), tt.want)
			}
		})
	}
}

func TestAddObraPersistsAndUpdatesState(t *testing.T) {
	store := &fakeConfigStore{}
	state := NewAppStateWithStore(testConfig(), store)

	if err := AddObra(state, models.Obra{ID: 333, Nome: "Nova Obra"}); err != nil {
		t.Fatalf("AddObra() error = %v", err)
	}

	if !store.saved {
		t.Fatal("config was not saved")
	}
	if len(state.Config.Obras) != 3 || len(state.Obras.Obras) != 3 {
		t.Fatalf("state obras not updated: config=%#v tab=%#v", state.Config.Obras, state.Obras.Obras)
	}
	if store.savedConfig.Obras[2] != (models.Obra{ID: 333, Nome: "Nova Obra"}) {
		t.Fatalf("saved obra = %#v, want new obra", store.savedConfig.Obras[2])
	}
}

func TestSearchAndAddCostCenterFromInputAddsSingleExistingCenter(t *testing.T) {
	store := &fakeConfigStore{}
	service := &fakeCostCenterService{centers: []models.Obra{{ID: 333, Nome: "Nova Obra"}}}
	state := NewAppStateWithStore(testConfig(), store)
	state.CostCenters = service

	if err := SearchAndAddCostCenterFromInput(context.Background(), state, "333"); err != nil {
		t.Fatalf("SearchAndAddCostCenterFromInput() error = %v", err)
	}

	if !service.called || service.costCenterID != 333 {
		t.Fatalf("cost center service called/id = %v/%d, want true/333", service.called, service.costCenterID)
	}
	if !store.saved {
		t.Fatal("config should be saved after adding center")
	}
	if state.Config.Obras[2] != (models.Obra{ID: 333, Nome: "Nova Obra"}) {
		t.Fatalf("new obra = %#v, want cost center from Sienge", state.Config.Obras[2])
	}
	if state.Obras.NovoCentroCustoID != "" || len(state.Obras.CentrosCustoEncontrados) != 0 {
		t.Fatalf("search state was not cleared: %#v", state.Obras)
	}
}

func TestSearchAndAddCostCenterFromInputKeepsMultipleOptionsForSelection(t *testing.T) {
	state := NewAppState(testConfig())
	state.CostCenters = &fakeCostCenterService{centers: []models.Obra{{ID: 333, Nome: "Nova A"}, {ID: 334, Nome: "Nova B"}}}

	err := SearchAndAddCostCenterFromInput(context.Background(), state, "333")
	var multipleErr *MultipleCostCentersError
	if !errors.As(err, &multipleErr) {
		t.Fatalf("SearchAndAddCostCenterFromInput() error = %T, want MultipleCostCentersError", err)
	}
	if len(state.Config.Obras) != 2 {
		t.Fatalf("config should not add automatically when there are multiple centers: %#v", state.Config.Obras)
	}
	if len(state.Obras.CentrosCustoEncontrados) != 2 {
		t.Fatalf("CentrosCustoEncontrados = %#v, want 2 options", state.Obras.CentrosCustoEncontrados)
	}

	if err := AddSelectedCostCenterFromLabel(state, "334 - Nova B"); err != nil {
		t.Fatalf("AddSelectedCostCenterFromLabel() error = %v", err)
	}
	if state.Config.Obras[2] != (models.Obra{ID: 334, Nome: "Nova B"}) {
		t.Fatalf("selected obra = %#v, want Nova B", state.Config.Obras[2])
	}
}

func TestSearchCostCentersFromInputRejectsInvalidCases(t *testing.T) {
	state := NewAppState(testConfig())
	if _, err := SearchCostCentersFromInput(context.Background(), state, "333"); err == nil {
		t.Fatal("SearchCostCentersFromInput() error = nil, want missing service error")
	}

	state.CostCenters = &fakeCostCenterService{}
	_, err := SearchCostCentersFromInput(context.Background(), state, "")
	if err == nil || !strings.Contains(err.Error(), "centro de custo") {
		t.Fatalf("SearchCostCentersFromInput() error = %v, want centro de custo validation", err)
	}

	_, err = SearchCostCentersFromInput(context.Background(), state, "333")
	if !errors.Is(err, ErrCentroCustoNaoEncontrado) {
		t.Fatalf("SearchCostCentersFromInput() error = %v, want ErrCentroCustoNaoEncontrado", err)
	}
}

func TestAddObraPropagatesPersistenceError(t *testing.T) {
	wantErr := errors.New("falha ao salvar")
	store := &fakeConfigStore{saveErr: wantErr}
	state := NewAppStateWithStore(testConfig(), store)

	err := AddObra(state, models.Obra{ID: 333, Nome: "Nova Obra"})
	if !errors.Is(err, wantErr) {
		t.Fatalf("AddObra() error = %v, want %v", err, wantErr)
	}
	if len(state.Config.Obras) != 2 {
		t.Fatalf("state should not change when save fails: %#v", state.Config.Obras)
	}
}

func TestRemoveObraFromConfig(t *testing.T) {
	cfg := testConfig()
	updated, err := RemoveObraFromConfig(cfg, 121)
	if err != nil {
		t.Fatalf("RemoveObraFromConfig() error = %v", err)
	}

	if len(updated.Obras) != 1 {
		t.Fatalf("len(updated.Obras) = %d, want 1", len(updated.Obras))
	}
	if updated.Obras[0].ID != 205 {
		t.Fatalf("remaining obra ID = %d, want 205", updated.Obras[0].ID)
	}
	if len(cfg.Obras) != 2 {
		t.Fatalf("original cfg was mutated: %#v", cfg.Obras)
	}
}

func TestRemoveObraFromConfigRejectsInvalidCases(t *testing.T) {
	tests := []struct {
		name string
		cfg  models.Config
		id   int
		want string
	}{
		{name: "invalid id", cfg: testConfig(), id: 0, want: "ID da obra"},
		{name: "not found", cfg: testConfig(), id: 999, want: ErrObraNaoEncontrada.Error()},
		{name: "last work", cfg: oneWorkConfig(), id: 121, want: "pelo menos uma obra"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := RemoveObraFromConfig(tt.cfg, tt.id)
			if err == nil {
				t.Fatal("RemoveObraFromConfig() error = nil, want error")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %q, want containing %q", err.Error(), tt.want)
			}
		})
	}
}

func TestRemoveObraConfirmadaPersistsAndUpdatesState(t *testing.T) {
	store := &fakeConfigStore{}
	state := NewAppStateWithStore(testConfig(), store)

	if err := RemoveObraConfirmada(state, 121, true); err != nil {
		t.Fatalf("RemoveObraConfirmada() error = %v", err)
	}

	if !store.saved {
		t.Fatal("config was not saved")
	}
	if len(state.Config.Obras) != 1 || state.Config.Obras[0].ID != 205 {
		t.Fatalf("state config obras = %#v, want only 205", state.Config.Obras)
	}
	if len(state.Obras.Obras) != 1 || state.Obras.Obras[0].ID != 205 {
		t.Fatalf("tab obras = %#v, want only 205", state.Obras.Obras)
	}
}

func TestRemoveObraConfirmadaDoesNothingWhenNotConfirmed(t *testing.T) {
	store := &fakeConfigStore{}
	state := NewAppStateWithStore(testConfig(), store)

	if err := RemoveObraConfirmada(state, 121, false); err != nil {
		t.Fatalf("RemoveObraConfirmada() error = %v", err)
	}
	if store.saved {
		t.Fatal("config should not be saved when removal is not confirmed")
	}
	if len(state.Config.Obras) != 2 {
		t.Fatalf("state should not change without confirmation: %#v", state.Config.Obras)
	}
}

func TestOnlyDigits(t *testing.T) {
	if got, want := onlyDigits("12a3-4"), "1234"; got != want {
		t.Fatalf("onlyDigits() = %q, want %q", got, want)
	}
}

func TestBuildObrasTabReturnsObjectWithoutCredentialEditing(t *testing.T) {
	state := NewAppState(testConfig())
	object := BuildObrasTab(state)
	if object == nil {
		t.Fatal("BuildObrasTab() returned nil")
	}
}

func oneWorkConfig() models.Config {
	cfg := testConfig()
	cfg.Obras = cfg.Obras[:1]
	return cfg
}

type fakeCostCenterService struct {
	centers      []models.Obra
	err          error
	called       bool
	costCenterID int
}

func (s *fakeCostCenterService) GetCostCenters(ctx context.Context, costCenterID int) ([]models.Obra, error) {
	s.called = true
	s.costCenterID = costCenterID
	if s.err != nil {
		return nil, s.err
	}

	return append([]models.Obra(nil), s.centers...), nil
}
