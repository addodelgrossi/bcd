package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	_ "modernc.org/sqlite"
)

// Estruturas de dados
type Empresa struct {
	CNPJBasico       string `json:"cnpj_basico"`
	RazaoSocial      string `json:"razao_social"`
	NaturezaJuridica string `json:"natureza_juridica"`
	QualificacaoResp string `json:"qualificacao_responsavel"`
	CapitalSocial    string `json:"capital_social"`
	PorteEmpresa     string `json:"porte_empresa"`
	EnteFederativo   string `json:"ente_federativo"`
}

type Estabelecimento struct {
	CNPJBasico            string `json:"cnpj_basico"`
	CNPJOrdem             string `json:"cnpj_ordem"`
	CNPJDV                string `json:"cnpj_dv"`
	CNPJCompleto          string `json:"cnpj_completo"` // Calculado
	IdentificadorMF       string `json:"identificador_matriz_filial"`
	NomeFantasia          string `json:"nome_fantasia"`
	SituacaoCadastral     string `json:"situacao_cadastral"`
	DataSituacaoCadastral string `json:"data_situacao_cadastral"`
	CNAEFiscalPrincipal   string `json:"cnae_fiscal_principal"`
	Logradouro            string `json:"logradouro"`
	Numero                string `json:"numero"`
	Bairro                string `json:"bairro"`
	CEP                   string `json:"cep"`
	UF                    string `json:"uf"`
	Municipio             string `json:"municipio"`
	Telefone1             string `json:"telefone1"`
	CorreioEletronico     string `json:"correio_eletronico"`
}

type EmpresaCompleta struct {
	Empresa         Empresa         `json:"empresa"`
	Estabelecimento Estabelecimento `json:"estabelecimento"`
}

// Database wrapper
type DB struct {
	conn *sql.DB
}

// NewDB inicializa conexão otimizada para leitura
func NewDB(dbPath string) (*DB, error) {
	// Connection string com PRAGMAs otimizados para read-only
	dsn := fmt.Sprintf("file:%s?mode=ro&cache=shared&_pragma=journal_mode(WAL)&_pragma=query_only(1)", dbPath)

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Connection pool otimizado para leituras concorrentes
	db.SetMaxOpenConns(10) // WAL permite múltiplos readers
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(time.Hour)

	// Testar conexão
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Printf("Database connection established: %s", dbPath)
	return &DB{conn: db}, nil
}

// BuscarEmpresaPorCNPJ retorna empresa + estabelecimento principal
func (db *DB) BuscarEmpresaPorCNPJ(ctx context.Context, cnpjBasico string) (*EmpresaCompleta, error) {
	query := `
		SELECT
			e.cnpj_basico, e.razao_social, e.natureza_juridica,
			e.qualificacao_responsavel, e.capital_social, e.porte_empresa, e.ente_federativo,
			est.cnpj_ordem, est.cnpj_dv, est.identificador_matriz_filial, est.nome_fantasia,
			est.situacao_cadastral, est.data_situacao_cadastral, est.cnae_fiscal_principal,
			est.logradouro, est.numero, est.bairro, est.cep, est.uf, est.municipio,
			est.telefone1, est.correio_eletronico
		FROM empresas e
		JOIN estabelecimentos est ON e.cnpj_basico = est.cnpj_basico
		WHERE e.cnpj_basico = ?
			AND est.identificador_matriz_filial = '1'  -- Apenas matriz
		LIMIT 1
	`

	var result EmpresaCompleta
	var emp Empresa
	var est Estabelecimento

	err := db.conn.QueryRowContext(ctx, query, cnpjBasico).Scan(
		&emp.CNPJBasico, &emp.RazaoSocial, &emp.NaturezaJuridica,
		&emp.QualificacaoResp, &emp.CapitalSocial, &emp.PorteEmpresa, &emp.EnteFederativo,
		&est.CNPJOrdem, &est.CNPJDV, &est.IdentificadorMF, &est.NomeFantasia,
		&est.SituacaoCadastral, &est.DataSituacaoCadastral, &est.CNAEFiscalPrincipal,
		&est.Logradouro, &est.Numero, &est.Bairro, &est.CEP, &est.UF, &est.Municipio,
		&est.Telefone1, &est.CorreioEletronico,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}

	// Calcular CNPJ completo
	est.CNPJBasico = emp.CNPJBasico
	est.CNPJCompleto = fmt.Sprintf("%s%s%s", emp.CNPJBasico, est.CNPJOrdem, est.CNPJDV)

	result.Empresa = emp
	result.Estabelecimento = est
	return &result, nil
}

