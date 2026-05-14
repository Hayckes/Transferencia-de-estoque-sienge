package ui

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"sienge-transfer/api"
	"sienge-transfer/models"
)

func ShowInsumoDetailsModal(window fyne.Window, item models.Insumo) {
	if window == nil {
		return
	}
	reconciliation := models.ReconcileStockAndAppropriations(item.Quantidade, item.Apropriacoes)
	rows := []fyne.CanvasObject{
		selectableWrappedLabel(fmt.Sprintf("Insumo: %d - %s", item.ID, item.Nome)),
		selectableWrappedLabel(fmt.Sprintf("Detalhe: %s | DetailID: %d", emptyAsDash(item.Detalhe), item.DetalheID)),
		selectableWrappedLabel(fmt.Sprintf("Marca: %s | TrademarkID: %d", emptyAsDash(item.Marca), item.MarcaID)),
		selectableWrappedLabel(fmt.Sprintf("Unidade: %s", emptyAsDash(item.Unidade))),
		selectableWrappedLabel(fmt.Sprintf("Estoque do item: %s", models.FormatQuantidade(item.Quantidade, item.Unidade))),
		widget.NewSeparator(),
		selectableWrappedLabel("Codigo | Descricao | Unidade construtiva | Item orcamento | Quantidade"),
	}
	if !reconciliation.OK {
		rows = append(rows, selectableWrappedLabel("Atencao: a soma das apropriacoes retornadas nao bate com o estoque do item. Isso pode indicar filtro incorreto, reserva, paginacao incompleta ou divergencia na API."))
		rows = append(rows, selectableWrappedLabel(fmt.Sprintf("Soma das apropriacoes: %s | Diferenca: %.4f", models.FormatQuantidade(reconciliation.AppropriationsQuantity, item.Unidade), reconciliation.Difference)))
		rows = append(rows, widget.NewSeparator())
	}
	for _, appropriation := range item.Apropriacoes {
		rows = append(rows, selectableWrappedLabel(fmt.Sprintf("%s | %s | %d | %d | %s", appropriation.Codigo, appropriationDisplayName(appropriation), appropriation.BuildingUnitID, appropriation.SheetItemID, models.FormatQuantidade(appropriation.Quantidade, item.Unidade))))
	}
	content := container.NewVScroll(container.NewVBox(rows...))
	d := dialog.NewCustom("Detalhes do insumo", "Fechar", content, window)
	d.Resize(sizeAtLeastWindowRatio(d.MinSize(), window.Canvas().Size(), insumoSelectionDialogWidthRatio, insumoSelectionDialogHeightRatio))
	d.Show()
}

func emptyAsDash(value string) string {
	if strings.TrimSpace(value) == "" {
		return "-"
	}
	return strings.TrimSpace(value)
}

func ShowInsumoSelectionModal(window fyne.Window, options []models.Insumo, onSelect func(models.Insumo)) {
	if window == nil || len(options) == 0 {
		return
	}
	var d dialog.Dialog
	selecting := false
	selectedIndex := -1
	var selectButton *widget.Button
	selectButton = widget.NewButton("Selecionar", func() {
		if selecting {
			return
		}
		selected, err := ResolveSelectedInsumo(options, selectedIndex)
		if err != nil {
			return
		}
		selecting = true
		selectButton.Disable()
		if d != nil {
			d.Hide()
		}
		if onSelect != nil {
			onSelect(selected)
		}
	})
	selectButton.Disable()
	tableRows := BuildTransferInsumoSelectionRows(options)
	table := newInsumoSelectionTable(&tableRows, func(index int) {
		selectedIndex = index
		selectButton.Enable()
	})
	scroll := container.NewScroll(table)
	scroll.SetMinSize(fyne.NewSize(860, 360))
	content := container.NewBorder(
		container.NewVBox(
			selectableWrappedLabel("Foram encontrados multiplos insumos para este ID."),
			selectableWrappedLabel("Clique na linha correta e confirme."),
		),
		container.NewHBox(selectButton),
		nil,
		nil,
		scroll,
	)
	d = dialog.NewCustom("Selecione o insumo", "Fechar", content, window)
	d.Resize(sizeAtLeastWindowRatio(fyne.NewSize(900, 500), window.Canvas().Size(), insumoSelectionDialogWidthRatio, insumoSelectionDialogHeightRatio))
	d.Show()
}

