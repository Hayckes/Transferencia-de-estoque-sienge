# Planejamento de Desenvolvimento — Botão Verde e Recalcular Saldos ao Sair do Campo de Quantidade

## 1. Objetivo

Implementar duas melhorias na aba **Transferência** do app **Sienge Transfer**:

1. O botão **Enviar Transferência** deve ter destaque visual em **verde**.
2. A mesma função executada pelo botão **Recalcular saldos** deve ser executada automaticamente quando o usuário terminar de editar a quantidade e clicar fora do input de quantidade.

O botão **Recalcular saldos** deve continuar existindo, mas passa a ser uma ação manual opcional, caso o usuário queira recalcular novamente.

---

## 2. Escopo

## 2.1 Alteração visual

Alterar o botão:

```text
Enviar Transferência
```

Para que seja exibido em verde, indicando ação principal/positiva.

## 2.2 Alteração comportamental

Hoje, o usuário precisa clicar no botão:

```text
Recalcular saldos
```

para atualizar:

```text
Estoque atual de origem
Estoque atual de destino
Saldo de origem após transferência
Saldo de destino após transferência
```

Com a mudança, a função de recalcular deve executar automaticamente quando o usuário:

1. Digitar ou alterar a quantidade.
2. Clicar fora do input de quantidade.
3. O valor digitado for válido.

O botão **Recalcular saldos** deve continuar disponível, mas apenas como reforço manual.

---

## 3. Regras funcionais

## 3.1 Botão “Enviar Transferência” verde

O botão deve:

1. Ter aparência verde.
2. Continuar executando a mesma ação atual de envio.
3. Continuar respeitando validações existentes.
4. Continuar sendo desabilitado enquanto uma transferência estiver em envio, se já houver essa proteção.
5. Não perder comportamento de feedback/sucesso/erro.

Texto do botão:

```text
Enviar Transferência
```

Cor desejada:

```text
Verde
```

Sugestão de cor:

```text
#2E7D32
```

ou:

```text
#16A34A
```

Se o app usa tema Fyne customizado, preferir usar uma cor semântica configurada no tema.

---

## 3.2 Recalcular saldos ao sair do input

Quando o usuário editar a quantidade e sair do campo, o app deve executar a mesma lógica do botão:

```text
Recalcular saldos
```

Ou seja, o comportamento automático deve reaproveitar a função existente, não duplicar regra de cálculo.

Exemplo:

```text
1. Usuário digita 5,0000 no input de quantidade.
2. Usuário clica em outro campo ou fora do input.
3. O app normaliza/valida a quantidade.
4. O app recalcula os saldos automaticamente.
5. O app atualiza os textos de saldo na tela.
```

---

## 4. Comportamento esperado

## 4.1 Quantidade válida

Entrada:

```text
1
```

Ao sair do campo:

```text
1,0000
```

E recalcula os saldos.

Entrada:

```text
1,5
```

Ao sair do campo:

```text
1,5000
```

E recalcula os saldos.

Entrada:

```text
1.5
```

Ao sair do campo:

```text
1,5000
```

E recalcula os saldos.

---

## 4.2 Quantidade inválida

Entrada:

```text
abc
```

Ao sair do campo:

1. Não recalcular.
2. Exibir feedback:

```text
Quantidade inválida. Informe um valor no formato 0,0000.
```

---

## 4.3 Quantidade zero ou negativa

Entrada:

```text
0,0000
```

ou:

```text
-1,0000
```

Ao sair do campo:

1. Não recalcular.
2. Exibir feedback:

```text
A quantidade a transferir deve ser maior que zero.
```

---

## 4.4 Quantidade maior que saldo de origem

Se o usuário informar quantidade maior que o saldo disponível da apropriação ou do estoque de origem:

1. Recalcular pode ser executado, mas deve retornar erro de validação.
2. A tela deve mostrar feedback:

```text
A quantidade informada é maior que o saldo disponível na origem.
```

3. O saldo após transferência não deve ser exibido como valor válido negativo sem aviso.

---

## 5. Avaliação técnica em Fyne

## 5.1 Atenção sobre evento de “perda de foco”

O Fyne pode não oferecer um callback simples e universal de “on blur” em `widget.Entry` padrão, dependendo da versão usada.

A IA do CLI deve verificar no código e na versão do Fyne:

