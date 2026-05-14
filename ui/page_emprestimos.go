package ui

import (
	"fmt"
	"image/color"
	"sort"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"sienge-transfer/models"
)

type EmprestimosTabState struct {
	Loans        []models.LoanRecord
	Search       string
	ShowPending  bool
	ShowPartial  bool
	ShowReturned bool
	Status       string
}

type LoanTableRow struct {
	ID               string
	DestinationWork  string
	Solicitor        string
	Type             string
	LoanDate         string
	LoanDateValue    time.Time
	ReturnDate       string
	LoanedQuantity   string
	ReturnedQuantity string
	Status           string
	StatusColor      string
	ItemCount        string
	CanReturn        bool
}

type LoanReturnSelectionState struct {
	Items           []models.LoanItem
	SelectedIndexes map[int]bool
	SelectAll       bool
}

func NewEmprestimosTabState() EmprestimosTabState {
	return EmprestimosTabState{ShowPending: true, ShowPartial: true}
}

func BuildEmprestimosTab(state *AppState) fyne.CanvasObject {
	status := NewStatusView(state.Window, state.Emprestimos.Status)
	if err := RefreshEmprestimos(state); err != nil {
		setEmprestimosStatus(state, status, err.Error())
	}
	initializing := true

	search := widget.NewEntry()
	search.SetPlaceHolder("Pesquisar...")
	search.SetText(state.Emprestimos.Search)
	search.OnChanged = func(value string) {
		state.Emprestimos.Search = value
		state.RefreshTab(TabEmprestimos)
	}

	pending := widget.NewCheck(models.LoanStatusLabel(models.LoanStatusPending), func(checked bool) {
		if initializing {
			return
		}
		state.Emprestimos.ShowPending = checked
		state.RefreshTab(TabEmprestimos)
	})
	pending.SetChecked(state.Emprestimos.ShowPending)
	partial := widget.NewCheck(models.LoanStatusLabel(models.LoanStatusPartiallyReturned), func(checked bool) {
		if initializing {
			return
		}
		state.Emprestimos.ShowPartial = checked
		state.RefreshTab(TabEmprestimos)
	})
	partial.SetChecked(state.Emprestimos.ShowPartial)
	returned := widget.NewCheck(models.LoanStatusLabel(models.LoanStatusReturned), func(checked bool) {
		if initializing {
			return
		}
		state.Emprestimos.ShowReturned = checked
		state.RefreshTab(TabEmprestimos)
	})
	returned.SetChecked(state.Emprestimos.ShowReturned)
	initializing = false
	refresh := widget.NewButton("Atualizar", func() {
		if err := RefreshEmprestimos(state); err != nil {
			setEmprestimosStatus(state, status, err.Error())
			return
		}
		setEmprestimosStatus(state, status, "Emprestimos atualizados.")
		state.RefreshTab(TabEmprestimos)
	})

	rows := BuildLoanTableRows(state.Emprestimos.Loans, LoanTableFilter{
		Search:       state.Emprestimos.Search,
		ShowPending:  state.Emprestimos.ShowPending,
		ShowPartial:  state.Emprestimos.ShowPartial,
		ShowReturned: state.Emprestimos.ShowReturned,
	})
	objects := []fyne.CanvasObject{loanTableHeader()}
	for _, row := range rows {
		loan, ok := LoanByID(state.Emprestimos.Loans, row.ID)
		if !ok {
			continue
		}
		detailsButton := widget.NewButton("Detalhes", func() { ShowLoanDetailsModal(state.Window, loan) })
		returnButton := widget.NewButton("Devolver", func() { StartLoanReturn(state, loan) })
		if !row.CanReturn {
			returnButton.Disable()
		}
		objects = append(objects, container.NewHBox(
			withMinObjectWidth(widget.NewLabel(row.DestinationWork), 220),
			withMinObjectWidth(widget.NewLabel(row.Solicitor), 150),
			withMinObjectWidth(widget.NewLabel(row.Type), 90),
			withMinObjectWidth(widget.NewLabel(row.LoanDate), 110),
			withMinObjectWidth(widget.NewLabel(row.ReturnDate), 110),
			withMinObjectWidth(widget.NewLabel(row.LoanedQuantity), 110),
			withMinObjectWidth(widget.NewLabel(row.ReturnedQuantity), 110),
			withMinObjectWidth(statusText(row.Status, row.StatusColor), 170),
			withMinObjectWidth(widget.NewLabel(row.ItemCount), 70),
			container.NewHBox(detailsButton, returnButton),
		))
	}
	if len(rows) == 0 {
		objects = append(objects, widget.NewLabel("Nenhum emprestimo encontrado para os filtros informados."))
	}

	return scrollablePage(
		widget.NewLabel("Emprestimos"),
		responsiveRow(expandingInput(search), refresh),
		container.NewHBox(pending, partial, returned),
		status.Object(),
		container.NewHScroll(container.NewVBox(objects...)),
	)
}

