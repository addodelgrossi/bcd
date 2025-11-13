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
	"regexp"
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
		logger.Info("load done", slog.String("db", flagOutDB))
		return nil
	},
}

func init() { RootCmd.AddCommand(loadCmd) }

func openSQLite(path string) (*sql.DB, error) {
	// Best-effort pragmas for bulk load; we'll set more after open
	dsn := fmt.Sprintf("file:%s?_pragma=journal_mode(MEMORY)&_pragma=synchronous(OFF)&_pragma=temp_store(MEMORY)", path)
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
			cnpj_basico TEXT,
			cnpj_ordem TEXT,
			cnpj_dv TEXT,
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
			data_situacao_especial TEXT
		);`,
		`CREATE TABLE IF NOT EXISTS cnaes (
			codigo TEXT PRIMARY KEY,
			descricao TEXT
		);`,
		`CREATE TABLE IF NOT EXISTS municipios (
			codigo TEXT PRIMARY KEY,
			descricao TEXT
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
	stmts := []string{
		`CREATE INDEX IF NOT EXISTS idx_estab_cnpj ON estabelecimentos(cnpj_basico);`,
		`CREATE INDEX IF NOT EXISTS idx_estab_mun ON estabelecimentos(municipio, uf);`,
		`CREATE INDEX IF NOT EXISTS idx_estab_cnae ON estabelecimentos(cnae_fiscal_principal);`,
	}
	for _, q := range stmts {
		if err := exec(ctx, db, q); err != nil {
			return err
		}
	}
	return nil
}

func loadAll(ctx context.Context, db *sql.DB, dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	// We accept both upper/lower .csv
	csvRe := regexp.MustCompile(`(?i)\.csv$`)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !csvRe.MatchString(name) {
			continue
		}
		path := filepath.Join(dir, name)
		lower := strings.ToLower(name)
		switch {
		case strings.HasPrefix(lower, "empresas"):
			if err := loadEmpresas(ctx, db, path); err != nil {
				return err
			}
		case strings.HasPrefix(lower, "estabelecimentos"):
			if err := loadEstabelecimentos(ctx, db, path); err != nil {
				return err
			}
		case strings.HasPrefix(lower, "cnaes"):
			if err := loadCnaes(ctx, db, path); err != nil {
				return err
			}
		case strings.HasPrefix(lower, "municipios"):
			if err := loadMunicipios(ctx, db, path); err != nil {
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
	place := make([]string, len(cols))
	for i := range cols {
		place[i] = "?"
	}
	stmt, err := tx.PrepareContext(ctx, fmt.Sprintf("INSERT OR REPLACE INTO %s(%s) VALUES(%s)", table, strings.Join(cols, ","), strings.Join(place, ",")))
	if err != nil {
		tx.Rollback()
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
		tx.Rollback()
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
	cols := []string{"cnpj_basico", "razao_social", "natureza_juridica", "qualificacao_responsavel", "capital_social", "porte_empresa", "ente_federativo"}
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
		"cnpj_basico", "cnpj_ordem", "cnpj_dv", "identificador_matriz_filial", "nome_fantasia", "situacao_cadastral", "data_situacao_cadastral", "motivo_situacao_cadastral", "nome_cidade_exterior", "pais", "data_inicio_atividade", "cnae_fiscal_principal", "cnae_fiscal_secundaria", "tipo_logradouro", "logradouro", "numero", "complemento", "bairro", "cep", "uf", "municipio", "ddd1", "telefone1", "ddd2", "telefone2", "ddd_fax", "fax", "correio_eletronico", "situacao_especial", "data_situacao_especial",
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

func joinRest(ss []string) string {
	for i := range ss {
		ss[i] = strings.TrimSpace(ss[i])
	}
	return strings.Join(ss, ";")
}
