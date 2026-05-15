package ui

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"sienge-transfer/api"
	"sienge-transfer/models"
)

var (
	ErrObraOrigemObrigatoria                = errors.New("selecione a obra de origem")
	ErrObraDestinoObrigatoria               = errors.New("selecione a obra de destino")
	ErrObrasTransferenciaIguais             = errors.New("obra de origem deve ser diferente da obra de destino")
	ErrMultiplosInsumosEncontrados          = errors.New("foram encontrados multiplos insumos com este ID; selecione detalhe e marca")
	ErrTransferQuantityGreaterThanAvailable = errors.New("A quantidade informada e maior que o saldo disponivel na origem.")
)

type RecalculateTrigger string

const (
	RecalculateByButton            RecalculateTrigger = "button"
	RecalculateByQuantityFocusLost RecalculateTrigger = "quantity_focus_lost"
)

type MultipleInsumosError struct {
	Options []models.Insumo
}

func (e *MultipleInsumosError) Error() string {
	return ErrMultiplosInsumosEncontrados.Error()
}

type TransferenciaItemState struct {
	Insumo                        models.Insumo
	ApropriacoesOrigem            []models.Apropriacao
	ApropriacoesDestino           []models.Apropriacao
	ApropriacaoOrigemSelecionada  string
	ApropriacaoDestinoSelecionada string
	QuantidadeDisponivel          float64
	QuantidadeTransferir          string
	EstoqueOrigemAntes            float64
	EstoqueDestinoAntes           float64
	StockPresenceFeedback         string
}

func (item TransferenciaItemState) selectedOriginAppropriation() models.Apropriacao {
	for _, appropriation := range item.ApropriacoesOrigem {
		if appropriationMatchesSelection(appropriation, item.ApropriacaoOrigemSelecionada) {
			return appropriation
		}
	}

	return models.Apropriacao{}
}

func (item TransferenciaItemState) selectedDestinationAppropriation() models.Apropriacao {
	for _, appropriation := range item.ApropriacoesDestino {
		if appropriationMatchesSelection(appropriation, item.ApropriacaoDestinoSelecionada) {
			return appropriation
		}
	}

	return models.Apropriacao{}
}

type TransferenciaTabState struct {
	ObraOrigem              string
	ObraDestino             string
	Solicitante             string
	Observacao              string
	CodigoDocumento         string
	CodigoMovimento         string
	InsumoIDInput           string
	TransferKind            models.TransferKind
	SelectedLoanID          string
	AvailableLoansForReturn []models.LoanRecord
	Itens                   []TransferenciaItemState
	IsSubmitting            bool
	FeedbackMessage         string
}

func NewTransferenciaTabState() TransferenciaTabState {
	return TransferenciaTabState{
		CodigoDocumento: "TR",
		CodigoMovimento: "3",
		TransferKind:    models.TransferKindNotApplicable,
	}
}