```text
fyne.io/fyne/v2
```

Opções possíveis:

1. Usar `OnSubmitted`, caso o usuário pressione Enter.
2. Criar um componente customizado que estende `widget.Entry` e sobrescreve `FocusLost`.
3. Usar `OnChanged` com debounce.
4. Usar o botão manual como fallback.
5. Recalcular quando o usuário clicar em outro controle da tela, se houver mecanismo centralizado.

A solução preferencial é criar um componente customizado de entrada de quantidade com `FocusLost`, se a versão do Fyne permitir.

---

## 5.2 Componente sugerido: `QuantityEntry`

Criar um componente reutilizável:

```go
type QuantityEntry struct {
    widget.Entry
    OnFocusLost func(value string)
}
```

Implementar:

```go
func NewQuantityEntry(onFocusLost func(value string)) *QuantityEntry
```

Sobrescrever:

```go
func (e *QuantityEntry) FocusLost() {
    e.Entry.FocusLost()

    if e.OnFocusLost != nil {
        e.OnFocusLost(e.Text)
    }
}
```

Uso:

```go
quantityEntry := NewQuantityEntry(func(value string) {
    normalized, quantity, err := NormalizeQuantityInput(value)
    if err != nil {
        state.SetStatus("Quantidade inválida. Informe um valor no formato 0,0000.")
        return
    }

    quantityEntry.SetText(normalized)
    item.Quantidade = quantity

    recalculateTransferBalances(state, item)
})
```

Atenção: evitar loop infinito se `SetText` disparar `OnChanged`. Proteger com flag interna se necessário.

---

## 5.3 Alternativa com debounce em `OnChanged`

Se `FocusLost` não for viável, implementar debounce:

```text
Usuário digita quantidade
Aguarda 400-700ms sem digitar
Se valor válido, recalcula saldos
```

Mas essa alternativa é secundária, porque o requisito pede especificamente ao clicar fora do input.

---

# 6. Reutilização da função “Recalcular saldos”

## 6.1 Problema a evitar

Não duplicar a regra de cálculo.

Errado:

```text
Botão Recalcular usa uma função.
FocusLost usa outra função parecida.
```

Correto:

```text
Botão Recalcular e FocusLost chamam a mesma função.
```

## 6.2 Função central sugerida

Criar ou consolidar:

```go
func RecalculateTransferItemBalances(state *AppState, itemIndex int) error
```

ou, se a lógica for pura:

```go
func CalculateTransferBalances(input TransferBalanceInput) (TransferBalanceOutput, error)
```

E uma função de UI:

```go
func ApplyTransferBalancesToUI(state *AppState, itemIndex int, output TransferBalanceOutput)
```

O fluxo deve ser:

```text
Input de quantidade perdeu foco
    -> normaliza quantidade
    -> atualiza quantidade do item
    -> chama RecalculateTransferItemBalances
    -> atualiza UI/status
```

Botão Recalcular:

```text
Clique no botão
    -> chama RecalculateTransferItemBalances para todos os itens ou item selecionado
    -> atualiza UI/status
```

---

# 7. Botão “Recalcular saldos” continua existindo

Mesmo com o cálculo automático, o botão deve continuar disponível.

Motivo:

1. O usuário pode querer recalcular manualmente após mudar apropriação.
2. O usuário pode querer confirmar saldos antes de enviar.
3. Pode haver casos em que o evento de perda de foco não seja disparado conforme esperado.
4. Serve como fallback de usabilidade.

Texto atual:

```text
Recalcular saldos
```

Pode manter o mesmo texto.

Feedback ao clicar:

```text
Saldos recalculados com sucesso.
```

Ou em caso de erro:

```text
Não foi possível recalcular os saldos: {erro}
```

---

# 8. Regras de atualização de saldos

Ao recalcular, o app deve considerar:

## 8.1 Quando houver apropriação de origem

```text
Estoque atual de origem = saldo da apropriação de origem selecionada
Saldo origem após transferência = saldo da apropriação origem - quantidade
```

## 8.2 Quando não houver apropriação de origem

```text
Estoque atual de origem = estoque total do item na origem
Saldo origem após transferência = estoque total origem - quantidade
```

## 8.3 Quando houver apropriação de destino

