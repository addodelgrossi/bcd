# Guia de Performance e Deployment

## Melhorias Implementadas

### 1. Schema Otimizado

#### PRIMARY KEY Composta em Estabelecimentos
```sql
PRIMARY KEY (cnpj_basico, cnpj_ordem, cnpj_dv)
```
**Benefício**: Garante unicidade e acelera JOINs com a tabela `empresas`.

### 2. Índices Estratégicos

Foram adicionados 7 índices baseados nos padrões de consulta mais comuns:

```sql
-- Busca por CNPJ (JOIN com empresas)
CREATE INDEX idx_estab_cnpj ON estabelecimentos(cnpj_basico);

-- Busca geográfica (cidade + estado)
CREATE INDEX idx_estab_mun_uf ON estabelecimentos(municipio, uf);

-- Busca por estado
CREATE INDEX idx_estab_uf ON estabelecimentos(uf);

-- Busca por CNAE (atividade econômica)
CREATE INDEX idx_estab_cnae ON estabelecimentos(cnae_fiscal_principal);

-- Filtro por situação cadastral (ativo/inativo)
CREATE INDEX idx_estab_situacao ON estabelecimentos(situacao_cadastral);

-- Busca por CEP
CREATE INDEX idx_estab_cep ON estabelecimentos(cep);

-- Agregações matriz/filial
CREATE INDEX idx_estab_matriz_filial ON estabelecimentos(identificador_matriz_filial);
```

### 3. Otimização Pós-Carga

Após a carga de dados, o banco é otimizado:

- **ANALYZE**: Atualiza estatísticas do query planner para escolher os melhores planos de execução
- **VACUUM**: Desfragmenta o arquivo e remove espaço não utilizado (pode reduzir 20-30% do tamanho)

### 4. PRAGMAs de Leitura

Após o load, o banco é reconfigurado para otimizar **leitura** (sua API):

```sql
PRAGMA journal_mode=WAL;        -- Write-Ahead Logging (reads não bloqueiam)
PRAGMA synchronous=NORMAL;      -- Balanço segurança/performance
PRAGMA mmap_size=268435456;     -- 256MB memory-mapped I/O
PRAGMA temp_store=MEMORY;       -- Temporários em RAM
```

**Nota:** O `cache_size` é configurado durante o load (64MB) e persiste no arquivo. A API pode ajustar via connection string se necessário.

## Deployment: Onde Colocar o SQLite?

### ✅ Opção 1: Dentro do Container (RECOMENDADO para Read-Only)

**Vantagens:**
- Startup instantâneo
- Zero latência de rede
- Imagem self-contained
- Ideal para deploys imutáveis (Kubernetes, Cloud Run, Lambda)

**Como fazer:**
```dockerfile
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o api .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/

# Copiar o SQLite para dentro da imagem
COPY --from=builder /app/api .
COPY cnpj.sqlite /data/cnpj.sqlite

# Montar /data como tmpfs para cache (opcional)
VOLUME /tmp

CMD ["./api"]
```

**Configuração da API em Go:**
```go
db, err := sql.Open("sqlite", "file:/data/cnpj.sqlite?mode=ro&cache=shared")
if err != nil {
    log.Fatal(err)
}
db.SetMaxOpenConns(10) // Múltiplas reads concorrentes com WAL
```

**Tamanho esperado da imagem:**
- Base Alpine: ~5MB
- Binário Go: ~10-15MB
- SQLite (CNPJ completo): ~15-25GB
- **Total: ~15-25GB** (imagem grande, mas aceitável para read-only)

---

### ✅ Opção 2: Volume Persistente (Para Atualizações Mensais)

Se você precisa atualizar o banco mensalmente **sem rebuildar a imagem**:

**Docker Compose:**
```yaml
version: '3.8'
services:
  api:
    image: sua-api:latest
    volumes:
      - ./cnpj.sqlite:/data/cnpj.sqlite:ro  # Read-only mount
    environment:
      - DB_PATH=/data/cnpj.sqlite
```

**Kubernetes:**
```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: cnpj-db
spec:
  accessModes:
    - ReadOnlyMany
  resources:
    requests:
      storage: 30Gi
---
apiVersion: apps/v1
kind: Deployment
spec:
  template:
    spec:
      volumes:
        - name: db
          persistentVolumeClaim:
            claimName: cnpj-db
            readOnly: true
      containers:
        - name: api
          volumeMounts:
            - name: db
              mountPath: /data
              readOnly: true
```

---

