-- ============================================================================
-- QUERIES SQL OTIMIZADAS - CNPJ Database
-- ============================================================================
-- Este arquivo contém exemplos de queries otimizadas para o banco de dados
-- de CNPJs da Receita Federal do Brasil
-- ============================================================================

-- ----------------------------------------------------------------------------
-- 1. BUSCA POR CNPJ COMPLETO (com razão social)
-- ----------------------------------------------------------------------------
-- Uso: Buscar dados completos de uma empresa
-- Performance: < 1ms (usa PRIMARY KEY)
-- ----------------------------------------------------------------------------
SELECT
    e.cnpj_basico,
    e.razao_social,
    e.natureza_juridica,
    e.capital_social,
    e.porte_empresa,
    est.cnpj_basico || est.cnpj_ordem || est.cnpj_dv as cnpj_completo,
    est.nome_fantasia,
    est.situacao_cadastral,
    est.cnae_fiscal_principal,
    est.logradouro || ', ' || est.numero || ' - ' || est.bairro as endereco,
    est.cep,
    est.municipio,
    est.uf,
    est.telefone1,
    est.correio_eletronico
FROM empresas e
JOIN estabelecimentos est ON e.cnpj_basico = est.cnpj_basico
WHERE e.cnpj_basico = '12345678'
    AND est.cnpj_ordem = '0001'
    AND est.cnpj_dv = '95';

-- ----------------------------------------------------------------------------
-- 2. BUSCAR APENAS MATRIZ (não filiais)
-- ----------------------------------------------------------------------------
-- Uso: Lista somente estabelecimentos matriz
-- Performance: 1-5ms (usa idx_estab_cnpj + idx_estab_matriz_filial)
-- ----------------------------------------------------------------------------
SELECT
    e.cnpj_basico,
    e.razao_social,
    est.nome_fantasia,
    est.situacao_cadastral,
    est.uf,
    est.municipio
FROM empresas e
JOIN estabelecimentos est ON e.cnpj_basico = est.cnpj_basico
WHERE est.identificador_matriz_filial = '1'  -- 1 = Matriz
    AND est.situacao_cadastral = '2'         -- 2 = Ativa
LIMIT 100;

-- ----------------------------------------------------------------------------
-- 3. EMPRESAS ATIVAS POR MUNICÍPIO
-- ----------------------------------------------------------------------------
-- Uso: Listar todas empresas ativas de uma cidade
-- Performance: 10-50ms (usa idx_estab_mun_uf)
-- ----------------------------------------------------------------------------
SELECT
    est.cnpj_basico || est.cnpj_ordem || est.cnpj_dv as cnpj,
    e.razao_social,
    est.nome_fantasia,
    est.cnae_fiscal_principal,
    est.logradouro,
    est.bairro,
    est.telefone1
FROM estabelecimentos est
JOIN empresas e ON est.cnpj_basico = e.cnpj_basico
WHERE est.uf = 'SP'
    AND est.municipio = '7107'  -- Código do município (ex: São Paulo)
    AND est.situacao_cadastral = '2'  -- Ativa
ORDER BY e.razao_social
LIMIT 100;

-- ----------------------------------------------------------------------------
-- 4. EMPRESAS POR CNAE (Atividade Econômica)
-- ----------------------------------------------------------------------------
-- Uso: Encontrar empresas de um setor específico
-- Performance: 20-100ms (usa idx_estab_cnae)
-- ----------------------------------------------------------------------------
SELECT
    est.cnpj_basico || est.cnpj_ordem || est.cnpj_dv as cnpj,
    e.razao_social,
    est.nome_fantasia,
    c.descricao as cnae_descricao,
    est.uf,
    est.municipio
FROM estabelecimentos est
JOIN empresas e ON est.cnpj_basico = e.cnpj_basico
LEFT JOIN cnaes c ON est.cnae_fiscal_principal = c.codigo
WHERE est.cnae_fiscal_principal = '6201501'  -- Desenvolvimento de software
    AND est.situacao_cadastral = '2'
    AND est.uf = 'SP'
LIMIT 100;

-- ----------------------------------------------------------------------------
-- 5. AGREGAÇÃO: TOP 10 CIDADES COM MAIS EMPRESAS ATIVAS
-- ----------------------------------------------------------------------------
-- Uso: Ranking de cidades por quantidade de empresas
-- Performance: 100-500ms (full table scan com agregação)
-- ----------------------------------------------------------------------------
SELECT
    est.uf,
    m.descricao as cidade,
    COUNT(*) as total_empresas,
    COUNT(DISTINCT est.cnpj_basico) as total_cnpj_basicos
FROM estabelecimentos est
LEFT JOIN municipios m ON est.municipio = m.codigo
WHERE est.situacao_cadastral = '2'  -- Apenas ativas
    AND est.uf = 'SP'
GROUP BY est.uf, est.municipio
ORDER BY total_empresas DESC
LIMIT 10;

