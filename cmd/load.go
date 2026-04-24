package cmd

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"
	_ "modernc.org/sqlite"
)

const (
	// Cache sizes (negative values = KB)
	cacheSize64MB = -64000 // 64MB for normal operations
	cacheSize16MB = -16000 // 16MB for VACUUM (reduced to free memory)
)

var loadCmd = &cobra.Command{
	Use:   "load",
	Short: "Cria o SQLite e carrega CSVs extraídos",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		db, err := openSQLite(flagOutDB)
		if err != nil {
			return err
		}
		defer func() { _ = db.Close() }()
		if err := createSchema(ctx, db); err != nil {
			return err
		}
		extracted := filepath.Join(flagWorkdir, "extracted")
		if err := loadAll(ctx, db, extracted); err != nil {
			return err
		}
		if err := createIndexes(ctx, db); err != nil {
			return err
		}
		if err := showStats(ctx, db); err != nil {
			return err
		}
		logger.Info("load done", slog.String("db", flagOutDB))
		return nil
	},
}

func init() { RootCmd.AddCommand(loadCmd) }

func openSQLite(path string) (*sql.DB, error) {
	// PRAGMAs otimizados para bulk load (escrita rápida)
	// journal_mode=MEMORY: evita I/O de journal durante carga
	// synchronous=OFF: maximiza velocidade (seguro apenas durante import)
	// temp_store=MEMORY: tabelas temporárias em RAM
	// cache_size=-64000: 64MB de cache (negativo = KB)
	dsn := fmt.Sprintf("file:%s?_pragma=journal_mode(MEMORY)&_pragma=synchronous(OFF)&_pragma=temp_store(MEMORY)&_pragma=cache_size(%d)", path, cacheSize64MB)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1) // SQLite recommends 1 writer
	return db, nil
}

func exec(ctx context.Context, db *sql.DB, q string) error {
	_, err := db.ExecContext(ctx, q)
	return err
}

func createSchema(ctx context.Context, db *sql.DB) error {
	stmts := []string{
		`PRAGMA foreign_keys=OFF;`,
		`CREATE TABLE IF NOT EXISTS empresas (
			cnpj_basico TEXT PRIMARY KEY NOT NULL,
			razao_social TEXT NOT NULL,
			natureza_juridica INTEGER,
			qualificacao_responsavel INTEGER,
			capital_social REAL DEFAULT 0.0,
			porte_empresa INTEGER,
			ente_federativo TEXT
		);`,
		`CREATE TABLE IF NOT EXISTS estabelecimentos (
			cnpj_basico TEXT NOT NULL,
			cnpj_ordem TEXT NOT NULL,
			cnpj_dv TEXT NOT NULL,
			identificador_matriz_filial INTEGER,
			nome_fantasia TEXT,
			situacao_cadastral INTEGER,
			data_situacao_cadastral TEXT,
			motivo_situacao_cadastral INTEGER,
			nome_cidade_exterior TEXT,
			pais INTEGER,
			data_inicio_atividade TEXT,
			cnae_fiscal_principal INTEGER,
			cnae_fiscal_secundaria TEXT,
			tipo_logradouro TEXT,
			logradouro TEXT,
			numero TEXT,
			complemento TEXT,
			bairro TEXT,
			cep TEXT,
			uf TEXT,
			municipio INTEGER,
			ddd1 TEXT, telefone1 TEXT,
			ddd2 TEXT, telefone2 TEXT,
			ddd_fax TEXT, fax TEXT,
			correio_eletronico TEXT,
			situacao_especial TEXT,
			data_situacao_especial TEXT,
			PRIMARY KEY (cnpj_basico, cnpj_ordem, cnpj_dv)
		);`,
		`CREATE TABLE IF NOT EXISTS cnaes (
			codigo TEXT PRIMARY KEY,
			descricao TEXT
		);`,
		`CREATE TABLE IF NOT EXISTS municipios (
			codigo TEXT PRIMARY KEY,
			descricao TEXT
		);`,
		`CREATE TABLE IF NOT EXISTS paises (
			codigo TEXT PRIMARY KEY,
			descricao TEXT
		);`,
		`CREATE TABLE IF NOT EXISTS qualificacoes (
			codigo TEXT PRIMARY KEY,
			descricao TEXT
		);`,
		`CREATE TABLE IF NOT EXISTS naturezas_juridicas (
			codigo TEXT PRIMARY KEY,
			descricao TEXT
		);`,
		`CREATE TABLE IF NOT EXISTS motivos (
			codigo TEXT PRIMARY KEY,
			descricao TEXT
		);`,
		`CREATE TABLE IF NOT EXISTS simples (
			cnpj_basico TEXT PRIMARY KEY NOT NULL,
			opcao_simples TEXT,
			data_opcao_simples TEXT,
			data_exclusao_simples TEXT,
			opcao_mei TEXT,
			data_opcao_mei TEXT,
			data_exclusao_mei TEXT
		);`,
		`CREATE TABLE IF NOT EXISTS socios (
			cnpj_basico TEXT NOT NULL,
			identificador_socio INTEGER,
			nome_socio TEXT,
			cnpj_cpf_socio TEXT,
			qualificacao_socio INTEGER,
			data_entrada_sociedade TEXT,
			pais INTEGER,
			representante_legal TEXT,
			nome_representante TEXT,
			qualificacao_representante_legal INTEGER,
			faixa_etaria INTEGER
		);`,
	}
	for _, q := range stmts {
		if err := exec(ctx, db, q); err != nil {
			return err
		}
	}
	return nil
}