### ✅ Opção 3: Disco SSD Local (Bare Metal/VPS)

Para servidores dedicados:

**Localização ideal:**
```bash
# NVMe SSD (melhor performance)
/mnt/nvme/cnpj.sqlite

# SSD SATA (bom)
/mnt/ssd/cnpj.sqlite

# ❌ Evite HDD tradicional (latência alta)
```

**Permissões:**
```bash
chown api-user:api-user /mnt/nvme/cnpj.sqlite
chmod 444 /mnt/nvme/cnpj.sqlite  # Read-only
```

---

### ❌ Opção NÃO Recomendada: Network Storage

**Evite:**
- NFS
- Network-attached storage
- Amazon EFS (latência alta)

**Motivo:** SQLite precisa de I/O rápido e consistente. Network storage adiciona latência (10-100ms) que mata a performance.

---

## Performance Esperada da API

Com as otimizações implementadas:

### Queries Simples (sem JOIN)
```sql
-- Buscar empresa por CNPJ
SELECT * FROM empresas WHERE cnpj_basico = '12345678';
```
**Latência esperada:** < 1ms

### Queries Complexas (com JOIN)
```sql
-- Buscar empresa + estabelecimento
SELECT e.razao_social, est.nome_fantasia, est.uf
FROM empresas e
JOIN estabelecimentos est ON e.cnpj_basico = est.cnpj_basico
WHERE e.cnpj_basico = '12345678';
```
**Latência esperada:** 1-5ms

### Agregações Geográficas
```sql
-- Contar empresas por cidade
SELECT municipio, COUNT(*) as total
FROM estabelecimentos
WHERE uf = 'SP' AND situacao_cadastral = '2'
GROUP BY municipio
ORDER BY total DESC
LIMIT 20;
```
**Latência esperada:** 50-200ms (depende do volume)

---

## Recomendações para a API Golang

### 1. Connection Pool
```go
db.SetMaxOpenConns(10)        // WAL permite múltiplos readers
db.SetMaxIdleConns(5)
db.SetConnMaxLifetime(time.Hour)
```

### 2. Prepared Statements
```go
// Cachear prepared statements para queries frequentes
stmt, _ := db.Prepare("SELECT * FROM empresas WHERE cnpj_basico = ?")
defer stmt.Close()
```

### 3. Context com Timeout
```go
ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
defer cancel()
rows, err := db.QueryContext(ctx, query, args...)
```

### 4. Cache de Resultados (Opcional)
Para queries muito frequentes, use cache in-memory:
- Redis
- go-cache
- bigcache

---

## Monitoramento

### Métricas Importantes
```sql
-- Verificar uso de índices
EXPLAIN QUERY PLAN SELECT ...;

-- Estatísticas do cache
PRAGMA cache_hit_rate;

-- Tamanho do arquivo
PRAGMA page_count;
PRAGMA page_size;
```

### Logs da API
Monitore:
- Latência p50, p95, p99
- Query mais lentas (> 100ms)
- Uso de CPU/memória

---

## Atualizações Mensais

### Estratégia Blue-Green

1. Gerar novo `cnpj_v2.sqlite`
2. Testar em staging
3. Swap atômico:
```bash
# Opção 1: Symlink
ln -sfn /data/cnpj_v2.sqlite /data/cnpj.sqlite

# Opção 2: Atomic rename
mv /data/cnpj.sqlite /data/cnpj.old.sqlite
mv /data/cnpj_v2.sqlite /data/cnpj.sqlite
```
4. Restart graceful da API (sem downtime com load balancer)

---

## Troubleshooting

### Queries Lentas
```bash
# Verificar se índices estão sendo usados
sqlite3 cnpj.sqlite "EXPLAIN QUERY PLAN SELECT ..."
```

### Banco Corrompido
```bash
# Verificar integridade
sqlite3 cnpj.sqlite "PRAGMA integrity_check;"
```

### Alto Uso de Memória
```sql
-- Reduzir cache
PRAGMA cache_size=-32000;  -- 32MB ao invés de 64MB
```

---

## Benchmark Sugerido

Antes de ir pra produção, faça load testing:

```bash
# Apache Bench
ab -n 10000 -c 100 http://localhost:8080/api/empresas/12345678

# Vegeta
echo "GET http://localhost:8080/api/empresas/12345678" | vegeta attack -duration=30s -rate=1000 | vegeta report
```

**Meta de performance:**
- p50 < 10ms
- p95 < 50ms
- p99 < 100ms
- Throughput > 1000 req/s (em hardware moderno)