func ShowConfirmTransferModal(window fyne.Window, transfer models.Transferencia, onConfirm func()) {
	if window == nil {
		if onConfirm != nil {
			onConfirm()
		}
		return
	}
	summary := BuildTransferConfirmationText(transfer)
	content := container.NewBorder(
		nil,
		widget.NewButton("Copiar resumo", func() { copyTextToClipboard(summary) }),
		nil,
		nil,
		container.NewVScroll(selectableWrappedLabel(summary)),
	)
	d := dialog.NewCustomConfirm("Confirmar Transferencia", "Enviar", "Cancelar", content, func(confirm bool) {
		if confirm && onConfirm != nil {
			onConfirm()
		}
	}, window)
	d.Resize(sizeAtLeastWindowRatio(d.MinSize(), window.Canvas().Size(), 0.55, 0.55))
	d.Show()
}

func ShowConfirmRemoveObra(window fyne.Window, onConfirm func()) {
	if window == nil {
		if onConfirm != nil {
			onConfirm()
		}
		return
	}
	dialog.ShowConfirm("Remover obra", "Confirma a remocao desta obra? O historico local nao sera apagado.", func(confirm bool) {
		if confirm && onConfirm != nil {
			onConfirm()
		}
	}, window)
}

func TransferSummaryText(transfer models.Transferencia) string {
	return BuildTransferConfirmationText(transfer)
}

func BuildTransferConfirmationText(transfer models.Transferencia) string {
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("Origem: %d - %s\n", transfer.ObraOrigemID, transfer.ObraOrigemNome))
	builder.WriteString(fmt.Sprintf("Destino: %d - %s\n", transfer.ObraDestinoID, transfer.ObraDestinoNome))
	builder.WriteString(fmt.Sprintf("Solicitante: %s\n", transfer.Solicitante))
	if strings.TrimSpace(transfer.Observacao) != "" {
		builder.WriteString(fmt.Sprintf("Observacao: %s\n", transfer.Observacao))
	}
	builder.WriteString("\nInsumos:\n")
	for _, item := range transfer.Insumos {
		builder.WriteString(fmt.Sprintf("- %d %s %s %s\n", item.ID, item.Nome, item.Detalhe, item.Marca))
		builder.WriteString(fmt.Sprintf("  Origem antes: %s | Quantidade enviada: %s | Origem depois: %s\n", models.FormatQuantidade(item.QuantidadeEstoqueOrigemAntes, item.Unidade), models.FormatQuantidade(quantityOrFallbackUI(item.QuantidadeEnviada, item.Quantidade), item.Unidade), models.FormatQuantidade(item.QuantidadeEstoqueOrigemDepois, item.Unidade)))
		builder.WriteString(fmt.Sprintf("  Destino antes: %s | Quantidade recebida: %s | Destino depois: %s\n", models.FormatQuantidade(item.QuantidadeEstoqueDestinoAntes, item.Unidade), models.FormatQuantidade(quantityOrFallbackUI(item.QuantidadeRecebida, item.Quantidade), item.Unidade), models.FormatQuantidade(item.QuantidadeEstoqueDestinoDepois, item.Unidade)))
		builder.WriteString(fmt.Sprintf("  Apropriacao origem: %s | Saldo antes: %s | Saldo depois: %s\n", appropriationTextOrNA(item.ApropriacaoOrigemLabel, item.Apropriacao, item.ApropriacaoDescricao), formatOptionalQuantity(item.QuantidadeApropriacaoOrigemAntes, item.Unidade), formatOptionalQuantity(item.QuantidadeApropriacaoOrigemDepois, item.Unidade)))
		builder.WriteString(fmt.Sprintf("  Apropriacao destino: %s | Saldo antes: %s | Saldo depois: %s\n", appropriationTextOrNA(item.ApropriacaoDestinoLabel, item.ApropriacaoDestino, item.ApropriacaoDestinoDescricao), formatOptionalQuantity(item.QuantidadeApropriacaoDestinoAntes, item.Unidade), formatOptionalQuantity(item.QuantidadeApropriacaoDestinoDepois, item.Unidade)))
	}
	return builder.String()
}

