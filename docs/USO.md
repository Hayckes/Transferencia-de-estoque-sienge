# Sienge Transfer

Aplicacao desktop local para consulta e transferencia de insumos entre obras no Sienge.

## Stack

- Go 1.24+ no estado atual do `go.mod`.
- Fyne v2.6.3 para interface desktop.
- JSON para configuracao e historico local.
- excelize para `transferencias.xlsx`.

## Arquivos Locais

Os dados ficam no diretorio retornado por `os.UserConfigDir()` dentro da pasta `sienge-transfer`.

Arquivos usados:

- `config.json`: usuario, empresa, credenciais criptografadas e obras.
- `secret.key`: chave AES-256-GCM local de 32 bytes.
- `historico.json`: historico em array JSON simples.
- `transferencias.xlsx`: planilha com uma linha por insumo transferido.

## API Sienge

Base URL:

```text
https://{subdominio}.sienge.com.br/sienge/api/public/v1
```

Endpoints usados:

- `GET /buildings?limit=1`: validar credenciais.
- `GET /stock-inventories/{costCenterId}/items`: consultar estoque.
- `GET /stock-inventories/{costCenterId}/items/{resourceId}/building-appropriation`: consultar apropriações.
- `POST /stock-transfers`: registrar transferencia.

## Testes

Rodar todos os testes:

```bash
go test ./...
```

Rodar testes de um pacote:

```bash
go test ./api
go test ./config
go test ./models
go test ./storage
go test ./ui
```

Teste opcional contra o Sienge real, sem gravar credenciais no projeto:

```bash
SIENGE_SUBDOMAIN="seu-subdominio" SIENGE_USER="seu-usuario" SIENGE_PASSWORD="sua-senha" go test -tags=integration ./api -run TestSiengeCredentialsIntegration
```

No Windows PowerShell:

```powershell
$env:SIENGE_SUBDOMAIN="seu-subdominio"
$env:SIENGE_USER="seu-usuario"
$env:SIENGE_PASSWORD="sua-senha"
go test -tags=integration ./api -run TestSiengeCredentialsIntegration
```

## Build

Build dos pacotes:

```bash
go build ./...
```

Build do executavel local:

```bash
go build -o sienge-transfer.exe .
```

No ambiente atual, `CGO_ENABLED=0` gera o fallback sem interface grafica real. Para executar a interface Fyne completa, habilite CGO e tenha as dependencias graficas da plataforma instaladas.

Windows PowerShell:

```powershell
$env:CGO_ENABLED="1"
go build -o sienge-transfer.exe .
```

No Windows, se aparecer erro parecido com `C compiler "gcc" not found`, instale um compilador C compatível, como MSYS2/MinGW-w64, e garanta que `gcc` esteja no `PATH` antes de executar o build gráfico.

Linux/macOS:

```bash
CGO_ENABLED=1 go build -o sienge-transfer .
```

## Segurança

- A senha da API nao e salva em texto puro.
- `config.json` armazena a senha criptografada.
- `secret.key` e criado localmente com 32 bytes.
- Em Linux/macOS, o sistema tenta usar permissao `0600` nos arquivos sensiveis.
- Se alguem copiar a pasta inteira, tambem leva `secret.key`; esta protecao evita exposicao casual do `config.json`, nao substitui cofre do sistema operacional.

## Estado Atual

O projeto ja possui pacotes testados para modelos, configuracao, API, estoque, transferencia, historico, Excel e estados de UI. A interface Fyne atual e uma base funcional com abas e acoes principais em estrutura simples; refinamentos visuais e fluxos modais completos podem ser feitos em etapas posteriores.
