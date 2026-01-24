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

var loadCmd = &cobra.Command{
	Use:   "load",
	Short: "Cria o SQLite e carrega CSVs extraídos",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		db, err := openSQLite(flagOutDB)
		if err != nil {
			return err
		}
		defer db.Close()
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
	dsn := fmt.Sprintf("file:%s?_pragma=journal_mode(MEMORY)&_pragma=synchronous(OFF)&_pragma=temp_store(MEMORY)&_pragma=cache_size(-64000)", path)
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
			identificador_matriz_filial TEXT,
			nome_fantasia TEXT,
			situacao_cadastral TEXT,
			data_situacao_cadastral TEXT,
			motivo_situacao_cadastral TEXT,
			nome_cidade_exterior TEXT,
			pais TEXT,
			data_inicio_atividade TEXT,
			cnae_fiscal_principal TEXT,
			cnae_fiscal_secundaria TEXT,
			tipo_logradouro TEXT,
			logradouro TEXT,
			numero TEXT,
			complemento TEXT,
			bairro TEXT,
			cep TEXT,
			uf TEXT,
			municipio TEXT,
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
			cnpj_basico TEXT PRIMARY KEY,
			opcao_simples TEXT,
			data_opcao_simples TEXT,
			data_exclusao_simples TEXT,
			opcao_mei TEXT,
			data_opcao_mei TEXT,
			data_exclusao_mei TEXT
		);`,
		`CREATE TABLE IF NOT EXISTS socios (
			cnpj_basico TEXT NOT NULL,
			identificador_socio TEXT,
			nome_socio TEXT,
			cnpj_cpf_socio TEXT,
			qualificacao_socio TEXT,
			data_entrada_sociedade TEXT,
			pais TEXT,
			representante_legal TEXT,
			nome_representante TEXT,
			qualificacao_representante_legal TEXT,
			faixa_etaria TEXT
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
		// Índices para estabelecimentos (consultas mais comuns)
		`CREATE INDEX IF NOT EXISTS idx_estab_cnpj ON estabelecimentos(cnpj_basico);`,
		`CREATE INDEX IF NOT EXISTS idx_estab_mun_uf ON estabelecimentos(municipio, uf);`,
		`CREATE INDEX IF NOT EXISTS idx_estab_uf ON estabelecimentos(uf);`,
		`CREATE INDEX IF NOT EXISTS idx_estab_cnae ON estabelecimentos(cnae_fiscal_principal);`,
		`CREATE INDEX IF NOT EXISTS idx_estab_situacao ON estabelecimentos(situacao_cadastral);`,
		`CREATE INDEX IF NOT EXISTS idx_estab_cep ON estabelecimentos(cep);`,

		// Índice para matriz/filial (útil para agregações)
		`CREATE INDEX IF NOT EXISTS idx_estab_matriz_filial ON estabelecimentos(identificador_matriz_filial);`,

		// Índices para socios (consultas por empresa)
		`CREATE INDEX IF NOT EXISTS idx_socios_cnpj ON socios(cnpj_basico);`,
	}
	for _, q := range stmts {
		logger.Info("creating index", slog.String("stmt", q))
		if err := exec(ctx, db, q); err != nil {
			return err
		}
	}

	// ANALYZE atualiza estatísticas do query planner
	logger.Info("analyzing tables for query optimization...")
	if err := exec(ctx, db, `ANALYZE;`); err != nil {
		return err
	}

	// VACUUM otimiza o arquivo físico (remove espaço livre, desfragmenta)
	logger.Info("vacuuming database - this may take a while...")
	if err := exec(ctx, db, `VACUUM;`); err != nil {
		return err
	}

	// Otimiza PRAGMAs para LEITURA (depois do load)
	// IMPORTANTE: Alguns PRAGMAs precisam fechar/reabrir a conexão
	logger.Info("optimizing database for read performance...")
	optimizeForReads := []string{
		`PRAGMA journal_mode=WAL;`,    // WAL mode para reads concorrentes
		`PRAGMA synchronous=NORMAL;`,  // Balanço segurança/performance
		`PRAGMA temp_store=MEMORY;`,   // Mantém temp em RAM
		`PRAGMA mmap_size=268435456;`, // 256MB memory-mapped I/O
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
	defer stmt.Close()

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
	defer c.Close()
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
	defer c.Close()
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
		for {
			rec, err := r.Read()
			if errors.Is(err, io.EOF) {
				return nil
			}
			if err != nil {
				return err
			}
			vals := make([]any, len(cols))
			for i := range cols {
				if i < len(rec) {
					vals[i] = strings.TrimSpace(rec[i])
				}
			}
			if err := emit(vals); err != nil {
				return err
			}
		}
	})
}

func loadCnaes(ctx context.Context, db *sql.DB, path string) error {
	r, c, err := newLatin1CSV(path)
	if err != nil {
		return err
	}
	defer c.Close()
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
	defer c.Close()
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
	defer c.Close()
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
	defer c.Close()
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
		for {
			rec, err := r.Read()
			if errors.Is(err, io.EOF) {
				return nil
			}
			if err != nil {
				return err
			}
			// RFB layout Simples: 7 colunas (cnpj_basico + 6 campos do Simples Nacional)
			vals := make([]any, len(cols))
			for i := range cols {
				if i < len(rec) {
					vals[i] = strings.TrimSpace(rec[i])
				} else {
					vals[i] = nil
				}
			}
			if err := emit(vals); err != nil {
				return err
			}
		}
	})
}

func loadSocios(ctx context.Context, db *sql.DB, path string) error {
	r, c, err := newLatin1CSV(path)
	if err != nil {
		return err
	}
	defer c.Close()
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
		for {
			rec, err := r.Read()
			if errors.Is(err, io.EOF) {
				return nil
			}
			if err != nil {
				return err
			}
			// RFB layout Socios: 11 colunas (conforme metadados da RFB)
			vals := make([]any, len(cols))
			for i := range cols {
				if i < len(rec) {
					vals[i] = strings.TrimSpace(rec[i])
				} else {
					vals[i] = nil
				}
			}
			if err := emit(vals); err != nil {
				return err
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
