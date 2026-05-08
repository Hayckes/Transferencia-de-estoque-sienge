package models

import (
	"strconv"
	"time"
)

type Config struct {
	Usuario Usuario `json:"usuario"`
	Empresa Empresa `json:"empresa"`
	Obras   []Obra  `json:"obras"`
}

type Usuario struct {
	Nome  string `json:"nome"`
	Cargo string `json:"cargo"`
}

type Empresa struct {
	Nome         string `json:"nome"`
	Subdominio   string `json:"subdominio"`
	APIUsuario   string `json:"api_usuario"`
	APISenha     string `json:"api_senha"`
	SenhaCifrada bool   `json:"senha_cifrada,omitempty"`
}

type Obra struct {
	ID   int    `json:"id"`
	Nome string `json:"nome"`
}

type Apropriacao struct {
	Codigo         string  `json:"codigo"`
	Descricao      string  `json:"descricao"`
	Referencia     string  `json:"referencia,omitempty"`
	BuildingUnitID int     `json:"building_unit_id,omitempty"`
	SheetItemID    int     `json:"sheet_item_id,omitempty"`
	Quantidade     float64 `json:"quantidade"`
}

type Insumo struct {
	ID           int           `json:"id"`
	Nome         string        `json:"nome"`
	Detalhe      string        `json:"detalhe"`
	DetalheID    int           `json:"detalhe_id,omitempty"`
	Marca        string        `json:"marca"`
	MarcaID      int           `json:"marca_id,omitempty"`
	Unidade      string        `json:"unidade"`
	Quantidade   float64       `json:"quantidade"`
	PrecoMedio   float64       `json:"preco_medio,omitempty"`
	Apropriacoes []Apropriacao `json:"apropriacoes"`
	OriginalJSON string        `json:"original_json,omitempty"`
}

type Transferencia struct {
	IDMovimento         string            `json:"id_movimento"`
	DataHora            time.Time         `json:"data_hora"`
	Usuario             string            `json:"usuario"`
	Cargo               string            `json:"cargo"`
	Solicitante         string            `json:"solicitante"`
	Observacao          string            `json:"observacao,omitempty"`
	ObraOrigemID        int               `json:"obra_origem_id"`
	ObraOrigemNome      string            `json:"obra_origem_nome"`
	ObraDestinoID       int               `json:"obra_destino_id"`
	ObraDestinoNome     string            `json:"obra_destino_nome"`
	CodigoTipoDocumento string            `json:"codigo_tipo_documento"`
	CodigoTipoMovimento int               `json:"codigo_tipo_movimento"`
	Insumos             []ItemTransferido `json:"insumos"`
}

type ItemTransferido struct {
	ID                               int     `json:"id"`
	Nome                             string  `json:"nome"`
	Detalhe                          string  `json:"detalhe"`
	DetalheID                        int     `json:"detalhe_id,omitempty"`
	Marca                            string  `json:"marca"`
	MarcaID                          int     `json:"marca_id,omitempty"`
	Unidade                          string  `json:"unidade,omitempty"`
	PrecoUnitario                    float64 `json:"preco_unitario,omitempty"`
	Apropriacao                      string  `json:"apropriacao"`
	ApropriacaoDescricao             string  `json:"apropriacao_descricao,omitempty"`
	ApropriacaoOrigemBuildingUnitID  int     `json:"apropriacao_origem_building_unit_id,omitempty"`
	ApropriacaoOrigemSheetItemID     int     `json:"apropriacao_origem_sheet_item_id,omitempty"`
	ApropriacaoDestino               string  `json:"apropriacao_destino,omitempty"`
	ApropriacaoDestinoDescricao      string  `json:"apropriacao_destino_descricao,omitempty"`
	ApropriacaoDestinoBuildingUnitID int     `json:"apropriacao_destino_building_unit_id,omitempty"`
	ApropriacaoDestinoSheetItemID    int     `json:"apropriacao_destino_sheet_item_id,omitempty"`
	Quantidade                       float64 `json:"quantidade"`
	QuantidadeDisponivel             float64 `json:"quantidade_disponivel,omitempty"`
}

func (o Obra) Label() string {
	return fmtInt(o.ID) + " - " + o.Nome
}

func fmtInt(v int) string {
	return strconv.Itoa(v)
}
