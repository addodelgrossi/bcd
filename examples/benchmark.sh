#!/bin/bash
# Script de benchmark para a API CNPJ
# Requer: apache-bench (ab) ou wrk

set -e

API_URL="${API_URL:-http://localhost:8080}"
DURATION="${DURATION:-30}"
CONNECTIONS="${CONNECTIONS:-100}"

echo "========================================="
echo "CNPJ API Benchmark"
echo "========================================="
echo "URL: $API_URL"
echo "Duration: ${DURATION}s"
echo "Concurrent connections: $CONNECTIONS"
echo "========================================="
echo ""

# Verificar se a API está rodando
if ! curl -s "$API_URL/health" > /dev/null; then
    echo "❌ API não está respondendo em $API_URL"
    exit 1
fi

echo "✅ API está online"
echo ""

# Função para executar benchmark com ab
benchmark_ab() {
    local endpoint=$1
    local description=$2

    echo "----------------------------------------"
    echo "Testing: $description"
    echo "Endpoint: $endpoint"
    echo "----------------------------------------"

    ab -n 10000 -c $CONNECTIONS -k "$API_URL$endpoint" 2>&1 | grep -E "(Requests per second|Time per request|Transfer rate|Failed requests|Percentage)"

    echo ""
}

# Função para executar benchmark com wrk (se disponível)
benchmark_wrk() {
    local endpoint=$1
    local description=$2

    echo "----------------------------------------"
    echo "Testing: $description"
    echo "Endpoint: $endpoint"
    echo "----------------------------------------"

    wrk -t4 -c$CONNECTIONS -d${DURATION}s "$API_URL$endpoint"

    echo ""
}

# Detectar ferramenta disponível
if command -v wrk &> /dev/null; then
    echo "Usando wrk para benchmark..."
    TOOL="wrk"
    BENCH_FUNC=benchmark_wrk
elif command -v ab &> /dev/null; then
    echo "Usando Apache Bench (ab) para benchmark..."
    TOOL="ab"
    BENCH_FUNC=benchmark_ab
else
    echo "❌ Nenhuma ferramenta de benchmark encontrada!"
    echo "Instale 'wrk' ou 'apache-bench' (ab):"
    echo ""
    echo "  Ubuntu/Debian: sudo apt-get install apache2-utils"
    echo "  macOS: brew install wrk"
    echo "  Fedora: sudo dnf install httpd-tools"
    exit 1
fi

echo ""

# Testes
$BENCH_FUNC "/health" "Healthcheck"
$BENCH_FUNC "/api/empresa?cnpj=00000000" "Busca por CNPJ (empresa pequena)"
$BENCH_FUNC "/api/municipio?uf=SP&municipio=7107" "Busca por município (São Paulo)"

echo "========================================="
echo "Benchmark completo!"
echo "========================================="

# Análise de performance esperada
echo ""
echo "📊 Metas de Performance:"
echo "  - Throughput: > 1000 req/s"
echo "  - Latência p50: < 10ms"
echo "  - Latência p95: < 50ms"
echo "  - Latência p99: < 100ms"
echo "  - Taxa de erro: 0%"
echo ""
echo "⚡ Dicas para melhorar performance:"
echo "  1. Use SSD/NVMe para o arquivo .sqlite"
echo "  2. Aumente PRAGMA cache_size no banco"
echo "  3. Use cache de resultados (Redis/Memcached)"
echo "  4. Configure connection pooling adequadamente"
echo "  5. Use CDN para servir dados estáticos"
