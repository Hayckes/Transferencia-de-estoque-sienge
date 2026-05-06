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
	Observacao      string
	Resultados      []models.Insumo
	DetalheAberto   *models.Insumo
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

	observacao := widget.NewMultiLineEntry()
	observacao.SetPlaceHolder("Observacao local da consulta")
	observacao.SetText(state.Consulta.Observacao)
	observacao.OnChanged = func(value string) {
		state.Consulta.Observacao = value
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
			state.Refresh()
		})
	})

	resultRows := make([]fyne.CanvasObject, 0, len(state.Consulta.Resultados)+1)
	resultRows = append(resultRows, widget.NewLabel("ID | Nome | Detalhe | Marca | Qtd. em Estoque"))
	for index, item := range state.Consulta.Resultados {
		rowIndex := index
		label := widget.NewLabel(ConsultaResultRow(item))
		detalhes := widget.NewButton("Detalhes", func() {
			item, err := LoadConsultaDetalhe(context.Background(), state, rowIndex)
			if err != nil {
				if MaybeShowCredentialReonboarding(state, err, status.SetText) {
					return
				}
				status.SetText(err.Error())
				return
			}
			ShowInsumoDetailsModal(state.Window, item)
			status.SetText("Detalhes carregados.")
		})
		resultRows = append(resultRows, container.NewHBox(label, detalhes))
	}

	limpar := widget.NewButton("Limpar", func() {
		ClearConsulta(state)
		obraSelect.ClearSelected()
		idsEntry.SetText("")
		observacao.SetText("")
		status.SetText("Consulta limpa.")
		state.Refresh()
	})

	return container.NewVBox(
		widget.NewLabel("Consulta de estoque"),
		container.NewHBox(obraSelect, idsEntry, consultar, limpar),
		observacao,
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
	state.Consulta.DetalheAberto = nil
	return nil
}

func LoadConsultaDetalhe(ctx context.Context, state *AppState, resultIndex int) (models.Insumo, error) {
	if state.Stock == nil {
		return models.Insumo{}, errors.New("servico de estoque nao configurado")
	}
	if resultIndex < 0 || resultIndex >= len(state.Consulta.Resultados) {
		return models.Insumo{}, errors.New("insumo selecionado nao encontrado")
	}

	obraID, ok := ObraIDFromLabel(state.Config.Obras, state.Consulta.ObraSelecionada)
	if !ok {
		return models.Insumo{}, ErrObraConsultaObrigatoria
	}

	item := state.Consulta.Resultados[resultIndex]
	appropriations, err := state.Stock.GetBuildingAppropriations(ctx, obraID, item.ID)
	if err != nil {
		return models.Insumo{}, err
	}

	item.Apropriacoes = append([]models.Apropriacao(nil), appropriations...)
	state.Consulta.Resultados[resultIndex] = item
	state.Consulta.DetalheAberto = &item
	return item, nil
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
	label = strings.TrimSpace(label)
	for _, obra := range obras {
		if obra.Label() == label {
			return obra.ID, true
		}
	}

	return 0, false
}

func ConsultaResultRow(item models.Insumo) string {
	return fmt.Sprintf("%d | %s | %s | %s | %s", item.ID, item.Nome, item.Detalhe, item.Marca, models.FormatQuantidade(item.Quantidade, item.Unidade))
}

func BuildAppropriationDetailsText(item models.Insumo) string {
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("%s %s - %s", item.Nome, item.Detalhe, item.Marca))
	for _, appropriation := range item.Apropriacoes {
		builder.WriteString(fmt.Sprintf("\n%s | %s | %s", appropriation.Codigo, appropriation.Descricao, models.FormatQuantidade(appropriation.Quantidade, item.Unidade)))
	}

	return builder.String()
}
