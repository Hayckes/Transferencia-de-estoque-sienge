package ui

import (
	"testing"

	"sienge-transfer/models"
)

func TestToggleSelectAllWorks_ChecksAllWorks(t *testing.T) {
	works := []models.Obra{{ID: 1}, {ID: 2}}
	state := ToggleSelectAllWorks(works, true)
	if !state.ConsultarTodasObras || len(state.ObrasSelecionadas) != 2 {
		t.Fatalf("state = %#v, want all selected", state)
	}
}

func TestToggleSelectAllWorks_UnchecksAllWorks(t *testing.T) {
	state := ToggleSelectAllWorks([]models.Obra{{ID: 1}}, false)
	if state.ConsultarTodasObras || len(state.ObrasSelecionadas) != 0 {
		t.Fatalf("state = %#v, want none selected", state)
	}
}

func TestToggleSingleWork_UnchecksSelectAllWhenOneWorkIsUnchecked(t *testing.T) {
	works := []models.Obra{{ID: 1}, {ID: 2}}
	state := ToggleSingleWork(works, ConsultaSelectionState{ObrasSelecionadas: works, ConsultarTodasObras: true}, works[0], false)
	if state.ConsultarTodasObras || len(state.ObrasSelecionadas) != 1 {
		t.Fatalf("state = %#v, want select all unchecked", state)
	}
}

func TestToggleSingleWork_ChecksSelectAllWhenAllWorksAreSelectedManually(t *testing.T) {
	works := []models.Obra{{ID: 1}, {ID: 2}}
	state := ToggleSingleWork(works, ConsultaSelectionState{ObrasSelecionadas: []models.Obra{works[0]}}, works[1], true)
	if !state.ConsultarTodasObras || len(state.ObrasSelecionadas) != 2 {
		t.Fatalf("state = %#v, want select all checked", state)
	}
}
