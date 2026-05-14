# Analise de melhorias e refatoracao

Data: 2026-05-14

Escopo revisado: codigo Go, testes, docs, configuracao de build e estado Git do repositorio. Esta analise nao aplica correcoes de codigo; ela registra pontos priorizados para correcoes futuras.

## Achados prioritarios

| Prioridade | Referencia | Achado | Impacto | Sugestao |
| --- | --- | --- | --- | --- |
| Critica | `ui/page_transferencia.go:898` | `RecalculateTransferSaldos` so atualiza `EstoqueOrigemAntes` quando o saldo recalculado e maior que zero. | Se o estoque cair para zero ou o item sumir, o estado mantem saldo antigo e `RevalidateTransferBeforeSend` pode permitir envio com saldo obsoleto. | Atualizar o saldo da origem incondicionalmente, inclusive zero, e tratar item nao encontrado como saldo zero ou erro explicito. |
| Alta | `models/loan.go:245` | `ApplyReturnToLoan` salva `transfer.LinkedLoanID` em `ReturnTransferIDs`. Esse valor e o ID do emprestimo, nao da transferencia de devolucao. | Historico de devolucoes fica incorreto e repete o proprio ID do emprestimo no campo de retornos. | Gravar o identificador real da transferencia de devolucao, ou remover/renomear `ReturnTransferIDs` se esse dado nao existir. |
| Alta | `ui/page_emprestimos.go:410`, `ui/page_emprestimos.go:414` | O select de emprestimo para devolucao usa label nao unica: obra destino, solicitante e data. | Dois emprestimos no mesmo dia para a mesma obra/solicitante podem vincular a devolucao ao emprestimo errado. | Incluir `loan.ID` no label ou manter mapa interno label/id gerado pela UI. |
| Alta | `models/loan.go:263` | Matching de item devolvido usa apenas recurso, detalhe e marca. | Itens iguais em insumo/detalhe/marca, mas de apropriacoes diferentes, podem abater no item errado ou rejeitar devolucao valida. | Incluir chaves de apropriacao no matching ou consolidar explicitamente itens equivalentes ao criar o emprestimo. |
| Alta | `storage/historico.go:138-148`, `config/config.go:133-143` | `replaceFile` remove o arquivo destino antes de renomear o temporario quando `os.Rename` falha. | Em Windows, antivirus, lock de arquivo ou queda de energia podem causar perda de `historico.json`, `emprestimos.json`, `config.json` ou Excel. | Usar estrategia atomica por plataforma ou manter backup antes da substituicao. |
| Alta | `storage/historico.go:151-158`, `storage/loan_store.go:59-72`, `storage/excel.go:110-147` | Operacoes de append/upsert fazem ciclo read-modify-write sem serializacao. | Duas operacoes simultaneas podem sobrescrever historico, emprestimos ou linhas do Excel. | Adicionar mutex compartilhado por store e considerar lock de arquivo para multiplas instancias. |
| Alta | `api/transferencia.go:92-93` | `SourceDepartmentID` e `DestinationDepartmentID` recebem `BuildingUnitID` do primeiro item. | Campo semanticamente incorreto pode causar rejeicao ou apropriacao errada no Sienge, especialmente com multiplos itens. | Remover se nao for obrigatorio ou criar campos explicitos para departamento com validacao. |
| Alta | `ui/page_transferencia.go:248`, `ui/page_transferencia.go:807`, `ui/page_transferencia.go:857`, `ui/shared_async.go:15-23` | Workers assicronos leem e alteram `state.Transferencia` e outros estados compartilhados. | Risco de race entre callbacks Fyne, rebuilds e edicao do usuario durante envio/recalculo. | Capturar snapshot validado antes da goroutine e restringir mutacoes de UI/estado ao dispatch principal. |
| Alta | `ui/page_emprestimos.go:314-329` | Modal de selecao de itens usa `SetChecked` dentro de callbacks sem guarda de atualizacao programatica. | Checkboxes podem disparar callbacks uns dos outros e alternar selecao incorretamente. | Adicionar flag `updatingChecks` para ignorar callbacks enquanto a UI sincroniza checks. |
| Alta | `ui/page_transferencia.go:346-354` | `OnChanged` e `OnSubmitted` de quantidade acessam `state.Transferencia.Itens[rowIndex]` sem validar limites. | Callback antigo pode panicar ou alterar item errado apos remocao/rebuild da linha. | Reusar a checagem de indice ja aplicada no `OnFocusLost`. |
| Media | `ui/page_transferencia.go:657`, `api/transferencia.go:167` | Validacao de quantidade maior que disponivel so roda quando `QuantidadeDisponivel > 0`. | Se a disponibilidade carregada for zero, uma quantidade positiva pode passar por validacao local. | Diferenciar disponibilidade nao carregada de disponibilidade zero usando ponteiro/flag; validar contra estoque total quando nao houver apropriacao. |
| Media | `api/estoque.go:145-156` | Consulta de apropriacoes usa `limit=100` sem paginacao. | Apropriacoes acima de 100 sao truncadas silenciosamente. | Paginar ate a pagina vir menor que o limite, como no fluxo de solicitacao de compra. |
| Media | `ui/app_services.go:42-43`, `ui/page_emprestimos.go:454-456` | Erros de refresh inicial e carregamento de emprestimos para devolucao sao ignorados. | Usuario pode ver listas vazias sem saber que houve falha de storage. | Persistir erro em status da aba e evitar falha silenciosa. |
| Media | `ui/page_emprestimos.go:55-65` | `BuildEmprestimosTab` reler storage durante montagem; filtro de busca reconstrui a aba. | Digitar no filtro pode causar IO sincrono e perda de responsividade. | Separar carregamento de dados de filtragem visual; carregar sob demanda ou em fluxo assicrono. |
| Media | `ui/page_transferencia.go:140`, `ui/page_emprestimos.go:450-466` | `BuildTransferenciaTab` chama `loadReturnLoans` em toda montagem. | Rebuilds da tela fazem IO sincrono repetido. | Carregar apenas ao selecionar tipo Devolucao ou via botao refresh. |
| Media | `models/transfer_balance.go:36-38`, `models/transfer_stock_snapshot.go:30-34` | Comparacoes de `float64` usam igualdade/maior direto, sem tolerancia consistente. | Quantidades decimalmente equivalentes podem ser rejeitadas por erro de representacao. | Centralizar tolerancia de quantidade e aplicar nas validacoes de saldo. |
| Media | `models/transfer_balance.go:26-38`, `models/transfer_stock_snapshot.go:30-34` | `CalculateTransferBalances` valida o saldo da apropriacao quando existe, mas nao valida tambem estoque total; snapshot valida ambos. | UI pode mostrar operacao possivel e snapshot rejeitar depois. | Alinhar invariantes entre calculo de exibicao e snapshot final. |
| Media | `models/loan.go:116-122` | `PendingQuantity` mascara estado superdevolvido retornando zero quando devolvido maior que emprestado. | JSON corrompido ou estado invalido pode aparecer como devolvido normal. | Adicionar validacao de integridade `ReturnedQuantity <= LoanedQuantity`. |
| Media | `api/client.go:505-510` | Redacao de texto sensivel substitui so palavras como `password`, nao os valores. | Corpo nao JSON pode expor segredo, por exemplo `Password=segredo`. | Usar regex case-insensitive que remova par chave/valor inteiro. |
| Media | `api/client.go:124-129` | `NewClientWithBaseURL` aceita `http://`. | Credenciais Basic Auth podem trafegar sem TLS se usado fora de testes. | Rejeitar HTTP por padrao ou exigir opcao explicita de teste. |
| Media | `api/purchase_requests.go:35-39` | Deduplicacao de itens de solicitacao usa apenas recurso/detalhe/marca. | Linhas distintas da mesma solicitacao podem virar um unico item. | Usar ID/linha da solicitacao quando existir ou evitar deduplicacao sem chave confiavel. |
| Media | `ui/onboarding_view.go:125`, `ui/onboarding_view.go:154`, `ui/modals.go:240` | Chamadas de rede/validacao rodam diretamente em callbacks da UI. | Janela pode congelar durante API lenta e permite duplo clique. | Usar `AsyncRunner`, status de carregamento e bloqueio temporario dos botoes. |
| Baixa | `docs/USO.md:27-36`, `api/client.go:121`, `api/transferencia.go:16` | Documentacao de base URL e endpoints diverge do codigo. | Troubleshooting e integracao manual podem seguir URLs erradas. | Atualizar docs para `https://api.sienge.com.br/{subdominio}/public/api/v1` e endpoints reais. |
| Baixa | `ui/page_transferencia.go:209-227`, `ui/page_transferencia.go:358-360`, `ui/page_historico.go:29-40` | Varios feedbacks usam `status.SetText` direto sem persistir no estado. | Mensagem pode sumir apos rebuild/troca de aba. | Padronizar helpers `set*Status` por aba e usa-los antes de refresh. |
| Baixa | `api/transferencia.go:307-309` | `ExtractMovementID` extrai ultimo segmento de `Location` por split simples. | URL com query string pode gerar ID com `?x=1`. | Parsear com `net/url` e usar apenas `Path`. |
| Baixa | `api/estoque.go:363-370` | Conversao numerica flexivel trunca `float64` para `int`. | Resposta malformada pode gerar IDs incorretos. | Rejeitar floats nao inteiros e validar range. |
| Baixa | `models/loan.go:308-328` | `sanitizeLoanIDPart` pode retornar string vazia apos remover hifens. | IDs podem terminar com sufixo vazio, reduzindo rastreabilidade. | Aplicar fallback apos `strings.Trim`. |
| Baixa | `sienge-transfer.exe` | Executavel binario esta versionado no Git. | Aumenta o repositorio, dificulta diffs e pode gerar conflitos frequentes. | Preferir GitHub Releases ou artefatos de build; manter no Git apenas se for decisao explicita de distribuicao. |