func BuildTransferenciaTab(state *AppState) fyne.CanvasObject {
	origemSelect := widget.NewSelect(ObraLabels(state.Config.Obras), func(value string) {
		state.Transferencia.ObraOrigem = value
	})
	origemSelect.PlaceHolder = "Obra de origem"
	origemSelect.SetSelected(state.Transferencia.ObraOrigem)

	destinoSelect := widget.NewSelect(ObraLabels(state.Config.Obras), func(value string) {
		state.Transferencia.ObraDestino = value
	})
	destinoSelect.PlaceHolder = "Obra de destino"
	destinoSelect.SetSelected(state.Transferencia.ObraDestino)

	solicitanteEntry := widget.NewEntry()
	solicitanteEntry.SetPlaceHolder("Solicitante")
	solicitanteEntry.SetText(state.Transferencia.Solicitante)
	solicitanteEntry.OnChanged = func(value string) { state.Transferencia.Solicitante = value }

	movimentoEntry := widget.NewEntry()
	movimentoEntry.SetPlaceHolder("Codigo tipo movimento")
	movimentoEntry.SetText(state.Transferencia.CodigoMovimento)
	movimentoEntry.OnChanged = func(value string) { state.Transferencia.CodigoMovimento = value }

	observacaoEntry := widget.NewMultiLineEntry()
	observacaoEntry.SetPlaceHolder("Observacao da transferencia")
	observacaoEntry.SetText(state.Transferencia.Observacao)
	observacaoEntry.OnChanged = func(value string) { state.Transferencia.Observacao = value }

	insumoEntry := widget.NewEntry()
	insumoEntry.SetPlaceHolder("ID do insumo")
	insumoEntry.SetText(state.Transferencia.InsumoIDInput)
	insumoEntry.OnChanged = func(value string) {
		filtered := onlyDigits(value)
		if filtered != value {
			insumoEntry.SetText(filtered)
			return
		}
		state.Transferencia.InsumoIDInput = filtered
	}

	status := NewStatusView(state.Window, state.Transferencia.FeedbackMessage)
	if api.TransferDryRunEnabled() {
		setTransferStatus(state, status, "Modo seguro ativo: TRANSFER_DRY_RUN=true. O envio real ao Sienge esta bloqueado.")
	}
	initializingLoanSelect := true
	loanSelect := widget.NewSelect(LoanReturnOptionLabels(state.Transferencia.AvailableLoansForReturn), func(value string) {
		if initializingLoanSelect {
			return
		}
		loan, ok := LoanByReturnOptionLabel(state.Transferencia.AvailableLoansForReturn, value)
		if !ok {
			state.Transferencia.SelectedLoanID = ""
			return
		}
		PrepareTransferReturnFromLoan(state, loan, loan.PendingItems())
		setTransferStatus(state, status, "Emprestimo selecionado para devolucao. Revise os saldos antes de enviar.")
		state.RefreshTab(TabTransferencia)
	})
	loanSelect.PlaceHolder = "Emprestimo para devolver (opcional)"
	if state.Transferencia.SelectedLoanID != "" {
		if loan, ok := LoanByID(state.Transferencia.AvailableLoansForReturn, state.Transferencia.SelectedLoanID); ok {
			loanSelect.SetSelected(LoanReturnOptionLabel(loan))
		}
	}
	initializingTransferKind := true
	transferKindRadio := widget.NewRadioGroup(TransferKindLabels(), func(value string) {
		if initializingTransferKind {
			return
		}
		state.Transferencia.TransferKind = models.TransferKindFromLabel(value)
		if state.Transferencia.TransferKind == models.TransferKindReturn {
			if err := RefreshReturnLoansForTransfer(state); err != nil {
				setTransferStatus(state, status, "Nao foi possivel carregar emprestimos para devolucao: "+err.Error())
				state.RefreshTab(TabTransferencia)
				return
			}
			if len(state.Transferencia.AvailableLoansForReturn) == 0 {
				setTransferStatus(state, status, "Nao ha emprestimos pendentes ou parcialmente devolvidos para devolucao.")
			}
		} else {
			state.Transferencia.SelectedLoanID = ""
		}
		state.RefreshTab(TabTransferencia)
	})
	transferKindRadio.Horizontal = true
	transferKindRadio.SetSelected(models.TransferKindLabel(effectiveTransferKindState(state)))
	initializingTransferKind = false
	initializingLoanSelect = false
	loanSelectorSection := container.NewVBox(widget.NewLabel("Emprestimo vinculado"), expandingInput(loanSelect))
	if effectiveTransferKindState(state) != models.TransferKindReturn {
		loanSelectorSection.Hide()
	}
	addButton := widget.NewButton("Adicionar Insumo", func() {
		state.ActiveTab = TabTransferencia
		status.SetText(StatusLoading)
		runLoadTransferInsumo(state.Runner, func() (TransferenciaItemState, error) {
			return LoadTransferInsumoFromInput(context.Background(), state, state.Transferencia.InsumoIDInput)
		}, func(itemState TransferenciaItemState, err error) {
			if err != nil {
				var multipleErr *MultipleInsumosError
				if errors.As(err, &multipleErr) {
					ShowInsumoSelectionModal(state.Window, multipleErr.Options, func(item models.Insumo) {
						status.SetText(StatusLoading)
						runLoadTransferInsumo(state.Runner, func() (TransferenciaItemState, error) {
							return LoadSelectedTransferInsumo(context.Background(), state, item)
						}, func(selectedItemState TransferenciaItemState, err error) {
							if err != nil {
								if MaybeShowCredentialReonboarding(state, err, status.SetText) {
									return
								}
								status.SetText(err.Error())
								return
							}
							AddPreparedTransferInsumo(state, selectedItemState)
							insumoEntry.SetText("")
							status.SetText(TransferInsumoAddedFeedback(selectedItemState))
							state.RefreshTab(TabTransferencia)
						})
					})
					status.SetText(err.Error())
					return
				}
				if MaybeShowCredentialReonboarding(state, err, status.SetText) {
					return
				}
				status.SetText(err.Error())
				return
			}
			AddPreparedTransferInsumo(state, itemState)
			insumoEntry.SetText("")
			status.SetText(TransferInsumoAddedFeedback(itemState))
			state.RefreshTab(TabTransferencia)
		})
	})

	sendButtonViewModel := BuildTransferSubmitButtonViewModel()
	var sendButton *widget.Button
	sendButton = NewSuccessButton(sendButtonViewModel.Label, func() {
		if state.Transferencia.IsSubmitting {
			setTransferStatus(state, status, "Transferencia ja esta em envio. Aguarde a conclusao.")
			return
		}
		state.ActiveTab = TabTransferencia
		transfer, err := BuildTransferenciaFromState(state)
		if err != nil {
			setTransferStatus(state, status, err.Error())
			return
		}
		ShowConfirmTransferModal(state.Window, transfer, func() {
			release, err := BeginTransferSubmission(state)
			if err != nil {
				setTransferStatus(state, status, err.Error())
				return
			}
			setTransferStatus(state, status, StatusLoading)
			sendButton.Disable()
			movementID := ""
			state.Runner.Run(func() error {
				var err error
				movementID, err = SendPreparedTransferencia(context.Background(), state, transfer)
				return err
			}, func(err error) {
				release()
				sendButton.Enable()
				if err != nil {
					if MaybeShowCredentialReonboarding(state, err, status.SetText) {
						return
					}
					setTransferStatus(state, status, "Erro ao enviar transferencia: "+err.Error())
					return
				}
				if state.HistoryStore != nil {
					_ = RefreshHistorico(state)
				}
				if state.LoanStore != nil {
					_ = RefreshEmprestimos(state)
				}
				ClearTransferencia(state)
				setTransferStatus(state, status, TransferSuccessFeedback(movementID))
				state.RefreshTab(TabTransferencia)
			})
		})
	})
	if state.Transferencia.IsSubmitting || api.TransferDryRunEnabled() {
		sendButton.Disable()
	}

	recalculateButton := widget.NewButton("Recalcular saldos", func() {
		originID, destinationID, items, err := SnapshotTransferRecalculationInput(state, -1, RecalculateByButton)
		if err != nil {
			setTransferStatus(state, status, transferRecalculateErrorFeedback(err))
			return
		}
		setTransferStatus(state, status, StatusLoading)
		var updatedItems []TransferenciaItemState
		state.Runner.Run(func() error {
			var err error
			updatedItems, err = RecalculateTransferSaldosForItems(context.Background(), state.Stock, originID, destinationID, items)
			return err
		}, func(err error) {
			if err != nil {
				if MaybeShowCredentialReonboarding(state, err, status.SetText) {
					return
				}
				setTransferStatus(state, status, "Nao foi possivel recalcular os saldos: "+err.Error())
				return
			}
			state.Transferencia.Itens = updatedItems
			setTransferStatus(state, status, "Saldos recalculados com sucesso.")
			state.RefreshTab(TabTransferencia)
		})
	})

	rows := make([]fyne.CanvasObject, 0, len(state.Transferencia.Itens)+1)
	rows = append(rows, widget.NewLabel("ID | Nome/Detalhe/Marca | Aprop. Origem | Aprop. Destino | Disponivel | Transferir"))
	for index, item := range state.Transferencia.Itens {
		rowIndex := index
		originAppropriationSelect := widget.NewSelect(AppropriationLabels(item.ApropriacoesOrigem), nil)
		originAppropriationSelect.PlaceHolder = "Apropriacao origem"
		originAppropriationSelect.SetSelected(item.ApropriacaoOrigemSelecionada)
		availableLabel := widget.NewLabel(models.FormatQuantidade(item.QuantidadeDisponivel, item.Insumo.Unidade))
		originAppropriationSelect.OnChanged = func(value string) {
			if value == "Nao se aplica" {
				status.SetText(NoAppropriationsFeedback(true))
				return
			}
			if rowIndex < 0 || rowIndex >= len(state.Transferencia.Itens) || value == state.Transferencia.Itens[rowIndex].ApropriacaoOrigemSelecionada {
				return
			}
			if err := SetTransferItemOriginAppropriation(state, rowIndex, value); err == nil {
				availableLabel.SetText(models.FormatQuantidade(state.Transferencia.Itens[rowIndex].QuantidadeDisponivel, state.Transferencia.Itens[rowIndex].Insumo.Unidade))
			}
		}
		destinationAppropriationSelect := widget.NewSelect(AppropriationLabels(item.ApropriacoesDestino), nil)
		destinationAppropriationSelect.PlaceHolder = "Apropriacao destino"
		destinationAppropriationSelect.SetSelected(item.ApropriacaoDestinoSelecionada)
		destinationAppropriationSelect.OnChanged = func(value string) {
			if value == "Nao se aplica" {
				status.SetText(NoAppropriationsFeedback(false))
				return
			}
			if rowIndex < 0 || rowIndex >= len(state.Transferencia.Itens) || value == state.Transferencia.Itens[rowIndex].ApropriacaoDestinoSelecionada {
				return
			}
			_ = SetTransferItemDestinationAppropriation(state, rowIndex, value)
		}
		var quantityEntry *QuantityEntry
		quantityEntry = NewQuantityEntry(func(value string) {
			if rowIndex < 0 || rowIndex >= len(state.Transferencia.Itens) {
				return
			}
			state.Transferencia.Itens[rowIndex].QuantidadeTransferir = value
			originID, destinationID, items, err := SnapshotTransferRecalculationInput(state, rowIndex, RecalculateByQuantityFocusLost)
			if err != nil {
				setTransferStatus(state, status, transferRecalculateErrorFeedback(err))
				return
			}
			setTransferStatus(state, status, StatusLoading)
			var updatedItems []TransferenciaItemState
			state.Runner.Run(func() error {
				var err error
				updatedItems, err = RecalculateTransferSaldosForItems(context.Background(), state.Stock, originID, destinationID, items)
				return err
			}, func(err error) {
				if err != nil {
					if MaybeShowCredentialReonboarding(state, err, status.SetText) {
						return
					}
					setTransferStatus(state, status, transferRecalculateErrorFeedback(err))
					return
				}
				state.Transferencia.Itens = updatedItems
				if rowIndex >= 0 && rowIndex < len(state.Transferencia.Itens) {
					quantityEntry.SetText(state.Transferencia.Itens[rowIndex].QuantidadeTransferir)
				}
				setTransferStatus(state, status, "Saldos recalculados.")
				state.RefreshTab(TabTransferencia)
			})
		})
		quantityEntry.SetPlaceHolder("Qtd.")
		quantityEntry.SetText(item.QuantidadeTransferir)
		quantityEntry.OnChanged = func(value string) {
			if rowIndex < 0 || rowIndex >= len(state.Transferencia.Itens) {
				return
			}
			state.Transferencia.Itens[rowIndex].QuantidadeTransferir = value
		}
		quantityEntry.OnSubmitted = func(value string) {
			if rowIndex < 0 || rowIndex >= len(state.Transferencia.Itens) {
				return
			}
			formatted, _, err := NormalizeQuantityInput(value)
			if err != nil {
				setTransferStatus(state, status, transferRecalculateErrorFeedback(err))
				return
			}
			quantityEntry.SetText(formatted)
			state.Transferencia.Itens[rowIndex].QuantidadeTransferir = formatted
			setTransferStatus(state, status, "Quantidade atualizada. Confira os saldos recalculados.")
		}
		removeButton := widget.NewButton("Remover", func() {
			_ = RemoveTransferItem(state, rowIndex)
			setTransferStatus(state, status, "Insumo removido.")
			state.RefreshTab(TabTransferencia)
		})
		rows = append(rows, container.NewVBox(
			container.NewHBox(
				withMinObjectWidth(widget.NewLabel(TransferItemLabel(item.Insumo)), 280),
				withMinObjectWidth(originAppropriationSelect, 340),
				withMinObjectWidth(destinationAppropriationSelect, 340),
				withMinObjectWidth(availableLabel, 120),
				withMinObjectWidth(quantityEntry, 120),
				removeButton,
			),
			BuildTransferItemSummaryObject(item),
		))
	}

	clearButton := widget.NewButton("Limpar", func() {
		ClearTransferencia(state)
		origemSelect.ClearSelected()
		destinoSelect.ClearSelected()
		solicitanteEntry.SetText("")
		observacaoEntry.SetText("")
		movimentoEntry.SetText("3")
		insumoEntry.SetText("")
		status.SetText("Transferencia limpa.")
		state.RefreshTab(TabTransferencia)
	})

	workRow := container.NewGridWithColumns(2, origemSelect, destinoSelect)
	typeSection := container.NewVBox(widget.NewLabel("Tipo da transferencia"), transferKindRadio, loanSelectorSection)
	requesterRow := container.NewBorder(nil, nil, nil, container.NewHBox(widget.NewLabel("Documento: TR"), withMinObjectWidth(movimentoEntry, 180)), solicitanteEntry)
	itemInputRow := container.NewBorder(nil, nil, nil, container.NewHBox(addButton, recalculateButton, sendButton, clearButton), insumoEntry)

	return scrollablePage(
		widget.NewLabel("Transferencia de insumos"),
		typeSection,
		workRow,
		requesterRow,
		observacaoEntry,
		itemInputRow,
		status.Object(),
		horizontalScrollbarOnly(container.NewVBox(rows...)),
	)
}

