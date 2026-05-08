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
		if AppropriationLabel(appropriation) == item.ApropriacaoOrigemSelecionada || appropriation.Codigo == item.ApropriacaoOrigemSelecionada {
			return appropriation
		}
	}

	return models.Apropriacao{}
}

func (item TransferenciaItemState) selectedDestinationAppropriation() models.Apropriacao {
	for _, appropriation := range item.ApropriacoesDestino {
		if AppropriationLabel(appropriation) == item.ApropriacaoDestinoSelecionada || appropriation.Codigo == item.ApropriacaoDestinoSelecionada {
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

	status := widget.NewLabel("")
	addButton := widget.NewButton("Adicionar Insumo", func() {
		state.ActiveTab = TabTransferencia
		status.SetText(StatusLoading)
		state.Runner.Run(func() error {
			return AddTransferInsumoFromInput(context.Background(), state, state.Transferencia.InsumoIDInput)
		}, func(err error) {
			if err != nil {
				var multipleErr *MultipleInsumosError
				if errors.As(err, &multipleErr) {
					ShowInsumoSelectionModal(state.Window, multipleErr.Options, func(item models.Insumo) {
						state.Runner.Run(func() error {
							return AddSelectedTransferInsumo(context.Background(), state, item)
						}, func(err error) {
							if err != nil {
								status.SetText(err.Error())
								return
							}
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
		originAppropriationSelect := widget.NewSelect(AppropriationLabels(item.ApropriacoesOrigem), func(value string) {
			if err := SetTransferItemOriginAppropriation(state, rowIndex, value); err == nil {
				state.RefreshTab(TabTransferencia)
			}
		})
		originAppropriationSelect.PlaceHolder = "Apropriacao origem"
		originAppropriationSelect.SetSelected(item.ApropriacaoOrigemSelecionada)
		destinationAppropriationSelect := widget.NewSelect(AppropriationLabels(item.ApropriacoesDestino), func(value string) {
			if err := SetTransferItemDestinationAppropriation(state, rowIndex, value); err == nil {
				state.RefreshTab(TabTransferencia)
			}
		})
		destinationAppropriationSelect.PlaceHolder = "Apropriacao destino"
		destinationAppropriationSelect.SetSelected(item.ApropriacaoDestinoSelecionada)
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
			widget.NewLabel(TransferItemLabel(item.Insumo)),
			originAppropriationSelect,
			destinationAppropriationSelect,
			widget.NewLabel(models.FormatQuantidade(item.QuantidadeDisponivel, item.Insumo.Unidade)),
			withMinTypingInputWidth(quantityEntry),
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

	return container.NewVBox(
		widget.NewLabel("Transferencia de insumos"),
		container.NewHBox(withMinTypingInputWidth(origemSelect), withMinTypingInputWidth(destinoSelect)),
		container.NewHBox(withMinTypingInputWidth(solicitanteEntry), widget.NewLabel("Documento: TR"), withMinTypingInputWidth(movimentoEntry)),
		withMinTypingInputWidth(observacaoEntry),
		container.NewHBox(withMinTypingInputWidth(insumoEntry), addButton, sendButton, clearButton),
		status,
		container.NewVBox(rows...),
	)
}

func AddTransferInsumoFromInput(ctx context.Context, state *AppState, input string) error {
	id, err := parseObraID(input)
	if err != nil {
		return err
	}

	return AddTransferInsumo(ctx, state, id)
}

func AddTransferInsumo(ctx context.Context, state *AppState, supplyID int) error {
	if state.Stock == nil {
		return errors.New("servico de estoque nao configurado")
	}
	originID, err := TransferOriginID(state)
	if err != nil {
		return err
	}
	if _, err := TransferDestinationID(state); err != nil {
		return err
	}

	items, err := state.Stock.GetStockItemsByIDs(ctx, originID, []int{supplyID})
	if err != nil {
		return err
	}
	if len(items) == 0 {
		return errors.New("insumo nao encontrado na obra de origem")
	}
	if len(items) > 1 {
		return &MultipleInsumosError{Options: items}
	}

	return AddSelectedTransferInsumo(ctx, state, items[0])
}

func AddSelectedTransferInsumo(ctx context.Context, state *AppState, item models.Insumo) error {
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

	appropriations, err := state.Stock.GetBuildingAppropriations(ctx, originID, item.ID)
	if err != nil {
		return err
	}
	destinationAppropriations, err := state.Stock.GetBuildingAppropriations(ctx, destinationID, item.ID)
	if err != nil {
		return err
	}
	originAppropriations := AppropriationsWithStock(appropriations)
	item.Apropriacoes = append([]models.Apropriacao(nil), originAppropriations...)
	state.Transferencia.Itens = append(state.Transferencia.Itens, NewTransferenciaItemState(item, originAppropriations, destinationAppropriations))
	state.Transferencia.InsumoIDInput = ""
	return nil
}

func NewTransferenciaItemState(item models.Insumo, originAppropriations, destinationAppropriations []models.Apropriacao) TransferenciaItemState {
	itemState := TransferenciaItemState{
		Insumo:              item,
		ApropriacoesOrigem:  append([]models.Apropriacao(nil), originAppropriations...),
		ApropriacoesDestino: append([]models.Apropriacao(nil), destinationAppropriations...),
	}
	if len(itemState.ApropriacoesOrigem) == 1 {
		appropriation := itemState.ApropriacoesOrigem[0]
		itemState.ApropriacaoOrigemSelecionada = AppropriationLabel(appropriation)
		itemState.QuantidadeDisponivel = appropriation.Quantidade
	}
	if len(itemState.ApropriacoesDestino) == 1 {
		itemState.ApropriacaoDestinoSelecionada = AppropriationLabel(itemState.ApropriacoesDestino[0])
	}

	return itemState
}

func AppropriationsWithStock(appropriations []models.Apropriacao) []models.Apropriacao {
	filtered := make([]models.Apropriacao, 0, len(appropriations))
	for _, appropriation := range appropriations {
		if appropriation.Quantidade > 0 {
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
		if AppropriationLabel(appropriation) == code || appropriation.Codigo == code {
			state.Transferencia.Itens[index].ApropriacaoOrigemSelecionada = AppropriationLabel(appropriation)
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
		if AppropriationLabel(appropriation) == code || appropriation.Codigo == code {
			state.Transferencia.Itens[index].ApropriacaoDestinoSelecionada = AppropriationLabel(appropriation)
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
		originAppropriationCode, originAppropriationDescription := SplitAppropriationLabel(item.ApropriacaoOrigemSelecionada)
		destinationAppropriationCode, destinationAppropriationDescription := SplitAppropriationLabel(item.ApropriacaoDestinoSelecionada)
		items = append(items, models.ItemTransferido{
			ID:                               item.Insumo.ID,
			Nome:                             item.Insumo.Nome,
			Detalhe:                          item.Insumo.Detalhe,
			DetalheID:                        item.Insumo.DetalheID,
			Marca:                            item.Insumo.Marca,
			MarcaID:                          item.Insumo.MarcaID,
			Unidade:                          item.Insumo.Unidade,
			PrecoUnitario:                    item.Insumo.PrecoMedio,
			Apropriacao:                      originAppropriationCode,
			ApropriacaoDescricao:             originAppropriationDescription,
			ApropriacaoOrigemBuildingUnitID:  state.Transferencia.Itens[index].selectedOriginAppropriation().BuildingUnitID,
			ApropriacaoOrigemSheetItemID:     state.Transferencia.Itens[index].selectedOriginAppropriation().SheetItemID,
			ApropriacaoDestino:               destinationAppropriationCode,
			ApropriacaoDestinoDescricao:      destinationAppropriationDescription,
			ApropriacaoDestinoBuildingUnitID: state.Transferencia.Itens[index].selectedDestinationAppropriation().BuildingUnitID,
			ApropriacaoDestinoSheetItemID:    state.Transferencia.Itens[index].selectedDestinationAppropriation().SheetItemID,
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
		labels = append(labels, AppropriationLabel(appropriation))
	}

	return labels
}

func AppropriationLabel(appropriation models.Apropriacao) string {
	description := strings.TrimSpace(appropriation.Descricao)
	if description == "" {
		description = strings.TrimSpace(appropriation.Referencia)
	}
	if description == "" {
		return appropriation.Codigo
	}

	return appropriation.Codigo + " - " + description
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