// BuscarPorMunicipio retorna empresas ativas em uma cidade
func (db *DB) BuscarPorMunicipio(ctx context.Context, uf, municipio string, limit int) ([]Estabelecimento, error) {
	query := `
		SELECT
			cnpj_basico, cnpj_ordem, cnpj_dv, identificador_matriz_filial, nome_fantasia,
			situacao_cadastral, cnae_fiscal_principal, logradouro, numero, bairro, cep, uf, municipio
		FROM estabelecimentos
		WHERE uf = ?
			AND municipio = ?
			AND situacao_cadastral = '2'  -- Apenas ativos
		LIMIT ?
	`

	rows, err := db.conn.QueryContext(ctx, query, uf, municipio, limit)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var result []Estabelecimento
	for rows.Next() {
		var est Estabelecimento
		err := rows.Scan(
			&est.CNPJBasico, &est.CNPJOrdem, &est.CNPJDV, &est.IdentificadorMF, &est.NomeFantasia,
			&est.SituacaoCadastral, &est.CNAEFiscalPrincipal, &est.Logradouro, &est.Numero,
			&est.Bairro, &est.CEP, &est.UF, &est.Municipio,
		)
		if err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}
		est.CNPJCompleto = fmt.Sprintf("%s%s%s", est.CNPJBasico, est.CNPJOrdem, est.CNPJDV)
		result = append(result, est)
	}

	return result, rows.Err()
}

// HTTP Handlers
type API struct {
	db *DB
}

func (api *API) handleBuscarEmpresa(w http.ResponseWriter, r *http.Request) {
	cnpj := r.URL.Query().Get("cnpj")
	if cnpj == "" {
		http.Error(w, "parâmetro 'cnpj' obrigatório", http.StatusBadRequest)
		return
	}

	// Timeout de 3 segundos por query
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	result, err := api.db.BuscarEmpresaPorCNPJ(ctx, cnpj)
	if err != nil {
		log.Printf("ERROR: %v", err)
		http.Error(w, "erro ao buscar empresa", http.StatusInternalServerError)
		return
	}

	if result == nil {
		http.Error(w, "empresa não encontrada", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (api *API) handleBuscarPorMunicipio(w http.ResponseWriter, r *http.Request) {
	uf := r.URL.Query().Get("uf")
	municipio := r.URL.Query().Get("municipio")

	if uf == "" || municipio == "" {
		http.Error(w, "parâmetros 'uf' e 'municipio' obrigatórios", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	result, err := api.db.BuscarPorMunicipio(ctx, uf, municipio, 100)
	if err != nil {
		log.Printf("ERROR: %v", err)
		http.Error(w, "erro ao buscar empresas", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (api *API) handleHealth(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 1*time.Second)
	defer cancel()

	if err := api.db.conn.PingContext(ctx); err != nil {
		http.Error(w, "database unhealthy", http.StatusServiceUnavailable)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
}

func main() {
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "./cnpj.sqlite"
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Inicializar database
	db, err := NewDB(dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.conn.Close()

	// Inicializar API
	api := &API{db: db}

	// Rotas
	http.HandleFunc("/health", api.handleHealth)
	http.HandleFunc("/api/empresa", api.handleBuscarEmpresa)
	http.HandleFunc("/api/municipio", api.handleBuscarPorMunicipio)

	// Iniciar servidor
	addr := ":" + port
	log.Printf("Starting server on %s", addr)
	log.Printf("Routes:")
	log.Printf("  GET /health")
	log.Printf("  GET /api/empresa?cnpj=12345678")
	log.Printf("  GET /api/municipio?uf=SP&municipio=7107")

	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
