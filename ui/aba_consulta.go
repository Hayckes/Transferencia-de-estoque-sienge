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
	TipoConsulta           models.ConsultaTipo
	ObraSelecionada        string
	ObrasSelecionadas      []models.Obra
	ConsultarTodasObras    bool
	InsumoIDsInput         string
	SolicitacaoCompraID    string
	SolicitacaoObraID      string
	Resultados             []models.Insumo
	ResultadosNormalizados []models.ConsultaResultado
	DetalheAberto          *models.Insumo
	SolicitacaoItensCount  int
}

type ConsultaViewModel struct {
	Tipo                  models.ConsultaTipo
	ShowInsumoInput       bool
	ShowSolicitacaoInputs bool
	ShowObrasSelector     bool
}

var ErrObraConsultaObrigatoria = errors.New("selecione uma obra para consultar")

func NewConsultaTabState() ConsultaTabState {
	return ConsultaTabState{TipoConsulta: models.ConsultaPorInsumo}
}

func BuildConsultaTab(state *AppState) fyne.CanvasObject {
	tipoSelect := widget.NewSelect([]string{"Por insumo", "Por solicitacao de compra"}, nil)
	tipoSelect.SetSelected(consultaTipoLabel(effectiveConsultaTipo(state.Consulta.TipoConsulta)))

	idsEntry := widget.NewEntry()
	idsEntry.SetPlaceHolder("IDs dos insumos separados por virgula ou espaco")
	idsEntry.SetText(state.Consulta.InsumoIDsInput)
	idsEntry.OnChanged = func(value string) {
		state.Consulta.InsumoIDsInput = value
	}
	solicitacaoEntry := widget.NewEntry()
	solicitacaoEntry.SetPlaceHolder("ID da solicitacao de compra")
	solicitacaoEntry.SetText(state.Consulta.SolicitacaoCompraID)
	solicitacaoEntry.OnChanged = func(value string) { state.Consulta.SolicitacaoCompraID = value }
	solicitacaoObraEntry := widget.NewEntry()
	solicitacaoObraEntry.SetPlaceHolder("ID da obra da solicitacao")
	solicitacaoObraEntry.SetText(state.Consulta.SolicitacaoObraID)
	solicitacaoObraEntry.OnChanged = func(value string) { state.Consulta.SolicitacaoObraID = value }

	updatingWorkChecks := false
	var workCheckWidgets []*widget.Check
	allWorksCheck := widget.NewCheck("Todas as obras cadastradas", func(checked bool) {
		if updatingWorkChecks {
			return
		}
		selection := ToggleSelectAllWorks(state.Config.Obras, checked)
		state.Consulta.ObrasSelecionadas = selection.ObrasSelecionadas
		state.Consulta.ConsultarTodasObras = selection.ConsultarTodasObras
		updatingWorkChecks = true
		for _, check := range workCheckWidgets {
			check.SetChecked(checked)
		}
		updatingWorkChecks = false
	})
	allWorksCheck.SetChecked(state.Consulta.ConsultarTodasObras)
	workChecks := []fyne.CanvasObject{allWorksCheck}
	for _, obra := range state.Config.Obras {
		selectedObra := obra
		check := widget.NewCheck(obra.Label(), func(checked bool) {
			if updatingWorkChecks {
				return
			}
			selection := ToggleSingleWork(state.Config.Obras, ConsultaSelectionState{ObrasSelecionadas: state.Consulta.ObrasSelecionadas, ConsultarTodasObras: state.Consulta.ConsultarTodasObras}, selectedObra, checked)
			state.Consulta.ObrasSelecionadas = selection.ObrasSelecionadas
			state.Consulta.ConsultarTodasObras = selection.ConsultarTodasObras
			updatingWorkChecks = true
			allWorksCheck.SetChecked(selection.ConsultarTodasObras)
			updatingWorkChecks = false
		})
		check.SetChecked(isConsultaObraSelecionada(state.Consulta.ObrasSelecionadas, selectedObra.ID))
		workCheckWidgets = append(workCheckWidgets, check)
		workChecks = append(workChecks, check)
	}
	worksSelector := container.NewVBox(workChecks...)

	status := NewStatusView(state.Window, "")
	consultar := widget.NewButton("Consultar", func() {
		status.SetText(StatusLoading)
		state.Runner.Run(func() error {
			return RunConsulta(context.Background(), state)
		}, func(err error) {
			if err != nil {
				if MaybeShowCredentialReonboarding(state, err, status.SetText) {
					return
				}
				status.SetText(consultaErrorFeedback(state, err))
				return
			}
			status.SetText(consultaSuccessFeedback(state))
			state.RefreshTab(TabConsulta)
		})
	})

	selectedResultIndex := -1
	detailsButton := widget.NewButton("Detalhes do item selecionado", func() {
		if selectedResultIndex < 0 {
			status.SetText("Selecione um item na tabela.")
			return
		}
		status.SetText(StatusLoading)
		state.Runner.Run(func() error {
			_, err := LoadConsultaDetalhe(context.Background(), state, selectedResultIndex)
			return err
		}, func(err error) {
			if err != nil {
				if MaybeShowCredentialReonboarding(state, err, status.SetText) {
					return
				}
				status.SetText(err.Error())
				return
			}
			if state.Consulta.DetalheAberto != nil {
				ShowInsumoDetailsModal(state.Window, *state.Consulta.DetalheAberto)
				status.SetText("Detalhes carregados.")
			}
		})
	})
	detailsButton.Disable()
	results := ConsultaResultsForDisplay(state)
	resultTableRows := BuildConsultaInsumoRows(results)
	resultsTable := newInsumoResultsTable(&resultTableRows, func(index int) {
		selectedResultIndex = index
		detailsButton.Enable()
	})

	limpar := widget.NewButton("Limpar", func() {
		ClearConsulta(state)
		idsEntry.SetText("")
		solicitacaoEntry.SetText("")
		solicitacaoObraEntry.SetText("")
		status.SetText("Consulta limpa.")
		state.RefreshTab(TabConsulta)
	})

	resultsBox := container.NewBorder(nil, detailsButton, nil, nil, resultsTable)
	insumoSection := container.NewVBox(
		widget.NewLabel("IDs dos insumos"),
		responsiveRow(expandingInput(idsEntry)),
	)
	solicitacaoSection := container.NewVBox(
		widget.NewLabel("Dados da solicitacao de compra"),
		responsiveRow(expandingInput(solicitacaoEntry), expandingInput(solicitacaoObraEntry)),
	)
	applyConsultaViewModel(BuildConsultaViewModel(state.Consulta), insumoSection, solicitacaoSection)
	tipoSelect.OnChanged = func(value string) {
		state.Consulta.TipoConsulta = consultaTipoFromLabel(value)
		ClearConsultaResults(state)
		applyConsultaViewModel(BuildConsultaViewModel(state.Consulta), insumoSection, solicitacaoSection)
		selectedResultIndex = -1
		detailsButton.Disable()
		resultTableRows = nil
		resultsTable.Refresh()
	}
	worksScroll := container.NewVScroll(worksSelector)
	worksScroll.SetMinSize(fyne.NewSize(0, 130))
	topContent := container.NewPadded(container.NewVBox(
		widget.NewLabel("Consulta de estoque"),
		responsiveRow(expandingInput(tipoSelect), consultar, limpar),
		widget.NewLabel("Obras onde buscar estoque"),
		worksScroll,
		insumoSection,
		solicitacaoSection,
		status.Object(),
	))

	return container.NewBorder(topContent, nil, nil, nil, resultsBox)
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
	if effectiveConsultaTipo(state.Consulta.TipoConsulta) == models.ConsultaPorSolicitacaoCompra {
		return RunConsultaPorSolicitacao(ctx, state)
	}
	return RunConsultaPorInsumo(ctx, state)
}

