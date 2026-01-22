#!/bin/bash
# Script para verificar qual é a última versão disponível do CNPJ
# sem fazer download completo

echo "🔍 Verificando última versão disponível do CNPJ..."
echo ""

# Gerar lista de meses para verificar (atual + 6 anteriores)
current_month=$(date +%Y-%m)
months_to_check=()

# Adicionar mês atual
months_to_check+=("$current_month")

# Adicionar 6 meses anteriores
for i in {1..6}; do
    prev_month=$(date -d "$current_month-01 -$i month" +%Y-%m 2>/dev/null || date -v-${i}m -j -f "%Y-%m" "$current_month" +%Y-%m 2>/dev/null)
    if [ -n "$prev_month" ]; then
        months_to_check+=("$prev_month")
    fi
done

echo "Verificando disponibilidade dos últimos 7 meses:"
echo ""

# URL base da Receita Federal (CORRIGIDA!)
BASE_URL="https://arquivos.receitafederal.gov.br/dados/cnpj/dados_abertos_cnpj"

LATEST_FOUND=""

for ym in "${months_to_check[@]}"; do
    # Construir URL com o ano-mês e testar arquivo Empresas0.zip
    test_url="$BASE_URL/$ym/Empresas0.zip"

    echo -n "  $ym: "

    # Fazer HEAD request para verificar se existe (timeout de 10 segundos)
    if curl -s -f -I --max-time 10 "$test_url" > /dev/null 2>&1; then
        echo "✅ DISPONÍVEL"
        if [ -z "$LATEST_FOUND" ]; then
            LATEST_FOUND="$ym"
        fi
    else
        echo "❌ não disponível"
    fi
done

echo ""

if [ -n "$LATEST_FOUND" ]; then
    echo "============================================"
    echo "📅 Última versão disponível: $LATEST_FOUND"
    echo "============================================"
    echo ""
    echo "Para baixar e processar, execute:"
    echo ""
    echo "  ./bcd download --ym $LATEST_FOUND --workdir /tmp/cnpj_rf"
    echo "  ./bcd extract --ym $LATEST_FOUND --workdir /tmp/cnpj_rf"
    echo "  ./bcd load --ym $LATEST_FOUND --workdir /tmp/cnpj_rf --out ./cnpj.sqlite"
    echo ""
    echo "Ou use o script automatizado:"
    echo ""
    echo "  ./scripts/download_latest.sh"
    echo ""
else
    echo "❌ Nenhuma versão disponível encontrada!"
    echo ""
    echo "Isso pode indicar:"
    echo "  1. URL da Receita Federal mudou"
    echo "  2. Problema de conectividade"
    echo ""
    echo "Verifique manualmente em:"
    echo "  https://arquivos.receitafederal.gov.br/dados/cnpj/dados_abertos_cnpj/"
fi