```text
Estoque atual de destino = saldo da apropriação de destino selecionada
Saldo destino após transferência = saldo da apropriação destino + quantidade
```

## 8.4 Quando não houver apropriação de destino

```text
Estoque atual de destino = estoque total do item no destino
Saldo destino após transferência = estoque total destino + quantidade
```

---

# 9. Feedback para o usuário

## 9.1 Quando recalcular automaticamente com sucesso

Mostrar feedback discreto:

```text
Saldos recalculados.
```

Evitar feedback muito intrusivo a cada edição.

## 9.2 Quando falhar

Mostrar feedback claro:

```text
Não foi possível recalcular os saldos. Verifique a quantidade informada.
```

Com detalhe técnico copiável no `StatusView`, se houver.

## 9.3 Quando o valor estiver vazio

Se o usuário apagar o campo e sair dele:

```text
Informe a quantidade a transferir.
```

Não recalcular.

---

# 10. Alteração visual do botão verde

## 10.1 Opções em Fyne

A IA do CLI deve verificar como os botões estão sendo renderizados.

Possibilidades:

1. `widget.Button` comum.
2. `widget.NewButtonWithIcon`.
3. Botão com tema customizado.
4. Botão dentro de componente próprio.
5. Uso de `canvas.Rectangle` ou `container` customizado para simular botão colorido.

O `widget.Button` padrão do Fyne não permite alterar cor diretamente por instância em todas as versões. Por isso, pode ser necessário criar um botão customizado.

---

## 10.2 Opção recomendada: componente `PrimarySuccessButton`

Criar helper:

```go
func NewSuccessButton(label string, tapped func()) fyne.CanvasObject
```

Implementação possível:

```go
button := widget.NewButton(label, tapped)
button.Importance = widget.HighImportance
```

Se `HighImportance` não ficar verde no tema atual, criar componente custom com:

- fundo verde;
- texto branco;
- hover/click compatível se possível.

Exemplo conceitual:

```go
bg := canvas.NewRectangle(color.NRGBA{R: 22, G: 163, B: 74, A: 255})
text := canvas.NewText(label, color.White)
```

Mas preferir usar os componentes padrões do Fyne se o projeto já possui tema.

---

## 10.3 Cuidado com tema global

Não alterar a cor de todos os botões do app.

A mudança deve afetar apenas:

```text
Enviar Transferência
```

Não deve deixar verdes botões como:

```text
Limpar
Recalcular saldos
Detalhes
Fechar
Copiar
```

---

# 11. TDD obrigatório

## 11.1 Testes de normalização de quantidade

Arquivo sugerido:

```text
ui/quantity_input_test.go
```

Testes:

```go
func TestNormalizeQuantityInput_FormatsIntegerWithFourDecimals(t *testing.T)
```

```go
func TestNormalizeQuantityInput_FormatsCommaDecimal(t *testing.T)
```

```go
func TestNormalizeQuantityInput_FormatsDotDecimal(t *testing.T)
```

```go
func TestNormalizeQuantityInput_RejectsInvalidValue(t *testing.T)
```

```go
func TestNormalizeQuantityInput_RejectsZero(t *testing.T)
```

---

## 11.2 Testes da função central de recálculo

Arquivo sugerido:

```text
models/transfer_balance_test.go
```

Testes:

```go
func TestCalculateTransferBalances_UsesOriginAppropriationWhenSelected(t *testing.T)
```

```go
func TestCalculateTransferBalances_UsesDestinationAppropriationWhenSelected(t *testing.T)
```

```go
func TestCalculateTransferBalances_UsesTotalStockWhenNoAppropriation(t *testing.T)
```

```go
func TestCalculateTransferBalances_RejectsNegativeOriginAfterTransfer(t *testing.T)
```

---

## 11.3 Testes para reaproveitamento da mesma função

Arquivo sugerido:

```text
ui/transfer_recalculate_trigger_test.go
```

Criar uma abstração testável:

```go
type RecalculateTrigger string

const (
    RecalculateByButton RecalculateTrigger = "button"
    RecalculateByQuantityFocusLost RecalculateTrigger = "quantity_focus_lost"
)
```

Criar função:

```go
func HandleRecalculateTrigger(state *TransferState, itemIndex int, trigger RecalculateTrigger) error
```

Testes:

```go
func TestHandleRecalculateTrigger_ButtonUsesSameRecalculationAsFocusLost(t *testing.T)
```