func RunConsultaPorInsumo(ctx context.Context, state *AppState) error {
	ids, err := models.ParseInsumoIDs(state.Consulta.InsumoIDsInput)
	if err != nil {
		return err
	}
	obras, err := ResolveConsultaObras(state)
	if err != nil {
		return err
	}

	stockByWork := make(map[int][]models.Insumo, len(obras))
	for _, obra := range obras {
		items, err := state.Stock.GetStockItemsByIDs(ctx, obra.ID, ids)
		if err != nil {
			return err
		}
		stockByWork[obra.ID] = items
	}

	state.Consulta.ResultadosNormalizados = models.BuildConsultaPorInsumoResults(obras, stockByWork)
	state.Consulta.Resultados = consultaResultsToInsumos(state.Consulta.ResultadosNormalizados)
	state.Consulta.DetalheAberto = nil
	return nil
}

func RunConsultaPorSolicitacao(ctx context.Context, state *AppState) error {
	if state.PurchaseRequests == nil {
		return errors.New("servico de solicitacao de compra nao configurado")
	}
	purchaseRequestID, err := parseObraID(state.Consulta.SolicitacaoCompraID)
	if err != nil {
		return err
	}
	requestBuildingID, err := parseObraID(state.Consulta.SolicitacaoObraID)
	if err != nil {
		return err
	}
	obras, err := ResolveConsultaObras(state)
	if err != nil {
		return err
	}

	requestItems, err := state.PurchaseRequests.GetPurchaseRequestItems(ctx, purchaseRequestID, requestBuildingID)
	if err != nil {
		return err
	}
	state.Consulta.SolicitacaoItensCount = len(requestItems)
	ids := uniqueRequestResourceIDs(requestItems)
	if len(ids) == 0 {
		state.Consulta.Resultados = nil
		state.Consulta.ResultadosNormalizados = nil
		state.Consulta.DetalheAberto = nil
		return nil
	}

	stockByWork := make(map[int][]models.Insumo, len(obras))
	for _, obra := range obras {
		items, err := state.Stock.GetStockItemsByIDs(ctx, obra.ID, ids)
		if err != nil {
			return err
		}
		stockByWork[obra.ID] = items
	}

	state.Consulta.ResultadosNormalizados = models.BuildConsultaPorSolicitacaoResults(obras, requestItems, stockByWork)
	state.Consulta.Resultados = consultaResultsToInsumos(state.Consulta.ResultadosNormalizados)
	state.Consulta.DetalheAberto = nil
	return nil
}

