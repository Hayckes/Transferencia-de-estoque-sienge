package ui

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"sienge-transfer/models"
)

type ConsultaTabState struct {
	ObraSelecionada string
	InsumoIDsInput  string
	Resultados      []models.Insumo
}

var ErrObraConsultaObrigatoria = errors.New("selecione uma obra para consultar")

func NewConsultaTabState() ConsultaTabState {
	return ConsultaTabState{}
}

func BuildConsultaTab(state *AppState) fyne.CanvasObject {
	obraSelect := widget.NewSelect(ObraLabels(state.Config.Obras), func(value string) {
		state.Consulta.ObraSelecionada = value
	})
	obraSelect.PlaceHolder = "Selecione a obra"
	obraSelect.SetSelected(state.Consulta.ObraSelecionada)

	idsEntry := widget.NewEntry()
	idsEntry.SetPlaceHolder("IDs dos insumos separados por virgula ou espaco")
	idsEntry.SetText(state.Consulta.InsumoIDsInput)
	idsEntry.OnChanged = func(value string) {
		state.Consulta.InsumoIDsInput = value
	}

	status := widget.NewLabel("")
	consultar := widget.NewButton("Consultar", func() {
		status.SetText(StatusLoading)
		state.Runner.Run(func() error {
			return RunConsulta(context.Background(), state)
		}, func(err error) {
			if err != nil {
				if MaybeShowCredentialReonboarding(state, err, status.SetText) {
					return
				}
				status.SetText(err.Error())
				return
			}
			status.SetText(fmt.Sprintf("Consulta concluida. %d item(ns) encontrado(s).", len(state.Consulta.Resultados)))
			state.RefreshTab(TabConsulta)
		})
	})

	resultRows := make([]fyne.CanvasObject, 0, len(state.Consulta.Resultados)+1)
	resultRows = append(resultRows, widget.NewLabel("ID | Nome | Detalhe | Marca | Qtd. em Estoque"))
	for _, item := range state.Consulta.Resultados {
		resultRows = append(resultRows, widget.NewLabel(ConsultaResultRow(item)))
	}

	limpar := widget.NewButton("Limpar", func() {
		ClearConsulta(state)
		obraSelect.ClearSelected()
		idsEntry.SetText("")
		status.SetText("Consulta limpa.")
		state.RefreshTab(TabConsulta)
	})

	return container.NewVBox(
		widget.NewLabel("Consulta de estoque"),
		container.NewHBox(obraSelect, withMinTypingInputWidth(idsEntry), consultar, limpar),
		status,
		container.NewVBox(resultRows...),
	)
}

func ValidateConsultaInput(state *AppState) (int, []int, error) {
	obraID, ok := ObraIDFromLabel(state.Config.Obras, state.Consulta.ObraSelecionada)
	if !ok {
		return 0, nil, ErrObraConsultaObrigatoria
	}

	ids, err := models.ParseInsumoIDs(state.Consulta.InsumoIDsInput)
	if err != nil {
		return 0, nil, err
	}

	return obraID, ids, nil
}

func RunConsulta(ctx context.Context, state *AppState) error {
	if state.Stock == nil {
		return errors.New("servico de estoque nao configurado")
	}

	obraID, ids, err := ValidateConsultaInput(state)
	if err != nil {
		return err
	}

	items, err := state.Stock.GetStockItemsByIDs(ctx, obraID, ids)
	if err != nil {
		return err
	}

	state.Consulta.Resultados = append([]models.Insumo(nil), items...)
	return nil
}

func ClearConsulta(state *AppState) {
	state.Consulta = NewConsultaTabState()
}

func ObraLabels(obras []models.Obra) []string {
	labels := make([]string, 0, len(obras))
	for _, obra := range obras {
		labels = append(labels, obra.Label())
	}

	return labels
}

func ObraIDFromLabel(obras []models.Obra, label string) (int, bool) {
	obra, ok := ObraFromLabel(obras, label)
	if !ok {
		return 0, false
	}

	return obra.ID, true
}

func ObraFromLabel(obras []models.Obra, label string) (models.Obra, bool) {
	label = strings.TrimSpace(label)
	for _, obra := range obras {
		if obra.Label() == label {
			return obra, true
		}
	}

	return models.Obra{}, false
}

func ConsultaResultRow(item models.Insumo) string {
	return fmt.Sprintf("%d | %s | %s | %s | %s", item.ID, item.Nome, item.Detalhe, item.Marca, models.FormatQuantidade(item.Quantidade, item.Unidade))
}