func quantityOrFallbackUI(value float64, fallback float64) float64 {
	if value == 0 {
		return fallback
	}
	return value
}

func appropriationTextOrNA(label string, code string, description string) string {
	if strings.TrimSpace(label) != "" {
		return strings.TrimSpace(label)
	}
	if strings.TrimSpace(code) == "" {
		return "Nao se aplica"
	}
	return itemAppropriationText(code, description)
}

func formatOptionalQuantity(value *float64, unit string) string {
	if value == nil {
		return "Nao se aplica"
	}
	return models.FormatQuantidade(*value, unit)
}

func itemAppropriationText(code, description string) string {
	if strings.TrimSpace(description) == "" {
		return strings.TrimSpace(code)
	}
	return strings.TrimSpace(code) + " - " + strings.TrimSpace(description)
}

func appropriationDisplayName(appropriation models.Apropriacao) string {
	if strings.TrimSpace(appropriation.Descricao) != "" {
		return strings.TrimSpace(appropriation.Descricao)
	}
	if strings.TrimSpace(appropriation.Referencia) != "" {
		return strings.TrimSpace(appropriation.Referencia)
	}
	return strings.TrimSpace(appropriation.Codigo)
}

func MaybeShowCredentialReonboarding(state *AppState, err error, status func(string)) bool {
	if !IsAuthError(err) || state == nil || state.Window == nil || state.Store == nil {
		return false
	}
	ShowCredentialsReonboardingModal(state, status)
	return true
}

func IsAuthError(err error) bool {
	var apiErr *api.APIError
	return errors.As(err, &apiErr) && (apiErr.StatusCode == http.StatusUnauthorized || apiErr.StatusCode == http.StatusForbidden)
}

func ShowCredentialsReonboardingModal(state *AppState, status func(string)) {
	empresa := widget.NewEntry()
	empresa.SetText(state.Config.Empresa.Nome)
	subdominio := widget.NewEntry()
	subdominio.SetText(state.Config.Empresa.Subdominio)
	usuario := widget.NewEntry()
	usuario.SetText(state.Config.Empresa.APIUsuario)
	senha := widget.NewPasswordEntry()
	senha.SetPlaceHolder("Nova senha API")
	message := widget.NewLabel("Atualize as credenciais da API Sienge.")

	content := container.NewVBox(
		message,
		widget.NewLabel("Nome da empresa"), expandingInput(empresa),
		widget.NewLabel("Subdominio"), expandingInput(subdominio),
		widget.NewLabel("Usuario API"), expandingInput(usuario),
		widget.NewLabel("Senha API"), expandingInput(senha),
	)

	d := dialog.NewCustomConfirm("Refazer Credenciais", "Salvar", "Cancelar", content, func(confirm bool) {
		if !confirm {
			return
		}
		service := OnboardingService{Store: state.Store, Validator: SiengeCredentialValidator{}}
		newEmpresa, err := service.UpdateCredentials(context.Background(), CredentialsInput{
			EmpresaNome: empresa.Text,
			Subdominio:  subdominio.Text,
			APIUsuario:  usuario.Text,
			APISenha:    senha.Text,
		})
		if err != nil {
			if status != nil {
				status("Credenciais nao atualizadas: " + err.Error())
			}
			return
		}
		state.Config.Empresa = newEmpresa
		if err := ConfigureAPIClient(state); err != nil && status != nil {
			status(err.Error())
			return
		}
		if status != nil {
			status("Credenciais atualizadas com sucesso.")
		}
		state.Refresh()
	}, state.Window)
	d.Show()
}
