-- =============================================================================
-- BCD - Brazil Companies Database
-- Schema de referencia do banco de dados SQLite
-- Fonte: Dados abertos da Receita Federal do Brasil (RFB)
-- =============================================================================

PRAGMA foreign_keys=OFF;

-- =============================================================================
-- TABELAS PRINCIPAIS
-- =============================================================================

-- Empresas (dados cadastrais basicos)
CREATE TABLE IF NOT EXISTS empresas (
    cnpj_basico TEXT PRIMARY KEY NOT NULL,   -- CNPJ basico (8 digitos)
    razao_social TEXT NOT NULL,              -- Razao social
    natureza_juridica INTEGER,               -- Codigo natureza juridica (ver naturezas_juridicas)
    qualificacao_responsavel INTEGER,         -- Codigo qualificacao (ver qualificacoes)
    capital_social REAL DEFAULT 0.0,         -- Capital social
    porte_empresa INTEGER,                   -- 01=Micro, 03=Pequeno, 05=Medio/Grande
    ente_federativo TEXT                     -- Ente federativo (apenas tipo 1XXX)
);

-- Estabelecimentos (matrizes e filiais)
CREATE TABLE IF NOT EXISTS estabelecimentos (
    cnpj_basico TEXT NOT NULL,               -- CNPJ basico (referencia empresas)
    cnpj_ordem TEXT NOT NULL,                -- Numero de ordem
    cnpj_dv TEXT NOT NULL,                   -- Digito verificador
    identificador_matriz_filial INTEGER,     -- 1=Matriz, 2=Filial
    nome_fantasia TEXT,                      -- Nome fantasia
    situacao_cadastral INTEGER,              -- 02=Ativa, 03=Suspensa, 04=Inapta, 08=Baixada
    data_situacao_cadastral TEXT,            -- Data da situacao (AAAAMMDD)
    motivo_situacao_cadastral INTEGER,       -- Codigo do motivo (ver motivos)
    nome_cidade_exterior TEXT,               -- Cidade no exterior (se aplicavel)
    pais INTEGER,                            -- Codigo do pais (ver paises)
    data_inicio_atividade TEXT,              -- Data inicio atividade (AAAAMMDD)
    cnae_fiscal_principal INTEGER,           -- CNAE principal (ver cnaes)
    cnae_fiscal_secundaria TEXT,             -- CNAEs secundarios (separados por virgula)
    tipo_logradouro TEXT,                    -- Tipo logradouro (Rua, Av, etc)
    logradouro TEXT,                         -- Logradouro
    numero TEXT,                             -- Numero
    complemento TEXT,                        -- Complemento
    bairro TEXT,                             -- Bairro
    cep TEXT,                                -- CEP
    uf TEXT,                                 -- Unidade federativa (2 caracteres)
    municipio INTEGER,                       -- Codigo do municipio (ver municipios)
    ddd1 TEXT, telefone1 TEXT,               -- Telefone 1
    ddd2 TEXT, telefone2 TEXT,               -- Telefone 2
    ddd_fax TEXT, fax TEXT,                  -- Fax
    correio_eletronico TEXT,                 -- Email
    situacao_especial TEXT,                  -- Situacao especial
    data_situacao_especial TEXT,             -- Data da situacao especial (AAAAMMDD)
    PRIMARY KEY (cnpj_basico, cnpj_ordem, cnpj_dv)
);

-- Socios (quadro societario)
CREATE TABLE IF NOT EXISTS socios (
    cnpj_basico TEXT NOT NULL,               -- CNPJ basico (referencia empresas)
    identificador_socio INTEGER,             -- 1=PJ, 2=PF, 3=Estrangeiro
    nome_socio TEXT,                         -- Nome do socio
    cnpj_cpf_socio TEXT,                     -- CPF/CNPJ do socio (ofuscado)
    qualificacao_socio INTEGER,              -- Codigo qualificacao (ver qualificacoes)
    data_entrada_sociedade TEXT,             -- Data de entrada (AAAAMMDD)
    pais INTEGER,                            -- Codigo do pais (ver paises)
    representante_legal TEXT,                -- CPF do representante (ofuscado)
    nome_representante TEXT,                 -- Nome do representante legal
    qualificacao_representante_legal INTEGER, -- Qualificacao do representante
    faixa_etaria INTEGER                    -- Faixa etaria (1=0-12, 2=13-20, ..., 9=>80)
);

