package models

import (
	"encoding/json"
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
	Bloqueado      bool    `json:"bloqueado,omitempty"`
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
	TransferKind        TransferKind      `json:"transfer_kind,omitempty"`
	LinkedLoanID        string            `json:"linked_loan_id,omitempty"`
	LoanStatus          LoanStatus        `json:"loan_status,omitempty"`
	Insumos             []ItemTransferido `json:"insumos"`
}

type ItemTransferido struct {
	ID                                  int      `json:"id"`
	Nome                                string   `json:"nome"`
	Detalhe                             string   `json:"detalhe"`
	DetalheID                           int      `json:"detalhe_id,omitempty"`
	Marca                               string   `json:"marca"`
	MarcaID                             int      `json:"marca_id,omitempty"`
	Unidade                             string   `json:"unidade,omitempty"`
	PrecoUnitario                       float64  `json:"preco_unitario,omitempty"`
	Apropriacao                         string   `json:"apropriacao"`
	ApropriacaoDescricao                string   `json:"apropriacao_descricao,omitempty"`
	ApropriacaoOrigemBuildingUnitID     int      `json:"apropriacao_origem_building_unit_id,omitempty"`
	ApropriacaoOrigemSheetItemID        int      `json:"apropriacao_origem_sheet_item_id,omitempty"`
	ApropriacaoDestino                  string   `json:"apropriacao_destino,omitempty"`
	ApropriacaoDestinoDescricao         string   `json:"apropriacao_destino_descricao,omitempty"`
	ApropriacaoDestinoBuildingUnitID    int      `json:"apropriacao_destino_building_unit_id,omitempty"`
	ApropriacaoDestinoSheetItemID       int      `json:"apropriacao_destino_sheet_item_id,omitempty"`
	ApropriacaoOrigemObrigatoria        bool     `json:"apropriacao_origem_obrigatoria,omitempty"`
	ApropriacaoDestinoObrigatoria       bool     `json:"apropriacao_destino_obrigatoria,omitempty"`
	Quantidade                          float64  `json:"quantidade"`
	QuantidadeDisponivel                float64  `json:"quantidade_disponivel,omitempty"`
	QuantidadeEstoqueOrigemAntes        float64  `json:"quantidade_estoque_origem_antes,omitempty"`
	QuantidadeEstoqueOrigemDepois       float64  `json:"quantidade_estoque_origem_depois,omitempty"`
	QuantidadeEstoqueDestinoAntes       float64  `json:"quantidade_estoque_destino_antes,omitempty"`
	QuantidadeEstoqueDestinoDepois      float64  `json:"quantidade_estoque_destino_depois,omitempty"`
	QuantidadeApropriacaoOrigemAntes    *float64 `json:"quantidade_apropriacao_origem_antes,omitempty"`
	QuantidadeApropriacaoOrigemDepois   *float64 `json:"quantidade_apropriacao_origem_depois,omitempty"`
	QuantidadeApropriacaoDestinoAntes   *float64 `json:"quantidade_apropriacao_destino_antes,omitempty"`
	QuantidadeApropriacaoDestinoDepois  *float64 `json:"quantidade_apropriacao_destino_depois,omitempty"`
	QuantidadeEnviada                   float64  `json:"quantidade_enviada,omitempty"`
	QuantidadeRecebida                  float64  `json:"quantidade_recebida,omitempty"`
	ApropriacaoOrigemCodigo             string   `json:"apropriacao_origem_codigo,omitempty"`
	ApropriacaoOrigemDescricao          string   `json:"apropriacao_origem_descricao,omitempty"`
	ApropriacaoOrigemLabel              string   `json:"apropriacao_origem_label,omitempty"`
	ApropriacaoDestinoCodigo            string   `json:"apropriacao_destino_codigo,omitempty"`
	ApropriacaoDestinoDescricaoSnapshot string   `json:"apropriacao_destino_descricao_snapshot,omitempty"`
	ApropriacaoDestinoLabel             string   `json:"apropriacao_destino_label,omitempty"`
}

type PurchaseRequestItem struct {
	PurchaseRequestID int             `json:"purchase_request_id"`
	BuildingID        int             `json:"building_id"`
	ResourceID        int             `json:"resource_id"`
	ResourceName      string          `json:"resource_name"`
	Detail            string          `json:"detail"`
	DetailID          int             `json:"detail_id,omitempty"`
	Brand             string          `json:"brand"`
	BrandID           int             `json:"brand_id,omitempty"`
	Unit              string          `json:"unit"`
	Quantity          float64         `json:"quantity"`
	OriginalJSON      json.RawMessage `json:"original_json,omitempty"`
}

type ConsultaTipo string

const (
	ConsultaPorInsumo            ConsultaTipo = "por_insumo"
	ConsultaPorSolicitacaoCompra ConsultaTipo = "por_solicitacao_compra"
)

type ConsultaResultado struct {
	ObraID       int           `json:"obra_id"`
	ObraNome     string        `json:"obra_nome"`
	InsumoID     int           `json:"insumo_id"`
	InsumoNome   string        `json:"insumo_nome"`
	Detalhe      string        `json:"detalhe"`
	DetalheID    int           `json:"detalhe_id,omitempty"`
	Marca        string        `json:"marca"`
	MarcaID      int           `json:"marca_id,omitempty"`
	Unidade      string        `json:"unidade"`
	Quantidade   float64       `json:"quantidade"`
	Apropriacoes []Apropriacao `json:"apropriacoes,omitempty"`
}

func (o Obra) Label() string {
	return fmtInt(o.ID) + " - " + o.Nome
}

func fmtInt(v int) string {
	return strconv.Itoa(v)
}