-- ----------------------------------------------------------------------------
-- 6. EMPRESAS POR PORTE
-- ----------------------------------------------------------------------------
-- Uso: Listar empresas por tamanho (ME, EPP, etc)
-- Performance: 10-50ms (depende dos filtros)
-- ----------------------------------------------------------------------------
-- Códigos de porte:
-- 01 = Micro Empresa
-- 03 = Empresa de Pequeno Porte
-- 05 = Demais (médio/grande porte)

SELECT
    e.cnpj_basico,
    e.razao_social,
    e.porte_empresa,
    e.capital_social,
    est.nome_fantasia,
    est.uf,
    est.municipio
FROM empresas e
JOIN estabelecimentos est ON e.cnpj_basico = est.cnpj_basico
WHERE e.porte_empresa = '05'  -- Médio/Grande porte
    AND est.identificador_matriz_filial = '1'
    AND est.situacao_cadastral = '2'
    AND est.uf = 'SP'
LIMIT 100;

-- ----------------------------------------------------------------------------
-- 7. BUSCA POR CEP (Range)
-- ----------------------------------------------------------------------------
-- Uso: Empresas em uma região por CEP
-- Performance: 20-100ms (usa idx_estab_cep)
-- ----------------------------------------------------------------------------
SELECT
    est.cnpj_basico || est.cnpj_ordem || est.cnpj_dv as cnpj,
    e.razao_social,
    est.nome_fantasia,
    est.logradouro || ', ' || est.numero as endereco,
    est.cep,
    est.bairro
FROM estabelecimentos est
JOIN empresas e ON est.cnpj_basico = e.cnpj_basico
WHERE est.cep BETWEEN '01000000' AND '01999999'  -- Região Centro SP
    AND est.situacao_cadastral = '2'
ORDER BY est.cep
LIMIT 100;

-- ----------------------------------------------------------------------------
-- 8. CONTAR FILIAIS DE UMA EMPRESA
-- ----------------------------------------------------------------------------
-- Uso: Ver quantas filiais uma empresa possui
-- Performance: < 5ms (usa idx_estab_cnpj)
-- ----------------------------------------------------------------------------
SELECT
    e.razao_social,
    COUNT(*) as total_estabelecimentos,
    SUM(CASE WHEN est.identificador_matriz_filial = '1' THEN 1 ELSE 0 END) as matrizes,
    SUM(CASE WHEN est.identificador_matriz_filial = '2' THEN 1 ELSE 0 END) as filiais,
    SUM(CASE WHEN est.situacao_cadastral = '2' THEN 1 ELSE 0 END) as ativos
FROM empresas e
JOIN estabelecimentos est ON e.cnpj_basico = est.cnpj_basico
WHERE e.cnpj_basico = '12345678'
GROUP BY e.cnpj_basico, e.razao_social;

-- ----------------------------------------------------------------------------
-- 9. BUSCA POR RAZÃO SOCIAL (LIKE)
-- ----------------------------------------------------------------------------
-- Uso: Busca textual por nome de empresa
-- Performance: LENTA (100-1000ms) - não há índice FTS
-- Recomendação: Use cache ou adicione índice FTS (Full Text Search)
-- ----------------------------------------------------------------------------
SELECT
    e.cnpj_basico,
    e.razao_social,
    est.nome_fantasia,
    est.uf,
    est.municipio,
    est.situacao_cadastral
FROM empresas e
JOIN estabelecimentos est ON e.cnpj_basico = est.cnpj_basico
WHERE e.razao_social LIKE '%TECH%'
    AND est.identificador_matriz_filial = '1'
LIMIT 50;

-- ⚠️ Para otimizar buscas textuais, considere:
-- CREATE VIRTUAL TABLE empresas_fts USING fts5(cnpj_basico, razao_social);

-- ----------------------------------------------------------------------------
-- 10. EMPRESAS INATIVAS (para análise de churn)
-- ----------------------------------------------------------------------------
-- Uso: Listar empresas que encerraram atividades
-- Performance: 50-200ms (usa idx_estab_situacao)
-- ----------------------------------------------------------------------------
SELECT
    est.cnpj_basico || est.cnpj_ordem || est.cnpj_dv as cnpj,
    e.razao_social,
    est.situacao_cadastral,
    est.data_situacao_cadastral,
    est.motivo_situacao_cadastral,
    est.uf,
    est.municipio
FROM estabelecimentos est
JOIN empresas e ON est.cnpj_basico = e.cnpj_basico
WHERE est.situacao_cadastral != '2'  -- Não ativas
    AND est.uf = 'SP'
    AND est.identificador_matriz_filial = '1'
ORDER BY est.data_situacao_cadastral DESC
LIMIT 100;

-- ----------------------------------------------------------------------------
-- 11. ESTATÍSTICAS GERAIS DO BANCO
-- ----------------------------------------------------------------------------
-- Uso: Overview do banco de dados
-- Performance: 1-5s (full table scan)
-- ----------------------------------------------------------------------------
SELECT
    'Empresas' as tabela,
    COUNT(*) as total_registros
