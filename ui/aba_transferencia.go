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
	Insumo                 models.Insumo
	ApropriacaoSelecionada string
	QuantidadeDisponivel   float64
	QuantidadeTransferir   string
}

type TransferenciaTabState struct {
	ObraOrigem      string
	ObraDestino     string
	Solicitante     string
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

	insumoEntry := widget.NewEntry()
	insumoEntry.SetPlaceHolder("ID do insumo")
	insumoEntry.SetText(state.Transferencia.InsumoIDInput)
	insumoEntry.OnChanged = func(value string) { state.Transferencia.InsumoIDInput = onlyDigits(value) }

	status := widget.NewLabel("")
	addButton := widget.NewButton("Adicionar Insumo", func() {
		status.SetText(StatusLoading)
		state.Runner.Run(func() error {
			return AddTransferInsumoFromInput(context.Background(), state, state.Transferencia.InsumoIDInput)
		}, func(err error) {
			if err != nil {
				status.SetText(err.Error())
				return
			}
			insumoEntry.SetText("")
			status.SetText("Insumo adicionado.")
		})
	})

	sendButton := widget.NewButton("Enviar Transferencia", func() {
		status.SetText(StatusLoading)
		state.Runner.Run(func() error {
			_, err := SendTransferencia(context.Background(), state)
			return err
		}, func(err error) {
			if err != nil {
				status.SetText(err.Error())
				return
			}
			status.SetText("Transferencia enviada com sucesso.")
		})
	})

	rows := make([]fyne.CanvasObject, 0, len(state.Transferencia.Itens)+1)
	rows = append(rows, widget.NewLabel("ID | Nome/Detalhe/Marca | Apropriacao | Disponivel | Transferir"))
	for index, item := range state.Transferencia.Itens {
		rowIndex := index
		appropriationLabels := AppropriationLabels(item.Insumo.Apropriacoes)
		appropriationSelect := widget.NewSelect(appropriationLabels, func(value string) {
			_ = SetTransferItemAppropriation(state, rowIndex, value)
		})
		appropriationSelect.SetSelected(item.ApropriacaoSelecionada)
		quantityEntry := widget.NewEntry()
		quantityEntry.SetPlaceHolder("Qtd.")
		quantityEntry.SetText(item.QuantidadeTransferir)
		quantityEntry.OnChanged = func(value string) {
			state.Transferencia.Itens[rowIndex].QuantidadeTransferir = value
		}
		removeButton := widget.NewButton("Remover", func() {
			_ = RemoveTransferItem(state, rowIndex)
			status.SetText("Insumo removido.")
		})
		rows = append(rows, container.NewHBox(
			widget.NewLabel(TransferItemLabel(item.Insumo)),
			appropriationSelect,
			widget.NewLabel(models.FormatQuantidade(item.QuantidadeDisponivel, item.Insumo.Unidade)),
			quantityEntry,
			removeButton,
		))
	}

	clearButton := widget.NewButton("Limpar", func() {
		ClearTransferencia(state)
		origemSelect.ClearSelected()
		destinoSelect.ClearSelected()
		solicitanteEntry.SetText("")
		movimentoEntry.SetText("3")
		insumoEntry.SetText("")
		status.SetText("Transferencia limpa.")
	})

	return container.NewVBox(
		widget.NewLabel("Transferencia de insumos"),
		container.NewHBox(origemSelect, destinoSelect),
		container.NewHBox(solicitanteEntry, widget.NewLabel("Documento: TR"), movimentoEntry),
		container.NewHBox(insumoEntry, addButton, sendButton, clearButton),
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

	appropriations, err := state.Stock.GetBuildingAppropriations(ctx, originID, item.ID)
	if err != nil {
		return err
	}
	item.Apropriacoes = append([]models.Apropriacao(nil), appropriations...)
	state.Transferencia.Itens = append(state.Transferencia.Itens, TransferenciaItemState{Insumo: item})
	state.Transferencia.InsumoIDInput = ""
	return nil
}

func SetTransferItemAppropriation(state *AppState, index int, code string) error {
	if index < 0 || index >= len(state.Transferencia.Itens) {
		return errors.New("insumo da transferencia nao encontrado")
	}
	code = strings.TrimSpace(code)
	for _, appropriation := range state.Transferencia.Itens[index].Insumo.Apropriacoes {
		if AppropriationLabel(appropriation) == code || appropriation.Codigo == code {
			state.Transferencia.Itens[index].ApropriacaoSelecionada = AppropriationLabel(appropriation)
			state.Transferencia.Itens[index].QuantidadeDisponivel = appropriation.Quantidade
			return nil
		}
	}

	return errors.New("apropriacao selecionada nao encontrada")
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
		appropriationCode, appropriationDescription := SplitAppropriationLabel(item.ApropriacaoSelecionada)
		items = append(items, models.ItemTransferido{
			ID:                   item.Insumo.ID,
			Nome:                 item.Insumo.Nome,
			Detalhe:              item.Insumo.Detalhe,
			Marca:                item.Insumo.Marca,
			Apropriacao:          appropriationCode,
			ApropriacaoDescricao: appropriationDescription,
			Quantidade:           quantity,
			QuantidadeDisponivel: item.QuantidadeDisponivel,
		})
	}

	transfer := models.Transferencia{
		DataHora:            time.Now(),
		Usuario:             state.Config.Usuario.Nome,
		Cargo:               state.Config.Usuario.Cargo,
		Solicitante:         strings.TrimSpace(state.Transferencia.Solicitante),
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
			validationErrors = append(validationErrors, prefix+": apropriacao obrigatoria")
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
	input = strings.ReplaceAll(strings.TrimSpace(input), ",", ".")
	if input == "" {
		return 0, errors.New("quantidade obrigatoria")
	}
	quantity, err := strconv.ParseFloat(input, 64)
	if err != nil || quantity <= 0 {
		return 0, errors.New("quantidade deve ser numerica positiva")
	}

	return quantity, nil
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
	if strings.TrimSpace(appropriation.Descricao) == "" {
		return appropriation.Codigo
	}

	return appropriation.Codigo + " - " + appropriation.Descricao
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
