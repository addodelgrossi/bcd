# Documentação do BCD (Brazil Companies Database)

Esta pasta contém documentação técnica e guias do projeto.

## 📚 Guias Principais

### [PERFORMANCE.md](PERFORMANCE.md)
Guia completo de performance e deployment para API de alta performance.

**Conteúdo:**
- Otimizações implementadas (índices, PRAGMAs, VACUUM)
- Estratégias de deployment (container, volume, SSD)
- Benchmarks esperados e métricas
- Configuração da API Golang
- Troubleshooting

**Quando usar:** Antes de colocar em produção ou ao debugar problemas de performance.

---

### [OPTIMIZATION_STRATEGY.md](OPTIMIZATION_STRATEGY.md)
Análise detalhada de estratégias de otimização: load completo vs otimização em runtime.

**Conteúdo:**
- Comparação de abordagens (tudo de uma vez vs load rápido)
- Timeline estimado de cada processo
- Quando usar cada estratégia
- Otimizações já implementadas

**Quando usar:** Para entender as decisões de arquitetura do projeto.

---

### [REVIEW_SUMMARY.md](REVIEW_SUMMARY.md)
Resumo executivo das otimizações implementadas no projeto.

**Conteúdo:**
- Problemas identificados e soluções
- Performance esperada (antes vs depois)
- Arquivos criados (exemplos, scripts)
- Recomendações de deployment
- Checklist de validação

**Quando usar:** Para overview rápido de todas as melhorias.

---

### [STRATEGY_COMPARISON.md](STRATEGY_COMPARISON.md)
Comparação visual entre estratégias de otimização com diagramas e tabelas.

**Conteúdo:**
- Fluxogramas dos processos
- Tabelas comparativas
- Análise de custo/benefício
- Cenários de problemas
- Decisão final justificada

**Quando usar:** Para decisões técnicas sobre estratégia de build/deploy.

---

### [BUGFIX_URL.md](BUGFIX_URL.md)
Documentação de correção de bug na URL da Receita Federal.

**Conteúdo:**
- Bug identificado (URL incorreta)
- Correção aplicada
- Estrutura oficial da URL
- Testes de validação

**Quando usar:** Referência histórica de bugs corrigidos.

---

### [PROJECT_STRUCTURE.md](PROJECT_STRUCTURE.md)
Documentação da estrutura de pastas e arquivos do projeto.

**Conteúdo:**
- Árvore de diretórios completa
- Descrição de cada pasta
- Convenções de nomenclatura
- Fluxos de navegação
- Como adicionar novos arquivos

**Quando usar:** Para entender a organização do projeto ou decidir onde adicionar novos arquivos.

---

## 🎯 Guia Rápido: Qual Documento Ler?

### Você quer...

**...colocar em produção?**
→ Leia [PERFORMANCE.md](PERFORMANCE.md)

**...entender por que foi otimizado assim?**
→ Leia [OPTIMIZATION_STRATEGY.md](OPTIMIZATION_STRATEGY.md)

**...ver um resumo de tudo que foi feito?**
→ Leia [REVIEW_SUMMARY.md](REVIEW_SUMMARY.md)

**...decidir entre estratégias de deployment?**
→ Leia [STRATEGY_COMPARISON.md](STRATEGY_COMPARISON.md)

**...debugar problema com URLs da Receita?**
→ Leia [BUGFIX_URL.md](BUGFIX_URL.md)

---

## 📖 Outros Recursos

- **[../README.md](../README.md)** - README principal do projeto
- **[../RELEASE.md](../RELEASE.md)** - Guia de como criar releases
- **[../examples/](../examples/)** - Exemplos de código (API, queries SQL, benchmarks)
- **[../scripts/](../scripts/)** - Scripts utilitários (download, check version)

---

## 🚀 Quick Start

Se você é novo no projeto, recomendo ler nesta ordem:

1. **[../README.md](../README.md)** - Entenda o que é o projeto
2. **[REVIEW_SUMMARY.md](REVIEW_SUMMARY.md)** - Veja todas as otimizações
3. **[PERFORMANCE.md](PERFORMANCE.md)** - Aprenda a fazer deploy

Tempo estimado de leitura: **15-20 minutos** para os 3 documentos.

---

## 📝 Contribuindo

Ao adicionar nova documentação:

1. ✅ Use títulos claros e descritivos
2. ✅ Inclua exemplos de código quando relevante
3. ✅ Adicione referências cruzadas entre documentos
4. ✅ Atualize este README.md com link para o novo doc
5. ✅ Use emoji para facilitar scanning visual (opcional)

---

## 📊 Estrutura do Projeto

```
bcd/
├── README.md              # README principal
├── RELEASE.md             # Guia de releases
├── docs/                  # 📁 Você está aqui!
│   ├── README.md          # Este arquivo
│   ├── PERFORMANCE.md     # Guia de performance
│   ├── OPTIMIZATION_STRATEGY.md
│   ├── REVIEW_SUMMARY.md
│   ├── STRATEGY_COMPARISON.md
│   └── BUGFIX_URL.md
├── examples/              # Exemplos de código
│   ├── README.md
│   ├── api_example.go
│   ├── benchmark.sh
│   └── queries.sql
├── scripts/               # Scripts utilitários
│   ├── README.md
│   ├── check_latest.sh
│   └── download_latest.sh
├── cmd/                   # Código Go (comandos)
│   ├── root.go
│   ├── download.go
│   ├── extract.go
│   └── load.go
└── main.go                # Entry point
```

---

## 🔗 Links Úteis

- **Portal da Receita Federal:** https://arquivos.receitafederal.gov.br/dados/cnpj/dados_abertos_cnpj/
- **Dados Abertos Brasil:** https://dados.gov.br/dados/conjuntos-dados/cadastro-nacional-da-pessoa-juridica---cnpj
- **GitHub Issues:** https://github.com/addodelgrossi/bcd/issues
- **Releases:** https://github.com/addodelgrossi/bcd/releases

---

**Última atualização:** Janeiro 2026
