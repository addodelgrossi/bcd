#!/bin/bash
# Pipeline completo: Download → Extract → Load → Upload S3
# Este script automatiza todo o processo mensal de atualização

set -e

# Configurações
YEAR_MONTH="${YEAR_MONTH:-}"
WORKDIR="${WORKDIR:-/tmp/cnpj_rf}"
OUTPUT="${OUTPUT:-./cnpj.sqlite}"
S3_BUCKET="${S3_BUCKET:-}"
S3_PREFIX="${S3_PREFIX:-cnpj}"
BCD="${BCD:-./bcd}"

# Cores
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo "============================================"
echo "Pipeline Completo CNPJ"
echo "============================================"
echo ""

# Validar configurações
if [ -z "$YEAR_MONTH" ]; then
    echo -e "${YELLOW}YEAR_MONTH não definido, tentando detectar automaticamente...${NC}"
    # Usar script de download automático
    YEAR_MONTH="auto"
fi

if [ -z "$S3_BUCKET" ]; then
    echo -e "${YELLOW}⚠️  S3_BUCKET não definido. Upload para S3 será PULADO.${NC}"
    echo "Para habilitar upload, defina: S3_BUCKET=meu-bucket"
    SKIP_S3=1
fi

echo "Configuração:"
echo "  Ano-Mês:    ${YEAR_MONTH}"
echo "  Workdir:    ${WORKDIR}"
echo "  Output:     ${OUTPUT}"
echo "  S3 Bucket:  ${S3_BUCKET:-<não configurado>}"
echo ""

# ==================================================
# ETAPA 1: Download e Extract
# ==================================================

if [ "$YEAR_MONTH" = "auto" ]; then
    echo -e "${GREEN}[1/4] Download e Extract (auto-detectando versão)...${NC}"
    ./scripts/download_latest.sh

    # Detectar qual versão foi baixada
    YEAR_MONTH=$(ls -1 "$WORKDIR/extracted/" | grep -E 'Empresas[0-9]' | head -1 | grep -oE '[0-9]{4}-[0-9]{2}' || echo "")
    if [ -z "$YEAR_MONTH" ]; then
        # Fallback: tentar mês atual
        YEAR_MONTH=$(date +%Y-%m)
    fi
    echo "  Detectado: $YEAR_MONTH"
else
    echo -e "${GREEN}[1/4] Download...${NC}"
    $BCD download --ym "$YEAR_MONTH" --workdir "$WORKDIR"

    echo ""
    echo -e "${GREEN}[2/4] Extract...${NC}"
    $BCD extract --ym "$YEAR_MONTH" --workdir "$WORKDIR"
fi

# ==================================================
# ETAPA 2: Load (com otimizações)
# ==================================================

echo ""
echo -e "${GREEN}[3/4] Load (isso pode demorar 60-90 minutos)...${NC}"
START_TIME=$(date +%s)

# Adicionar timestamp ao nome do arquivo
OUTPUT_VERSIONED="${OUTPUT%.sqlite}_${YEAR_MONTH}.sqlite"

$BCD load --ym "$YEAR_MONTH" --workdir "$WORKDIR" --out "$OUTPUT_VERSIONED"

END_TIME=$(date +%s)
DURATION=$((END_TIME - START_TIME))
DURATION_MIN=$((DURATION / 60))

echo ""
echo -e "${GREEN}✅ Load concluído em ${DURATION_MIN} minutos!${NC}"

# Criar link simbólico para arquivo sem versão
ln -sfn "$OUTPUT_VERSIONED" "$OUTPUT"
echo "  Arquivo principal: $OUTPUT_VERSIONED"
echo "  Link simbólico:    $OUTPUT -> $OUTPUT_VERSIONED"

# ==================================================
# ETAPA 3: Upload para S3 (opcional)
# ==================================================

if [ "${SKIP_S3:-0}" = "0" ]; then
    echo ""
    echo -e "${GREEN}[4/4] Upload para S3...${NC}"

    S3_BUCKET="$S3_BUCKET" \
    S3_PREFIX="$S3_PREFIX" \
    SQLITE_FILE="$OUTPUT_VERSIONED" \
    VERSION="$YEAR_MONTH" \
    SKIP_CONFIRM="${SKIP_CONFIRM:-0}" \
    CREATE_LATEST_LINK=1 \
    ./scripts/upload_to_s3.sh
else
    echo ""
    echo -e "${YELLOW}[4/4] Upload para S3: PULADO${NC}"
fi

# ==================================================
# ETAPA 4: Resumo Final
# ==================================================

echo ""
echo "============================================"
echo -e "${GREEN}✨ Pipeline Concluído!${NC}"
echo "============================================"
echo ""
echo "Resumo:"
echo "  Versão:          $YEAR_MONTH"
echo "  Banco gerado:    $OUTPUT_VERSIONED"
echo "  Tamanho:         $(du -h "$OUTPUT_VERSIONED" | cut -f1)"
echo "  Tempo total:     ${DURATION_MIN} minutos"

if [ "${SKIP_S3:-0}" = "0" ]; then
    echo "  Upload S3:       ✅ Concluído"
    echo "  S3 Path:         s3://${S3_BUCKET}/${S3_PREFIX}/${YEAR_MONTH}/cnpj.sqlite"
else
    echo "  Upload S3:       ⏭️  Pulado"
fi

echo ""
echo "Próximos passos:"
echo "  1. Validar banco:"
echo "     sqlite3 $OUTPUT_VERSIONED 'SELECT COUNT(*) FROM estabelecimentos;'"
echo ""
echo "  2. Deploy para produção:"
if [ "${SKIP_S3:-0}" = "0" ]; then
    echo "     aws s3 cp s3://${S3_BUCKET}/${S3_PREFIX}/${YEAR_MONTH}/cnpj.sqlite /data/"
else
    echo "     docker run -v $OUTPUT_VERSIONED:/data/cnpj.sqlite:ro sua-api"
fi
echo ""
echo "  3. Restart da API:"
echo "     docker restart cnpj-api"
echo ""
