package models

import (
	"errors"
	"strings"
)

var ErrObrasConsultaObrigatorias = errors.New("selecione pelo menos uma obra para consultar")

func ResolveObrasParaConsulta(todas []Obra, selecionadas []Obra, consultarTodas bool) ([]Obra, error) {
	registered := make(map[int]Obra, len(todas))
	for _, obra := range todas {
		registered[obra.ID] = obra
	}

	if consultarTodas {
		if len(todas) == 0 {
			return nil, ErrObrasConsultaObrigatorias
		}
		return append([]Obra(nil), todas...), nil
	}
	if len(selecionadas) == 0 {
		return nil, ErrObrasConsultaObrigatorias
	}

	resolved := make([]Obra, 0, len(selecionadas))
	seen := make(map[int]bool, len(selecionadas))
	for _, selected := range selecionadas {
		registeredObra, ok := registered[selected.ID]
		if !ok {
			return nil, errors.New("obra selecionada nao esta cadastrada")
		}
		if !seen[selected.ID] {
			resolved = append(resolved, registeredObra)
			seen[selected.ID] = true
		}
	}

	return resolved, nil
}

func BuildConsultaPorInsumoResults(obras []Obra, estoques map[int][]Insumo) []ConsultaResultado {
	results := make([]ConsultaResultado, 0)
	for _, obra := range obras {
		for _, item := range estoques[obra.ID] {
			results = append(results, consultaResultadoFromInsumo(obra, item))
		}
	}
	return results
}

func BuildConsultaPorSolicitacaoResults(obras []Obra, requestItems []PurchaseRequestItem, stockByWork map[int][]Insumo) []ConsultaResultado {
	results := make([]ConsultaResultado, 0)
	for _, obra := range obras {
		for _, stockItem := range stockByWork[obra.ID] {
			if stockItem.Quantidade <= 0 || !stockItemMatchesAnyRequest(stockItem, requestItems) {
				continue
			}
			results = append(results, consultaResultadoFromInsumo(obra, stockItem))
		}
	}
	return results
}

func stockItemMatchesAnyRequest(stockItem Insumo, requestItems []PurchaseRequestItem) bool {
	for _, requestItem := range requestItems {
		if stockItem.ID != requestItem.ResourceID {
			continue
		}
		if requestItem.DetailID > 0 && requestItem.DetailID != stockItem.DetalheID {
			continue
		}
		if requestItem.BrandID != stockItem.MarcaID {
			continue
		}
		return true
	}
	return false
}

func consultaResultadoFromInsumo(obra Obra, item Insumo) ConsultaResultado {
	return ConsultaResultado{
		ObraID:       obra.ID,
		ObraNome:     obra.Nome,
		InsumoID:     item.ID,
		InsumoNome:   item.Nome,
		Detalhe:      item.Detalhe,
		DetalheID:    item.DetalheID,
		Marca:        item.Marca,
		MarcaID:      item.MarcaID,
		Unidade:      item.Unidade,
		Quantidade:   item.Quantidade,
		Apropriacoes: append([]Apropriacao(nil), item.Apropriacoes...),
	}
}

func AppropriationLabel(appropriation Apropriacao) string {
	code := strings.TrimSpace(appropriation.Codigo)
	description := AppropriationDescription(appropriation)
	if code == "" {
		return description
	}
	if description == "" {
		return code
	}
	return code + " - " + description
}

func AppropriationDescription(appropriation Apropriacao) string {
	description := strings.TrimSpace(appropriation.Descricao)
	if description == "" {
		description = strings.TrimSpace(appropriation.Referencia)
	}
	return description
}