type LoanTableFilter struct {
	Search       string
	ShowPending  bool
	ShowPartial  bool
	ShowReturned bool
}

func BuildLoanTableRows(loans []models.LoanRecord, filter LoanTableFilter) []LoanTableRow {
	rows := make([]LoanTableRow, 0, len(loans))
	for _, loan := range loans {
		loan.Recalculate()
		if !loanStatusAllowed(loan.Status, filter) {
			continue
		}
		row := LoanTableRowFromRecord(loan)
		if !loanRowMatchesSearch(row, filter.Search) {
			continue
		}
		rows = append(rows, row)
	}
	sort.SliceStable(rows, func(i, j int) bool { return rows[i].LoanDateValue.After(rows[j].LoanDateValue) })
	return rows
}

func LoanTableRowFromRecord(loan models.LoanRecord) LoanTableRow {
	loan.Recalculate()
	returnDate := "-"
	if loan.LastReturnDate != nil {
		returnDate = loan.LastReturnDate.Format("02/01/2006")
	}
	return LoanTableRow{
		ID:               loan.ID,
		DestinationWork:  models.Obra{ID: loan.DestinationWorkID, Nome: loan.DestinationWorkName}.Label(),
		Solicitor:        loan.Solicitor,
		Type:             models.TransferKindLabel(models.TransferKindLoan),
		LoanDate:         loan.LoanDate.Format("02/01/2006"),
		LoanDateValue:    loan.LoanDate,
		ReturnDate:       returnDate,
		LoanedQuantity:   FormatLoanQuantity(loan.TotalLoanedQuantity),
		ReturnedQuantity: FormatLoanQuantity(loan.TotalReturnedQuantity),
		Status:           models.LoanStatusLabel(loan.Status),
		StatusColor:      LoanStatusColor(loan.Status),
		ItemCount:        strconv.Itoa(loan.ItemCount),
		CanReturn:        models.CanReturnLoan(loan.Status),
	}
}

func RefreshEmprestimos(state *AppState) error {
	if state == nil || state.LoanStore == nil {
		return nil
	}
	loans, err := state.LoanStore.ListLoans()
	if err != nil {
		return err
	}
	state.Emprestimos.Loans = append([]models.LoanRecord(nil), loans...)
	return nil
}

func StartLoanReturn(state *AppState, loan models.LoanRecord) {
	pendingItems := loan.PendingItems()
	if len(pendingItems) == 0 {
		state.Emprestimos.Status = "Este emprestimo ja foi devolvido."
		state.RefreshTab(TabEmprestimos)
		return
	}
	if len(pendingItems) == 1 || state.Window == nil {
		PrepareTransferReturnFromLoan(state, loan, pendingItems)
		state.RefreshTab(TabTransferencia)
		return
	}
	ShowLoanReturnSelectionModal(state, loan, pendingItems)
}