func createIndexes(ctx context.Context, db *sql.DB) error {
	logger.Info("creating indexes - this may take several minutes...")
	stmts := []string{
		// Índices simples — usados por lookups pontuais e filtros isolados.
		`CREATE INDEX IF NOT EXISTS idx_estab_cnpj ON estabelecimentos(cnpj_basico);`,
		`CREATE INDEX IF NOT EXISTS idx_estab_mun_uf ON estabelecimentos(municipio, uf);`,
		`CREATE INDEX IF NOT EXISTS idx_estab_uf ON estabelecimentos(uf);`,
		`CREATE INDEX IF NOT EXISTS idx_estab_cnae ON estabelecimentos(cnae_fiscal_principal);`,
		`CREATE INDEX IF NOT EXISTS idx_estab_situacao ON estabelecimentos(situacao_cadastral);`,
		`CREATE INDEX IF NOT EXISTS idx_estab_cep ON estabelecimentos(cep);`,
		`CREATE INDEX IF NOT EXISTS idx_estab_matriz_filial ON estabelecimentos(identificador_matriz_filial);`,

		// Covering indexes para listagem paginada da API (/api/v1/companies).
		// A API filtra por uma coluna e ordena pelo PK composto para cursor
		// pagination; sem combinar filtro + PK no mesmo índice o SQLite cai
		// num sort em temp B-tree que estoura o timeout HTTP em filtros
		// amplos (ex.: uf=SP ≈ 20 M linhas).
		`CREATE INDEX IF NOT EXISTS idx_estab_uf_pk ON estabelecimentos(uf, cnpj_basico, cnpj_ordem, cnpj_dv);`,
		`CREATE INDEX IF NOT EXISTS idx_estab_mun_pk ON estabelecimentos(municipio, cnpj_basico, cnpj_ordem, cnpj_dv);`,
		`CREATE INDEX IF NOT EXISTS idx_estab_cnae_pk ON estabelecimentos(cnae_fiscal_principal, cnpj_basico, cnpj_ordem, cnpj_dv);`,
		`CREATE INDEX IF NOT EXISTS idx_estab_situacao_pk ON estabelecimentos(situacao_cadastral, cnpj_basico, cnpj_ordem, cnpj_dv);`,

		// Sócios — lookups por empresa e busca reversa por CPF ofuscado.
		`CREATE INDEX IF NOT EXISTS idx_socios_cnpj ON socios(cnpj_basico);`,
		`CREATE INDEX IF NOT EXISTS idx_socios_cpf ON socios(cnpj_cpf_socio);`,
		// Lookup reverso por representante legal — consumido pelo
		// RepresentativeRelator no grafo de relações da API.
		`CREATE INDEX IF NOT EXISTS idx_socios_representante ON socios(representante_legal);`,
	}
	for _, q := range stmts {
		logger.Info("creating index", slog.String("stmt", q))
		if err := exec(ctx, db, q); err != nil {
			return err
		}
	}

	// FTS5 virtual tables para busca textual em razão social / nome fantasia.
	// Roda antes de ANALYZE/VACUUM para aproveitar journal_mode=MEMORY do bulk
	// load; o VACUUM que vem depois compacta também as shadow tables do FTS.
	if err := createFTSTables(ctx, db); err != nil {
		return err
	}

	// ANALYZE atualiza estatísticas do query planner
	logger.Info("analyzing tables for query optimization...")
	if err := exec(ctx, db, `ANALYZE;`); err != nil {
		return err
	}

	// VACUUM otimiza o arquivo físico (remove espaço livre, desfragmenta)
	// Pode ser pulado em ambientes com pouca memória usando --skip-vacuum
	if !flagSkipVacuum {
		logger.Info("vacuuming database - this may take a while...")
		logger.Info("note: VACUUM requires available memory (~2x database size)")
		logger.Info("if you encounter out-of-memory errors, use --skip-vacuum flag")
		
		// Reduzir cache temporariamente para liberar memória para VACUUM
		// VACUUM precisa de memória para criar uma cópia do banco
		if err := exec(ctx, db, fmt.Sprintf(`PRAGMA cache_size=%d;`, cacheSize16MB)); err != nil {
			return err
		}
		
		if err := exec(ctx, db, `VACUUM;`); err != nil {
			return fmt.Errorf("VACUUM failed - try using --skip-vacuum flag: %w", err)
		}
		
		logger.Info("vacuum completed successfully")
	} else {
		logger.Info("skipping VACUUM operation (--skip-vacuum enabled)")
		logger.Info("note: database size may be larger without VACUUM")
	}

	// Otimiza PRAGMAs para LEITURA (depois do load)
	// IMPORTANTE: Alguns PRAGMAs precisam fechar/reabrir a conexão
	logger.Info("optimizing database for read performance...")
	optimizeForReads := []string{
		`PRAGMA journal_mode=WAL;`,    // WAL mode para reads concorrentes
		`PRAGMA synchronous=NORMAL;`,  // Balanço segurança/performance
		`PRAGMA temp_store=MEMORY;`,   // Mantém temp em RAM
		`PRAGMA mmap_size=268435456;`, // 256MB memory-mapped I/O
		fmt.Sprintf(`PRAGMA cache_size=%d;`, cacheSize64MB), // Restaurar 64MB de cache
	}
	for _, q := range optimizeForReads {
		if err := exec(ctx, db, q); err != nil {
			return err
		}
	}

	logger.Info("indexes and optimizations completed")
	logger.Info("database is ready for high-performance reads")
	return nil
}