func setTransferStatus(state *AppState, status *StatusView, message string) {
	if state != nil {
		state.Transferencia.FeedbackMessage = message
		state.Status = message
	}
	if status != nil {
		status.SetText(message)
	}
}

func runLoadTransferInsumo(runner AsyncRunner, operation func() (TransferenciaItemState, error), done func(TransferenciaItemState, error)) {
	dispatch := runner.Dispatch
	if dispatch == nil {
		dispatch = func(fn func()) { fn() }
	}

	go func() {
		itemState, err := operation()
		dispatch(func() {
			if done != nil {
				done(itemState, err)
			}
		})
	}()
}

func AddTransferInsumoFromInput(ctx context.Context, state *AppState, input string) error {
	itemState, err := LoadTransferInsumoFromInput(ctx, state, input)
	if err != nil {
		return err
	}
	AddPreparedTransferInsumo(state, itemState)
	return nil
}

func LoadTransferInsumoFromInput(ctx context.Context, state *AppState, input string) (TransferenciaItemState, error) {
	id, err := parseObraID(input)
	if err != nil {
		return TransferenciaItemState{}, err
	}

	return LoadTransferInsumo(ctx, state, id)
}

func AddTransferInsumo(ctx context.Context, state *AppState, supplyID int) error {
	itemState, err := LoadTransferInsumo(ctx, state, supplyID)
	if err != nil {
		return err
	}
	AddPreparedTransferInsumo(state, itemState)
	return nil
}

func LoadTransferInsumo(ctx context.Context, state *AppState, supplyID int) (TransferenciaItemState, error) {
	if state.Stock == nil {
		return TransferenciaItemState{}, errors.New("servico de estoque nao configurado")
	}
	originID, err := TransferOriginID(state)
	if err != nil {
		return TransferenciaItemState{}, err
	}
	if _, err := TransferDestinationID(state); err != nil {
		return TransferenciaItemState{}, err
	}

	items, err := state.Stock.GetStockItemsByIDs(ctx, originID, []int{supplyID})
	if err != nil {
		return TransferenciaItemState{}, err
	}
	if len(items) == 0 {
		return TransferenciaItemState{}, errors.New(BuildStockPresenceFeedback(false, false))
	}
	if len(items) > 1 {
		return TransferenciaItemState{}, &MultipleInsumosError{Options: items}
	}

	return LoadSelectedTransferInsumo(ctx, state, items[0])
}

func AddSelectedTransferInsumo(ctx context.Context, state *AppState, item models.Insumo) error {
	itemState, err := LoadSelectedTransferInsumo(ctx, state, item)
	if err != nil {
		return err
	}
	AddPreparedTransferInsumo(state, itemState)
	return nil
}

func LoadSelectedTransferInsumo(ctx context.Context, state *AppState, item models.Insumo) (TransferenciaItemState, error) {
	if state.Stock == nil {
		return TransferenciaItemState{}, errors.New("servico de estoque nao configurado")
	}
	originID, err := TransferOriginID(state)
	if err != nil {
		return TransferenciaItemState{}, err
	}
	destinationID, err := TransferDestinationID(state)
	if err != nil {
		return TransferenciaItemState{}, err
	}

	appropriations, err := state.Stock.GetStockAppropriationsWithDescriptionsForItem(ctx, originID, item)
	if err != nil {
		return TransferenciaItemState{}, err
	}
	destinationAppropriations, err := state.Stock.GetStockAppropriationsWithDescriptionsForItem(ctx, destinationID, item)
	if err != nil {
		return TransferenciaItemState{}, err
	}
	destinationItems, err := state.Stock.GetStockItemsByIDs(ctx, destinationID, []int{item.ID})
	if err != nil {
		return TransferenciaItemState{}, err
	}
	destinationStock := matchingStockQuantity(destinationItems, item)
	originAppropriations := AppropriationsWithStock(appropriations)
	destinationAppropriations = AppropriationsAvailableForTransfer(destinationAppropriations)
	item.Apropriacoes = append([]models.Apropriacao(nil), originAppropriations...)
	itemState := NewTransferenciaItemStateWithDestinationStock(item, originAppropriations, destinationAppropriations, destinationStock)
	itemState.StockPresenceFeedback = BuildStockPresenceFeedback(true, len(destinationItems) > 0)
	return itemState, nil
}

