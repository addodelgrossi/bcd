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

-- ============================================================================
-- QUERIES COM TABELAS DE REFERÊNCIA (NOVOS)
-- ============================================================================
-- As queries abaixo demonstram o uso das tabelas de referência para
-- decodificar códigos em descrições legíveis
-- ============================================================================

-- ----------------------------------------------------------------------------
-- 16. BUSCA COMPLETA COM DESCRIÇÕES DE REFERÊNCIA
-- ----------------------------------------------------------------------------
-- Uso: Buscar empresa com todas as descrições decodificadas
-- Performance: < 5ms (múltiplos JOINs com PKs)
-- ----------------------------------------------------------------------------
SELECT
    e.cnpj_basico,
    e.razao_social,
    nj.descricao as natureza_juridica,
    q.descricao as qualificacao_responsavel,
    e.capital_social,
    e.porte_empresa,
    est.cnpj_basico || est.cnpj_ordem || est.cnpj_dv as cnpj_completo,
    est.nome_fantasia,
    est.situacao_cadastral,
    m.descricao as motivo_situacao,
    p.descricao as pais,
    mun.descricao as municipio,
    est.uf,
    c.descricao as cnae_principal
FROM empresas e
JOIN estabelecimentos est ON e.cnpj_basico = est.cnpj_basico
LEFT JOIN naturezas_juridicas nj ON e.natureza_juridica = nj.codigo
LEFT JOIN qualificacoes q ON e.qualificacao_responsavel = q.codigo
LEFT JOIN motivos m ON est.motivo_situacao_cadastral = m.codigo
LEFT JOIN paises p ON est.pais = p.codigo
LEFT JOIN municipios mun ON est.municipio = mun.codigo
LEFT JOIN cnaes c ON est.cnae_fiscal_principal = c.codigo
WHERE e.cnpj_basico = '12345678'
LIMIT 10;

-- ----------------------------------------------------------------------------
-- 17. EMPRESAS NO SIMPLES NACIONAL
-- ----------------------------------------------------------------------------
-- Uso: Listar empresas optantes pelo Simples Nacional
-- Performance: 10-50ms (JOIN com simples)
-- ----------------------------------------------------------------------------
SELECT
    e.cnpj_basico,
    e.razao_social,
    s.opcao_simples,
    s.data_opcao_simples,
    s.opcao_mei,
    s.data_opcao_mei,
    est.uf,
    mun.descricao as municipio
FROM empresas e
JOIN simples s ON e.cnpj_basico = s.cnpj_basico
JOIN estabelecimentos est ON e.cnpj_basico = est.cnpj_basico
LEFT JOIN municipios mun ON est.municipio = mun.codigo
WHERE s.opcao_simples = 'S'
    AND est.identificador_matriz_filial = '1'  -- Apenas matriz
    AND est.situacao_cadastral = '2'  -- Ativa
LIMIT 100;

-- ----------------------------------------------------------------------------
-- 18. DISTRIBUIÇÃO POR NATUREZA JURÍDICA
-- ----------------------------------------------------------------------------
-- Uso: Estatísticas de tipos de empresa
-- Performance: 100-500ms (agregação)
-- ----------------------------------------------------------------------------
SELECT
    nj.descricao as natureza_juridica,
    COUNT(*) as total_empresas,
    ROUND(CAST(COUNT(*) AS FLOAT) / (SELECT COUNT(*) FROM empresas) * 100, 2) as percentual
FROM empresas e
LEFT JOIN naturezas_juridicas nj ON e.natureza_juridica = nj.codigo
GROUP BY e.natureza_juridica, nj.descricao
ORDER BY total_empresas DESC
LIMIT 20;

-- ----------------------------------------------------------------------------
-- 19. TOP PAÍSES DE ORIGEM DAS EMPRESAS ESTRANGEIRAS
-- ----------------------------------------------------------------------------
-- Uso: Empresas com sede no exterior
-- Performance: 50-200ms
-- ----------------------------------------------------------------------------
SELECT
    p.descricao as pais,
    COUNT(DISTINCT est.cnpj_basico) as total_empresas,
    COUNT(*) as total_estabelecimentos
FROM estabelecimentos est
JOIN paises p ON est.pais = p.codigo
WHERE est.pais != '105'  -- 105 = Brasil
    AND est.pais IS NOT NULL
GROUP BY est.pais, p.descricao
ORDER BY total_empresas DESC
LIMIT 20;

-- ----------------------------------------------------------------------------
-- 20. MOTIVOS DE BAIXA/INATIVAÇÃO MAIS COMUNS
-- ----------------------------------------------------------------------------
-- Uso: Análise de razões para empresas inativas
-- Performance: 100-500ms
-- ----------------------------------------------------------------------------
SELECT
    m.descricao as motivo,
    COUNT(*) as total_estabelecimentos,
    ROUND(CAST(COUNT(*) AS FLOAT) / 
          (SELECT COUNT(*) FROM estabelecimentos WHERE situacao_cadastral != '2') * 100, 2) as percentual
FROM estabelecimentos est
JOIN motivos m ON est.motivo_situacao_cadastral = m.codigo
WHERE est.situacao_cadastral != '2'  -- Não ativas
    AND est.motivo_situacao_cadastral IS NOT NULL
    AND est.motivo_situacao_cadastral != '00'  -- Desconsiderar "sem motivo"
GROUP BY est.motivo_situacao_cadastral, m.descricao
ORDER BY total_estabelecimentos DESC
LIMIT 20;

