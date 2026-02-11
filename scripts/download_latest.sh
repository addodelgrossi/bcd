#!/bin/bash
# Script para baixar, extrair e carregar a última versão do CNPJ
# Suporta MODE=zip (default) ou MODE=tar

set -e

WORKDIR="${WORKDIR:-/tmp/cnpj_rf}"
OUTPUT="${OUTPUT:-./cnpj.sqlite}"
BCD="${BCD:-./bcd}"
MODE="${MODE:-zip}"

echo "============================================"
echo "CNPJ - Download da Última Versão Disponível"
echo "============================================"
echo ""
echo "Modo: $MODE"
echo "Workdir: $WORKDIR"
echo "Output: $OUTPUT"
echo ""

if [ "$MODE" = "tar" ]; then
    echo "Baixando cnpj.tar.gz (~60GB)..."
    $BCD download --mode tar --workdir "$WORKDIR"

    echo ""
    echo "Extraindo tar.gz..."
    $BCD extract --mode tar --workdir "$WORKDIR"
else
    echo "Baixando ZIPs (auto-detectando mês mais recente)..."
    $BCD download --workdir "$WORKDIR"

    echo ""
    echo "Extraindo ZIPs..."
    $BCD extract --workdir "$WORKDIR"
fi

echo ""
echo "============================================"
echo "Carregando no SQLite..."
echo "============================================"
echo "Isso pode demorar 60-90 minutos."
echo ""

$BCD load --workdir "$WORKDIR" --out "$OUTPUT"

echo ""
echo "============================================"
echo "Processo concluído com sucesso!"
echo "============================================"
echo ""
echo "Banco gerado: $OUTPUT"
echo ""
echo "Próximos passos:"
echo "  1. Validar o banco:"
echo "     sqlite3 $OUTPUT 'SELECT COUNT(*) FROM estabelecimentos;'"
echo ""
echo "  2. Verificar tamanho:"
echo "     du -h $OUTPUT"
echo ""