func AddPreparedTransferInsumo(state *AppState, itemState TransferenciaItemState) {
	state.Transferencia.Itens = append(state.Transferencia.Itens, itemState)
	state.Transferencia.InsumoIDInput = ""
}

func NewTransferenciaItemState(item models.Insumo, originAppropriations, destinationAppropriations []models.Apropriacao) TransferenciaItemState {
	return NewTransferenciaItemStateWithDestinationStock(item, originAppropriations, destinationAppropriations, 0)
}

func NewTransferenciaItemStateWithDestinationStock(item models.Insumo, originAppropriations, destinationAppropriations []models.Apropriacao, destinationStock float64) TransferenciaItemState {
	itemState := TransferenciaItemState{
		Insumo:              item,
		ApropriacoesOrigem:  append([]models.Apropriacao(nil), originAppropriations...),
		ApropriacoesDestino: append([]models.Apropriacao(nil), destinationAppropriations...),
		EstoqueOrigemAntes:  item.Quantidade,
		EstoqueDestinoAntes: destinationStock,
	}
	if len(itemState.ApropriacoesOrigem) == 1 {
		appropriation := itemState.ApropriacoesOrigem[0]
		itemState.ApropriacaoOrigemSelecionada = AppropriationOptionLabel(appropriation)
		itemState.QuantidadeDisponivel = appropriation.Quantidade
	}
	if len(itemState.ApropriacoesDestino) == 1 {
		itemState.ApropriacaoDestinoSelecionada = AppropriationOptionLabel(itemState.ApropriacoesDestino[0])
	}

	return itemState
}

func matchingStockQuantity(items []models.Insumo, selected models.Insumo) float64 {
	for _, item := range items {
		if item.ID != selected.ID {
			continue
		}
		if selected.DetalheID > 0 && item.DetalheID > 0 && selected.DetalheID != item.DetalheID {
			continue
		}
		if selected.MarcaID > 0 && item.MarcaID > 0 && selected.MarcaID != item.MarcaID {
			continue
		}
		return item.Quantidade
	}
	return 0
}

func AppropriationsWithStock(appropriations []models.Apropriacao) []models.Apropriacao {
	return filterAppropriations(appropriations, true)
}

func AppropriationsAvailableForTransfer(appropriations []models.Apropriacao) []models.Apropriacao {
	return filterAppropriations(appropriations, false)
}

func filterAppropriations(appropriations []models.Apropriacao, requireStock bool) []models.Apropriacao {
	filtered := make([]models.Apropriacao, 0, len(appropriations))
	for _, appropriation := range appropriations {
		if !appropriation.Bloqueado && (!requireStock || appropriation.Quantidade > 0) {
			filtered = append(filtered, appropriation)
		}
	}

	return filtered
}

func SetTransferItemAppropriation(state *AppState, index int, code string) error {
	return SetTransferItemOriginAppropriation(state, index, code)
}

func SetTransferItemOriginAppropriation(state *AppState, index int, code string) error {
	return setTransferItemOriginAppropriation(state.Transferencia.Itens, index, code)
}

func SetTransferItemDestinationAppropriation(state *AppState, index int, code string) error {
	return setTransferItemDestinationAppropriation(state.Transferencia.Itens, index, code)
}

func setTransferItemOriginAppropriation(items []TransferenciaItemState, index int, code string) error {
	if index < 0 || index >= len(items) {
		return errors.New("insumo da transferencia nao encontrado")
	}
	code = strings.TrimSpace(code)
	for _, appropriation := range items[index].ApropriacoesOrigem {
		if appropriationMatchesSelection(appropriation, code) {
			items[index].ApropriacaoOrigemSelecionada = AppropriationOptionLabel(appropriation)
			items[index].QuantidadeDisponivel = appropriation.Quantidade
			return nil
		}
	}

	return errors.New("apropriacao de origem selecionada nao encontrada")
}

func setTransferItemDestinationAppropriation(items []TransferenciaItemState, index int, code string) error {
	if index < 0 || index >= len(items) {
		return errors.New("insumo da transferencia nao encontrado")
	}
	code = strings.TrimSpace(code)
	for _, appropriation := range items[index].ApropriacoesDestino {
		if appropriationMatchesSelection(appropriation, code) {
			items[index].ApropriacaoDestinoSelecionada = AppropriationOptionLabel(appropriation)
			return nil
		}
	}

	return errors.New("apropriacao de destino selecionada nao encontrada")
}

func RemoveTransferItem(state *AppState, index int) error {
	if index < 0 || index >= len(state.Transferencia.Itens) {
		return errors.New("insumo da transferencia nao encontrado")
	}

	state.Transferencia.Itens = append(state.Transferencia.Itens[:index], state.Transferencia.Itens[index+1:]...)
	return nil
}

func cloneTransferItems(items []TransferenciaItemState) []TransferenciaItemState {
	cloned := make([]TransferenciaItemState, len(items))
	for index, item := range items {
		cloned[index] = item
		cloned[index].ApropriacoesOrigem = append([]models.Apropriacao(nil), item.ApropriacoesOrigem...)
		cloned[index].ApropriacoesDestino = append([]models.Apropriacao(nil), item.ApropriacoesDestino...)
	}
	return cloned
}