func PrepareTransferReturnFromLoan(state *AppState, loan models.LoanRecord, items []models.LoanItem) {
	state.Transferencia = NewTransferenciaTabState()
	state.Transferencia.TransferKind = models.TransferKindReturn
	state.Transferencia.SelectedLoanID = loan.ID
	state.Transferencia.ObraOrigem = models.Obra{ID: loan.DestinationWorkID, Nome: loan.DestinationWorkName}.Label()
	state.Transferencia.ObraDestino = models.Obra{ID: loan.OriginWorkID, Nome: loan.OriginWorkName}.Label()
	state.Transferencia.Solicitante = loan.Solicitor
	state.Transferencia.Observacao = "Devolucao do emprestimo " + loan.ID
	state.Transferencia.AvailableLoansForReturn = []models.LoanRecord{loan}
	state.Transferencia.Itens = make([]TransferenciaItemState, 0, len(items))
	for _, item := range items {
		state.Transferencia.Itens = append(state.Transferencia.Itens, TransferItemStateFromLoanItem(item))
	}
}

func TransferItemStateFromLoanItem(item models.LoanItem) TransferenciaItemState {
	originAppropriation := models.Apropriacao{
		Codigo:         item.DestinationAppropriationCode,
		Descricao:      item.DestinationAppropriationDescription,
		BuildingUnitID: ptrValueUI(item.DestinationBuildingUnitID),
		SheetItemID:    ptrValueUI(item.DestinationSheetItemID),
		Quantidade:     item.PendingQuantity(),
	}
	destinationAppropriation := models.Apropriacao{
		Codigo:         item.OriginAppropriationCode,
		Descricao:      item.OriginAppropriationDescription,
		BuildingUnitID: ptrValueUI(item.OriginBuildingUnitID),
		SheetItemID:    ptrValueUI(item.OriginSheetItemID),
	}
	return TransferenciaItemState{
		Insumo: models.Insumo{
			ID:         item.ResourceID,
			Nome:       item.ResourceName,
			Detalhe:    item.DetailName,
			DetalheID:  ptrValueUI(item.DetailID),
			Marca:      item.BrandName,
			MarcaID:    ptrValueUI(item.BrandID),
			Unidade:    item.Unit,
			Quantidade: item.PendingQuantity(),
			PrecoMedio: item.UnitPrice,
		},
		ApropriacoesOrigem:            optionalAppropriation(originAppropriation),
		ApropriacoesDestino:           optionalAppropriation(destinationAppropriation),
		ApropriacaoOrigemSelecionada:  AppropriationOptionLabel(originAppropriation),
		ApropriacaoDestinoSelecionada: AppropriationOptionLabel(destinationAppropriation),
		QuantidadeDisponivel:          item.PendingQuantity(),
		QuantidadeTransferir:          FormatBrazilianDecimal(item.PendingQuantity()),
		EstoqueOrigemAntes:            item.PendingQuantity(),
	}
}

func ShowLoanDetailsModal(window fyne.Window, loan models.LoanRecord) {
	loan.Recalculate()
	rows := []fyne.CanvasObject{
		selectableWrappedLabel("ID do emprestimo: " + loan.ID),
		selectableWrappedLabel("ID movimento original: " + emptyAsDash(loan.OriginalMovementID)),
		selectableWrappedLabel("Data do emprestimo: " + loan.LoanDate.Format("02/01/2006 15:04:05")),
		selectableWrappedLabel(fmt.Sprintf("Origem: %d - %s", loan.OriginWorkID, loan.OriginWorkName)),
		selectableWrappedLabel(fmt.Sprintf("Destino: %d - %s", loan.DestinationWorkID, loan.DestinationWorkName)),
		selectableWrappedLabel("Solicitante: " + loan.Solicitor),
		selectableWrappedLabel("Usuario: " + loan.User),
		selectableWrappedLabel("Cargo: " + loan.Role),
		selectableWrappedLabel("Observacao: " + emptyAsDash(loan.Observation)),
		selectableWrappedLabel("Status: " + models.LoanStatusLabel(loan.Status)),
		selectableWrappedLabel("Qtd emprestada: " + FormatLoanQuantity(loan.TotalLoanedQuantity)),
		selectableWrappedLabel("Qtd devolvida: " + FormatLoanQuantity(loan.TotalReturnedQuantity)),
		widget.NewSeparator(),
		selectableWrappedLabel("Itens:"),
	}
	for _, item := range loan.Items {
		rows = append(rows, selectableWrappedLabel(fmt.Sprintf("%d - %s | %s | %s | Emprestado: %s %s | Devolvido: %s %s | Pendente: %s %s", item.ResourceID, item.ResourceName, item.DetailName, item.BrandName, FormatBrazilianDecimal(item.LoanedQuantity), item.Unit, FormatBrazilianDecimal(item.ReturnedQuantity), item.Unit, FormatBrazilianDecimal(item.PendingQuantity()), item.Unit)))
	}
	rows = append(rows, widget.NewSeparator(), selectableWrappedLabel("Historico de devolucoes:"))
	if len(loan.ReturnMovementIDs) == 0 {
		rows = append(rows, selectableWrappedLabel("Nenhuma devolucao registrada."))
	} else {
		for _, movementID := range loan.ReturnMovementIDs {
			rows = append(rows, selectableWrappedLabel("Movimento: "+movementID))
		}
	}
	if window == nil {
		return
	}
	d := dialog.NewCustom("Detalhes do emprestimo", "Fechar", container.NewVScroll(container.NewVBox(rows...)), window)
	d.Resize(sizeAtLeastWindowRatio(d.MinSize(), window.Canvas().Size(), 0.55, 0.6))
	d.Show()
}