-- ============================================================================
-- QUERIES COM SÓCIOS
-- ============================================================================
-- As queries abaixo demonstram consultas à tabela de sócios/administradores
-- ============================================================================

-- ----------------------------------------------------------------------------
-- 21. BUSCAR SÓCIOS DE UMA EMPRESA
-- ----------------------------------------------------------------------------
-- Uso: Listar todos os sócios de uma empresa específica
-- Performance: < 5ms (usa idx_socios_cnpj)
-- ----------------------------------------------------------------------------
SELECT
    s.cnpj_basico,
    e.razao_social,
    s.identificador_socio,  -- 1=PJ, 2=PF, 3=Estrangeiro
    s.nome_socio,
    s.cnpj_cpf_socio,
    q.descricao as qualificacao,
    s.data_entrada_sociedade,
    p.descricao as pais,
    s.nome_representante,
    qr.descricao as qualificacao_representante
FROM socios s
JOIN empresas e ON s.cnpj_basico = e.cnpj_basico
LEFT JOIN qualificacoes q ON s.qualificacao_socio = q.codigo
LEFT JOIN paises p ON s.pais = p.codigo
LEFT JOIN qualificacoes qr ON s.qualificacao_representante_legal = qr.codigo
WHERE s.cnpj_basico = '12345678'
ORDER BY s.nome_socio;

-- ----------------------------------------------------------------------------
-- 22. ENCONTRAR EMPRESAS POR SÓCIO (Pessoa Física)
-- ----------------------------------------------------------------------------
-- Uso: Descobrir todas as empresas onde uma pessoa é sócia
-- Performance: 20-100ms (scan na tabela socios)
-- ----------------------------------------------------------------------------
SELECT
    e.cnpj_basico,
    e.razao_social,
    s.qualificacao_socio,
    q.descricao as qualificacao,
    s.data_entrada_sociedade,
    est.uf,
    mun.descricao as municipio
FROM socios s
JOIN empresas e ON s.cnpj_basico = e.cnpj_basico
JOIN estabelecimentos est ON e.cnpj_basico = est.cnpj_basico
LEFT JOIN qualificacoes q ON s.qualificacao_socio = q.codigo
LEFT JOIN municipios mun ON est.municipio = mun.codigo
WHERE s.cnpj_cpf_socio = '12345678901'  -- CPF do sócio
    AND s.identificador_socio = '2'  -- Pessoa Física
    AND est.identificador_matriz_filial = '1'  -- Apenas matriz
ORDER BY e.razao_social;

-- ----------------------------------------------------------------------------
-- 23. SÓCIOS PESSOA JURÍDICA (empresas que são sócias de outras)
-- ----------------------------------------------------------------------------
-- Uso: Encontrar holdings ou grupos empresariais
-- Performance: 50-200ms
-- ----------------------------------------------------------------------------
SELECT
    s.cnpj_basico,
    e.razao_social as empresa,
    s.nome_socio as socio_pj,
    s.cnpj_cpf_socio as cnpj_socio,
    q.descricao as qualificacao
FROM socios s
JOIN empresas e ON s.cnpj_basico = e.cnpj_basico
LEFT JOIN qualificacoes q ON s.qualificacao_socio = q.codigo
WHERE s.identificador_socio = '1'  -- Pessoa Jurídica
    AND s.cnpj_cpf_socio IS NOT NULL
ORDER BY s.nome_socio
LIMIT 100;

-- ----------------------------------------------------------------------------
-- 24. SÓCIOS ESTRANGEIROS
-- ----------------------------------------------------------------------------
-- Uso: Identificar participação estrangeira em empresas brasileiras
-- Performance: 50-200ms
-- ----------------------------------------------------------------------------
SELECT
    s.cnpj_basico,
    e.razao_social,
    s.nome_socio,
    p.descricao as pais_origem,
    s.nome_representante,
    est.uf,
    mun.descricao as municipio
FROM socios s
JOIN empresas e ON s.cnpj_basico = e.cnpj_basico
JOIN estabelecimentos est ON e.cnpj_basico = est.cnpj_basico
LEFT JOIN paises p ON s.pais = p.codigo
LEFT JOIN municipios mun ON est.municipio = mun.codigo
WHERE s.identificador_socio = '3'  -- Estrangeiro
    AND s.pais != '105'  -- Diferente de Brasil
    AND est.identificador_matriz_filial = '1'
ORDER BY s.pais, s.nome_socio
LIMIT 100;

-- ----------------------------------------------------------------------------
-- 25. DISTRIBUIÇÃO POR FAIXA ETÁRIA DOS SÓCIOS
-- ----------------------------------------------------------------------------
-- Uso: Análise demográfica dos sócios
-- Performance: 100-500ms (agregação)
-- ----------------------------------------------------------------------------
SELECT
    s.faixa_etaria,
    CASE s.faixa_etaria
        WHEN '1' THEN '0-12 anos'
        WHEN '2' THEN '13-20 anos'
        WHEN '3' THEN '21-30 anos'
        WHEN '4' THEN '31-40 anos'
        WHEN '5' THEN '41-50 anos'
        WHEN '6' THEN '51-60 anos'
        WHEN '7' THEN '61-70 anos'
        WHEN '8' THEN '71-80 anos'
        WHEN '9' THEN 'Mais de 80 anos'
        ELSE 'Não informado'
    END as descricao_faixa,
    COUNT(*) as total_socios
FROM socios s
WHERE s.identificador_socio = '2'  -- Apenas pessoas físicas
    AND s.faixa_etaria IS NOT NULL
GROUP BY s.faixa_etaria
ORDER BY s.faixa_etaria;

-- ============================================================================
