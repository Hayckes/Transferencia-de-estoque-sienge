package ui

import (
	"errors"
	"strings"

	"sienge-transfer/models"
)

type ConsultaLayoutSection string

const (
	SectionTitle                ConsultaLayoutSection = "title"
	SectionTypeActions          ConsultaLayoutSection = "type_actions"
	SectionInsumoInput          ConsultaLayoutSection = "insumo_input"
	SectionPurchaseRequestInput ConsultaLayoutSection = "purchase_request_input"
	SectionWorksSelection       ConsultaLayoutSection = "works_selection"
	SectionResults              ConsultaLayoutSection = "results"
)

var (
	ErrConsultaObrasObrigatorias              = errors.New("selecione ao menos uma obra ou marque Todas as obras cadastradas")
	ErrConsultaSolicitacaoCompraIDObrigatoria = errors.New("informe o ID da solicitacao de compra")
	ErrConsultaSolicitacaoObraIDObrigatoria   = errors.New("informe o ID da obra/centro de custo")
)

type ConsultaLayoutViewModel struct {
	Sections                  []ConsultaLayoutSection
	ShowInsumoInput           bool
	ShowPurchaseRequestInputs bool
	ShowWorksSelection        bool
	ShowResults               bool
}

func BuildConsultaLayoutViewModel(state ConsultaTabState) ConsultaLayoutViewModel {
	tipo := effectiveConsultaTipo(state.TipoConsulta)
	sections := []ConsultaLayoutSection{SectionTitle, SectionTypeActions}
	showInsumoInput := tipo == models.ConsultaPorInsumo
	showPurchaseRequestInputs := tipo == models.ConsultaPorSolicitacaoCompra
	if showPurchaseRequestInputs {
		sections = append(sections, SectionPurchaseRequestInput)
	} else {
		sections = append(sections, SectionInsumoInput)
	}
	sections = append(sections, SectionWorksSelection, SectionResults)

	return ConsultaLayoutViewModel{
		Sections:                  sections,
		ShowInsumoInput:           showInsumoInput,
		ShowPurchaseRequestInputs: showPurchaseRequestInputs,
		ShowWorksSelection:        true,
		ShowResults:               true,
	}
}

func ValidateConsultaForm(state ConsultaTabState) error {
	tipo := effectiveConsultaTipo(state.TipoConsulta)
	if tipo == models.ConsultaPorSolicitacaoCompra {
		if strings.TrimSpace(state.SolicitacaoCompraID) == "" {
			return ErrConsultaSolicitacaoCompraIDObrigatoria
		}
		if strings.TrimSpace(state.SolicitacaoObraID) == "" {
			return ErrConsultaSolicitacaoObraIDObrigatoria
		}
	} else if strings.TrimSpace(state.InsumoIDsInput) == "" {
		return models.ErrIDsInsumoObrigatorios
	}

	if !state.ConsultarTodasObras && len(state.ObrasSelecionadas) == 0 && strings.TrimSpace(state.ObraSelecionada) == "" {
		return ErrConsultaObrasObrigatorias
	}
	return nil
}