func ShowLoanReturnSelectionModal(state *AppState, loan models.LoanRecord, items []models.LoanItem) {
	selection := NewLoanReturnSelectionState(items)
	checks := make([]*widget.Check, len(selection.Items))
	var selectAll *widget.Check
	var confirm *widget.Button
	selectAll = widget.NewCheck("Selecionar todos", func(checked bool) {
		selection = ToggleLoanReturnSelectAll(selection, checked)
		for index, check := range checks {
			check.SetChecked(selection.SelectedIndexes[index])
		}
		if confirm != nil {
			confirm.Enable()
		}
	})
	rows := []fyne.CanvasObject{selectAll}
	for index, item := range selection.Items {
		itemIndex := index
		checks[index] = widget.NewCheck(LoanReturnItemLabel(item), func(checked bool) {
			selection = ToggleLoanReturnItem(selection, itemIndex, checked)
			selectAll.SetChecked(selection.SelectAll)
			if confirm != nil {
				if len(SelectedLoanReturnItems(selection)) == 0 {
					confirm.Disable()
				} else {
					confirm.Enable()
				}
			}
		})
		rows = append(rows, checks[index])
	}
	var d dialog.Dialog
	confirm = widget.NewButton("Confirmar", func() {
		selected := SelectedLoanReturnItems(selection)
		if len(selected) == 0 {
			return
		}
		PrepareTransferReturnFromLoan(state, loan, selected)
		if d != nil {
			d.Hide()
		}
		state.RefreshTab(TabTransferencia)
	})
	confirm.Disable()
	content := container.NewBorder(nil, container.NewHBox(confirm), nil, nil, container.NewVScroll(container.NewVBox(rows...)))
	d = dialog.NewCustom("Selecionar itens para devolver", "Cancelar", content, state.Window)
	d.Resize(sizeAtLeastWindowRatio(d.MinSize(), state.Window.Canvas().Size(), 0.45, 0.55))
	d.Show()
}

func NewLoanReturnSelectionState(items []models.LoanItem) LoanReturnSelectionState {
	pending := make([]models.LoanItem, 0, len(items))
	for _, item := range items {
		if item.PendingQuantity() > 0 {
			pending = append(pending, item)
		}
	}
	return LoanReturnSelectionState{Items: pending, SelectedIndexes: map[int]bool{}}
}

func ToggleLoanReturnSelectAll(state LoanReturnSelectionState, checked bool) LoanReturnSelectionState {
	state.SelectAll = checked
	state.SelectedIndexes = make(map[int]bool, len(state.Items))
	if checked {
		for index := range state.Items {
			state.SelectedIndexes[index] = true
		}
	}
	return state
}

