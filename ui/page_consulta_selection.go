package ui

import "sienge-transfer/models"

type ConsultaSelectionState struct {
	ObrasSelecionadas   []models.Obra
	ConsultarTodasObras bool
}

func ToggleSelectAllWorks(allWorks []models.Obra, checked bool) ConsultaSelectionState {
	if !checked {
		return ConsultaSelectionState{}
	}
	return ConsultaSelectionState{ObrasSelecionadas: append([]models.Obra(nil), allWorks...), ConsultarTodasObras: true}
}

func ToggleSingleWork(allWorks []models.Obra, current ConsultaSelectionState, obra models.Obra, checked bool) ConsultaSelectionState {
	selected := append([]models.Obra(nil), current.ObrasSelecionadas...)
	if checked {
		if !isConsultaObraSelecionada(selected, obra.ID) {
			selected = append(selected, obra)
		}
	} else {
		filtered := selected[:0]
		for _, currentObra := range selected {
			if currentObra.ID != obra.ID {
				filtered = append(filtered, currentObra)
			}
		}
		selected = filtered
	}
	return ConsultaSelectionState{ObrasSelecionadas: selected, ConsultarTodasObras: len(allWorks) > 0 && len(selected) == len(allWorks)}
}
