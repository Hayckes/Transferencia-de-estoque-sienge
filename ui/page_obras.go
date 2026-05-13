package ui

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"sienge-transfer/models"
)

var (
	ErrObraNaoEncontrada                = errors.New("obra nao encontrada")
	ErrCentroCustoNaoEncontrado         = errors.New("centro de custo nao encontrado no Sienge")
	ErrCentroCustoSelecaoObrigatoria    = errors.New("selecione o centro de custo que deseja adicionar")
	ErrMultiplosCentrosCustoEncontrados = errors.New("foram encontrados multiplos centros de custo; selecione qual deseja adicionar")
)

type MultipleCostCentersError struct {
	Options []models.Obra
}

func (e *MultipleCostCentersError) Error() string {
	return ErrMultiplosCentrosCustoEncontrados.Error()
}

type ObrasTabState struct {
	UsuarioNome             string
	UsuarioCargo            string
	Obras                   []models.Obra
	NovoCentroCustoID       string
	CentrosCustoEncontrados []models.Obra
	CentroCustoSelecionado  string
	Status                  string
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
	idEntry.SetPlaceHolder("ID do centro de custo")
	idEntry.SetText(state.Obras.NovoCentroCustoID)
	idEntry.OnChanged = func(value string) {
		filtered := onlyDigits(value)
		if filtered != value {
			idEntry.SetText(filtered)
			return
		}
		state.Obras.NovoCentroCustoID = filtered
	}

	status := NewStatusView(state.Window, state.Obras.Status)
	buscarButton := widget.NewButton("Buscar", func() {
		setObrasStatus(state, status, StatusLoading)
		state.Runner.Run(func() error {
			return SearchAndAddCostCenterFromInput(context.Background(), state, state.Obras.NovoCentroCustoID)
		}, func(err error) {
			if err != nil {
				var multipleErr *MultipleCostCentersError
				if errors.As(err, &multipleErr) {
					setObrasStatus(state, status, err.Error())
					state.Refresh()
					return
				}
				if MaybeShowCredentialReonboarding(state, err, func(message string) { setObrasStatus(state, status, message) }) {
					return
				}
				setObrasStatus(state, status, err.Error())
				return
			}
			idEntry.SetText("")
			setObrasStatus(state, status, "Obra adicionada com sucesso.")
			state.Refresh()
		})
	})

	items := make([]fyne.CanvasObject, 0, len(state.Obras.Obras)+6)
	items = append(items,
		widget.NewLabel(fmt.Sprintf("Usuario: %s", state.Obras.UsuarioNome)),
		widget.NewLabel(fmt.Sprintf("Cargo: %s", state.Obras.UsuarioCargo)),
		responsiveRow(expandingInput(idEntry), buscarButton),
	)
	if len(state.Obras.CentrosCustoEncontrados) > 1 {
		centroCustoSelect := widget.NewSelect(ObraLabels(state.Obras.CentrosCustoEncontrados), func(value string) {
			state.Obras.CentroCustoSelecionado = value
		})
		centroCustoSelect.PlaceHolder = "Selecione o centro de custo"
		centroCustoSelect.SetSelected(state.Obras.CentroCustoSelecionado)
		items = append(items, responsiveRow(
			expandingInput(centroCustoSelect),
			widget.NewButton("Adicionar selecionado", func() {
				if err := AddSelectedCostCenterFromLabel(state, state.Obras.CentroCustoSelecionado); err != nil {
					setObrasStatus(state, status, err.Error())
					return
				}
				setObrasStatus(state, status, "Obra adicionada com sucesso.")
				state.Refresh()
			}),
		))
	}
	items = append(items, status.Object())
	for _, obra := range state.Obras.Obras {
		obraID := obra.ID
		items = append(items, container.NewHBox(
			widget.NewLabel(obra.Label()),
			widget.NewButton("Remover", func() {
				remove := func() {
					if err := RemoveObraConfirmada(state, obraID, true); err != nil {
						setObrasStatus(state, status, err.Error())
						return
					}
					setObrasStatus(state, status, "Obra removida com sucesso.")
					state.Refresh()
				}
				if state.Window == nil {
					remove()
					return
				}
				ShowConfirmRemoveObra(state.Window, remove)
			}),
		))
	}

	return scrollablePage(items...)
}

func SearchAndAddCostCenterFromInput(ctx context.Context, state *AppState, idInput string) error {
	centers, err := SearchCostCentersFromInput(ctx, state, idInput)
	if err != nil {
		return err
	}
	if len(centers) > 1 {
		state.Obras.CentrosCustoEncontrados = append([]models.Obra(nil), centers...)
		state.Obras.CentroCustoSelecionado = ""
		return &MultipleCostCentersError{Options: centers}
	}

	return AddObra(state, centers[0])
}

func SearchCostCentersFromInput(ctx context.Context, state *AppState, idInput string) ([]models.Obra, error) {
	if state.CostCenters == nil {
		return nil, errors.New("servico de centro de custo nao configurado")
	}
	id, err := parseCostCenterID(idInput)
	if err != nil {
		return nil, err
	}

	state.Obras.CentrosCustoEncontrados = nil
	state.Obras.CentroCustoSelecionado = ""
	centers, err := state.CostCenters.GetCostCenters(ctx, id)
	if err != nil {
		return nil, err
	}
	if len(centers) == 0 {
		return nil, ErrCentroCustoNaoEncontrado
	}

	return append([]models.Obra(nil), centers...), nil
}

func AddSelectedCostCenterFromLabel(state *AppState, label string) error {
	obra, ok := ObraFromLabel(state.Obras.CentrosCustoEncontrados, label)
	if !ok {
		return ErrCentroCustoSelecaoObrigatoria
	}

	return AddObra(state, obra)
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
	state.Obras.NovoCentroCustoID = ""
	state.Obras.CentrosCustoEncontrados = nil
	state.Obras.CentroCustoSelecionado = ""
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

func parseCostCenterID(input string) (int, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return 0, &ValidationError{Errors: []string{"ID do centro de custo obrigatorio"}}
	}
	id, err := strconv.Atoi(input)
	if err != nil || id <= 0 {
		return 0, &ValidationError{Errors: []string{"ID do centro de custo deve ser numerico positivo"}}
	}

	return id, nil
}

func setObrasStatus(state *AppState, label interface{ SetText(string) }, message string) {
	state.Obras.Status = message
	label.SetText(message)
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
