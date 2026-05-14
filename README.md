# Transferencia de Estoque Sienge

Aplicacao desktop local, escrita em Go com Fyne, para consultar estoque e registrar transferencias de insumos entre obras/centros de custo no Sienge.

O projeto foi pensado como um MVP operacional: configuracao local, consulta de estoque, preparacao de transferencia, envio para a API do Sienge, historico local em JSON e exportacao/atualizacao de planilha Excel.

## Principais Recursos

| Recurso | Descricao |
| --- | --- |
| Onboarding | Cadastro inicial de empresa, subdominio, credenciais da API, usuario e obras. |
| Obras | Busca e cadastro local de centros de custo/obras pelo Sienge. |
| Consulta por insumo | Consulta saldo de um ou mais insumos nas obras selecionadas. |
| Consulta por solicitacao | Busca itens de uma solicitacao de compra e cruza com estoque disponivel. |
| Transferencia | Monta transferencia entre obra origem e destino, com apropriacoes e validacoes. |
| Envio via API | Envia a transferencia para o endpoint de movimentacao de estoque do Sienge. |
| Historico local | Salva transferencias em `historico.json`. |
| Excel local | Mantem `transferencias.xlsx` com uma linha por insumo transferido. |
| Modo seguro | `TRANSFER_DRY_RUN=true` bloqueia POST real de transferencia. |

## Stack

| Camada | Tecnologia |
| --- | --- |
| Linguagem | Go 1.24.0 |
| Interface desktop | Fyne v2.6.3 |
| API externa | Sienge Public API |
| Persistencia local | JSON e XLSX |
| Excel | `github.com/xuri/excelize/v2` |
| Criptografia local | AES-256-GCM |
| Testes | `go test` |

## Estrutura Do Projeto

```text
.
|-- api/                 # Cliente HTTP, parsers e payloads da API Sienge
|-- config/              # Configuracao local, criptografia e store de credenciais
|-- docs/                # Documentacao complementar
|-- models/              # Modelos e regras de dominio
|-- storage/             # Historico JSON e planilha Excel
|-- ui/                  # Interface Fyne, abas, componentes, dialogs e servicos de UI
|-- main.go              # Ponto de entrada
|-- go.mod               # Dependencias Go
|-- package.json         # Dependencia local do opencode usada no workspace
`-- sienge-transfer.exe  # Executavel Windows gerado localmente
```

## Organizacao Interna Da UI

Os arquivos de `ui/` seguem uma convencao simples para facilitar manutencao do MVP.

| Prefixo | Responsabilidade |
| --- | --- |
| `app_*` | Inicializacao, estado global, servicos e abas principais. |
| `page_*` | Telas/abas principais: obras, consulta, transferencia e historico. |
| `components_*` | Componentes reutilizaveis da interface. |
| `shared_*` | Utilitarios compartilhados de layout, status, feedback e async. |
| `modals.go` | Dialogs e modais usados pela interface. |
| `onboarding*` | Fluxo de configuracao inicial e atualizacao de credenciais. |

## Fluxo Geral

```text
main.go
  -> ui.Run()
  -> carrega config local ou abre onboarding
  -> cria AppState com cliente Sienge e stores locais
  -> exibe abas Obras, Consulta, Transferencia e Historico
```

Fluxo de transferencia:

```text
Selecionar origem/destino
  -> adicionar insumo
  -> carregar estoque e apropriacoes
  -> informar quantidade
  -> recalcular saldos
  -> montar payload
  -> enviar POST ao Sienge
  -> salvar historico JSON
  -> atualizar Excel
