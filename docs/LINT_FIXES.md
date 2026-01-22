# Correções de Lint

Este documento descreve as correções de lint aplicadas ao projeto.

---

## 🐛 Problema: Error return value not checked

### Erro Original
```
cmd/load.go:248: Error return value of `tx.Rollback` is not checked (errcheck)
cmd/load.go:267: Error return value of `tx.Rollback` is not checked (errcheck)
```

### Código Problemático (Antes)

```go
func txInsert(ctx context.Context, db *sql.DB, table string, cols []string, iter func(func([]any) error) error) error {
    tx, err := db.BeginTx(ctx, &sql.TxOptions{})
    if err != nil {
        return err
    }

    // ... código ...

    stmt, err := tx.PrepareContext(ctx, fmt.Sprintf(...))
    if err != nil {
        tx.Rollback()  // ❌ Erro não verificado (linha 248)
        return err
    }

    // ... código ...

    err = iter(...)
    if err != nil {
        tx.Rollback()  // ❌ Erro não verificado (linha 267)
        return err
    }

    return tx.Commit()
}
```

**Problema:** O linter `errcheck` detecta que o retorno de `tx.Rollback()` não está sendo verificado.

---

## ✅ Solução Implementada

### Código Corrigido (Depois)

```go
func txInsert(ctx context.Context, db *sql.DB, table string, cols []string, iter func(func([]any) error) error) error {
    tx, err := db.BeginTx(ctx, &sql.TxOptions{})
    if err != nil {
        return err
    }

    // ✅ Defer rollback - executado automaticamente ao sair da função
    defer func() {
        if err := tx.Rollback(); err != nil && err != sql.ErrTxDone {
            logger.Error("rollback failed", slog.String("table", table), slog.String("error", err.Error()))
        }
    }()

    // ... resto do código ...
    // Não precisa mais chamar tx.Rollback() manualmente!

    if err := tx.Commit(); err != nil {
        return err
    }
    return nil
}
```

---

## 🎯 Vantagens da Solução

### 1. ✅ Passa no Linter
O erro de `tx.Rollback()` agora é verificado e tratado.

### 2. ✅ Código Mais Limpo
Não precisa chamar `tx.Rollback()` em múltiplos lugares.

### 3. ✅ Mais Seguro
O rollback é **sempre** executado se a transação não for commitada, mesmo em caso de panic.

### 4. ✅ Ignora `sql.ErrTxDone`
```go
err != sql.ErrTxDone
```

Quando `tx.Commit()` é bem-sucedido, `tx.Rollback()` retorna `sql.ErrTxDone` (esperado). A condição ignora esse erro específico.

---

## 📚 Padrão Recomendado para Transações SQL

### ✅ MELHOR PRÁTICA (defer rollback)

```go
func executeWithTx(ctx context.Context, db *sql.DB) error {
    tx, err := db.BeginTx(ctx, nil)
    if err != nil {
        return err
    }

    // Sempre use defer para rollback
    defer func() {
        if err := tx.Rollback(); err != nil && err != sql.ErrTxDone {
            log.Printf("rollback failed: %v", err)
        }
    }()

    // Fazer operações...
    if err := doStuff(tx); err != nil {
        return err  // Rollback automático via defer
    }

    // Commit
    return tx.Commit()  // Se sucesso, defer rollback retorna ErrTxDone (ignorado)
}
```

### ❌ EVITE (rollback manual)

```go
func executeWithTx(ctx context.Context, db *sql.DB) error {
    tx, err := db.BeginTx(ctx, nil)
    if err != nil {
        return err
    }

    // ❌ Problemático: precisa lembrar de fazer rollback em TODOS os erros
    if err := doStuff(tx); err != nil {
        tx.Rollback()  // Pode esquecer de verificar erro
        return err
    }

    if err := doMoreStuff(tx); err != nil {
        tx.Rollback()  // Código duplicado
        return err
    }

    return tx.Commit()
}
```

---

## 🔍 Por Que `sql.ErrTxDone` é Esperado?

Quando você chama `tx.Commit()` com sucesso:
1. A transação é marcada como "concluída"
2. O `defer` executa `tx.Rollback()`
3. SQLite retorna `sql.ErrTxDone` (transação já está concluída)
4. Esse erro é **esperado** e pode ser ignorado

```go
// Fluxo normal (commit bem-sucedido):
tx.Commit()          // ✅ Sucesso, tx marcada como "done"
  ↓
defer tx.Rollback()  // ⚠️  Retorna sql.ErrTxDone (ignorado)
  ↓
return nil           // ✅ Função retorna sucesso
```

```go
// Fluxo de erro (antes do commit):
return err           // ❌ Erro aconteceu
  ↓
defer tx.Rollback()  // ✅ Rollback real acontece
  ↓
return err           // ❌ Função retorna erro
```

---

## 🧪 Como Testar

### Teste de Commit Bem-Sucedido
```go
// Deve passar sem erros
err := txInsert(ctx, db, "empresas", cols, func(emit func([]any) error) error {
    return emit([]any{"12345678", "Empresa Teste", ...})
})
// err deve ser nil
```

### Teste de Rollback em Erro
```go
// Deve fazer rollback automaticamente
err := txInsert(ctx, db, "empresas", cols, func(emit func([]any) error) error {
    emit([]any{"12345678", "Empresa 1", ...})
    return fmt.Errorf("erro simulado")  // Força rollback
})
// err deve ser "erro simulado"
// Nenhum dado deve ter sido inserido (rollback funcionou)
```

---

## 📝 Checklist de Validação

- [x] Código passa no `errcheck`
- [x] Código passa no `go vet`
- [x] Rollback é executado em caso de erro
- [x] Commit bem-sucedido não gera erros
- [x] Logs de rollback failures são registrados
- [x] `sql.ErrTxDone` é ignorado corretamente

---

## 🔗 Referências

- [Go Database/SQL Tutorial](https://go.dev/doc/database/execute-transactions)
- [errcheck linter](https://github.com/kisielk/errcheck)
- [Go SQL Best Practices](https://www.alexedwards.net/blog/working-with-transactions-in-go)

---

**Correção aplicada em:** Janeiro 2026
**Arquivo modificado:** `cmd/load.go`
**Linhas corrigidas:** 248, 267
