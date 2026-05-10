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
	rows := []fyne.CanvasObject{
		selectableWrappedLabel(fmt.Sprintf("%s %s - %s", item.Nome, item.Detalhe, item.Marca)),
		widget.NewSeparator(),
		selectableWrappedLabel("Codigo | Nome/Referencia | Quantidade"),
	}
	for _, appropriation := range item.Apropriacoes {
		rows = append(rows, selectableWrappedLabel(fmt.Sprintf("%s | %s | %s", appropriation.Codigo, appropriationDisplayName(appropriation), models.FormatQuantidade(appropriation.Quantidade, item.Unidade))))
	}
	content := container.NewVScroll(container.NewVBox(rows...))
	d := dialog.NewCustom("Detalhes do insumo", "Fechar", content, window)
	d.Resize(sizeAtLeastWindowRatio(d.MinSize(), window.Canvas().Size(), insumoSelectionDialogWidthRatio, insumoSelectionDialogHeightRatio))
	d.Show()
}

func ShowInsumoSelectionModal(window fyne.Window, options []models.Insumo, onSelect func(models.Insumo)) {
	if window == nil || len(options) == 0 {
		return
	}
	var d dialog.Dialog
	selecting := false
	buttons := make([]*widget.Button, 0, len(options))
	rows := make([]fyne.CanvasObject, 0, len(options))
	for _, option := range options {
		selected := option
		selectButton := widget.NewButton("Selecionar", func() {
			if selecting {
				return
			}
			selecting = true
			for _, button := range buttons {
				button.Disable()
			}
			if d != nil {
				d.Hide()
			}
			if onSelect != nil {
				onSelect(selected)
			}
		})
		buttons = append(buttons, selectButton)
		rows = append(rows, container.NewHBox(
			selectableWrappedLabel(TransferItemLabel(option)),
			selectButton,
		))
	}
	content := container.NewVScroll(container.NewVBox(rows...))
	d = dialog.NewCustom("Selecione o insumo", "Fechar", content, window)
	d.Resize(sizeAtLeastWindowRatio(d.MinSize(), window.Canvas().Size(), insumoSelectionDialogWidthRatio, insumoSelectionDialogHeightRatio))
	d.Show()
}

func ShowConfirmTransferModal(window fyne.Window, transfer models.Transferencia, onConfirm func()) {
	if window == nil {
		if onConfirm != nil {
			onConfirm()
		}
		return
	}
	summary := TransferSummaryText(transfer)
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
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("Origem: %d - %s\n", transfer.ObraOrigemID, transfer.ObraOrigemNome))
	builder.WriteString(fmt.Sprintf("Destino: %d - %s\n", transfer.ObraDestinoID, transfer.ObraDestinoNome))
	builder.WriteString(fmt.Sprintf("Solicitante: %s\n", transfer.Solicitante))
	if strings.TrimSpace(transfer.Observacao) != "" {
		builder.WriteString(fmt.Sprintf("Observacao: %s\n", transfer.Observacao))
	}
	builder.WriteString("\nInsumos:\n")
	for _, item := range transfer.Insumos {
		builder.WriteString(fmt.Sprintf("- %d %s %s %s | origem %s | destino %s | %s\n", item.ID, item.Nome, item.Detalhe, item.Marca, itemAppropriationText(item.Apropriacao, item.ApropriacaoDescricao), itemAppropriationText(item.ApropriacaoDestino, item.ApropriacaoDestinoDescricao), models.FormatQuantidade(item.Quantidade, item.Unidade)))
	}
	return builder.String()
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
