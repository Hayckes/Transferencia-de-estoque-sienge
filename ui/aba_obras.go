package ui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"sienge-transfer/models"
)

type ObrasTabState struct {
	UsuarioNome  string
	UsuarioCargo string
	Obras        []models.Obra
	NovoObraID   string
	NovoNome     string
}

func NewObrasTabState(cfg models.Config) ObrasTabState {
	return ObrasTabState{
		UsuarioNome:  cfg.Usuario.Nome,
		UsuarioCargo: cfg.Usuario.Cargo,
		Obras:        append([]models.Obra(nil), cfg.Obras...),
	}
}

func BuildObrasTab(state *AppState) fyne.CanvasObject {
	items := make([]fyne.CanvasObject, 0, len(state.Obras.Obras)+2)
	items = append(items,
		widget.NewLabel(fmt.Sprintf("Usuario: %s", state.Obras.UsuarioNome)),
		widget.NewLabel(fmt.Sprintf("Cargo: %s", state.Obras.UsuarioCargo)),
	)
	for _, obra := range state.Obras.Obras {
		items = append(items, widget.NewLabel(obra.Label()))
	}

	return container.NewVBox(items...)
}
