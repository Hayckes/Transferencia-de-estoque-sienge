package ui

import (
	"errors"

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
