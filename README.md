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

### Build a partir do Código

```bash
go mod tidy
go build -o bcd
```

## Uso

```bash
# Verificar versão instalada
./bcd --version

# 1) Download dos ZIPs do mês (ex.: 2025-10)
./bcd download --ym 2025-10 --workdir /tmp/cnpj_rf

# 2) Extrair todos os ZIPs
./bcd extract  --ym 2025-10 --workdir /tmp/cnpj_rf

# 3) Carregar no SQLite (gera ./cnpj.sqlite por padrão)
./bcd load     --ym 2025-10 --workdir /tmp/cnpj_rf --out ./cnpj.sqlite
```

## Tabelas criadas

- `empresas(cnpj_basico, razao_social, natureza_juridica, qualificacao_responsavel, capital_social, porte_empresa, ente_federativo)`
- `estabelecimentos(cnpj_basico, cnpj_ordem, cnpj_dv, identificador_matriz_filial, nome_fantasia, situacao_cadastral, data_situacao_cadastral, motivo_situacao_cadastral, nome_cidade_exterior, pais, data_inicio_atividade, cnae_fiscal_principal, cnae_fiscal_secundaria, tipo_logradouro, logradouro, numero, complemento, bairro, cep, uf, municipio, ddd1, telefone1, ddd2, telefone2, ddd_fax, fax, correio_eletronico, situacao_especial, data_situacao_especial)`
- `cnaes(codigo, descricao)`
- `municipios(codigo, descricao)`

> Dica de performance: o loader usa PRAGMAs para bulk load e transações; em máquinas lentas, rode em SSD e evite antivírus varrendo a pasta.

## Consultas úteis

- Empresas grandes por cidade, CNAE, etc. (adapte das queries do BigQuery: os nomes de colunas são equivalentes).

## Observações

- O parser assume **CSV com `;`** e **Latin-1 (ISO-8859-1)**, conforme a RFB. Tudo é convertido para **UTF-8** antes de inserir.
- A etapa `extract` grava todos os CSVs em `workdir/extracted`. O `load` varre esse diretório e carrega os arquivos que começam por `Empresas*`, `Estabelecimentos*`, `Cnaes*`, `Municipios*` (case-insensitive).
- Driver **pure Go**: `modernc.org/sqlite` (dispensa CGO).

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