// createFTSTables cria as virtual tables FTS5 (empresas_fts e
// estabelecimentos_fts) consumidas pela API em /api/v1/companies?q=<termo>.
// Tokenizer unicode61 + remove_diacritics faz "sao paulo" casar com
// "São Paulo". Drop-and-recreate é intencional: a RFB é batch mensal,
// triggers de sync incremental só atrasariam o bulk load sem benefício real.
func createFTSTables(ctx context.Context, db *sql.DB) error {
	if flagSkipFTS {
		logger.Info("skipping FTS5 virtual tables (--skip-fts)")
		return nil
	}
	logger.Info("creating FTS5 virtual tables - this may take a few minutes...")

	steps := []struct{ label, sql string }{
		{"drop empresas_fts", `DROP TABLE IF EXISTS empresas_fts;`},
		{"drop estabelecimentos_fts", `DROP TABLE IF EXISTS estabelecimentos_fts;`},
		{"create empresas_fts", `CREATE VIRTUAL TABLE empresas_fts USING fts5(
			cnpj_basico UNINDEXED,
			razao_social,
			tokenize = 'unicode61 remove_diacritics 2'
		);`},
		{"create estabelecimentos_fts", `CREATE VIRTUAL TABLE estabelecimentos_fts USING fts5(
			cnpj_basico UNINDEXED,
			cnpj_ordem UNINDEXED,
			cnpj_dv UNINDEXED,
			nome_fantasia,
			tokenize = 'unicode61 remove_diacritics 2'
		);`},
		{"populate empresas_fts", `INSERT INTO empresas_fts(cnpj_basico, razao_social)
			SELECT cnpj_basico, razao_social
			  FROM empresas
			 WHERE razao_social IS NOT NULL AND razao_social <> '';`},
		{"populate estabelecimentos_fts", `INSERT INTO estabelecimentos_fts(cnpj_basico, cnpj_ordem, cnpj_dv, nome_fantasia)
			SELECT cnpj_basico, cnpj_ordem, cnpj_dv, nome_fantasia
			  FROM estabelecimentos
			 WHERE nome_fantasia IS NOT NULL AND nome_fantasia <> '';`},
		{"optimize empresas_fts", `INSERT INTO empresas_fts(empresas_fts) VALUES('optimize');`},
		{"optimize estabelecimentos_fts", `INSERT INTO estabelecimentos_fts(estabelecimentos_fts) VALUES('optimize');`},
	}
	for _, s := range steps {
		logger.Info("fts step", slog.String("step", s.label))
		if err := exec(ctx, db, s.sql); err != nil {
			return fmt.Errorf("fts %s: %w", s.label, err)
		}
	}
	logger.Info("fts5 tables ready")
	return nil
}

// entityLoader representa um carregador de entidade
type entityLoader struct {
	name    string
	pattern string
	loader  func(context.Context, *sql.DB, string) error
}

