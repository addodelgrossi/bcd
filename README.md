# bcd

Brazil Companies Database

CLI em Go para **baixar → extrair → carregar** os dados abertos do CNPJ (RFB) em **SQLite**.

## Instalação

### Download de Binários Pré-Compilados

Baixe a versão mais recente para sua plataforma em [Releases](https://github.com/addodelgrossi/bcd/releases).

Plataformas suportadas:

- **Linux**: AMD64, ARM64
- **macOS**: Intel (AMD64), Apple Silicon (ARM64)
- **Windows**: AMD64, ARM64

> 💡 **Nota de Portabilidade:** O arquivo `.sqlite` gerado é **100% portável** entre todas as plataformas. Você pode gerar o banco em Linux x86_64 e usar em Linux ARM, Windows, macOS, etc. Apenas o executável `bcd` precisa ser específico da plataforma.

### Build a partir do Código

```bash
go mod tidy
go build -o bcd
```

## Uso

### Descobrir a Última Versão Disponível

```bash
# Verificar qual é a última versão publicada pela Receita Federal
./scripts/check_latest.sh
```

### Download e Processamento Manual

```bash
# Verificar versão instalada
./bcd --version

# 1) Download dos ZIPs do mês (ex.: 2025-01)
./bcd download --ym 2025-01 --workdir /tmp/cnpj_rf

# 2) Extrair todos os ZIPs
./bcd extract  --ym 2025-01 --workdir /tmp/cnpj_rf

# 3) Carregar no SQLite (gera ./cnpj.sqlite por padrão)
./bcd load     --ym 2025-01 --workdir /tmp/cnpj_rf --out ./cnpj.sqlite

# 3a) Se encontrar erro de memória, use --skip-vacuum
./bcd load     --ym 2025-01 --workdir /tmp/cnpj_rf --out ./cnpj.sqlite --skip-vacuum

# 3b) Para pular a criação das FTS5 (busca textual), --skip-fts
#     Ganha ~5–10 min + ~2–3 GB; em contrapartida /api/v1/companies?q= fica lento.
./bcd load     --ym 2025-01 --workdir /tmp/cnpj_rf --out ./cnpj.sqlite --skip-fts
```

> `bcd load` já cria as virtual tables `empresas_fts` e `estabelecimentos_fts`
> consumidas pela [API](https://github.com/brazildata/api) em `/api/v1/companies?q=`.
> Tokenizer `unicode61 remove_diacritics 2` (busca sem acento).

### Requisitos de Memória

O processo de load requer memória suficiente, especialmente durante a operação **VACUUM**:

- **Mínimo recomendado:** 8GB de RAM
- **Ideal:** 16GB+ de RAM
- **VACUUM desabilitado:** 4GB de RAM pode ser suficiente

**Se você encontrar erro "out of memory (7)":**

```bash
# Use a flag --skip-vacuum para pular a otimização VACUUM
./bcd load --ym 2025-01 --workdir /tmp/cnpj_rf --skip-vacuum
```

> ⚠️ **Nota:** Pular o VACUUM resultará em um arquivo `.sqlite` maior (~20-30% maior), mas o banco será totalmente funcional.
>
> Você pode executar VACUUM manualmente depois em uma máquina com mais memória:
> ```bash
> sqlite3 cnpj.sqlite "VACUUM;"
> ```

### Download Automatizado (Recomendado)

```bash
# Baixa automaticamente a última versão disponível
./scripts/download_latest.sh

# Com configuração customizada:
WORKDIR=/data/cnpj OUTPUT=/data/cnpj.sqlite ./scripts/download_latest.sh
```

**Nota:** A Receita Federal geralmente publica os dados no início de cada mês com os dados do mês anterior.

## Tabelas criadas

### Tabelas Principais
- `empresas(cnpj_basico, razao_social, natureza_juridica, qualificacao_responsavel, capital_social, porte_empresa, ente_federativo)`
- `estabelecimentos(cnpj_basico, cnpj_ordem, cnpj_dv, identificador_matriz_filial, nome_fantasia, situacao_cadastral, data_situacao_cadastral, motivo_situacao_cadastral, nome_cidade_exterior, pais, data_inicio_atividade, cnae_fiscal_principal, cnae_fiscal_secundaria, tipo_logradouro, logradouro, numero, complemento, bairro, cep, uf, municipio, ddd1, telefone1, ddd2, telefone2, ddd_fax, fax, correio_eletronico, situacao_especial, data_situacao_especial)`
- `socios(cnpj_basico, identificador_socio, nome_socio, cnpj_cpf_socio, qualificacao_socio, data_entrada_sociedade, pais, representante_legal, nome_representante, qualificacao_representante_legal, faixa_etaria)` - Sócios e administradores das empresas
- `simples(cnpj_basico, opcao_simples, data_opcao_simples, data_exclusao_simples, opcao_mei, data_opcao_mei, data_exclusao_mei)`

### Tabelas de Referência (Lookup)
- `cnaes(codigo, descricao)` - Classificação Nacional de Atividades Econômicas
- `municipios(codigo, descricao)` - Códigos de municípios brasileiros
- `paises(codigo, descricao)` - Códigos de países
- `qualificacoes(codigo, descricao)` - Qualificação do responsável pela empresa
- `naturezas_juridicas(codigo, descricao)` - Natureza jurídica da empresa
- `motivos(codigo, descricao)` - Motivos de situação cadastral

## Performance

O banco gerado é **otimizado para APIs de alta performance**:

✅ **7 índices estratégicos** para queries comuns (CNPJ, município, CNAE, CEP, etc.)
✅ **WAL mode** habilitado para leituras concorrentes sem bloqueio
✅ **VACUUM + ANALYZE** executados automaticamente após carga
✅ **64MB de cache** + 256MB memory-mapped I/O

**Latência esperada:**
- Busca por CNPJ: **< 1ms**
- JOINs empresa + estabelecimento: **1-5ms**
- Agregações geográficas: **50-200ms**

📖 Veja [docs/PERFORMANCE.md](docs/PERFORMANCE.md) para guia completo de deployment, benchmarks e troubleshooting.

> Dica: Para máximo desempenho, rode o `load` em SSD e mantenha o `.sqlite` em disco rápido (NVMe) ou dentro do container da API.

## Consultas úteis

Veja [examples/queries.sql](examples/queries.sql) para 15+ exemplos de queries otimizadas:
- Busca por CNPJ completo
- Empresas ativas por município
- Empresas por CNAE (setor)
- Top cidades com mais empresas
- Empresas por porte (ME, EPP, grande)
- Busca por CEP
- Contagem de filiais
- E muito mais...

## Observações

- O parser assume **CSV com `;`** e **Latin-1 (ISO-8859-1)**, conforme a RFB. Tudo é convertido para **UTF-8** antes de inserir.
- A etapa `extract` grava todos os CSVs em `workdir/extracted`. O `load` varre esse diretório e carrega os arquivos que começam por `Empresas*`, `Estabelecimentos*`, `Cnaes*`, `Municipios*` (case-insensitive).
- Driver **pure Go**: `modernc.org/sqlite` (dispensa CGO).

## 📚 Documentação

### Guias Técnicos
Veja a pasta **[docs/](docs/)** para documentação completa:
- [docs/PERFORMANCE.md](docs/PERFORMANCE.md) - Guia de performance e deployment
- [docs/REVIEW_SUMMARY.md](docs/REVIEW_SUMMARY.md) - Resumo das otimizações implementadas
- [docs/OPTIMIZATION_STRATEGY.md](docs/OPTIMIZATION_STRATEGY.md) - Estratégias de otimização
- [docs/STRATEGY_COMPARISON.md](docs/STRATEGY_COMPARISON.md) - Comparação visual de abordagens
- [docs/BUGFIX_URL.md](docs/BUGFIX_URL.md) - Histórico de correções

### Exemplos de Código
- [examples/queries.sql](examples/queries.sql) - 15+ queries SQL otimizadas
- [examples/benchmark.sh](examples/benchmark.sh) - Script de load testing

> Para a API HTTP em produção, veja o repo [brazildata/api](https://github.com/brazildata/api).
> Este projeto (`bcd`) existe apenas para gerar o `cnpj.sqlite` que aquela API consome.

### Deployment
- [Dockerfile.example](Dockerfile.example) - Dockerfile otimizado
- [docker-compose.example.yml](docker-compose.example.yml) - Setup completo com monitoramento

## Desenvolvimento

Para contribuir ou fazer releases, consulte:
- [RELEASE.md](RELEASE.md) - Guia completo de como criar releases

### Releases Automatizados

O projeto usa GitHub Actions para compilar binários automaticamente para todas as plataformas suportadas quando uma tag é criada:

```bash
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0
```

Os binários são automaticamente:

- Compilados em paralelo para Linux, macOS e Windows (AMD64 e ARM64)
- Empacotados com checksums SHA256
- Anexados ao GitHub Release
