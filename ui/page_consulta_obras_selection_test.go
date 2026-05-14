package ui

import "testing"

func TestToggleSelectAllWorks_ChecksAllWorks(t *testing.T) {
	obras := testConfig().Obras
	selection := ToggleSelectAllWorks(obras, true)

	if !selection.ConsultarTodasObras || len(selection.ObrasSelecionadas) != len(obras) {
		t.Fatalf("selection = %#v, want all works selected", selection)
	}
}

func TestToggleSelectAllWorks_UnchecksAllWorks(t *testing.T) {
	obras := testConfig().Obras
	selection := ToggleSelectAllWorks(obras, false)

	if selection.ConsultarTodasObras || len(selection.ObrasSelecionadas) != 0 {
		t.Fatalf("selection = %#v, want all works cleared", selection)
	}
}

func TestToggleSingleWork_UnchecksSelectAllWhenOneWorkIsUnchecked(t *testing.T) {
	obras := testConfig().Obras
	selection := ToggleSingleWork(obras, ToggleSelectAllWorks(obras, true), obras[0], false)

	if selection.ConsultarTodasObras || len(selection.ObrasSelecionadas) != len(obras)-1 {
		t.Fatalf("selection = %#v, want select all unchecked and one work removed", selection)
	}
}

func TestToggleSingleWork_ChecksSelectAllWhenAllWorksAreSelectedManually(t *testing.T) {
	obras := testConfig().Obras
	selection := ConsultaSelectionState{}
	for _, obra := range obras {
		selection = ToggleSingleWork(obras, selection, obra, true)
	}

	if !selection.ConsultarTodasObras || len(selection.ObrasSelecionadas) != len(obras) {
		t.Fatalf("selection = %#v, want select all checked after all manual selections", selection)
	}
}