```go
func TestHandleRecalculateTrigger_FocusLostRecalculatesWhenQuantityIsValid(t *testing.T)
```

```go
func TestHandleRecalculateTrigger_FocusLostDoesNotRecalculateWhenQuantityInvalid(t *testing.T)
```

---

## 11.4 Testes do botão verde

Como cor visual é difícil de testar em Fyne, criar helper testável.

Exemplo:

```go
type ButtonStyleKind string

const (
    ButtonStyleDefault ButtonStyleKind = "default"
    ButtonStyleSuccess ButtonStyleKind = "success"
)
```

Criar:

```go
func BuildTransferSubmitButtonViewModel() ButtonViewModel
```

Teste:

```go
func TestBuildTransferSubmitButtonViewModel_IsSuccessStyle(t *testing.T)
```

Esperado:

```text
Label = Enviar Transferência
Style = success
```

Se houver componente custom, testar que ele recebe a cor definida.

---

# 12. Implementação recomendada por etapas

## Etapa 1 — Identificar código atual

A IA do CLI deve localizar:

```text
Enviar Transferência
Recalcular saldos
Quantidade
Entry de quantidade
CalculateTransferBalances
Recalculate
Transferencia
```

Arquivos prováveis:

```text
ui/transferencia.go
ui/transfer_tab.go
ui/transfer_balance.go
ui/transfer_item.go
models/transferencia.go
models/transfer_balance.go
```

---

## Etapa 2 — Criar testes

Criar primeiro os testes de:

1. Normalização de quantidade.
2. Cálculo de saldos.
3. Trigger de recálculo por botão e por focus lost.
4. ViewModel do botão verde.

Rodar:

```bash
go test ./...
```

Os testes devem falhar antes da implementação.

---

## Etapa 3 — Consolidar função de recálculo

Garantir que exista uma única função central:

```go
RecalculateTransferItemBalances(...)
```

ou equivalente.

O botão **Recalcular saldos** deve chamar essa função.

O evento de perda de foco do input de quantidade também deve chamar essa função.

---

## Etapa 4 — Criar input de quantidade com FocusLost

Implementar `QuantityEntry` ou equivalente.

Regras:

1. Ao perder foco, normalizar quantidade.
2. Se válido, atualizar o item.
3. Chamar recálculo.
4. Se inválido, mostrar feedback e não recalcular.
5. Evitar loops causados por `SetText`.

---

## Etapa 5 — Aplicar ao item de transferência

Substituir o `widget.Entry` atual de quantidade pelo novo componente.

Garantir que cada item de transferência chame recálculo para o item correto.

Cuidado com closures em loops:

Errado:

```go
for i, item := range items {
    entry.OnFocusLost = func() {
        Recalculate(i)
    }
}
```

Se `i` for capturado incorretamente, todos os inputs podem recalcular o último item.

Correto:

```go
for i := range items {
    itemIndex := i
    entry.OnFocusLost = func() {
        Recalculate(itemIndex)
    }
}
```

---

## Etapa 6 — Botão verde

Alterar apenas o botão **Enviar Transferência**.

Se usar `Importance`:

```go
sendButton.Importance = widget.HighImportance
```

Se isso não for verde, criar helper específico:

```go
NewSuccessButton("Enviar Transferência", onSend)
```

Não mudar tema global de todos os botões.

---

## Etapa 7 — Feedback

Ao recalcular automaticamente:

```text
Saldos recalculados.
```

Ao recalcular manualmente:

```text
Saldos recalculados com sucesso.
```

Ao erro:

```text
Não foi possível recalcular os saldos: {erro}
```

---

# 13. Checklist de validação manual

## 13.1 Botão verde

1. Abrir aba Transferência.
2. Confirmar que **Enviar Transferência** está verde.
3. Confirmar que os outros botões não ficaram verdes:
   - Recalcular saldos
   - Limpar
   - Detalhes
   - Copiar
4. Confirmar que o botão continua clicável.
5. Confirmar que o botão respeita estado desabilitado durante envio, se aplicável.

---

## 13.2 Recalcular ao sair do input

1. Adicionar um insumo à transferência.
2. Selecionar apropriação de origem, se houver.
3. Selecionar apropriação de destino, se houver.
4. Digitar:

```text
1
```