func consultaSuccessFeedback(state *AppState) string {
	count := len(state.Consulta.Resultados)
	if effectiveConsultaTipo(state.Consulta.TipoConsulta) == models.ConsultaPorSolicitacaoCompra {
		if state.Consulta.SolicitacaoItensCount == 0 {
			return PurchaseRequestNoItemsFeedback(state.Consulta.SolicitacaoCompraID, state.Consulta.SolicitacaoObraID)
		}
		if count == 0 {
			return PurchaseRequestNoStockFeedback()
		}
		return fmt.Sprintf("Consulta concluida. %d item(ns) encontrado(s) em estoque.", count)
	}
	if count == 0 {
		return ConsultaNoResultsFeedback()
	}
	return fmt.Sprintf("Consulta concluida. %d item(ns) encontrado(s).", count)
}

func consultaErrorFeedback(state *AppState, err error) string {
	if effectiveConsultaTipo(state.Consulta.TipoConsulta) == models.ConsultaPorSolicitacaoCompra {
		return "Erro ao consultar solicitacao de compra: " + err.Error()
	}
	return "Erro ao consultar estoque: " + err.Error()
}

func ResolveConsultaObras(state *AppState) ([]models.Obra, error) {
	selected := append([]models.Obra(nil), state.Consulta.ObrasSelecionadas...)
	if len(selected) == 0 && strings.TrimSpace(state.Consulta.ObraSelecionada) != "" {
		if obra, ok := ObraFromLabel(state.Config.Obras, state.Consulta.ObraSelecionada); ok {
			selected = append(selected, obra)
		}
	}
	return models.ResolveObrasParaConsulta(state.Config.Obras, selected, state.Consulta.ConsultarTodasObras)
}

func uniqueRequestResourceIDs(items []models.PurchaseRequestItem) []int {
	ids := make([]int, 0, len(items))
	seen := make(map[int]bool, len(items))
	for _, item := range items {
		if item.ResourceID > 0 && !seen[item.ResourceID] {
			ids = append(ids, item.ResourceID)
			seen[item.ResourceID] = true
		}
	}
	return ids
}

func consultaResultsToInsumos(results []models.ConsultaResultado) []models.Insumo {
	items := make([]models.Insumo, 0, len(results))
	for _, result := range results {
		items = append(items, models.Insumo{
			ID:           result.InsumoID,
			Nome:         result.InsumoNome,
			Detalhe:      result.Detalhe,
			DetalheID:    result.DetalheID,
			Marca:        result.Marca,
			MarcaID:      result.MarcaID,
			Unidade:      result.Unidade,
			Quantidade:   result.Quantidade,
			Apropriacoes: append([]models.Apropriacao(nil), result.Apropriacoes...),
		})
	}
	return items
}

func LoadConsultaDetalhe(ctx context.Context, state *AppState, resultIndex int) (models.Insumo, error) {
	if state.Stock == nil {
		return models.Insumo{}, errors.New("servico de estoque nao configurado")
	}
	results := ConsultaResultsForDisplay(state)
	if resultIndex < 0 || resultIndex >= len(results) {
		return models.Insumo{}, errors.New("insumo selecionado nao encontrado")
	}
	result := results[resultIndex]
	item := models.Insumo{ID: result.InsumoID, Nome: result.InsumoNome, Detalhe: result.Detalhe, DetalheID: result.DetalheID, Marca: result.Marca, MarcaID: result.MarcaID, Unidade: result.Unidade, Quantidade: result.Quantidade}
	appropriations, err := state.Stock.GetStockAppropriationsWithDescriptionsForItem(ctx, result.ObraID, item)
	if err != nil {
		return models.Insumo{}, err
	}
	item.Apropriacoes = append([]models.Apropriacao(nil), appropriations...)
	if resultIndex < len(state.Consulta.Resultados) {
		state.Consulta.Resultados[resultIndex] = item
	}
	if resultIndex < len(state.Consulta.ResultadosNormalizados) {
		state.Consulta.ResultadosNormalizados[resultIndex].Apropriacoes = append([]models.Apropriacao(nil), appropriations...)
	}
	state.Consulta.DetalheAberto = &item
	return item, nil
}

