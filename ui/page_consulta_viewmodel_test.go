package ui

import (
	"errors"
	"reflect"
	"testing"

	"sienge-transfer/models"
)

func TestBuildConsultaLayoutViewModel_OrderForInsumo(t *testing.T) {
	viewModel := BuildConsultaLayoutViewModel(ConsultaTabState{TipoConsulta: models.ConsultaPorInsumo})
	want := []ConsultaLayoutSection{SectionTitle, SectionTypeActions, SectionInsumoInput, SectionWorksSelection, SectionResults}

	if !reflect.DeepEqual(viewModel.Sections, want) {
		t.Fatalf("Sections = %#v, want %#v", viewModel.Sections, want)
	}
	if !viewModel.ShowInsumoInput || viewModel.ShowPurchaseRequestInputs || !viewModel.ShowWorksSelection || !viewModel.ShowResults {
		t.Fatalf("viewModel flags = %#v, want insumo input only", viewModel)
	}
}

func TestBuildConsultaLayoutViewModel_OrderForPurchaseRequest(t *testing.T) {
	viewModel := BuildConsultaLayoutViewModel(ConsultaTabState{TipoConsulta: models.ConsultaPorSolicitacaoCompra})
	want := []ConsultaLayoutSection{SectionTitle, SectionTypeActions, SectionPurchaseRequestInput, SectionWorksSelection, SectionResults}

	if !reflect.DeepEqual(viewModel.Sections, want) {
		t.Fatalf("Sections = %#v, want %#v", viewModel.Sections, want)
	}
	if viewModel.ShowInsumoInput || !viewModel.ShowPurchaseRequestInputs || !viewModel.ShowWorksSelection || !viewModel.ShowResults {
		t.Fatalf("viewModel flags = %#v, want purchase request inputs only", viewModel)
	}
}

func TestBuildConsultaLayoutViewModel_WorksSelectionComesAfterInputSection(t *testing.T) {
	for _, state := range []ConsultaTabState{
		{TipoConsulta: models.ConsultaPorInsumo},
		{TipoConsulta: models.ConsultaPorSolicitacaoCompra},
	} {
		viewModel := BuildConsultaLayoutViewModel(state)
		worksIndex := consultaLayoutSectionIndex(viewModel.Sections, SectionWorksSelection)
		inputSection := SectionInsumoInput
		if effectiveConsultaTipo(state.TipoConsulta) == models.ConsultaPorSolicitacaoCompra {
			inputSection = SectionPurchaseRequestInput
		}
		inputIndex := consultaLayoutSectionIndex(viewModel.Sections, inputSection)
		if worksIndex <= inputIndex {
			t.Fatalf("Sections = %#v, want works after input section", viewModel.Sections)
		}
	}
}

func TestValidateConsultaForm_InsumoRequiresIDs(t *testing.T) {
	state := ConsultaTabState{TipoConsulta: models.ConsultaPorInsumo, ObrasSelecionadas: []models.Obra{{ID: 1}}}
	err := ValidateConsultaForm(state)
	if !errors.Is(err, models.ErrIDsInsumoObrigatorios) {
		t.Fatalf("ValidateConsultaForm() error = %v, want ErrIDsInsumoObrigatorios", err)
	}
}

func TestValidateConsultaForm_InsumoRequiresWorks(t *testing.T) {
	state := ConsultaTabState{TipoConsulta: models.ConsultaPorInsumo, InsumoIDsInput: "1001"}
	err := ValidateConsultaForm(state)
	if !errors.Is(err, ErrConsultaObrasObrigatorias) {
		t.Fatalf("ValidateConsultaForm() error = %v, want ErrConsultaObrasObrigatorias", err)
	}
}

func TestValidateConsultaForm_PurchaseRequestRequiresRequestID(t *testing.T) {
	state := ConsultaTabState{TipoConsulta: models.ConsultaPorSolicitacaoCompra, SolicitacaoObraID: "121", ObrasSelecionadas: []models.Obra{{ID: 1}}}
	err := ValidateConsultaForm(state)
	if !errors.Is(err, ErrConsultaSolicitacaoCompraIDObrigatoria) {
		t.Fatalf("ValidateConsultaForm() error = %v, want ErrConsultaSolicitacaoCompraIDObrigatoria", err)
	}
}

func TestValidateConsultaForm_PurchaseRequestRequiresBuildingID(t *testing.T) {
	state := ConsultaTabState{TipoConsulta: models.ConsultaPorSolicitacaoCompra, SolicitacaoCompraID: "99", ObrasSelecionadas: []models.Obra{{ID: 1}}}
	err := ValidateConsultaForm(state)
	if !errors.Is(err, ErrConsultaSolicitacaoObraIDObrigatoria) {
		t.Fatalf("ValidateConsultaForm() error = %v, want ErrConsultaSolicitacaoObraIDObrigatoria", err)
	}
}

func TestValidateConsultaForm_PurchaseRequestRequiresWorks(t *testing.T) {
	state := ConsultaTabState{TipoConsulta: models.ConsultaPorSolicitacaoCompra, SolicitacaoCompraID: "99", SolicitacaoObraID: "121"}
	err := ValidateConsultaForm(state)
	if !errors.Is(err, ErrConsultaObrasObrigatorias) {
		t.Fatalf("ValidateConsultaForm() error = %v, want ErrConsultaObrasObrigatorias", err)
	}
}

func consultaLayoutSectionIndex(sections []ConsultaLayoutSection, section ConsultaLayoutSection) int {
	for index, current := range sections {
		if current == section {
			return index
		}
	}
	return -1
}