func BuildTransferenciaFromState(state *AppState) (models.Transferencia, error) {
	originID, err := TransferOriginID(state)
	if err != nil {
		return models.Transferencia{}, err
	}
	destinationID, err := TransferDestinationID(state)
	if err != nil {
		return models.Transferencia{}, err
	}
	if originID == destinationID {
		return models.Transferencia{}, ErrObrasTransferenciaIguais
	}
	movementCode, err := strconv.Atoi(strings.TrimSpace(state.Transferencia.CodigoMovimento))
	if err != nil || movementCode <= 0 {
		return models.Transferencia{}, errors.New("codigo do tipo de movimento deve ser numerico positivo")
	}

	items := make([]models.ItemTransferido, 0, len(state.Transferencia.Itens))
	for index, item := range state.Transferencia.Itens {
		quantity, err := ParseQuantidadeTransferir(item.QuantidadeTransferir)
		if err != nil {
			return models.Transferencia{}, fmt.Errorf("insumo %d: %w", index+1, err)
		}
		if item.QuantidadeDisponivel > 0 && quantity > item.QuantidadeDisponivel {
			return models.Transferencia{}, fmt.Errorf("insumo %d: quantidade a transferir maior que a disponivel", index+1)
		}
		originAppropriation := item.selectedOriginAppropriation()
		destinationAppropriation := item.selectedDestinationAppropriation()
		if originAppropriation.Bloqueado {
			return models.Transferencia{}, fmt.Errorf("insumo %d: apropriacao de origem bloqueada no Sienge", index+1)
		}
		if destinationAppropriation.Bloqueado {
			return models.Transferencia{}, fmt.Errorf("insumo %d: apropriacao de destino bloqueada no Sienge", index+1)
		}
		items = append(items, models.ItemTransferido{
			ID:                               item.Insumo.ID,
			Nome:                             item.Insumo.Nome,
			Detalhe:                          item.Insumo.Detalhe,
			DetalheID:                        item.Insumo.DetalheID,
			Marca:                            item.Insumo.Marca,
			MarcaID:                          item.Insumo.MarcaID,
			Unidade:                          item.Insumo.Unidade,
			PrecoUnitario:                    item.Insumo.PrecoMedio,
			Apropriacao:                      originAppropriation.Codigo,
			ApropriacaoDescricao:             AppropriationDescription(originAppropriation),
			ApropriacaoOrigemBuildingUnitID:  originAppropriation.BuildingUnitID,
			ApropriacaoOrigemSheetItemID:     originAppropriation.SheetItemID,
			ApropriacaoDestino:               destinationAppropriation.Codigo,
			ApropriacaoDestinoDescricao:      AppropriationDescription(destinationAppropriation),
			ApropriacaoDestinoBuildingUnitID: destinationAppropriation.BuildingUnitID,
			ApropriacaoDestinoSheetItemID:    destinationAppropriation.SheetItemID,
			ApropriacaoOrigemObrigatoria:     len(item.ApropriacoesOrigem) > 0,
			ApropriacaoDestinoObrigatoria:    len(item.ApropriacoesDestino) > 0,
			Quantidade:                       quantity,
			QuantidadeDisponivel:             item.QuantidadeDisponivel,
		})
		if err := applyTransferSnapshot(&items[len(items)-1], item, originAppropriation, destinationAppropriation); err != nil {
			return models.Transferencia{}, fmt.Errorf("insumo %d: %w", index+1, err)
		}
	}

	transfer := models.Transferencia{
		DataHora:            time.Now(),
		Usuario:             state.Config.Usuario.Nome,
		Cargo:               state.Config.Usuario.Cargo,
		Solicitante:         strings.TrimSpace(state.Transferencia.Solicitante),
		Observacao:          strings.TrimSpace(state.Transferencia.Observacao),
		ObraOrigemID:        originID,
		ObraOrigemNome:      ObraNameByID(state.Config.Obras, originID),
		ObraDestinoID:       destinationID,
		ObraDestinoNome:     ObraNameByID(state.Config.Obras, destinationID),
		CodigoTipoDocumento: "TR",
		CodigoTipoMovimento: movementCode,
		TransferKind:        effectiveTransferKindState(state),
		LinkedLoanID:        strings.TrimSpace(state.Transferencia.SelectedLoanID),
		Insumos:             items,
	}
	if validationErrors := ValidateTransferenciaState(transfer); len(validationErrors) > 0 {
		return models.Transferencia{}, &ValidationError{Errors: validationErrors}
	}

	return transfer, nil
}

func ValidateTransferenciaState(transfer models.Transferencia) []string {
	validationErrors := ValidateTransferenciaRequiredFields(transfer)
	for _, errText := range apiValidationErrors(transfer) {
		validationErrors = append(validationErrors, errText)
	}

	return validationErrors
}

func ValidateTransferenciaRequiredFields(transfer models.Transferencia) []string {
	var validationErrors []string
	if strings.TrimSpace(transfer.Solicitante) == "" {
		validationErrors = append(validationErrors, "solicitante obrigatorio")
	}
	if len(transfer.Insumos) == 0 {
		validationErrors = append(validationErrors, "adicione pelo menos um insumo")
	}
	for index, item := range transfer.Insumos {
		prefix := fmt.Sprintf("insumo %d", index+1)
		if item.ApropriacaoOrigemObrigatoria && strings.TrimSpace(item.Apropriacao) == "" {
			validationErrors = append(validationErrors, prefix+": apropriacao de origem obrigatoria")
		}
		if item.ApropriacaoDestinoObrigatoria && strings.TrimSpace(item.ApropriacaoDestino) == "" {
			validationErrors = append(validationErrors, prefix+": apropriacao de destino obrigatoria")
		}
		if item.Quantidade <= 0 {
			validationErrors = append(validationErrors, prefix+": quantidade deve ser maior que zero")
		}
		if item.QuantidadeDisponivel > 0 && item.Quantidade > item.QuantidadeDisponivel {
			validationErrors = append(validationErrors, prefix+": quantidade a transferir maior que a disponivel")
		}
	}

	return validationErrors
}

func applyTransferSnapshot(item *models.ItemTransferido, stateItem TransferenciaItemState, originAppropriation models.Apropriacao, destinationAppropriation models.Apropriacao) error {
	originStock := stateItem.EstoqueOrigemAntes
	if originStock == 0 {
		originStock = stateItem.Insumo.Quantidade
	}
	if originStock == 0 {
		originStock = stateItem.QuantidadeDisponivel
	}
	var originAppropriationStock *float64
	if hasAppropriation(originAppropriation) {
		value := originAppropriation.Quantidade
		originAppropriationStock = &value
	}
	var destinationAppropriationStock *float64
	if hasAppropriation(destinationAppropriation) {
		value := destinationAppropriation.Quantidade
		destinationAppropriationStock = &value
	}

	snapshot, err := models.CalculateTransferStockSnapshot(models.TransferStockSnapshotInput{
		EstoqueOrigemAntes:      originStock,
		EstoqueDestinoAntes:     stateItem.EstoqueDestinoAntes,
		ApropriacaoOrigemAntes:  originAppropriationStock,
		ApropriacaoDestinoAntes: destinationAppropriationStock,
		Quantidade:              item.Quantidade,
	})
	if err != nil {
		return err
	}

	item.QuantidadeEstoqueOrigemAntes = snapshot.EstoqueOrigemAntes
	item.QuantidadeEstoqueOrigemDepois = snapshot.EstoqueOrigemDepois
	item.QuantidadeEstoqueDestinoAntes = snapshot.EstoqueDestinoAntes
	item.QuantidadeEstoqueDestinoDepois = snapshot.EstoqueDestinoDepois
	item.QuantidadeApropriacaoOrigemAntes = snapshot.ApropriacaoOrigemAntes
	item.QuantidadeApropriacaoOrigemDepois = snapshot.ApropriacaoOrigemDepois
	item.QuantidadeApropriacaoDestinoAntes = snapshot.ApropriacaoDestinoAntes
	item.QuantidadeApropriacaoDestinoDepois = snapshot.ApropriacaoDestinoDepois
	item.QuantidadeEnviada = snapshot.QuantidadeEnviada
	item.QuantidadeRecebida = snapshot.QuantidadeRecebida
	item.ApropriacaoOrigemCodigo = originAppropriation.Codigo
	item.ApropriacaoOrigemDescricao = models.AppropriationDescription(originAppropriation)
	item.ApropriacaoOrigemLabel = models.AppropriationLabel(originAppropriation)
	item.ApropriacaoDestinoCodigo = destinationAppropriation.Codigo
	item.ApropriacaoDestinoDescricaoSnapshot = models.AppropriationDescription(destinationAppropriation)
	item.ApropriacaoDestinoLabel = models.AppropriationLabel(destinationAppropriation)
	return nil
}

func hasAppropriation(appropriation models.Apropriacao) bool {
	return strings.TrimSpace(appropriation.Codigo) != "" || appropriation.BuildingUnitID > 0 || appropriation.SheetItemID > 0
}

