package ui

import (
	"errors"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"sienge-transfer/api"
	"sienge-transfer/config"
)

const (
	StatusReady   = "Pronto."
	StatusLoading = "Processando, aguarde..."
)

func StatusMessageForError(err error) string {
	if err == nil {
		return StatusReady
	}

	var apiErr *api.APIError
	if errors.As(err, &apiErr) {
		return apiErr.Message
	}
	if errors.Is(err, config.ErrConfigNotFound) {
		return "Configuracao inicial nao encontrada. Conclua o onboarding para iniciar."
	}
	if errors.Is(err, config.ErrInvalidConfig) {
		return "Configuracao local invalida. Verifique ou refaca a configuracao inicial."
	}

	return "Ocorreu um erro inesperado. Tente novamente."
}

type StatusView struct {
	window        fyne.Window
	label         *widget.Label
	copyButton    *widget.Button
	detailsButton *widget.Button
	content       fyne.CanvasObject
}

func NewStatusView(window fyne.Window, initial string) *StatusView {
	status := &StatusView{window: window}
	status.label = widget.NewLabel(initial)
	status.label.Wrapping = fyne.TextWrapWord
	status.label.Selectable = true
	status.copyButton = widget.NewButton("Copiar", func() {
		copyTextToClipboard(status.Text())
	})
	status.copyButton.Importance = widget.LowImportance
	status.detailsButton = widget.NewButton("Detalhes", func() {
		ShowCopyableMessageModal(status.window, "Detalhes da mensagem", status.Text())
	})
	status.detailsButton.Importance = widget.LowImportance

	actions := container.NewHBox(status.copyButton, status.detailsButton)
	status.content = container.NewBorder(nil, nil, nil, actions, status.label)
	status.SetText(initial)
	return status
}

func (s *StatusView) SetText(message string) {
	if s == nil || s.label == nil {
		return
	}
	s.label.SetText(message)
	s.label.Importance = statusImportance(message)
	s.label.Refresh()
	s.updateActions()
}

func (s *StatusView) Text() string {
	if s == nil || s.label == nil {
		return ""
	}
	return s.label.Text
}

func (s *StatusView) Object() fyne.CanvasObject {
	if s == nil || s.content == nil {
		return widget.NewLabel("")
	}
	return s.content
}

func (s *StatusView) updateActions() {
	if s.copyButton == nil || s.detailsButton == nil {
		return
	}
	if strings.TrimSpace(s.Text()) == "" {
		s.copyButton.Disable()
		s.copyButton.Hide()
		s.detailsButton.Disable()
		s.detailsButton.Hide()
		return
	}
	s.copyButton.Show()
	s.copyButton.Enable()
	if s.window == nil {
		s.detailsButton.Disable()
		s.detailsButton.Hide()
		return
	}
	s.detailsButton.Show()
	s.detailsButton.Enable()
}

func ShowCopyableMessageModal(window fyne.Window, title, message string) {
	if window == nil || strings.TrimSpace(message) == "" {
		copyTextToClipboard(message)
		return
	}
	messageLabel := widget.NewLabel(message)
	messageLabel.Wrapping = fyne.TextWrapWord
	messageLabel.Selectable = true
	copyButton := widget.NewButton("Copiar mensagem", func() {
		copyTextToClipboard(message)
	})
	content := container.NewBorder(nil, copyButton, nil, nil, container.NewVScroll(messageLabel))
	d := dialog.NewCustom(title, "Fechar", content, window)
	d.Resize(sizeAtLeastWindowRatio(d.MinSize(), window.Canvas().Size(), 0.55, 0.5))
	d.Show()
}

func copyTextToClipboard(text string) {
	app := fyne.CurrentApp()
	if app == nil {
		return
	}
	app.Clipboard().SetContent(text)
}

func statusImportance(message string) widget.Importance {
	message = strings.ToLower(strings.TrimSpace(message))
	if message == "" {
		return widget.LowImportance
	}
	if strings.Contains(message, "sucesso") || strings.Contains(message, "concluid") || strings.Contains(message, "atualizad") || message == strings.ToLower(StatusReady) {
		return widget.SuccessImportance
	}
	if strings.Contains(message, "processando") || strings.Contains(message, "aguarde") {
		return widget.WarningImportance
	}
	if strings.Contains(message, "erro") || strings.Contains(message, "falha") || strings.Contains(message, "inval") || strings.Contains(message, "obrigator") || strings.Contains(message, "recusou") || strings.Contains(message, "bloquead") || strings.Contains(message, "nao ") || strings.Contains(message, "não ") {
		return widget.DangerImportance
	}
	return widget.MediumImportance
}
