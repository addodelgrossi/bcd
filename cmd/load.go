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
			cnpj_basico TEXT PRIMARY KEY,
			razao_social TEXT,
			natureza_juridica TEXT,
			qualificacao_responsavel TEXT,
			capital_social TEXT,
			porte_empresa TEXT,
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

func loadAll(ctx context.Context, db *sql.DB, dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		path := filepath.Join(dir, name)
		upper := strings.ToUpper(name)

		// Detectar arquivos pelos padrões da Receita Federal
		switch {
		case strings.Contains(upper, "EMPRE"):
			if err := loadEmpresas(ctx, db, path); err != nil {
				return err
			}
		case strings.Contains(upper, "ESTABELE"):
			if err := loadEstabelecimentos(ctx, db, path); err != nil {
				return err
			}
		case strings.Contains(upper, "CNAE"):
			if err := loadCnaes(ctx, db, path); err != nil {
				return err
			}
		case strings.Contains(upper, "MUNIC"):
			if err := loadMunicipios(ctx, db, path); err != nil {
				return err
			}
		case strings.Contains(upper, "PAIS"):
			if err := loadPaises(ctx, db, path); err != nil {
				return err
			}
		case strings.Contains(upper, "QUALS"):
			if err := loadQualificacoes(ctx, db, path); err != nil {
				return err
			}
		case strings.Contains(upper, "NATJU"):
			if err := loadNaturezasJuridicas(ctx, db, path); err != nil {
				return err
			}
		case strings.Contains(upper, "MOTI"):
			if err := loadMotivos(ctx, db, path); err != nil {
				return err
			}
		case strings.Contains(upper, "SIMPLES"):
			if err := loadSimples(ctx, db, path); err != nil {
				return err
			}
		default:
			logger.Info("skip file", slog.String("name", name))
		}
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
		for {
			rec, err := r.Read()
			if errors.Is(err, io.EOF) {
				return nil
			}
			if err != nil {
				return err
			}
			// RFB layout Empresas: 7 colunas (vide dicionário)
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