func loadAll(ctx context.Context, db *sql.DB, dir string) error {
	// Ordem de carga: tabelas de referência primeiro, depois dados principais
	loaders := []entityLoader{
		// 1. Tabelas de referência (necessárias para validação)
		{name: "Países", pattern: "PAIS", loader: loadPaises},
		{name: "Municípios", pattern: "MUNIC", loader: loadMunicipios},
		{name: "CNAEs", pattern: "CNAE", loader: loadCnaes},
		{name: "Qualificações", pattern: "QUALS", loader: loadQualificacoes},
		{name: "Naturezas Jurídicas", pattern: "NATJU", loader: loadNaturezasJuridicas},
		{name: "Motivos", pattern: "MOTI", loader: loadMotivos},

		// 2. Dados principais (ordem lógica: empresa -> estabelecimento -> sócios)
		{name: "Empresas", pattern: "EMPRE", loader: loadEmpresas},
		{name: "Estabelecimentos", pattern: "ESTABELE", loader: loadEstabelecimentos},
		{name: "Simples Nacional", pattern: "SIMPLES", loader: loadSimples},
		{name: "Sócios", pattern: "SOCIO", loader: loadSocios},
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	// Mapear arquivos disponíveis
	files := make(map[string][]string)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		upper := strings.ToUpper(name)
		for _, l := range loaders {
			if strings.Contains(upper, l.pattern) {
				files[l.pattern] = append(files[l.pattern], filepath.Join(dir, name))
				break
			}
		}
	}

	// Carregar na ordem definida
	for _, l := range loaders {
		paths, ok := files[l.pattern]
		if !ok {
			logger.Warn("no files found for entity", slog.String("entity", l.name))
			continue
		}

		logger.Info("loading entity", slog.String("entity", l.name), slog.Int("files", len(paths)))

		for _, path := range paths {
			logger.Info("processing file",
				slog.String("entity", l.name),
				slog.String("file", filepath.Base(path)))

			if err := l.loader(ctx, db, path); err != nil {
				return fmt.Errorf("failed to load %s from %s: %w", l.name, filepath.Base(path), err)
			}
		}

		logger.Info("entity loaded successfully", slog.String("entity", l.name))
	}

	return nil
}

func newLatin1CSV(path string) (*csv.Reader, io.Closer, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	br := bufio.NewReaderSize(f, 4<<20)
	tr := transform.NewReader(br, charmap.ISO8859_1.NewDecoder())
	r := csv.NewReader(tr)
	r.Comma = ';'
	r.FieldsPerRecord = -1
	r.LazyQuotes = true
	return r, f, nil
}

func txInsert(ctx context.Context, db *sql.DB, table string, cols []string, iter func(func([]any) error) error) error {
	tx, err := db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return err
	}

	// Defer rollback - será ignorado se commit for bem-sucedido
	defer func() {
		if err := tx.Rollback(); err != nil && err != sql.ErrTxDone {
			logger.Error("rollback failed", slog.String("table", table), slog.String("error", err.Error()))
		}
	}()

	place := make([]string, len(cols))
	for i := range cols {
		place[i] = "?"
	}
	stmt, err := tx.PrepareContext(ctx, fmt.Sprintf("INSERT OR REPLACE INTO %s(%s) VALUES(%s)", table, strings.Join(cols, ","), strings.Join(place, ",")))
	if err != nil {
		return err
	}
	defer func() { _ = stmt.Close() }()

	count := 0
	lastLog := time.Now()
	flush := func(vals []any) error { _, err := stmt.ExecContext(ctx, vals...); return err }
	err = iter(func(vals []any) error {
		if err := flush(vals); err != nil {
			return err
		}
		count++
		if time.Since(lastLog) > 3*time.Second {
			logger.Info("loading", slog.String("table", table), slog.Int("rows", count))
			lastLog = time.Now()
		}
		return nil
	})
	if err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	logger.Info("table loaded", slog.String("table", table), slog.Int("rows", count))
	return nil
}

// ===== Loaders =====