## Refatoracoes recomendadas

1. Separar `ui/page_transferencia.go` em componentes menores: construcao da tela, mutacoes de estado, validacao, envio, recalc e montagem de item.
2. Separar `ui/page_emprestimos.go` em tabela/viewmodel, modal de selecao, helpers de dominio e side effects de storage.
3. Extrair escrita atomica duplicada de `config/config.go` e `storage/historico.go` para um utilitario interno unico.
4. Criar uma camada de servico para side effects de emprestimos, reduzindo acoplamento da UI com storage e dominio.
5. Centralizar tolerancia de quantidade e regras de estoque em `models`, para API/UI/storage usarem as mesmas invariantes.
6. Padronizar status/feedback por aba para evitar mensagens que somem apos refresh.
7. Unificar viewmodels reais e testados na consulta; hoje parte da UI monta manualmente uma estrutura diferente do viewmodel testado.

## Ordem sugerida de execucao

1. Corrigir riscos de envio incorreto: saldo zero no recalculo, validacao de disponibilidade zero, matching de devolucao e label nao unico.
2. Corrigir persistencia: replace seguro, locks de escrita e erros ignorados de refresh/storage.
3. Corrigir payload Sienge: departamentos, paginacao de apropriacao e deduplicacao de solicitacao de compra.
4. Reduzir riscos de UI assicrona: snapshots antes de goroutine, callbacks com guarda e network fora do thread de UI.
5. Refatorar arquivos grandes e centralizar regras/tolerancias depois que os bugs acima estiverem cobertos por testes.

## Estado observado

O branch estava sincronizado com `origin/feature/etapas`. Existia um arquivo nao rastreado `planejamento_funcionalidade_emprestimos.md`, mantido fora desta analise por ser um artefato de planejamento anterior.