func SendTransferencia(ctx context.Context, state *AppState) (string, error) {
	release, err := BeginTransferSubmission(state)
	if err != nil {
		return "", err
	}
	defer release()

	transfer, err := BuildTransferenciaFromState(state)
	if err != nil {
		return "", err
	}
	movementID, err := SendPreparedTransferencia(ctx, state, transfer)
	if err != nil {
		return "", err
	}
	if state.HistoryStore != nil {
		_ = RefreshHistorico(state)
	}
	if state.LoanStore != nil {
		_ = RefreshEmprestimos(state)
	}
	ClearTransferencia(state)
	return movementID, nil
}

func SendPreparedTransferencia(ctx context.Context, state *AppState, transfer models.Transferencia) (string, error) {
	if state == nil {
		return "", errors.New("estado da aplicacao nao configurado")
	}
	if state.Transfer == nil {
		return "", errors.New("servico de transferencia nao configurado")
	}
	if state.TransferStore == nil {
		return "", errors.New("armazenamento de transferencias nao configurado")
	}

	if err := ValidateLoanReturnBeforeSend(state, transfer); err != nil {
		return "", err
	}
	if err := RevalidateTransferSnapshotBeforeSend(ctx, state, transfer); err != nil {
		return "", err
	}
	if err := ValidateLoanReturnBeforeSend(state, transfer); err != nil {
		return "", err
	}
	movementID, err := state.Transfer.CreateStockTransfer(ctx, transfer)
	if err != nil {
		return "", err
	}
	transfer.IDMovimento = movementID
	applyLoanSideEffects := PrepareLoanSideEffectsAfterSend(state, &transfer)
	if err := state.TransferStore.AppendHistory(transfer); err != nil {
		return "", errors.New(TransferLocalHistoryErrorFeedback(err))
	}
	if err := state.TransferStore.AppendTransferToExcel(transfer); err != nil {
		return "", errors.New("Transferencia enviada com sucesso no Sienge, mas houve erro ao salvar a planilha local: " + err.Error())
	}
	if err := applyLoanSideEffects(); err != nil {
		return "", errors.New("Transferencia enviada com sucesso no Sienge e salva no historico local, mas houve erro ao atualizar emprestimos: " + err.Error())
	}
	return movementID, nil
}

func BeginTransferSubmission(state *AppState) (func(), error) {
	if state == nil {
		return nil, errors.New("estado da aplicacao nao configurado")
	}
	state.transferSubmitMu.Lock()
	defer state.transferSubmitMu.Unlock()

	if state.Transferencia.IsSubmitting {
		return nil, errors.New("Transferencia ja esta em envio. Aguarde a conclusao.")
	}
	state.Transferencia.IsSubmitting = true

	return func() {
		state.transferSubmitMu.Lock()
		defer state.transferSubmitMu.Unlock()
		state.Transferencia.IsSubmitting = false
	}, nil
}

func RecalculateTransferSaldos(ctx context.Context, state *AppState) error {
	if state.Stock == nil {
		return errors.New("servico de estoque nao configurado")
	}
	originID, err := TransferOriginID(state)
	if err != nil {
		return err
	}
	destinationID, err := TransferDestinationID(state)
	if err != nil {
		return err
	}
	updatedItems, err := RecalculateTransferSaldosForItems(ctx, state.Stock, originID, destinationID, state.Transferencia.Itens)
	if err != nil {
		return err
	}
	state.Transferencia.Itens = updatedItems
	return nil
}

func SnapshotTransferRecalculationInput(state *AppState, itemIndex int, trigger RecalculateTrigger) (int, int, []TransferenciaItemState, error) {
	if state == nil {
		return 0, 0, nil, errors.New("estado da aplicacao nao configurado")
	}
	originID, err := TransferOriginID(state)
	if err != nil {
		return 0, 0, nil, err
	}
	destinationID, err := TransferDestinationID(state)
	if err != nil {
		return 0, 0, nil, err
	}
	items := cloneTransferItems(state.Transferencia.Itens)
	if trigger == RecalculateByQuantityFocusLost {
		if itemIndex < 0 || itemIndex >= len(items) {
			return 0, 0, nil, errors.New("insumo da transferencia nao encontrado")
		}
		formatted, quantity, err := NormalizeQuantityInput(items[itemIndex].QuantidadeTransferir)
		if err != nil {
			return 0, 0, nil, err
		}
		if available := items[itemIndex].QuantidadeDisponivel; available > 0 && quantity > available {
			return 0, 0, nil, ErrTransferQuantityGreaterThanAvailable
		}
		items[itemIndex].QuantidadeTransferir = formatted
	}
	return originID, destinationID, items, nil
}

func RecalculateTransferSaldosForItems(ctx context.Context, stock StockService, originID int, destinationID int, items []TransferenciaItemState) ([]TransferenciaItemState, error) {
	if stock == nil {
		return nil, errors.New("servico de estoque nao configurado")
	}
	updatedItems := cloneTransferItems(items)
	for index := range updatedItems {
		item := &updatedItems[index]
		originItems, err := stock.GetStockItemsByIDs(ctx, originID, []int{item.Insumo.ID})
		if err != nil {
			return nil, err
		}
		if stock := matchingStockQuantity(originItems, item.Insumo); stock > 0 {
			item.EstoqueOrigemAntes = stock
			item.Insumo.Quantidade = stock
		}
		destinationItems, err := stock.GetStockItemsByIDs(ctx, destinationID, []int{item.Insumo.ID})
		if err != nil {
			return nil, err
		}
		item.EstoqueDestinoAntes = matchingStockQuantity(destinationItems, item.Insumo)

		originAppropriations, err := stock.GetStockAppropriationsWithDescriptionsForItem(ctx, originID, item.Insumo)
		if err != nil {
			return nil, err
		}
		destinationAppropriations, err := stock.GetStockAppropriationsWithDescriptionsForItem(ctx, destinationID, item.Insumo)
		if err != nil {
			return nil, err
		}
		item.ApropriacoesOrigem = AppropriationsWithStock(originAppropriations)
		item.ApropriacoesDestino = AppropriationsAvailableForTransfer(destinationAppropriations)
		if item.ApropriacaoOrigemSelecionada != "" {
			_ = setTransferItemOriginAppropriation(updatedItems, index, item.ApropriacaoOrigemSelecionada)
		}
		if item.ApropriacaoDestinoSelecionada != "" {
			_ = setTransferItemDestinationAppropriation(updatedItems, index, item.ApropriacaoDestinoSelecionada)
		}
	}
	return updatedItems, nil
}

func HandleRecalculateTrigger(ctx context.Context, state *AppState, itemIndex int, trigger RecalculateTrigger) error {
	if state == nil {
		return errors.New("estado da aplicacao nao configurado")
	}
	if trigger == RecalculateByQuantityFocusLost {
		if itemIndex < 0 || itemIndex >= len(state.Transferencia.Itens) {
			return errors.New("insumo da transferencia nao encontrado")
		}
		formatted, quantity, err := NormalizeQuantityInput(state.Transferencia.Itens[itemIndex].QuantidadeTransferir)
		if err != nil {
			return err
		}
		if available := state.Transferencia.Itens[itemIndex].QuantidadeDisponivel; available > 0 && quantity > available {
			return ErrTransferQuantityGreaterThanAvailable
		}
		state.Transferencia.Itens[itemIndex].QuantidadeTransferir = formatted
	}

	return RecalculateTransferSaldos(ctx, state)
}

func transferRecalculateErrorFeedback(err error) string {
	if err == nil {
		return ""
	}
	if errors.Is(err, ErrQuantityRequired) || errors.Is(err, ErrQuantityInvalidFormat) || errors.Is(err, ErrQuantityMustBePositive) || errors.Is(err, ErrTransferQuantityGreaterThanAvailable) {
		return err.Error()
	}
	return "Nao foi possivel recalcular os saldos: " + err.Error()
}