func loadEmpresas(ctx context.Context, db *sql.DB, path string) error {
	r, c, err := newLatin1CSV(path)
	if err != nil {
		return err
	}
	defer func() { _ = c.Close() }()
	cols := []string{
		"cnpj_basico",
		"razao_social",
		"natureza_juridica",
		"qualificacao_responsavel",
		"capital_social",
		"porte_empresa",
		"ente_federativo",
	}
	return txInsert(ctx, db, "empresas", cols, func(emit func([]any) error) error {
		lineNum := 0
		for {
			rec, err := r.Read()
			lineNum++
			if errors.Is(err, io.EOF) {
				return nil
			}
			if err != nil {
				return fmt.Errorf("line %d: %w", lineNum, err)
			}

			// Validação: RFB layout Empresas: 7 colunas obrigatórias
			if len(rec) < 7 {
				return fmt.Errorf("line %d: expected 7 columns, got %d", lineNum, len(rec))
			}

			vals := make([]any, len(cols))

			// Campo 0: CNPJ BÁSICO (8 dígitos) - TEXT, obrigatório
			cnpjBasico := strings.TrimSpace(rec[0])
			if cnpjBasico == "" {
				return fmt.Errorf("line %d: cnpj_basico cannot be empty", lineNum)
			}
			vals[0] = cnpjBasico

			// Campo 1: RAZÃO SOCIAL - TEXT, obrigatório
			razaoSocial := strings.TrimSpace(rec[1])
			if razaoSocial == "" {
				return fmt.Errorf("line %d: razao_social cannot be empty", lineNum)
			}
			vals[1] = razaoSocial

			// Campo 2: NATUREZA JURÍDICA - INTEGER (código)
			natureza := strings.TrimSpace(rec[2])
			if natureza == "" {
				vals[2] = nil
			} else {
				vals[2] = natureza // Será convertido para INTEGER pelo SQLite
			}

			// Campo 3: QUALIFICAÇÃO DO RESPONSÁVEL - INTEGER (código)
			qualif := strings.TrimSpace(rec[3])
			if qualif == "" {
				vals[3] = nil
			} else {
				vals[3] = qualif
			}

			// Campo 4: CAPITAL SOCIAL - REAL (valor monetário)
			capital := strings.TrimSpace(rec[4])
			if capital == "" || capital == "0" {
				vals[4] = 0.0
			} else {
				// Capital vem como string "10000,00" - converter para float
				capital = strings.ReplaceAll(capital, ",", ".")
				vals[4] = capital // SQLite converterá para REAL
			}

			// Campo 5: PORTE DA EMPRESA - INTEGER (código: 00, 01, 03, 05)
			porte := strings.TrimSpace(rec[5])
			if porte == "" || porte == "00" {
				vals[5] = nil
			} else {
				vals[5] = porte
			}

			// Campo 6: ENTE FEDERATIVO RESPONSÁVEL - TEXT (apenas para natureza 1XXX)
			ente := strings.TrimSpace(rec[6])
			if ente == "" {
				vals[6] = nil
			} else {
				vals[6] = ente
			}

			if err := emit(vals); err != nil {
				return fmt.Errorf("line %d: %w", lineNum, err)
			}
		}
	})
}

