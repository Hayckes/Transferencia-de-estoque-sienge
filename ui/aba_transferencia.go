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

	"sienge-transfer/models"
)

var (
	ErrObraOrigemObrigatoria       = errors.New("selecione a obra de origem")
	ErrObraDestinoObrigatoria      = errors.New("selecione a obra de destino")
	ErrObrasTransferenciaIguais    = errors.New("obra de origem deve ser diferente da obra de destino")
	ErrMultiplosInsumosEncontrados = errors.New("foram encontrados multiplos insumos com este ID; selecione detalhe e marca")
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
	ObraOrigem      string
	ObraDestino     string
	Solicitante     string
	Observacao      string
	CodigoDocumento string
	CodigoMovimento string
	InsumoIDInput   string
	Itens           []TransferenciaItemState
}

func NewTransferenciaTabState() TransferenciaTabState {
	return TransferenciaTabState{
		CodigoDocumento: "TR",
		CodigoMovimento: "3",
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

	status := NewStatusView(state.Window, "")
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
							status.SetText("Insumo adicionado.")
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
			status.SetText("Insumo adicionado.")
			state.RefreshTab(TabTransferencia)
		})
	})

	sendButton := widget.NewButton("Enviar Transferencia", func() {
		state.ActiveTab = TabTransferencia
		transfer, err := BuildTransferenciaFromState(state)
		if err != nil {
			status.SetText(err.Error())
			return
		}
		ShowConfirmTransferModal(state.Window, transfer, func() {
			status.SetText(StatusLoading)
			state.Runner.Run(func() error {
				_, err := SendTransferencia(context.Background(), state)
				return err
			}, func(err error) {
				if err != nil {
					if MaybeShowCredentialReonboarding(state, err, status.SetText) {
						return
					}
					status.SetText(err.Error())
					return
				}
				status.SetText("Transferencia enviada com sucesso.")
				state.RefreshTab(TabTransferencia)
			})
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
			if rowIndex < 0 || rowIndex >= len(state.Transferencia.Itens) || value == state.Transferencia.Itens[rowIndex].ApropriacaoDestinoSelecionada {
				return
			}
			_ = SetTransferItemDestinationAppropriation(state, rowIndex, value)
		}
		quantityEntry := widget.NewEntry()
		quantityEntry.SetPlaceHolder("Qtd.")
		quantityEntry.SetText(item.QuantidadeTransferir)
		quantityEntry.OnChanged = func(value string) {
			state.Transferencia.Itens[rowIndex].QuantidadeTransferir = value
		}
		removeButton := widget.NewButton("Remover", func() {
			_ = RemoveTransferItem(state, rowIndex)
			status.SetText("Insumo removido.")
			state.RefreshTab(TabTransferencia)
		})
		rows = append(rows, container.NewHBox(
			withMinObjectWidth(widget.NewLabel(TransferItemLabel(item.Insumo)), 280),
			withMinObjectWidth(originAppropriationSelect, 340),
			withMinObjectWidth(destinationAppropriationSelect, 340),
			withMinObjectWidth(availableLabel, 120),
			withMinObjectWidth(quantityEntry, 120),
			removeButton,
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
	requesterRow := container.NewBorder(nil, nil, nil, container.NewHBox(widget.NewLabel("Documento: TR"), withMinObjectWidth(movimentoEntry, 180)), solicitanteEntry)
	itemInputRow := container.NewBorder(nil, nil, nil, container.NewHBox(addButton, sendButton, clearButton), insumoEntry)

	return scrollablePage(
		widget.NewLabel("Transferencia de insumos"),
		workRow,
		requesterRow,
		observacaoEntry,
		itemInputRow,
		status.Object(),
		container.NewHScroll(container.NewVBox(rows...)),
	)
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
		return TransferenciaItemState{}, errors.New("insumo nao encontrado na obra de origem")
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

	appropriations, err := state.Stock.GetBuildingAppropriations(ctx, originID, item.ID)
	if err != nil {
		return TransferenciaItemState{}, err
	}
	destinationAppropriations, err := state.Stock.GetBuildingAppropriations(ctx, destinationID, item.ID)
	if err != nil {
		return TransferenciaItemState{}, err
	}
	originAppropriations := AppropriationsWithStock(appropriations)
	destinationAppropriations = AppropriationsAvailableForTransfer(destinationAppropriations)
	item.Apropriacoes = append([]models.Apropriacao(nil), originAppropriations...)
	return NewTransferenciaItemState(item, originAppropriations, destinationAppropriations), nil
}

func AddPreparedTransferInsumo(state *AppState, itemState TransferenciaItemState) {
	state.Transferencia.Itens = append(state.Transferencia.Itens, itemState)
	state.Transferencia.InsumoIDInput = ""
}

func NewTransferenciaItemState(item models.Insumo, originAppropriations, destinationAppropriations []models.Apropriacao) TransferenciaItemState {
	itemState := TransferenciaItemState{
		Insumo:              item,
		ApropriacoesOrigem:  append([]models.Apropriacao(nil), originAppropriations...),
		ApropriacoesDestino: append([]models.Apropriacao(nil), destinationAppropriations...),
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
	if index < 0 || index >= len(state.Transferencia.Itens) {
		return errors.New("insumo da transferencia nao encontrado")
	}
	code = strings.TrimSpace(code)
	for _, appropriation := range state.Transferencia.Itens[index].ApropriacoesOrigem {
		if appropriationMatchesSelection(appropriation, code) {
			state.Transferencia.Itens[index].ApropriacaoOrigemSelecionada = AppropriationOptionLabel(appropriation)
			state.Transferencia.Itens[index].QuantidadeDisponivel = appropriation.Quantidade
			return nil
		}
	}

	return errors.New("apropriacao de origem selecionada nao encontrada")
}

func SetTransferItemDestinationAppropriation(state *AppState, index int, code string) error {
	if index < 0 || index >= len(state.Transferencia.Itens) {
		return errors.New("insumo da transferencia nao encontrado")
	}
	code = strings.TrimSpace(code)
	for _, appropriation := range state.Transferencia.Itens[index].ApropriacoesDestino {
		if appropriationMatchesSelection(appropriation, code) {
			state.Transferencia.Itens[index].ApropriacaoDestinoSelecionada = AppropriationOptionLabel(appropriation)
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
			Quantidade:                       quantity,
			QuantidadeDisponivel:             item.QuantidadeDisponivel,
		})
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
		if strings.TrimSpace(item.Apropriacao) == "" {
			validationErrors = append(validationErrors, prefix+": apropriacao de origem obrigatoria")
		}
		if strings.TrimSpace(item.ApropriacaoDestino) == "" {
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

func SendTransferencia(ctx context.Context, state *AppState) (string, error) {
	if state.Transfer == nil {
		return "", errors.New("servico de transferencia nao configurado")
	}
	if state.TransferStore == nil {
		return "", errors.New("armazenamento de transferencias nao configurado")
	}

	transfer, err := BuildTransferenciaFromState(state)
	if err != nil {
		return "", err
	}
	movementID, err := state.Transfer.CreateStockTransfer(ctx, transfer)
	if err != nil {
		return "", err
	}
	transfer.IDMovimento = movementID
	if err := state.TransferStore.AppendHistory(transfer); err != nil {
		return "", err
	}
	if err := state.TransferStore.AppendTransferToExcel(transfer); err != nil {
		return "", err
	}
	if state.HistoryStore != nil {
		_ = RefreshHistorico(state)
	}

	ClearTransferencia(state)
	return movementID, nil
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