func RevalidateTransferBeforeSend(ctx context.Context, state *AppState, transfer models.Transferencia) error {
	if state.Stock == nil || len(state.Transferencia.Itens) != len(transfer.Insumos) {
		return nil
	}
	previousOrigin := make([]float64, len(state.Transferencia.Itens))
	previousDestination := make([]float64, len(state.Transferencia.Itens))
	for index, item := range state.Transferencia.Itens {
		previousOrigin[index] = item.EstoqueOrigemAntes
		previousDestination[index] = item.EstoqueDestinoAntes
	}
	if err := RecalculateTransferSaldos(ctx, state); err != nil {
		return err
	}
	for index, item := range state.Transferencia.Itens {
		if previousOrigin[index] != 0 && previousOrigin[index] != item.EstoqueOrigemAntes {
			return errors.New("o estoque foi alterado desde a ultima consulta. Revise os saldos antes de enviar a transferencia")
		}
		if previousDestination[index] != item.EstoqueDestinoAntes {
			return errors.New("o estoque foi alterado desde a ultima consulta. Revise os saldos antes de enviar a transferencia")
		}
	}
	return nil
}

func RevalidateTransferSnapshotBeforeSend(ctx context.Context, state *AppState, transfer models.Transferencia) error {
	if state == nil || state.Stock == nil {
		return nil
	}
	for _, item := range transfer.Insumos {
		stockItem := models.Insumo{ID: item.ID, DetalheID: item.DetalheID, MarcaID: item.MarcaID}
		originItems, err := state.Stock.GetStockItemsByIDs(ctx, transfer.ObraOrigemID, []int{item.ID})
		if err != nil {
			return err
		}
		originStock := matchingStockQuantity(originItems, stockItem)
		if item.QuantidadeEstoqueOrigemAntes != 0 && originStock > 0 && item.QuantidadeEstoqueOrigemAntes != originStock {
			return errors.New("o estoque foi alterado desde a ultima consulta. Revise os saldos antes de enviar a transferencia")
		}

		destinationItems, err := state.Stock.GetStockItemsByIDs(ctx, transfer.ObraDestinoID, []int{item.ID})
		if err != nil {
			return err
		}
		if destinationStock := matchingStockQuantity(destinationItems, stockItem); item.QuantidadeEstoqueDestinoAntes != destinationStock {
			return errors.New("o estoque foi alterado desde a ultima consulta. Revise os saldos antes de enviar a transferencia")
		}

		if err := validateSnapshotAppropriation(ctx, state.Stock, transfer.ObraOrigemID, stockItem, item.ApropriacaoOrigemCodigo, item.ApropriacaoOrigemBuildingUnitID, item.ApropriacaoOrigemSheetItemID, item.QuantidadeApropriacaoOrigemAntes); err != nil {
			return err
		}
		if err := validateSnapshotAppropriation(ctx, state.Stock, transfer.ObraDestinoID, stockItem, item.ApropriacaoDestino, item.ApropriacaoDestinoBuildingUnitID, item.ApropriacaoDestinoSheetItemID, item.QuantidadeApropriacaoDestinoAntes); err != nil {
			return err
		}
	}
	return nil
}

func validateSnapshotAppropriation(ctx context.Context, stock StockService, workID int, item models.Insumo, code string, buildingUnitID int, sheetItemID int, expected *float64) error {
	if expected == nil {
		return nil
	}
	appropriations, err := stock.GetStockAppropriationsWithDescriptionsForItem(ctx, workID, item)
	if err != nil {
		return err
	}
	current, ok := matchingAppropriationQuantity(appropriations, code, buildingUnitID, sheetItemID)
	if !ok || current != *expected {
		return errors.New("o estoque foi alterado desde a ultima consulta. Revise os saldos antes de enviar a transferencia")
	}
	return nil
}

func matchingAppropriationQuantity(appropriations []models.Apropriacao, code string, buildingUnitID int, sheetItemID int) (float64, bool) {
	for _, appropriation := range appropriations {
		if strings.TrimSpace(appropriation.Codigo) != strings.TrimSpace(code) {
			continue
		}
		if buildingUnitID > 0 && appropriation.BuildingUnitID != buildingUnitID {
			continue
		}
		if sheetItemID > 0 && appropriation.SheetItemID != sheetItemID {
			continue
		}
		return appropriation.Quantidade, true
	}
	return 0, false
}

func TransferOriginID(state *AppState) (int, error) {
	id, ok := ObraIDFromLabel(state.Config.Obras, state.Transferencia.ObraOrigem)
	if !ok {
		return 0, ErrObraOrigemObrigatoria
	}

	return id, nil
}

func TransferDestinationID(state *AppState) (int, error) {
	id, ok := ObraIDFromLabel(state.Config.Obras, state.Transferencia.ObraDestino)
	if !ok {
		return 0, ErrObraDestinoObrigatoria
	}

	return id, nil
}

func ClearTransferencia(state *AppState) {
	state.Transferencia = NewTransferenciaTabState()
}

func ParseQuantidadeTransferir(input string) (float64, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return 0, errors.New("quantidade obrigatoria")
	}
	if decimalPlaces(input) > 4 {
		return 0, errors.New("quantidade deve ter no maximo 4 casas decimais")
	}
	input = strings.ReplaceAll(input, ",", ".")
	quantity, err := strconv.ParseFloat(input, 64)
	if err != nil || quantity <= 0 {
		return 0, errors.New("quantidade deve ser numerica positiva")
	}

	return quantity, nil
}

func decimalPlaces(input string) int {
	separator := strings.LastIndex(input, ".")
	if comma := strings.LastIndex(input, ","); comma > separator {
		separator = comma
	}
	if separator == -1 {
		return 0
	}

	return len(input) - separator - 1
}

func TransferItemLabel(item models.Insumo) string {
	return fmt.Sprintf("%d | %s %s - %s", item.ID, item.Nome, item.Detalhe, item.Marca)
}

func AppropriationLabels(appropriations []models.Apropriacao) []string {
	if len(appropriations) == 0 {
		return []string{"Nao se aplica"}
	}
	labels := make([]string, 0, len(appropriations))
	for _, appropriation := range appropriations {
		labels = append(labels, AppropriationOptionLabel(appropriation))
	}

	return labels
}

func AppropriationOptionLabel(appropriation models.Apropriacao) string {
	label := AppropriationLabel(appropriation)
	details := make([]string, 0, 3)
	if appropriation.BuildingUnitID > 0 {
		details = append(details, "Unidade "+strconv.Itoa(appropriation.BuildingUnitID))
	}
	if appropriation.SheetItemID > 0 {
		details = append(details, "Item orcamento "+strconv.Itoa(appropriation.SheetItemID))
	}
	if appropriation.Bloqueado {
		details = append(details, "BLOQUEADO")
	}
	if len(details) == 0 {
		return label
	}

	return label + " | " + strings.Join(details, " | ")
}

func AppropriationLabel(appropriation models.Apropriacao) string {
	description := AppropriationDescription(appropriation)
	if description == "" {
		return appropriation.Codigo
	}

	return appropriation.Codigo + " - " + description
}

func AppropriationDescription(appropriation models.Apropriacao) string {
	description := strings.TrimSpace(appropriation.Descricao)
	if description == "" {
		description = strings.TrimSpace(appropriation.Referencia)
	}

	return description
}