func loadEstabelecimentos(ctx context.Context, db *sql.DB, path string) error {
	r, c, err := newLatin1CSV(path)
	if err != nil {
		return err
	}
	defer func() { _ = c.Close() }()
	cols := []string{
		"cnpj_basico",
		"cnpj_ordem",
		"cnpj_dv",
		"identificador_matriz_filial",
		"nome_fantasia",
		"situacao_cadastral",
		"data_situacao_cadastral",
		"motivo_situacao_cadastral",
		"nome_cidade_exterior",
		"pais",
		"data_inicio_atividade",
		"cnae_fiscal_principal",
		"cnae_fiscal_secundaria",
		"tipo_logradouro",
		"logradouro",
		"numero",
		"complemento",
		"bairro",
		"cep",
		"uf",
		"municipio",
		"ddd1",
		"telefone1",
		"ddd2",
		"telefone2",
		"ddd_fax",
		"fax",
		"correio_eletronico",
		"situacao_especial",
		"data_situacao_especial",
	}
	return txInsert(ctx, db, "estabelecimentos", cols, func(emit func([]any) error) error {
		lineNum := 0
		for {
			rec, err := r.Read()
			lineNum++
			if errors.Is(err, io.EOF) {
				return nil
			}
			if err != nil {
				return fmt.Errorf("line %d: %w", lineNum, err)
			}

			// Validação: layout Estabelecimentos: 30 colunas
			if len(rec) < 30 {
				return fmt.Errorf("line %d: expected 30 columns, got %d", lineNum, len(rec))
			}

			vals := make([]any, len(cols))

			// Campo 0: CNPJ BÁSICO - TEXT, obrigatório
			vals[0] = strings.TrimSpace(rec[0])
			if vals[0] == "" {
				return fmt.Errorf("line %d: cnpj_basico cannot be empty", lineNum)
			}

			// Campo 1: CNPJ ORDEM - TEXT, obrigatório
			vals[1] = strings.TrimSpace(rec[1])
			if vals[1] == "" {
				return fmt.Errorf("line %d: cnpj_ordem cannot be empty", lineNum)
			}

			// Campo 2: CNPJ DV - TEXT, obrigatório
			vals[2] = strings.TrimSpace(rec[2])
			if vals[2] == "" {
				return fmt.Errorf("line %d: cnpj_dv cannot be empty", lineNum)
			}

			// Campo 3: IDENTIFICADOR MATRIZ/FILIAL - INTEGER (1=matriz, 2=filial)
			matrizFilial := strings.TrimSpace(rec[3])
			vals[3] = parseIntOrNil(matrizFilial)

			// Campo 4: NOME FANTASIA - TEXT
			vals[4] = trimOrNil(rec[4])

			// Campo 5: SITUAÇÃO CADASTRAL - INTEGER (01=nula, 2=ativa, 3=suspensa, 4=inapta, 08=baixada)
			vals[5] = parseIntOrNil(strings.TrimSpace(rec[5]))

			// Campo 6: DATA SITUAÇÃO CADASTRAL - TEXT (formato: YYYYMMDD)
			vals[6] = trimOrNil(rec[6])

			// Campo 7: MOTIVO SITUAÇÃO CADASTRAL - INTEGER (código)
			vals[7] = parseIntOrNil(strings.TrimSpace(rec[7]))

			// Campo 8: NOME DA CIDADE NO EXTERIOR - TEXT
			vals[8] = trimOrNil(rec[8])

			// Campo 9: PAÍS - INTEGER (código)
			vals[9] = parseIntOrNil(strings.TrimSpace(rec[9]))

			// Campo 10: DATA DE INÍCIO ATIVIDADE - TEXT (formato: YYYYMMDD)
			vals[10] = trimOrNil(rec[10])

			// Campo 11: CNAE FISCAL PRINCIPAL - INTEGER (código)
			vals[11] = parseIntOrNil(strings.TrimSpace(rec[11]))

			// Campo 12: CNAE FISCAL SECUNDÁRIA - TEXT (separado por vírgula)
			vals[12] = trimOrNil(rec[12])

			// Campos 13-18: Endereço - TEXT
			vals[13] = trimOrNil(rec[13]) // tipo_logradouro
			vals[14] = trimOrNil(rec[14]) // logradouro
			vals[15] = trimOrNil(rec[15]) // numero
			vals[16] = trimOrNil(rec[16]) // complemento
			vals[17] = trimOrNil(rec[17]) // bairro
			vals[18] = trimOrNil(rec[18]) // cep

			// Campo 19: UF - TEXT (2 caracteres)
			vals[19] = trimOrNil(rec[19])

			// Campo 20: MUNICÍPIO - INTEGER (código)
			vals[20] = parseIntOrNil(strings.TrimSpace(rec[20]))

			// Campos 21-26: Telefones - TEXT
			vals[21] = trimOrNil(rec[21]) // ddd1
			vals[22] = trimOrNil(rec[22]) // telefone1
			vals[23] = trimOrNil(rec[23]) // ddd2
			vals[24] = trimOrNil(rec[24]) // telefone2
			vals[25] = trimOrNil(rec[25]) // ddd_fax
			vals[26] = trimOrNil(rec[26]) // fax

			// Campo 27: CORREIO ELETRÔNICO - TEXT
			vals[27] = trimOrNil(rec[27])

			// Campo 28: SITUAÇÃO ESPECIAL - TEXT
			vals[28] = trimOrNil(rec[28])

			// Campo 29: DATA DA SITUAÇÃO ESPECIAL - TEXT (formato: YYYYMMDD)
			vals[29] = trimOrNil(rec[29])

			if err := emit(vals); err != nil {
				return fmt.Errorf("line %d: %w", lineNum, err)
			}
		}
	})
}

func loadCnaes(ctx context.Context, db *sql.DB, path string) error {
	r, c, err := newLatin1CSV(path)
	if err != nil {
		return err
	}
	defer func() { _ = c.Close() }()
	cols := []string{"codigo", "descricao"}
	return txInsert(ctx, db, "cnaes", cols, func(emit func([]any) error) error {
		for {
			rec, err := r.Read()
			if errors.Is(err, io.EOF) {
				return nil
			}
			if err != nil {
				return err
			}
			vals := []any{strings.TrimSpace(rec[0]), strings.TrimSpace(joinRest(rec[1:]))}
			if err := emit(vals); err != nil {
				return err
			}
		}
	})
}