func ToggleLoanReturnItem(state LoanReturnSelectionState, index int, checked bool) LoanReturnSelectionState {
	if state.SelectedIndexes == nil {
		state.SelectedIndexes = map[int]bool{}
	}
	if checked {
		state.SelectedIndexes[index] = true
	} else {
		delete(state.SelectedIndexes, index)
	}
	state.SelectAll = len(state.Items) > 0 && len(state.SelectedIndexes) == len(state.Items)
	return state
}

func SelectedLoanReturnItems(state LoanReturnSelectionState) []models.LoanItem {
	items := make([]models.LoanItem, 0, len(state.SelectedIndexes))
	for index, item := range state.Items {
		if state.SelectedIndexes[index] {
			items = append(items, item)
		}
	}
	return items
}

func LoanReturnOptionLabels(loans []models.LoanRecord) []string {
	labels := make([]string, 0, len(loans))
	for _, loan := range loans {
		labels = append(labels, LoanReturnOptionLabel(loan))
	}
	return labels
}

func LoanReturnOptionLabel(loan models.LoanRecord) string {
	return fmt.Sprintf("%s | %s | %s", models.Obra{ID: loan.DestinationWorkID, Nome: loan.DestinationWorkName}.Label(), loan.Solicitor, loan.LoanDate.Format("02/01/2006"))
}

func LoanByReturnOptionLabel(loans []models.LoanRecord, label string) (models.LoanRecord, bool) {
	for _, loan := range loans {
		if LoanReturnOptionLabel(loan) == label {
			return loan, true
		}
	}
	return models.LoanRecord{}, false
}

func LoanByID(loans []models.LoanRecord, id string) (models.LoanRecord, bool) {
	for _, loan := range loans {
		if loan.ID == id {
			return loan, true
		}
	}
	return models.LoanRecord{}, false
}

func TransferKindLabels() []string {
	return []string{models.TransferKindLabel(models.TransferKindLoan), models.TransferKindLabel(models.TransferKindReturn), models.TransferKindLabel(models.TransferKindNotApplicable)}
}

func FormatLoanQuantity(value float64) string {
	return FormatBrazilianDecimal(value) + " total"
}

func LoanStatusColor(status models.LoanStatus) string {
	switch status {
	case models.LoanStatusPartiallyReturned:
		return "#2563EB"
	case models.LoanStatusReturned:
		return "#16A34A"
	default:
		return "#DC2626"
	}
}

func LoanReturnItemLabel(item models.LoanItem) string {
	return fmt.Sprintf("%d - %s / %s / %s - Pendente: %s %s", item.ResourceID, item.ResourceName, item.DetailName, item.BrandName, FormatBrazilianDecimal(item.PendingQuantity()), item.Unit)
}

func loadReturnLoans(state *AppState) {
	if state == nil || state.LoanStore == nil {
		return
	}
	loans, err := state.LoanStore.ListLoans()
	if err != nil {
		return
	}
	available := make([]models.LoanRecord, 0, len(loans))
	for _, loan := range loans {
		loan.Recalculate()
		if models.CanReturnLoan(loan.Status) {
			available = append(available, loan)
		}
	}
	state.Transferencia.AvailableLoansForReturn = available
}

func effectiveTransferKindState(state *AppState) models.TransferKind {
	if state == nil {
		return models.TransferKindNotApplicable
	}
	return models.EffectiveTransferKind(state.Transferencia.TransferKind)
}

func ValidateLoanReturnBeforeSend(state *AppState, transfer models.Transferencia) error {
	if state == nil || state.LoanStore == nil || models.EffectiveTransferKind(transfer.TransferKind) != models.TransferKindReturn || strings.TrimSpace(transfer.LinkedLoanID) == "" {
		return nil
	}
	loan, err := state.LoanStore.GetLoanByID(transfer.LinkedLoanID)
	if err != nil {
		return err
	}
	return models.ValidateReturnAgainstLoan(loan, transfer)
}

func ApplyLoanSideEffectsAfterSend(state *AppState, transfer *models.Transferencia) error {
	return PrepareLoanSideEffectsAfterSend(state, transfer)()
}

