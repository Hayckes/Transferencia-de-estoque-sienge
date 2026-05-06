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
	Codigo     string  `json:"codigo"`
	Descricao  string  `json:"descricao"`
	Quantidade float64 `json:"quantidade"`
}

type Insumo struct {
	ID           int           `json:"id"`
	Nome         string        `json:"nome"`
	Detalhe      string        `json:"detalhe"`
	Marca        string        `json:"marca"`
	Unidade      string        `json:"unidade"`
	Quantidade   float64       `json:"quantidade"`
	Apropriacoes []Apropriacao `json:"apropriacoes"`
	OriginalJSON string        `json:"original_json,omitempty"`
}

type Transferencia struct {
	IDMovimento         string            `json:"id_movimento"`
	DataHora            time.Time         `json:"data_hora"`
	Usuario             string            `json:"usuario"`
	Cargo               string            `json:"cargo"`
	Solicitante         string            `json:"solicitante"`
	ObraOrigemID        int               `json:"obra_origem_id"`
	ObraOrigemNome      string            `json:"obra_origem_nome"`
	ObraDestinoID       int               `json:"obra_destino_id"`
	ObraDestinoNome     string            `json:"obra_destino_nome"`
	CodigoTipoDocumento string            `json:"codigo_tipo_documento"`
	CodigoTipoMovimento int               `json:"codigo_tipo_movimento"`
	Insumos             []ItemTransferido `json:"insumos"`
}

type ItemTransferido struct {
	ID                   int     `json:"id"`
	Nome                 string  `json:"nome"`
	Detalhe              string  `json:"detalhe"`
	Marca                string  `json:"marca"`
	Apropriacao          string  `json:"apropriacao"`
	ApropriacaoDescricao string  `json:"apropriacao_descricao,omitempty"`
	Quantidade           float64 `json:"quantidade"`
	QuantidadeDisponivel float64 `json:"quantidade_disponivel,omitempty"`
}

func (o Obra) Label() string {
	return fmtInt(o.ID) + " - " + o.Nome
}

func fmtInt(v int) string {
	return strconv.Itoa(v)
}