func loadMunicipios(ctx context.Context, db *sql.DB, path string) error {
	r, c, err := newLatin1CSV(path)
	if err != nil {
		return err
	}
	defer func() { _ = c.Close() }()
	cols := []string{"codigo", "descricao"}
	return txInsert(ctx, db, "municipios", cols, func(emit func([]any) error) error {
		for {
			rec, err := r.Read()
			if errors.Is(err, io.EOF) {
				return nil
			}
			if err != nil {
				return err
			}
			vals := []any{strings.TrimSpace(rec[0]), strings.TrimSpace(joinRest(rec[1:]))}
			if err := emit(vals); err != nil {
				return err
			}
		}
	})
}

func loadPaises(ctx context.Context, db *sql.DB, path string) error {
	return loadReferenceTable(ctx, db, path, "paises")
}

func loadQualificacoes(ctx context.Context, db *sql.DB, path string) error {
	return loadReferenceTable(ctx, db, path, "qualificacoes")
}

func loadNaturezasJuridicas(ctx context.Context, db *sql.DB, path string) error {
	return loadReferenceTable(ctx, db, path, "naturezas_juridicas")
}

func loadMotivos(ctx context.Context, db *sql.DB, path string) error {
	return loadReferenceTable(ctx, db, path, "motivos")
}

// loadReferenceTable is a generic loader for two-column reference tables (codigo, descricao)
func loadReferenceTable(ctx context.Context, db *sql.DB, path, tableName string) error {
	r, c, err := newLatin1CSV(path)
	if err != nil {
		return err
	}
	defer func() { _ = c.Close() }()
	cols := []string{"codigo", "descricao"}
	return txInsert(ctx, db, tableName, cols, func(emit func([]any) error) error {
		for {
			rec, err := r.Read()
			if errors.Is(err, io.EOF) {
				return nil
			}
			if err != nil {
				return err
			}
			vals := []any{strings.TrimSpace(rec[0]), strings.TrimSpace(joinRest(rec[1:]))}
			if err := emit(vals); err != nil {
				return err
			}
		}
	})
}

func loadSimples(ctx context.Context, db *sql.DB, path string) error {
	r, c, err := newLatin1CSV(path)
	if err != nil {
		return err
	}
	defer func() { _ = c.Close() }()
	cols := []string{
		"cnpj_basico",
		"opcao_simples",
		"data_opcao_simples",
		"data_exclusao_simples",
		"opcao_mei",
		"data_opcao_mei",
		"data_exclusao_mei",
	}
	return txInsert(ctx, db, "simples", cols, func(emit func([]any) error) error {
		lineNum := 0
		for {
			rec, err := r.Read()
			lineNum++
			if errors.Is(err, io.EOF) {
				return nil
			}
			if err != nil {
				return fmt.Errorf("line %d: %w", lineNum, err)
			}

			// Validação: layout Simples: 7 colunas
			if len(rec) < 7 {
				return fmt.Errorf("line %d: expected 7 columns, got %d", lineNum, len(rec))
			}

			vals := make([]any, len(cols))

			// Campo 0: CNPJ BÁSICO - TEXT, obrigatório
			vals[0] = strings.TrimSpace(rec[0])
			if vals[0] == "" {
				return fmt.Errorf("line %d: cnpj_basico cannot be empty", lineNum)
			}

			// Campo 1: OPÇÃO PELO SIMPLES - TEXT (S/N/branco)
			vals[1] = trimOrNil(rec[1])

			// Campo 2: DATA DE OPÇÃO PELO SIMPLES - TEXT (formato: YYYYMMDD)
			vals[2] = trimOrNil(rec[2])

			// Campo 3: DATA DE EXCLUSÃO DO SIMPLES - TEXT (formato: YYYYMMDD)
			vals[3] = trimOrNil(rec[3])

			// Campo 4: OPÇÃO PELO MEI - TEXT (S/N/branco)
			vals[4] = trimOrNil(rec[4])

			// Campo 5: DATA DE OPÇÃO PELO MEI - TEXT (formato: YYYYMMDD)
			vals[5] = trimOrNil(rec[5])

			// Campo 6: DATA DE EXCLUSÃO DO MEI - TEXT (formato: YYYYMMDD)
			vals[6] = trimOrNil(rec[6])

			if err := emit(vals); err != nil {
				return fmt.Errorf("line %d: %w", lineNum, err)
			}
		}
	})
}

