# Correção de Bug: URL da Receita Federal

## 🐛 Problema Identificado

O script `scripts/check_latest.sh` tinha **2 bugs críticos** que impediam a detecção da última versão:

### Bug 1: URL Base Incorreta
```bash
# ❌ ERRADO (antigo)
BASE_URL="https://dadosabertos.rfb.gov.br/CNPJ"

# ✅ CORRETO (novo)
BASE_URL="https://arquivos.receitafederal.gov.br/dados/cnpj/dados_abertos_cnpj"
```

### Bug 2: URL Não Incluía o Ano-Mês
```bash
# ❌ ERRADO (antigo) - sempre testava a mesma URL
test_url="$BASE_URL/Empresas0.zip"

# ✅ CORRETO (novo) - testa URL específica de cada mês
test_url="$BASE_URL/$ym/Empresas0.zip"
```

---

## 📋 Detalhes da Correção

### Antes (Não Funcionava)

O script testava sempre:
```
https://dadosabertos.rfb.gov.br/CNPJ/Empresas0.zip
```

Isso **sempre falhava** porque:
1. O domínio `dadosabertos.rfb.gov.br` não existe mais (ou mudou)
2. A URL não incluía o ano-mês na estrutura

---

### Depois (Funcionando)

O script agora testa URLs específicas para cada mês:
```
https://arquivos.receitafederal.gov.br/dados/cnpj/dados_abertos_cnpj/2026-01/Empresas0.zip
https://arquivos.receitafederal.gov.br/dados/cnpj/dados_abertos_cnpj/2025-12/Empresas0.zip
https://arquivos.receitafederal.gov.br/dados/cnpj/dados_abertos_cnpj/2025-11/Empresas0.zip
... etc
```

Isso permite verificar corretamente qual versão está disponível.

---

## ✅ Resultado do Teste

Após a correção, o script detecta corretamente as versões disponíveis:

```bash
$ ./scripts/check_latest.sh

🔍 Verificando última versão disponível do CNPJ...

Verificando disponibilidade dos últimos 7 meses:

  2026-01: ✅ DISPONÍVEL
  2025-12: ✅ DISPONÍVEL
  2025-11: ✅ DISPONÍVEL
  2025-10: ✅ DISPONÍVEL
  2025-09: ✅ DISPONÍVEL
  2025-08: ✅ DISPONÍVEL
  2025-07: ✅ DISPONÍVEL

============================================
📅 Última versão disponível: 2026-01
============================================
```

---

## 🔍 Como Descobrimos o Bug

1. **Usuário reportou:** "Olhando o site tem dados desse mês: https://arquivos.receitafederal.gov.br/dados/cnpj/dados_abertos_cnpj/2026-01/"

2. **Análise do código:** Verificamos `scripts/check_latest.sh` e encontramos a URL errada

3. **Verificação do código Go:** Checamos `cmd/download.go` (linha 22) que tinha a URL correta:
   ```go
   base := fmt.Sprintf("https://arquivos.receitafederal.gov.br/dados/cnpj/dados_abertos_cnpj/%s/", flagYearMonth)
   ```

4. **Correção:** Atualizamos o script bash para usar a mesma URL do código Go

---

## 📝 Arquivos Modificados

### 1. `scripts/check_latest.sh`
- ✅ Corrigido URL base
- ✅ Adicionado ano-mês na URL de teste
- ✅ Adicionado timeout de 10 segundos no curl
- ✅ Atualizado link de verificação manual

### 2. `scripts/download_latest.sh`
- ✅ Adicionado comentário com URL correta no cabeçalho
- ℹ️ Não precisou mudança na lógica (já usa `./bcd download` que tem URL correta)

---

## 🎯 Estrutura Oficial da URL

A Receita Federal organiza os dados da seguinte forma:

```
https://arquivos.receitafederal.gov.br/dados/cnpj/dados_abertos_cnpj/
└── YYYY-MM/
    ├── Empresas0.zip
    ├── Empresas1.zip
    ├── Empresas2.zip
    ├── Estabelecimentos0.zip
    ├── Estabelecimentos1.zip
    ├── Estabelecimentos2.zip
    ├── Estabelecimentos3.zip
    ├── Estabelecimentos4.zip
    ├── Estabelecimentos5.zip
    ├── Estabelecimentos6.zip
    ├── Estabelecimentos7.zip
    ├── Estabelecimentos8.zip
    ├── Estabelecimentos9.zip
    ├── Cnaes.zip
    ├── Municipios.zip
    ├── Naturezas.zip
    ├── Paises.zip
    ├── Qualificacoes.zip
    └── Simples.zip
```

**Exemplo para Janeiro 2026:**
```
https://arquivos.receitafederal.gov.br/dados/cnpj/dados_abertos_cnpj/2026-01/Empresas0.zip
```

---

## 🧪 Como Testar

### Teste 1: Verificar versão disponível
```bash
./scripts/check_latest.sh
```

Deve mostrar `2026-01: ✅ DISPONÍVEL`

### Teste 2: Verificar URL manualmente
```bash
curl -I "https://arquivos.receitafederal.gov.br/dados/cnpj/dados_abertos_cnpj/2026-01/Empresas0.zip"
```

Deve retornar `HTTP/1.1 200 OK`

### Teste 3: Download completo
```bash
./scripts/download_latest.sh
```

Deve baixar automaticamente a versão 2026-01

---

## ✨ Lição Aprendida

**Sempre validar URLs de terceiros com testes reais!**

O código Go em `cmd/download.go` estava correto desde o início, mas os scripts bash tinham URL desatualizada. Ao criar scripts auxiliares, devemos:

1. ✅ Copiar URLs exatas do código principal
2. ✅ Testar com curl/wget antes de implementar
3. ✅ Adicionar comentários com estrutura da URL
4. ✅ Incluir timeout nos requests HTTP

---

## 🚀 Próximos Passos

1. ✅ Scripts corrigidos e testados
2. ✅ README atualizado
3. ⏭️ Usuário pode usar `./scripts/check_latest.sh`
4. ⏭️ Usuário pode usar `./scripts/download_latest.sh`

**Problema resolvido! 🎉**