func PrepareLoanSideEffectsAfterSend(state *AppState, transfer *models.Transferencia) func() error {
	noop := func() error { return nil }
	if state == nil || state.LoanStore == nil || transfer == nil {
		return noop
	}
	switch models.EffectiveTransferKind(transfer.TransferKind) {
	case models.TransferKindLoan:
		loan := models.CreateLoanRecordFromTransfer(*transfer)
		transfer.LinkedLoanID = loan.ID
		transfer.LoanStatus = loan.Status
		return func() error { return state.LoanStore.UpsertLoan(loan) }
	case models.TransferKindReturn:
		if strings.TrimSpace(transfer.LinkedLoanID) == "" {
			return noop
		}
		loan, err := state.LoanStore.GetLoanByID(transfer.LinkedLoanID)
		if err != nil {
			return func() error { return err }
		}
		updated, err := models.ApplyReturnToLoan(loan, *transfer)
		if err != nil {
			return func() error { return err }
		}
		transfer.LoanStatus = updated.Status
		return func() error { return state.LoanStore.UpsertLoan(updated) }
	default:
		return noop
	}
}

func optionalAppropriation(appropriation models.Apropriacao) []models.Apropriacao {
	if strings.TrimSpace(appropriation.Codigo) == "" && appropriation.BuildingUnitID <= 0 && appropriation.SheetItemID <= 0 {
		return nil
	}
	return []models.Apropriacao{appropriation}
}

func ptrValueUI(value *int) int {
	if value == nil {
		return 0
	}
	return *value
}

func setEmprestimosStatus(state *AppState, status *StatusView, message string) {
	if state != nil {
		state.Emprestimos.Status = message
		state.Status = message
	}
	if status != nil {
		status.SetText(message)
	}
}

func loanTableHeader() fyne.CanvasObject {
	return container.NewHBox(
		withMinObjectWidth(widget.NewLabel("Obra destino"), 220),
		withMinObjectWidth(widget.NewLabel("Solicitante"), 150),
		withMinObjectWidth(widget.NewLabel("Tipo"), 90),
		withMinObjectWidth(widget.NewLabel("Data emprestimo"), 110),
		withMinObjectWidth(widget.NewLabel("Data devolucao"), 110),
		withMinObjectWidth(widget.NewLabel("Qtd emprestada"), 110),
		withMinObjectWidth(widget.NewLabel("Qtd devolvida"), 110),
		withMinObjectWidth(widget.NewLabel("Status"), 170),
		withMinObjectWidth(widget.NewLabel("Qtd itens"), 70),
		withMinObjectWidth(widget.NewLabel("Acao"), 150),
	)
}

func statusText(text string, hexColor string) fyne.CanvasObject {
	label := canvas.NewText(text, parseHexColor(hexColor))
	label.TextStyle = fyne.TextStyle{Bold: true}
	return label
}

func parseHexColor(value string) color.Color {
	value = strings.TrimPrefix(strings.TrimSpace(value), "#")
	if len(value) != 6 {
		return color.Black
	}
	r, _ := strconv.ParseUint(value[0:2], 16, 8)
	g, _ := strconv.ParseUint(value[2:4], 16, 8)
	b, _ := strconv.ParseUint(value[4:6], 16, 8)
	return color.NRGBA{R: uint8(r), G: uint8(g), B: uint8(b), A: 255}
}

func loanStatusAllowed(status models.LoanStatus, filter LoanTableFilter) bool {
	switch status {
	case models.LoanStatusReturned:
		return filter.ShowReturned
	case models.LoanStatusPartiallyReturned:
		return filter.ShowPartial
	default:
		return filter.ShowPending
	}
}

func loanRowMatchesSearch(row LoanTableRow, search string) bool {
	search = strings.ToLower(strings.TrimSpace(search))
	if search == "" {
		return true
	}
	visible := strings.ToLower(strings.Join([]string{row.DestinationWork, row.Solicitor, row.Type, row.LoanDate, row.ReturnDate, row.LoanedQuantity, row.ReturnedQuantity, row.Status, row.ItemCount}, " "))
	return strings.Contains(visible, search)
}