func loadSocios(ctx context.Context, db *sql.DB, path string) error {
	r, c, err := newLatin1CSV(path)
	if err != nil {
		return err
	}
	defer func() { _ = c.Close() }()
	cols := []string{
		"cnpj_basico",
		"identificador_socio",
		"nome_socio",
		"cnpj_cpf_socio",
		"qualificacao_socio",
		"data_entrada_sociedade",
		"pais",
		"representante_legal",
		"nome_representante",
		"qualificacao_representante_legal",
		"faixa_etaria",
	}
	return txInsert(ctx, db, "socios", cols, func(emit func([]any) error) error {
		lineNum := 0
		for {
			rec, err := r.Read()
			lineNum++
			if errors.Is(err, io.EOF) {
				return nil
			}
			if err != nil {
				return fmt.Errorf("line %d: %w", lineNum, err)
			}

			// Validação: layout Sócios: 11 colunas
			if len(rec) < 11 {
				return fmt.Errorf("line %d: expected 11 columns, got %d", lineNum, len(rec))
			}

			vals := make([]any, len(cols))

			// Campo 0: CNPJ BÁSICO - TEXT, obrigatório
			vals[0] = strings.TrimSpace(rec[0])
			if vals[0] == "" {
				return fmt.Errorf("line %d: cnpj_basico cannot be empty", lineNum)
			}

			// Campo 1: IDENTIFICADOR DE SÓCIO - INTEGER (1=PJ, 2=PF, 3=estrangeiro)
			vals[1] = parseIntOrNil(strings.TrimSpace(rec[1]))

			// Campo 2: NOME DO SÓCIO (PF) OU RAZÃO SOCIAL (PJ) - TEXT
			vals[2] = trimOrNil(rec[2])

			// Campo 3: CNPJ/CPF DO SÓCIO - TEXT (descaracterizado: ***123456**)
			// Nota: CPF com 3 primeiros dígitos e 2 últimos ocultados
			vals[3] = trimOrNil(rec[3])

			// Campo 4: QUALIFICAÇÃO DO SÓCIO - INTEGER (código)
			vals[4] = parseIntOrNil(strings.TrimSpace(rec[4]))

			// Campo 5: DATA DE ENTRADA SOCIEDADE - TEXT (formato: YYYYMMDD)
			vals[5] = trimOrNil(rec[5])

			// Campo 6: PAÍS - INTEGER (código do país, sócio estrangeiro)
			vals[6] = parseIntOrNil(strings.TrimSpace(rec[6]))

			// Campo 7: REPRESENTANTE LEGAL - TEXT (CPF, descaracterizado)
			vals[7] = trimOrNil(rec[7])

			// Campo 8: NOME DO REPRESENTANTE - TEXT
			vals[8] = trimOrNil(rec[8])

			// Campo 9: QUALIFICAÇÃO DO REPRESENTANTE LEGAL - INTEGER (código)
			vals[9] = parseIntOrNil(strings.TrimSpace(rec[9]))

			// Campo 10: FAIXA ETÁRIA - INTEGER (0-9: 0=não aplica, 1=0-12, 2=13-20, ..., 9=>80)
			vals[10] = parseIntOrNil(strings.TrimSpace(rec[10]))

			if err := emit(vals); err != nil {
				return fmt.Errorf("line %d: %w", lineNum, err)
			}
		}
	})
}

func joinRest(ss []string) string {
	for i := range ss {
		ss[i] = strings.TrimSpace(ss[i])
	}
	return strings.Join(ss, ";")
}

// trimOrNil retorna o valor trimmed ou nil se vazio
func trimOrNil(s string) any {
	v := strings.TrimSpace(s)
	if v == "" {
		return nil
	}
	return v
}

// parseIntOrNil converte string para int ou retorna nil se vazio/inválido
func parseIntOrNil(s string) any {
	if s == "" || s == "0" || s == "00" {
		return nil
	}
	return s // SQLite converterá para INTEGER
}

func showStats(ctx context.Context, db *sql.DB) error {
	tables := []string{
		"empresas",
		"estabelecimentos",
		"socios",
		"cnaes",
		"municipios",
		"paises",
		"qualificacoes",
		"naturezas_juridicas",
		"motivos",
		"simples",
	}
	logger.Info("=== Database Statistics ===")

	for _, table := range tables {
		var count int
		if err := db.QueryRowContext(ctx, fmt.Sprintf("SELECT COUNT(*) FROM %s", table)).Scan(&count); err != nil {
			return err
		}
		logger.Info("table stats", slog.String("table", table), slog.Int("rows", count))
	}

	// Tamanho do arquivo
	var pageCount, pageSize int
	if err := db.QueryRowContext(ctx, "PRAGMA page_count").Scan(&pageCount); err != nil {
		return err
	}
	if err := db.QueryRowContext(ctx, "PRAGMA page_size").Scan(&pageSize); err != nil {
		return err
	}
	sizeGB := float64(pageCount*pageSize) / (1024 * 1024 * 1024)
	logger.Info("database size", slog.Float64("size_gb", sizeGB))

	return nil
}
