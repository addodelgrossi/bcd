#!/bin/bash
# Script para baixar a última versão disponível do CNPJ
# Tenta o mês atual e, se falhar, tenta os 3 meses anteriores
#
# URL base: https://arquivos.receitafederal.gov.br/dados/cnpj/dados_abertos_cnpj/YYYY-MM/

set -e

WORKDIR="${WORKDIR:-/tmp/cnpj_rf}"
OUTPUT="${OUTPUT:-./cnpj.sqlite}"
BCD="${BCD:-./bcd}"

echo "============================================"
echo "CNPJ - Download da Última Versão Disponível"
echo "============================================"
echo ""

# Função para tentar baixar um ano-mês específico
try_download() {
    local ym=$1
    echo "🔍 Tentando baixar $ym..."

    if $BCD download --ym "$ym" --workdir "$WORKDIR" 2>&1 | tee /tmp/bcd_download.log; then
        # Verificar se realmente baixou (não foi 404)
        if ! grep -q "404\|not found\|error" /tmp/bcd_download.log; then
            echo "✅ Download bem-sucedido para $ym!"
            return 0
        fi
    fi

    echo "❌ Versão $ym não disponível"
    return 1
}

# Gerar lista de meses para tentar (atual + 3 anteriores)
current_month=$(date +%Y-%m)
months_to_try=()

# Adicionar mês atual
months_to_try+=("$current_month")

# Adicionar 3 meses anteriores
for i in {1..3}; do
    prev_month=$(date -d "$current_month-01 -$i month" +%Y-%m 2>/dev/null || date -v-${i}m -j -f "%Y-%m" "$current_month" +%Y-%m 2>/dev/null)
    if [ -n "$prev_month" ]; then
        months_to_try+=("$prev_month")
    fi
done

echo "Tentando os seguintes meses (do mais recente ao mais antigo):"
printf '  - %s\n' "${months_to_try[@]}"
echo ""

# Tentar cada mês
FOUND_VERSION=""
for ym in "${months_to_try[@]}"; do
    if try_download "$ym"; then
        FOUND_VERSION="$ym"
        break
    fi
    echo ""
done

if [ -z "$FOUND_VERSION" ]; then
    echo "❌ Não foi possível encontrar nenhuma versão disponível!"
    echo "Meses testados: ${months_to_try[*]}"
    echo ""
    echo "Possíveis causas:"
    echo "  1. A Receita Federal ainda não publicou os dados recentes"
    echo "  2. Problema de conectividade"
    echo "  3. URL do portal mudou"
    echo ""
    echo "Tente verificar manualmente em:"
    echo "  https://dados.gov.br/dados/conjuntos-dados/cadastro-nacional-da-pessoa-juridica---cnpj"
    exit 1
fi

echo ""
echo "============================================"
echo "Versão encontrada: $FOUND_VERSION"
echo "============================================"
echo ""

# Extrair arquivos
echo "📦 Extraindo arquivos..."
$BCD extract --ym "$FOUND_VERSION" --workdir "$WORKDIR"

echo ""
echo "============================================"
echo "Carregando no SQLite..."
echo "============================================"
echo "Isso pode demorar 60-90 minutos."
echo ""

# Carregar no SQLite
$BCD load --ym "$FOUND_VERSION" --workdir "$WORKDIR" --out "$OUTPUT"

echo ""
echo "============================================"
echo "✅ Processo concluído com sucesso!"
echo "============================================"
echo ""
echo "Versão: $FOUND_VERSION"
echo "Banco gerado: $OUTPUT"
echo ""
echo "Próximos passos:"
echo "  1. Validar o banco:"
echo "     sqlite3 $OUTPUT 'SELECT COUNT(*) FROM estabelecimentos;'"
echo ""
echo "  2. Verificar tamanho:"
echo "     du -h $OUTPUT"
echo ""
echo "  3. Deploy em produção:"
echo "     docker run -v $OUTPUT:/data/cnpj.sqlite:ro sua-api"
echo ""