```

## API Sienge

A URL base e montada a partir do subdominio informado no onboarding.

```text
https://api.sienge.com.br/{subdominio}/public/api/v1
```

Endpoints usados pelo codigo atual:

| Acao | Endpoint |
| --- | --- |
| Validar/consultar centro de custo | `GET /cost-centers/{costCenterId}` |
| Consultar estoque da obra | `GET /stock-inventories/{costCenterId}/items` |
| Consultar apropriacoes | `GET /stock-inventories/{costCenterId}/items/{resourceId}/building-appropriation` |
| Buscar descricao de itens de planilha | `GET /building-cost-estimations/{costCenterId}/sheets/{buildingUnitId}/items` |
| Consultar itens de solicitacao | `GET /purchase-requests/all/items` |
| Enviar transferencia | `POST /stock-movements/transfer` |

A autenticacao usa Basic Auth com usuario e senha da API configurados localmente.

## Arquivos Locais

Os dados locais ficam no diretorio retornado por `os.UserConfigDir()` dentro da pasta `sienge-transfer`.

Em Windows, normalmente fica em:

```text
%AppData%\sienge-transfer
```

Arquivos locais principais:

| Arquivo | Conteudo |
| --- | --- |
| `config.json` | Usuario, empresa, obras e senha da API criptografada. |
| `secret.key` | Chave local AES-256-GCM de 32 bytes. |
| `historico.json` | Historico local das transferencias realizadas. |
| `transferencias.xlsx` | Planilha local com os detalhes de cada insumo transferido. |

## Seguranca

Medidas existentes:

| Medida | Descricao |
| --- | --- |
| Senha criptografada | A senha da API nao e persistida em texto puro no `config.json`. |
| AES-256-GCM | O projeto usa uma chave local de 32 bytes em `secret.key`. |
| Permissoes restritas | Arquivos sensiveis sao gravados com permissao `0600` quando a plataforma respeita esse modo. |
| Sanitizacao de erro | Respostas de erro da API removem chaves como senha, password, token e authorization. |
| Sanitizacao de JSON bruto | Campos sensiveis sao removidos de `OriginalJSON` antes de serem guardados nos modelos. |
| Modo dry-run | `TRANSFER_DRY_RUN=true` bloqueia o POST real de transferencia. |
| Circuit breaker | Falhas criticas de transferencia bloqueiam novos envios temporariamente. |
| Gate de envio | Evita transferencias concorrentes para a mesma empresa. |

Limitacao importante:

```text
Se alguem copiar a pasta local inteira, tambem copia o secret.key. A criptografia atual protege contra exposicao casual do config.json, mas nao substitui um cofre de credenciais do sistema operacional.
```

## Pre-Requisitos

Para desenvolvimento:

| Requisito | Observacao |
| --- | --- |
| Go 1.24+ | Versao definida em `go.mod`. |
| CGO habilitado | Necessario para executar/buildar a interface grafica Fyne completa. |
| Compilador C | No Windows, use MSYS2/MinGW-w64 ou equivalente com `gcc` no `PATH`. |
| Acesso Sienge | Subdominio, usuario e senha da API. |

Verificar ambiente Go:

```bash
go env GOOS GOARCH CGO_ENABLED
```

## Como Executar

Rodar em modo desenvolvimento:

```bash
go run .
```

Gerar o executavel Windows:

```bash
go build -o sienge-transfer.exe .
```

No PowerShell, se precisar garantir CGO:

```powershell
$env:CGO_ENABLED="1"
go build -o sienge-transfer.exe .
```

Se `CGO_ENABLED=0`, o binario compila o fallback sem interface grafica real e informa que Fyne requer CGO.

## Primeiro Uso

Na primeira execucao, o app abre o onboarding.

Campos solicitados:

| Campo | Exemplo |
| --- | --- |
| Nome da empresa | `Construtora Exemplo` |
| Subdominio | `minhaempresa` |
| Usuario API | `usuario.api` |
| Senha API | `senha da API` |
| Nome do usuario | `Joao Silva` |
| Cargo | `Engenheiro` |
| Obras | Centros de custo usados na operacao |

O subdominio deve ser apenas o identificador da empresa. O app normaliza entradas como `https://minhaempresa.sienge.com.br/` para `minhaempresa`.

## Uso Das Abas

| Aba | Finalidade |
| --- | --- |
| Obras | Adicionar/remover centros de custo cadastrados localmente. |
| Consulta | Consultar saldo por IDs de insumo ou por solicitacao de compra. |
| Transferencia | Preparar, validar e enviar transferencia entre obras. |
| Historico | Visualizar resumo local e abrir a planilha Excel. |

