#!/bin/bash
# Script para fazer upload do banco SQLite gerado para S3
# Requer: AWS CLI configurado (aws configure)

set -e

# Configurações (podem ser sobrescritas via env vars)
S3_BUCKET="${S3_BUCKET:-}"
S3_PREFIX="${S3_PREFIX:-cnpj}"
SQLITE_FILE="${SQLITE_FILE:-./cnpj.sqlite}"
VERSION="${VERSION:-}"
STORAGE_CLASS="${STORAGE_CLASS:-INTELLIGENT_TIERING}"

# Cores para output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "============================================"
echo "Upload do Banco CNPJ para S3"
echo "============================================"
echo ""

# Validações
if [ -z "$S3_BUCKET" ]; then
    echo -e "${RED}❌ Erro: S3_BUCKET não definido!${NC}"
    echo ""
    echo "Uso:"
    echo "  S3_BUCKET=meu-bucket ./scripts/upload_to_s3.sh"
    echo ""
    echo "Ou com todas as opções:"
    echo "  S3_BUCKET=meu-bucket \\"
    echo "  S3_PREFIX=cnpj \\"
    echo "  SQLITE_FILE=./cnpj.sqlite \\"
    echo "  VERSION=2026-01 \\"
    echo "  STORAGE_CLASS=INTELLIGENT_TIERING \\"
    echo "  ./scripts/upload_to_s3.sh"
    exit 1
fi

if [ ! -f "$SQLITE_FILE" ]; then
    echo -e "${RED}❌ Erro: Arquivo não encontrado: $SQLITE_FILE${NC}"
    exit 1
fi

# Verificar se AWS CLI está instalado
if ! command -v aws &> /dev/null; then
    echo -e "${RED}❌ Erro: AWS CLI não está instalado!${NC}"
    echo ""
    echo "Instale com:"
    echo "  macOS:   brew install awscli"
    echo "  Ubuntu:  sudo apt install awscli"
    echo "  pip:     pip install awscli"
    exit 1
fi

# Verificar credenciais AWS
if ! aws sts get-caller-identity &> /dev/null; then
    echo -e "${RED}❌ Erro: AWS CLI não está configurado!${NC}"
    echo ""
    echo "Configure com:"
    echo "  aws configure"
    exit 1
fi

# Auto-detectar versão do nome do arquivo se não fornecido
if [ -z "$VERSION" ]; then
    # Tentar extrair do nome do arquivo (ex: cnpj_2026-01.sqlite)
    VERSION=$(basename "$SQLITE_FILE" .sqlite | grep -oE '[0-9]{4}-[0-9]{2}' || echo "latest")
fi

# Calcular informações do arquivo
FILE_SIZE=$(du -h "$SQLITE_FILE" | cut -f1)
FILE_SIZE_BYTES=$(stat -f%z "$SQLITE_FILE" 2>/dev/null || stat -c%s "$SQLITE_FILE" 2>/dev/null)
FILE_MD5=$(md5sum "$SQLITE_FILE" 2>/dev/null | cut -d' ' -f1 || md5 -q "$SQLITE_FILE" 2>/dev/null)

# Construir S3 path
S3_PATH="s3://${S3_BUCKET}/${S3_PREFIX}/${VERSION}/cnpj.sqlite"

echo -e "${YELLOW}Configuração:${NC}"
echo "  Arquivo local:    $SQLITE_FILE"
echo "  Tamanho:          $FILE_SIZE ($FILE_SIZE_BYTES bytes)"
echo "  MD5:              $FILE_MD5"
echo "  Destino S3:       $S3_PATH"
echo "  Storage Class:    $STORAGE_CLASS"
echo "  Versão:           $VERSION"
echo ""