func ClearConsulta(state *AppState) {
	state.Consulta = NewConsultaTabState()
}

func ClearConsultaResults(state *AppState) {
	state.Consulta.Resultados = nil
	state.Consulta.ResultadosNormalizados = nil
	state.Consulta.DetalheAberto = nil
	state.Consulta.SolicitacaoItensCount = 0
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

func ConsultaResultRow(value any) string {
	switch item := value.(type) {
	case models.ConsultaResultado:
		return fmt.Sprintf("%d - %s | %d | %s | %s | %s | %s", item.ObraID, item.ObraNome, item.InsumoID, item.InsumoNome, item.Detalhe, item.Marca, models.FormatQuantidade(item.Quantidade, item.Unidade))
	case models.Insumo:
		return fmt.Sprintf("%d | %s | %s | %s | %s", item.ID, item.Nome, item.Detalhe, item.Marca, models.FormatQuantidade(item.Quantidade, item.Unidade))
	default:
		return ""
	}
}

func ConsultaResultsForDisplay(state *AppState) []models.ConsultaResultado {
	if len(state.Consulta.ResultadosNormalizados) > 0 {
		return append([]models.ConsultaResultado(nil), state.Consulta.ResultadosNormalizados...)
	}
	obraID, _ := ObraIDFromLabel(state.Config.Obras, state.Consulta.ObraSelecionada)
	obraNome := ObraNameByID(state.Config.Obras, obraID)
	obra := models.Obra{ID: obraID, Nome: obraNome}
	stockByWork := map[int][]models.Insumo{obraID: state.Consulta.Resultados}
	return models.BuildConsultaPorInsumoResults([]models.Obra{obra}, stockByWork)
}

func BuildConsultaViewModel(state ConsultaTabState) ConsultaViewModel {
	tipo := effectiveConsultaTipo(state.TipoConsulta)
	return ConsultaViewModel{
		Tipo:                  tipo,
		ShowInsumoInput:       tipo == models.ConsultaPorInsumo,
		ShowSolicitacaoInputs: tipo == models.ConsultaPorSolicitacaoCompra,
		ShowObrasSelector:     true,
	}
}

func applyConsultaViewModel(viewModel ConsultaViewModel, insumoSection fyne.CanvasObject, solicitacaoSection fyne.CanvasObject) {
	if viewModel.ShowInsumoInput {
		insumoSection.Show()
	} else {
		insumoSection.Hide()
	}
	if viewModel.ShowSolicitacaoInputs {
		solicitacaoSection.Show()
	} else {
		solicitacaoSection.Hide()
	}
	insumoSection.Refresh()
	solicitacaoSection.Refresh()
}

func effectiveConsultaTipo(tipo models.ConsultaTipo) models.ConsultaTipo {
	if tipo == "" {
		return models.ConsultaPorInsumo
	}
	return tipo
}

func consultaTipoLabel(tipo models.ConsultaTipo) string {
	if tipo == models.ConsultaPorSolicitacaoCompra {
		return "Por solicitacao de compra"
	}
	return "Por insumo"
}

func consultaTipoFromLabel(label string) models.ConsultaTipo {
	if label == "Por solicitacao de compra" {
		return models.ConsultaPorSolicitacaoCompra
	}
	return models.ConsultaPorInsumo
}

func isConsultaObraSelecionada(obras []models.Obra, id int) bool {
	for _, obra := range obras {
		if obra.ID == id {
			return true
		}
	}
	return false
}

func setConsultaObraSelecionada(state *AppState, obra models.Obra, selected bool) {
	state.Consulta.ConsultarTodasObras = false
	if selected {
		if !isConsultaObraSelecionada(state.Consulta.ObrasSelecionadas, obra.ID) {
			state.Consulta.ObrasSelecionadas = append(state.Consulta.ObrasSelecionadas, obra)
		}
		return
	}
	filtered := state.Consulta.ObrasSelecionadas[:0]
	for _, current := range state.Consulta.ObrasSelecionadas {
		if current.ID != obra.ID {
			filtered = append(filtered, current)
		}
	}
	state.Consulta.ObrasSelecionadas = filtered
}