5. Clicar fora do input.
6. Confirmar que o campo vira:

```text
1,0000
```

7. Confirmar que os saldos foram atualizados automaticamente.
8. Alterar para:

```text
2,5000
```

9. Clicar fora.
10. Confirmar novo recálculo.

---

## 13.3 Botão manual ainda funciona

1. Alterar quantidade.
2. Clicar em **Recalcular saldos**.
3. Confirmar que o recálculo acontece.
4. Confirmar feedback.

---

## 13.4 Quantidade inválida

1. Digitar:

```text
abc
```

2. Clicar fora.
3. Confirmar feedback de erro.
4. Confirmar que saldos não foram atualizados com valor inválido.

---

## 13.5 Quantidade acima do saldo

1. Digitar quantidade maior que o saldo de origem.
2. Clicar fora.
3. Confirmar feedback de erro.
4. Confirmar que o app não permite enviar transferência inválida.

---

# 14. Critérios de aceite

A implementação será considerada concluída quando:

```text
1. O botão Enviar Transferência estiver verde.
2. Apenas o botão Enviar Transferência for afetado pela cor verde.
3. O botão continuar executando a ação atual corretamente.
4. A quantidade for normalizada para 4 casas decimais ao sair do input.
5. O recálculo de saldos acontecer automaticamente ao sair do input de quantidade.
6. O botão Recalcular saldos continuar funcionando.
7. Botão e focus lost usarem a mesma função de recálculo.
8. Quantidades inválidas não recalcularem saldos.
9. Quantidade maior que saldo exibir erro antes do envio.
10. go test ./... passar.
```

---

# 15. Prompt direto para a IA do CLI

```text
Implemente duas melhorias na aba Transferência do app Sienge Transfer.

1. O botão “Enviar Transferência” deve ser verde.
- Alterar apenas esse botão.
- Não mudar a cor global de todos os botões.
- Manter o comportamento atual de envio, validação e feedback.
- Se widget.Button não permitir cor por instância, criar helper NewSuccessButton ou usar o padrão de tema existente.
- O botão Recalcular saldos, Limpar, Detalhes, Copiar e Fechar não devem ficar verdes.

2. A função do botão “Recalcular saldos” deve executar automaticamente quando o usuário clicar fora do input de quantidade.
- O botão Recalcular saldos deve continuar existindo.
- Não duplicar regra de cálculo.
- Botão manual e evento de perda de foco devem chamar a mesma função central de recálculo.
- Ao sair do input:
  - normalizar quantidade para 4 casas decimais no formato 0,0000;
  - aceitar vírgula e ponto;
  - validar quantidade > 0;
  - se válido, recalcular saldos;
  - se inválido, mostrar feedback e não recalcular.
- Se Fyne permitir, criar QuantityEntry sobrescrevendo FocusLost.
- Se FocusLost não for viável, implementar alternativa segura com OnSubmitted ou debounce, documentando a limitação.
- Cuidado com closures em loops para recalcular o item correto.

TDD obrigatório:
- TestNormalizeQuantityInput_FormatsIntegerWithFourDecimals
- TestNormalizeQuantityInput_FormatsCommaDecimal
- TestNormalizeQuantityInput_FormatsDotDecimal
- TestNormalizeQuantityInput_RejectsInvalidValue
- TestNormalizeQuantityInput_RejectsZero
- TestCalculateTransferBalances_UsesOriginAppropriationWhenSelected
- TestCalculateTransferBalances_UsesDestinationAppropriationWhenSelected
- TestCalculateTransferBalances_UsesTotalStockWhenNoAppropriation
- TestCalculateTransferBalances_RejectsNegativeOriginAfterTransfer
- TestHandleRecalculateTrigger_ButtonUsesSameRecalculationAsFocusLost
- TestHandleRecalculateTrigger_FocusLostRecalculatesWhenQuantityIsValid
- TestHandleRecalculateTrigger_FocusLostDoesNotRecalculateWhenQuantityInvalid
- TestBuildTransferSubmitButtonViewModel_IsSuccessStyle

Finalizar apenas quando:
- go test ./... passar;
- o botão Enviar Transferência estiver verde;
- os demais botões não forem afetados;
- o recálculo acontecer ao sair do campo de quantidade;
- o botão Recalcular saldos continuar funcionando.
```