func appropriationMatchesSelection(appropriation models.Apropriacao, selection string) bool {
	selection = strings.TrimSpace(selection)
	return selection != "" && (AppropriationOptionLabel(appropriation) == selection || AppropriationLabel(appropriation) == selection || appropriation.Codigo == selection)
}

func SplitAppropriationLabel(label string) (string, string) {
	parts := strings.SplitN(strings.TrimSpace(label), " - ", 2)
	if len(parts) == 1 {
		return strings.TrimSpace(parts[0]), ""
	}

	return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
}

func BuildTransferItemSummary(item TransferenciaItemState) string {
	quantity, _ := ParseQuantidadeTransferir(item.QuantidadeTransferir)
	balances := transferItemBalances(item, quantity)
	originAppropriation := item.selectedOriginAppropriation()
	destinationAppropriation := item.selectedDestinationAppropriation()

	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("Estoque origem atual: %s | ", models.FormatQuantidade(balances.OriginCurrentStock, item.Insumo.Unidade)))
	builder.WriteString(fmt.Sprintf("Quantidade enviada: %s | ", models.FormatQuantidade(quantity, item.Insumo.Unidade)))
	builder.WriteString(fmt.Sprintf("Saldo origem apos transferencia: %s\n", models.FormatQuantidade(balances.OriginAfterTransfer, item.Insumo.Unidade)))
	builder.WriteString(fmt.Sprintf("Estoque destino atual: %s | ", models.FormatQuantidade(balances.DestinationCurrentStock, item.Insumo.Unidade)))
	builder.WriteString(fmt.Sprintf("Quantidade recebida: %s | ", models.FormatQuantidade(quantity, item.Insumo.Unidade)))
	builder.WriteString(fmt.Sprintf("Saldo destino apos transferencia: %s\n", models.FormatQuantidade(balances.DestinationAfterTransfer, item.Insumo.Unidade)))
	builder.WriteString("Apropriacao origem: ")
	if hasAppropriation(originAppropriation) {
		builder.WriteString(models.AppropriationLabel(originAppropriation))
		builder.WriteString(fmt.Sprintf(" | Saldo: %s | Apos: %s", models.FormatQuantidade(originAppropriation.Quantidade, item.Insumo.Unidade), models.FormatQuantidade(originAppropriation.Quantidade-quantity, item.Insumo.Unidade)))
	} else {
		builder.WriteString("Nao se aplica")
	}
	builder.WriteString("\nApropriacao destino: ")
	if hasAppropriation(destinationAppropriation) {
		builder.WriteString(models.AppropriationLabel(destinationAppropriation))
		builder.WriteString(fmt.Sprintf(" | Saldo: %s | Apos: %s", models.FormatQuantidade(destinationAppropriation.Quantidade, item.Insumo.Unidade), models.FormatQuantidade(destinationAppropriation.Quantidade+quantity, item.Insumo.Unidade)))
	} else {
		builder.WriteString("Nao se aplica")
	}
	return builder.String()
}

func BuildTransferItemSummaryObject(item TransferenciaItemState) fyne.CanvasObject {
	quantity, _ := ParseQuantidadeTransferir(item.QuantidadeTransferir)
	balances := transferItemBalances(item, quantity)
	originAppropriation := item.selectedOriginAppropriation()
	destinationAppropriation := item.selectedDestinationAppropriation()
	return container.NewVBox(
		NewKeyValueLine("Estoque atual de origem", models.FormatQuantidade(balances.OriginCurrentStock, item.Insumo.Unidade)),
		NewKeyValueLine("Quantidade a transferir", models.FormatQuantidade(quantity, item.Insumo.Unidade)),
		NewKeyValueLine("Saldo de origem apos transferencia", models.FormatQuantidade(balances.OriginAfterTransfer, item.Insumo.Unidade)),
		NewKeyValueLine("Estoque atual de destino", models.FormatQuantidade(balances.DestinationCurrentStock, item.Insumo.Unidade)),
		NewKeyValueLine("Saldo de destino apos transferencia", models.FormatQuantidade(balances.DestinationAfterTransfer, item.Insumo.Unidade)),
		NewKeyValueLine("Apropriacao de origem", appropriationSummaryText(originAppropriation, true)),
		NewKeyValueLine("Apropriacao de destino", appropriationSummaryText(destinationAppropriation, false)),
	)
}

func transferItemBalances(item TransferenciaItemState, quantity float64) models.TransferBalanceOutput {
	var originAppropriationStock *float64
	if origin := item.selectedOriginAppropriation(); hasAppropriation(origin) {
		value := origin.Quantidade
		originAppropriationStock = &value
	}
	var destinationAppropriationStock *float64
	if destination := item.selectedDestinationAppropriation(); hasAppropriation(destination) {
		value := destination.Quantidade
		destinationAppropriationStock = &value
	}
	balances, err := models.CalculateTransferBalances(models.TransferBalanceInput{OriginTotalStock: item.EstoqueOrigemAntes, DestinationTotalStock: item.EstoqueDestinoAntes, OriginAppropriationStock: originAppropriationStock, DestinationAppropriationStock: destinationAppropriationStock, QuantityToTransfer: quantity})
	if err != nil {
		return models.TransferBalanceOutput{OriginCurrentStock: item.EstoqueOrigemAntes, DestinationCurrentStock: item.EstoqueDestinoAntes, OriginAfterTransfer: item.EstoqueOrigemAntes - quantity, DestinationAfterTransfer: item.EstoqueDestinoAntes + quantity}
	}
	return balances
}

func appropriationSummaryText(appropriation models.Apropriacao, origin bool) string {
	if hasAppropriation(appropriation) {
		return models.AppropriationLabel(appropriation)
	}
	if origin {
		return "Nao se aplica. " + NoAppropriationsFeedback(true)
	}
	return "Nao se aplica. " + NoAppropriationsFeedback(false)
}

func NewKeyValueLine(key string, value string) fyne.CanvasObject {
	keyLabel := widget.NewLabel(key + ":")
	keyLabel.TextStyle = fyne.TextStyle{Bold: true}
	return container.NewHBox(keyLabel, widget.NewLabel(value))
}

func BuildStockPresenceFeedback(originFound bool, destinationFound bool) string {
	switch {
	case originFound && destinationFound:
		return "Insumo encontrado na origem e no destino."
	case !originFound && !destinationFound:
		return "O insumo informado nao encontrado na obra de origem nem na obra de destino."
	case !originFound:
		return "O insumo nao encontrado no estoque da obra de origem. Nao e possivel transferir um item sem saldo de origem."
	default:
		return "O insumo foi encontrado na obra de origem, mas nao existe no estoque da obra de destino. Verifique se o Sienge permite transferir este item para o destino ou cadastre o item no estoque de destino."
	}
}

func TransferInsumoAddedFeedback(item TransferenciaItemState) string {
	if strings.TrimSpace(item.StockPresenceFeedback) == "" {
		return "Insumo adicionado a transferencia."
	}
	return "Insumo adicionado a transferencia. " + item.StockPresenceFeedback
}

func ObraNameByID(obras []models.Obra, id int) string {
	for _, obra := range obras {
		if obra.ID == id {
			return obra.Nome
		}
	}

	return ""
}

func apiValidationErrors(transfer models.Transferencia) []string {
	// Reusa as regras do pacote api sem expor dependência na UI pública.
	return nil
}