## Modo Seguro De Transferencia

Para bloquear envio real ao Sienge:

```bash
TRANSFER_DRY_RUN=true go run .
```

No PowerShell:

```powershell
$env:TRANSFER_DRY_RUN="true"
go run .
```

Valores aceitos como verdadeiro:

```text
1, true, sim, yes
```

Quando o dry-run esta ativo, o botao de envio fica bloqueado e nenhum `POST /stock-movements/transfer` e executado.

## Testes

Rodar toda a suite:

```bash
go test ./...
```

Rodar analise estatica basica:

```bash
go vet ./...
```

Rodar pacote especifico:

```bash
go test ./api
go test ./config
go test ./models
go test ./storage
go test ./ui
```

Teste opcional contra o Sienge real:

```bash
SIENGE_SUBDOMAIN="seu-subdominio" SIENGE_USER="seu-usuario" SIENGE_PASSWORD="sua-senha" go test -tags=integration ./api -run TestSiengeCredentialsIntegration
```

No PowerShell:

```powershell
$env:SIENGE_SUBDOMAIN="seu-subdominio"
$env:SIENGE_USER="seu-usuario"
$env:SIENGE_PASSWORD="sua-senha"
go test -tags=integration ./api -run TestSiengeCredentialsIntegration
```

## Build E Release Local

Sequencia recomendada antes de entregar novo executavel:

```bash
go test ./...
go vet ./...
go build -o sienge-transfer.exe .
```

O arquivo `sienge-transfer.exe` fica na raiz do projeto.

## Variaveis De Ambiente

| Variavel | Uso |
| --- | --- |
| `TRANSFER_DRY_RUN` | Bloqueia envio real de transferencia quando verdadeiro. |
| `SIENGE_TRANSFER_DEBUG_APPROPRIATIONS` | Exibe debug de apropriacoes quando igual a `1`. |
| `SIENGE_SUBDOMAIN` | Usado apenas em teste de integracao. |
| `SIENGE_USER` | Usado apenas em teste de integracao. |
| `SIENGE_PASSWORD` | Usado apenas em teste de integracao. |

## Boas Praticas Do Projeto

Antes de alterar fluxos criticos:

```bash
go test ./...
```

Antes de gerar executavel:

```bash
go test ./...
go vet ./...
go build -o sienge-transfer.exe .
```

Ao trabalhar com credenciais:

```text
Nao commitar .env, senhas reais, config.json local, secret.key local ou dados exportados de clientes.
```

## Solucao De Problemas

| Problema | Possivel causa | Acao sugerida |
| --- | --- | --- |
| `gcc not found` | CGO habilitado sem compilador C instalado | Instale MSYS2/MinGW-w64 e coloque `gcc` no `PATH`. |
| App diz que Fyne requer CGO | Build feito com `CGO_ENABLED=0` | Gere novamente com `CGO_ENABLED=1`. |
| Credenciais invalidas | Usuario/senha/API sem permissao | Refaca credenciais pelo modal de credenciais ou onboarding. |
| Excel nao atualiza | Arquivo aberto/bloqueado por outro processo | Feche a planilha e tente novamente. |
| Transferencia bloqueada temporariamente | Circuit breaker apos erro critico | Aguarde o periodo informado e confira Sienge antes de reenviar. |
| Historico corrompido | `historico.json` invalido | Corrigir/restaurar o arquivo local antes de abrir historico. |

## Estado Atual E Proximos Passos

O projeto ja possui testes para API, configuracao, modelos, persistencia e principais estados de UI.

Melhorias futuras recomendadas:

| Area | Sugestao |
| --- | --- |
| Credenciais | Integrar cofre nativo do sistema operacional. |
| Persistencia | Criar recuperacao guiada para `historico.json` corrompido. |
| UI | Dividir `page_transferencia.go` em arquivos menores por fluxo. |
| Observabilidade | Adicionar logs estruturados locais sem dados sensiveis. |
| Release | Automatizar build e assinatura do executavel. |
