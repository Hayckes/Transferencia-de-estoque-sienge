package ui

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"sienge-transfer/models"
)

var ErrObraNaoEncontrada = errors.New("obra nao encontrada")

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
	idEntry := widget.NewEntry()
	idEntry.SetPlaceHolder("ID da obra")
	idEntry.OnChanged = func(value string) {
		filtered := onlyDigits(value)
		if filtered != value {
			idEntry.SetText(filtered)
			return
		}
		state.Obras.NovoObraID = filtered
	}

	nomeEntry := widget.NewEntry()
	nomeEntry.SetPlaceHolder("Nome da obra")
	nomeEntry.OnChanged = func(value string) {
		state.Obras.NovoNome = value
	}

	status := widget.NewLabel("")
	addButton := widget.NewButton("Adicionar Obra", func() {
		if err := AddObraFromInput(state, state.Obras.NovoObraID, state.Obras.NovoNome); err != nil {
			status.SetText(err.Error())
			return
		}
		idEntry.SetText("")
		nomeEntry.SetText("")
		status.SetText("Obra adicionada com sucesso.")
	})

	items := make([]fyne.CanvasObject, 0, len(state.Obras.Obras)+4)
	items = append(items,
		widget.NewLabel(fmt.Sprintf("Usuario: %s", state.Obras.UsuarioNome)),
		widget.NewLabel(fmt.Sprintf("Cargo: %s", state.Obras.UsuarioCargo)),
		container.NewHBox(idEntry, nomeEntry, addButton),
		status,
	)
	for _, obra := range state.Obras.Obras {
		obraID := obra.ID
		items = append(items, container.NewHBox(
			widget.NewLabel(obra.Label()),
			widget.NewButton("Remover", func() {
				if err := RemoveObraConfirmada(state, obraID, true); err != nil {
					status.SetText(err.Error())
					return
				}
				status.SetText("Obra removida com sucesso.")
			}),
		))
	}

	return container.NewVBox(items...)
}

func AddObraFromInput(state *AppState, idInput, nomeInput string) error {
	id, err := parseObraID(idInput)
	if err != nil {
		return err
	}

	return AddObra(state, models.Obra{ID: id, Nome: nomeInput})
}

func AddObra(state *AppState, obra models.Obra) error {
	updated, err := AddObraToConfig(state.Config, obra)
	if err != nil {
		return err
	}
	if err := persistConfig(state, updated); err != nil {
		return err
	}

	state.Config = updated
	state.Obras.Obras = append([]models.Obra(nil), updated.Obras...)
	state.Obras.NovoObraID = ""
	state.Obras.NovoNome = ""
	return nil
}

func RemoveObraConfirmada(state *AppState, obraID int, confirmed bool) error {
	if !confirmed {
		return nil
	}

	updated, err := RemoveObraFromConfig(state.Config, obraID)
	if err != nil {
		return err
	}
	if err := persistConfig(state, updated); err != nil {
		return err
	}

	state.Config = updated
	state.Obras.Obras = append([]models.Obra(nil), updated.Obras...)
	return nil
}

func AddObraToConfig(cfg models.Config, obra models.Obra) (models.Config, error) {
	obra.Nome = strings.TrimSpace(obra.Nome)
	if obra.ID <= 0 {
		return models.Config{}, &ValidationError{Errors: []string{"ID da obra deve ser numerico positivo"}}
	}
	if obra.Nome == "" {
		return models.Config{}, &ValidationError{Errors: []string{"nome da obra obrigatorio"}}
	}
	for _, existing := range cfg.Obras {
		if existing.ID == obra.ID {
			return models.Config{}, &ValidationError{Errors: []string{"ID da obra duplicado"}}
		}
	}

	updated := cfg
	updated.Obras = append(append([]models.Obra(nil), cfg.Obras...), obra)
	return updated, nil
}

func RemoveObraFromConfig(cfg models.Config, obraID int) (models.Config, error) {
	if obraID <= 0 {
		return models.Config{}, &ValidationError{Errors: []string{"ID da obra deve ser numerico positivo"}}
	}
	if len(cfg.Obras) <= 1 {
		return models.Config{}, &ValidationError{Errors: []string{"mantenha pelo menos uma obra cadastrada"}}
	}

	updated := cfg
	updated.Obras = make([]models.Obra, 0, len(cfg.Obras)-1)
	found := false
	for _, obra := range cfg.Obras {
		if obra.ID == obraID {
			found = true
			continue
		}
		updated.Obras = append(updated.Obras, obra)
	}
	if !found {
		return models.Config{}, ErrObraNaoEncontrada
	}

	return updated, nil
}

func persistConfig(state *AppState, cfg models.Config) error {
	if state.Store == nil {
		return nil
	}

	return state.Store.Save(cfg)
}

func parseObraID(input string) (int, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return 0, &ValidationError{Errors: []string{"ID da obra obrigatorio"}}
	}
	id, err := strconv.Atoi(input)
	if err != nil || id <= 0 {
		return 0, &ValidationError{Errors: []string{"ID da obra deve ser numerico positivo"}}
	}

	return id, nil
}

func onlyDigits(value string) string {
	var builder strings.Builder
	for _, r := range value {
		if r >= '0' && r <= '9' {
			builder.WriteRune(r)
		}
	}

	return builder.String()
}