-- Simples Nacional / MEI
CREATE TABLE IF NOT EXISTS simples (
    cnpj_basico TEXT PRIMARY KEY NOT NULL,   -- CNPJ basico (referencia empresas)
    opcao_simples TEXT,                      -- Opcao Simples Nacional (S/N)
    data_opcao_simples TEXT,                 -- Data de opcao (AAAAMMDD)
    data_exclusao_simples TEXT,              -- Data de exclusao (AAAAMMDD)
    opcao_mei TEXT,                          -- Opcao MEI (S/N)
    data_opcao_mei TEXT,                     -- Data de opcao MEI (AAAAMMDD)
    data_exclusao_mei TEXT                   -- Data de exclusao MEI (AAAAMMDD)
);

-- =============================================================================
-- TABELAS DE REFERENCIA (LOOKUP)
-- =============================================================================

-- Classificacao Nacional de Atividades Economicas
CREATE TABLE IF NOT EXISTS cnaes (
    codigo TEXT PRIMARY KEY,
    descricao TEXT
);

-- Municipios
CREATE TABLE IF NOT EXISTS municipios (
    codigo TEXT PRIMARY KEY,
    descricao TEXT
);

-- Paises
CREATE TABLE IF NOT EXISTS paises (
    codigo TEXT PRIMARY KEY,
    descricao TEXT
);

-- Qualificacoes de socios
CREATE TABLE IF NOT EXISTS qualificacoes (
    codigo TEXT PRIMARY KEY,
    descricao TEXT
);

-- Naturezas juridicas
CREATE TABLE IF NOT EXISTS naturezas_juridicas (
    codigo TEXT PRIMARY KEY,
    descricao TEXT
);

-- Motivos de situacao cadastral
CREATE TABLE IF NOT EXISTS motivos (
    codigo TEXT PRIMARY KEY,
    descricao TEXT
);

-- =============================================================================
-- INDICES
-- =============================================================================

-- Indices simples - lookups pontuais e filtros isolados
CREATE INDEX IF NOT EXISTS idx_estab_cnpj ON estabelecimentos(cnpj_basico);
CREATE INDEX IF NOT EXISTS idx_estab_mun_uf ON estabelecimentos(municipio, uf);
CREATE INDEX IF NOT EXISTS idx_estab_uf ON estabelecimentos(uf);
CREATE INDEX IF NOT EXISTS idx_estab_cnae ON estabelecimentos(cnae_fiscal_principal);
CREATE INDEX IF NOT EXISTS idx_estab_situacao ON estabelecimentos(situacao_cadastral);
CREATE INDEX IF NOT EXISTS idx_estab_cep ON estabelecimentos(cep);
CREATE INDEX IF NOT EXISTS idx_estab_matriz_filial ON estabelecimentos(identificador_matriz_filial);

-- Covering indexes para a listagem paginada da API (/api/v1/companies).
-- A API filtra por uma coluna e ordena pelo PK composto; sem combinar
-- filtro + PK o SQLite cai num sort em temp B-tree que estoura o timeout
-- HTTP em filtros amplos (uf=SP -> ~20M linhas).
CREATE INDEX IF NOT EXISTS idx_estab_uf_pk       ON estabelecimentos(uf, cnpj_basico, cnpj_ordem, cnpj_dv);
CREATE INDEX IF NOT EXISTS idx_estab_mun_pk      ON estabelecimentos(municipio, cnpj_basico, cnpj_ordem, cnpj_dv);
CREATE INDEX IF NOT EXISTS idx_estab_cnae_pk     ON estabelecimentos(cnae_fiscal_principal, cnpj_basico, cnpj_ordem, cnpj_dv);
CREATE INDEX IF NOT EXISTS idx_estab_situacao_pk ON estabelecimentos(situacao_cadastral, cnpj_basico, cnpj_ordem, cnpj_dv);

-- Indices para socios (consultas por empresa e por CPF)
CREATE INDEX IF NOT EXISTS idx_socios_cnpj ON socios(cnpj_basico);
CREATE INDEX IF NOT EXISTS idx_socios_cpf ON socios(cnpj_cpf_socio);

-- =============================================================================
-- FTS5 (virtual tables) — criadas automaticamente pelo `bcd load`.
-- Nao use este arquivo para recria-las manualmente; rode `bcd load` (ou o
-- script scripts/setup_fts.sql do repo brazildata/api em bancos legados).
--
-- Tokenizer unicode61 + remove_diacritics faz "sao paulo" casar com "Sao Paulo".
-- =============================================================================
-- CREATE VIRTUAL TABLE empresas_fts USING fts5(
--     cnpj_basico UNINDEXED,
--     razao_social,
--     tokenize = 'unicode61 remove_diacritics 2'
-- );
-- CREATE VIRTUAL TABLE estabelecimentos_fts USING fts5(
--     cnpj_basico UNINDEXED,
--     cnpj_ordem UNINDEXED,
--     cnpj_dv UNINDEXED,
--     nome_fantasia,
--     tokenize = 'unicode61 remove_diacritics 2'
-- );
