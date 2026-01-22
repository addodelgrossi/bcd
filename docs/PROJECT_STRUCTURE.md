# Estrutura do Projeto BCD

## 📁 Organização de Pastas

```
bcd/
│
├── 📄 README.md                 # README principal do projeto
├── 📄 RELEASE.md                # Guia de como criar releases
├── 📄 go.mod / go.sum           # Dependências Go
├── 📄 main.go                   # Entry point da aplicação
│
├── 📁 cmd/                      # Comandos CLI (Cobra)
│   ├── root.go                  # Configuração raiz do CLI
│   ├── download.go              # Comando: baixar dados da Receita
│   ├── extract.go               # Comando: extrair ZIPs
│   └── load.go                  # Comando: carregar no SQLite
│
├── 📁 docs/                     # 📚 Documentação técnica
│   ├── README.md                # Índice da documentação
│   ├── PERFORMANCE.md           # Guia de performance e deployment
│   ├── OPTIMIZATION_STRATEGY.md # Análise de estratégias
│   ├── REVIEW_SUMMARY.md        # Resumo das otimizações
│   ├── STRATEGY_COMPARISON.md   # Comparação visual
│   ├── BUGFIX_URL.md            # Histórico de bugs
│   └── PROJECT_STRUCTURE.md     # Este arquivo
│
├── 📁 examples/                 # 💻 Exemplos de código
│   ├── README.md                # Documentação dos exemplos
│   ├── api_example.go           # API Golang completa
│   ├── queries.sql              # 15+ queries otimizadas
│   └── benchmark.sh             # Script de load testing
│
├── 📁 scripts/                  # 🔧 Scripts utilitários
│   ├── README.md                # Documentação dos scripts
│   ├── check_latest.sh          # Verifica última versão
│   └── download_latest.sh       # Download automatizado
│
├── 📄 Dockerfile.example        # Exemplo de Dockerfile
└── 📄 docker-compose.example.yml # Setup Docker completo

```

---

## 📂 Descrição das Pastas

### `/` (Raiz)
Arquivos essenciais do projeto:
- `README.md` - Primeira coisa que visitantes veem
- `RELEASE.md` - Para mantenedores criarem releases
- `main.go` - Entry point do código Go

**Mantidos na raiz:** Apenas arquivos essenciais e de configuração.

---

### `/cmd/`
Código Go dos comandos CLI (usando Cobra).

**Arquivos:**
- `root.go` - Setup do CLI, flags globais
- `download.go` - Lógica de download dos ZIPs
- `extract.go` - Lógica de extração
- `load.go` - Lógica de carga no SQLite

**Convenção:** Um arquivo por comando.

---

### `/docs/` ✨ NOVO
Documentação técnica e guias.

**Conteúdo:**
- Guias de performance
- Análises de decisões técnicas
- Comparações de abordagens
- Histórico de bugs e correções

**Quem lê:** Desenvolvedores, DevOps, arquitetos.

**Quando atualizar:** Ao fazer mudanças arquiteturais ou otimizações.

---

### `/examples/`
Exemplos práticos de código.

**Conteúdo:**
- API Golang completa e funcional
- Queries SQL otimizadas
- Scripts de benchmark

**Quem usa:** Desenvolvedores implementando APIs.

**Quando atualizar:** Ao adicionar novos casos de uso.

---

### `/scripts/`
Scripts bash utilitários.

**Conteúdo:**
- Automação de download
- Verificação de versões
- Helpers para CI/CD

**Quem usa:** DevOps, usuários finais.

**Quando atualizar:** Ao adicionar novas automações.

---

## 🎯 Fluxo de Navegação

### Novo Usuário
```
README.md
    ↓
scripts/check_latest.sh
    ↓
scripts/download_latest.sh
    ↓
docs/PERFORMANCE.md (para deploy)
```

### Desenvolvedor de API
```
README.md
    ↓
examples/api_example.go
    ↓
examples/queries.sql
    ↓
examples/benchmark.sh
```

### Arquiteto/DevOps
```
README.md
    ↓
docs/REVIEW_SUMMARY.md
    ↓
docs/PERFORMANCE.md
    ↓
docs/OPTIMIZATION_STRATEGY.md
```

---

## 📝 Convenções de Nomenclatura

### Arquivos Markdown
- **UPPERCASE.md** - Documentos importantes (README, RELEASE)
- **NOMEDESCRITIVO.md** - Documentos técnicos (sem prefixo)

### Scripts
- **snake_case.sh** - Scripts bash
- **sem extensão** - Binários Go compilados

### Código
- **snake_case.go** - Arquivos Go
- **camelCase** - Funções e variáveis Go

---

## 🔗 Links Entre Documentos

### Do README principal
```markdown
- [docs/PERFORMANCE.md](docs/PERFORMANCE.md)
- [examples/api_example.go](examples/api_example.go)
```

### De dentro de /docs/
```markdown
- [PERFORMANCE.md](PERFORMANCE.md)          # Mesmo diretório
- [../README.md](../README.md)              # Raiz
- [../examples/queries.sql](../examples/queries.sql)  # Outro diretório
```

---

## 📊 Estatísticas

```
Arquivos Markdown na raiz:
  Antes:  7 arquivos (.md poluindo)
  Depois: 2 arquivos (apenas essenciais)

Organização:
  ✅ docs/      - 6 arquivos
  ✅ examples/  - 4 arquivos
  ✅ scripts/   - 3 arquivos
```

---

## ✨ Benefícios da Reorganização

### ✅ Clareza
- Raiz do projeto limpa e profissional
- Fácil encontrar o que procura

### ✅ Escalabilidade
- Fácil adicionar nova documentação
- Estrutura cresce de forma organizada

### ✅ Manutenibilidade
- Documentação agrupada logicamente
- Menos chance de arquivos órfãos

### ✅ Primeira Impressão
- README fica mais focado
- Estrutura profissional no GitHub

---

## 🚀 Como Adicionar Novos Arquivos

### Nova documentação técnica?
→ Adicione em `/docs/` e atualize `/docs/README.md`

### Novo exemplo de código?
→ Adicione em `/examples/` e atualize `/examples/README.md`

### Novo script utilitário?
→ Adicione em `/scripts/` e atualize `/scripts/README.md`

### Novo comando CLI?
→ Adicione em `/cmd/` seguindo padrão Cobra

---

**Última atualização:** Janeiro 2026