# Confirmar com usuário (pode ser desabilitado com SKIP_CONFIRM=1)
if [ "${SKIP_CONFIRM:-0}" != "1" ]; then
    echo -e "${YELLOW}⚠️  Atenção: Isso vai fazer upload de ~${FILE_SIZE} para S3!${NC}"
    echo ""
    read -p "Deseja continuar? (y/n) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo "Upload cancelado."
        exit 0
    fi
    echo ""
fi

# Upload para S3
echo -e "${GREEN}📤 Fazendo upload...${NC}"
echo ""

# Metadata para incluir no objeto S3
METADATA="version=${VERSION},generated=$(date -u +%Y-%m-%dT%H:%M:%SZ),md5=${FILE_MD5},size=${FILE_SIZE_BYTES}"

# Comando de upload com opções otimizadas
aws s3 cp "$SQLITE_FILE" "$S3_PATH" \
    --storage-class "$STORAGE_CLASS" \
    --metadata "$METADATA" \
    --no-progress \
    2>&1 | tee /tmp/s3_upload.log

# Verificar se upload foi bem-sucedido
if [ $? -eq 0 ]; then
    echo ""
    echo -e "${GREEN}✅ Upload concluído com sucesso!${NC}"
    echo ""

    # Informações do objeto no S3
    echo -e "${YELLOW}Informações do objeto:${NC}"
    aws s3api head-object \
        --bucket "$S3_BUCKET" \
        --key "${S3_PREFIX}/${VERSION}/cnpj.sqlite" \
        --query '{Size:ContentLength,LastModified:LastModified,StorageClass:StorageClass,Metadata:Metadata}' \
        --output table 2>/dev/null || echo "  (não foi possível obter detalhes)"

    echo ""
    echo -e "${YELLOW}URLs de acesso:${NC}"
    echo "  S3 URI:    $S3_PATH"
    echo "  S3 Console: https://s3.console.aws.amazon.com/s3/object/${S3_BUCKET}?prefix=${S3_PREFIX}/${VERSION}/"
    echo ""

    # Gerar comando de download para outros usuários
    echo -e "${YELLOW}Para fazer download:${NC}"
    echo "  aws s3 cp $S3_PATH ./cnpj.sqlite"
    echo ""

    # Criar link simbólico "latest" (opcional)
    if [ "${CREATE_LATEST_LINK:-1}" = "1" ]; then
        echo -e "${YELLOW}Criando link 'latest'...${NC}"
        LATEST_PATH="s3://${S3_BUCKET}/${S3_PREFIX}/latest/cnpj.sqlite"
        aws s3 cp "$S3_PATH" "$LATEST_PATH" \
            --metadata "$METADATA,original-version=${VERSION}" \
            --storage-class "$STORAGE_CLASS" \
            --no-progress
        echo "  Link criado: $LATEST_PATH"
        echo ""
    fi

else
    echo ""
    echo -e "${RED}❌ Erro durante upload!${NC}"
    echo "Verifique o log: /tmp/s3_upload.log"
    exit 1
fi

# Opcional: Limpar versões antigas (manter apenas as últimas N)
if [ "${CLEANUP_OLD_VERSIONS:-0}" = "1" ]; then
    KEEP_VERSIONS="${KEEP_VERSIONS:-3}"
    echo -e "${YELLOW}Limpando versões antigas (mantendo últimas ${KEEP_VERSIONS})...${NC}"

    # Listar todas as versões
    aws s3 ls "s3://${S3_BUCKET}/${S3_PREFIX}/" | \
        grep -E '[0-9]{4}-[0-9]{2}' | \
        awk '{print $NF}' | \
        sort -r | \
        tail -n +$((KEEP_VERSIONS + 1)) | \
        while read old_version; do
            echo "  Removendo: ${old_version}"
            aws s3 rm "s3://${S3_BUCKET}/${S3_PREFIX}/${old_version}cnpj.sqlite"
        done
    echo ""
fi

echo "============================================"
echo -e "${GREEN}✨ Processo concluído!${NC}"
echo "============================================"