FROM empresas

UNION ALL

SELECT
    'Estabelecimentos (Total)' as tabela,
    COUNT(*) as total_registros
FROM estabelecimentos

UNION ALL

SELECT
    'Estabelecimentos (Ativos)' as tabela,
    COUNT(*) as total_registros
FROM estabelecimentos
WHERE situacao_cadastral = '2'

UNION ALL

SELECT
    'CNAEs' as tabela,
    COUNT(*) as total_registros
FROM cnaes

UNION ALL

SELECT
    'Municípios' as tabela,
    COUNT(*) as total_registros
FROM municipios;

-- ----------------------------------------------------------------------------
-- 12. TOP CNAEs MAIS COMUNS
-- ----------------------------------------------------------------------------
-- Uso: Descobrir os setores mais populares
-- Performance: 500ms-2s (full table scan + agregação)
-- ----------------------------------------------------------------------------
SELECT
    est.cnae_fiscal_principal,
    c.descricao,
    COUNT(*) as total_empresas
FROM estabelecimentos est
LEFT JOIN cnaes c ON est.cnae_fiscal_principal = c.codigo
WHERE est.situacao_cadastral = '2'
    AND est.cnae_fiscal_principal IS NOT NULL
    AND est.cnae_fiscal_principal != ''
GROUP BY est.cnae_fiscal_principal, c.descricao
ORDER BY total_empresas DESC
LIMIT 20;

-- ----------------------------------------------------------------------------
-- 13. VERIFICAR QUERY PLAN (para debugging de performance)
-- ----------------------------------------------------------------------------
-- Uso: Ver se a query está usando índices
-- Performance: N/A (análise apenas)
-- ----------------------------------------------------------------------------
EXPLAIN QUERY PLAN
SELECT *
FROM estabelecimentos
WHERE uf = 'SP'
    AND municipio = '7107'
    AND situacao_cadastral = '2';

-- Resultado esperado: "SEARCH estabelecimentos USING INDEX idx_estab_mun_uf"

-- ----------------------------------------------------------------------------
-- 14. EMPRESAS COM MÚLTIPLAS FILIAIS EM DIFERENTES ESTADOS
-- ----------------------------------------------------------------------------
-- Uso: Identificar empresas com presença nacional
-- Performance: 200ms-1s (agregação complexa)
-- ----------------------------------------------------------------------------
SELECT
    e.cnpj_basico,
    e.razao_social,
    COUNT(DISTINCT est.uf) as total_estados,
    COUNT(*) as total_filiais,
    GROUP_CONCAT(DISTINCT est.uf) as estados
FROM empresas e
JOIN estabelecimentos est ON e.cnpj_basico = est.cnpj_basico
WHERE est.situacao_cadastral = '2'
GROUP BY e.cnpj_basico, e.razao_social
HAVING COUNT(DISTINCT est.uf) >= 3  -- Presença em 3+ estados
ORDER BY total_estados DESC, total_filiais DESC
LIMIT 50;

-- ----------------------------------------------------------------------------
-- 15. DENSIDADE DE EMPRESAS POR ESTADO
-- ----------------------------------------------------------------------------
-- Uso: Comparar distribuição geográfica
-- Performance: 500ms-2s (full table scan)
-- ----------------------------------------------------------------------------
SELECT
    est.uf,
    COUNT(*) as total_estabelecimentos,
    COUNT(DISTINCT est.cnpj_basico) as total_empresas,
    SUM(CASE WHEN est.situacao_cadastral = '2' THEN 1 ELSE 0 END) as ativos,
    ROUND(CAST(SUM(CASE WHEN est.situacao_cadastral = '2' THEN 1 ELSE 0 END) AS FLOAT) / COUNT(*) * 100, 2) as percentual_ativos
FROM estabelecimentos est
GROUP BY est.uf
ORDER BY total_estabelecimentos DESC;

-- ============================================================================
-- DICAS DE OTIMIZAÇÃO
-- ============================================================================
--
-- 1. SEMPRE use índices nas cláusulas WHERE:
--    - uf, municipio → idx_estab_mun_uf
--    - cnae_fiscal_principal → idx_estab_cnae
--    - situacao_cadastral → idx_estab_situacao
--    - cnpj_basico → idx_estab_cnpj
--
-- 2. Use LIMIT quando possível para reduzir resultados
--
-- 3. Evite SELECT * - selecione apenas colunas necessárias
--
-- 4. Para queries frequentes, considere materializar views:
--    CREATE TABLE empresas_ativas_sp AS
--    SELECT ... FROM empresas WHERE ...
--
-- 5. Para buscas textuais frequentes, crie índice FTS:
--    CREATE VIRTUAL TABLE empresas_fts USING fts5(...)
--
-- 6. Use EXPLAIN QUERY PLAN para verificar uso de índices
--
-- 7. Para APIs, sempre use prepared statements + cache
--
-- ============================================================================
