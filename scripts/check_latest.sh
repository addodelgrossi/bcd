#!/bin/bash
# Script para verificar qual é a última versão disponível do CNPJ
# Usa WebDAV PROPFIND no Nextcloud da Receita Federal

SHARE_TOKEN="${SHARE_TOKEN:-YggdBLfdninEJX9}"
WEBDAV_URL="https://arquivos.receitafederal.gov.br/public.php/dav/files/${SHARE_TOKEN}/"

echo "Verificando última versão disponível do CNPJ via WebDAV..."
echo ""

# PROPFIND na raiz para listar pastas YYYY-MM
XML=$(curl -s -X PROPFIND -H "Depth: 1" "$WEBDAV_URL")

if [ -z "$XML" ]; then
    echo "Erro: não foi possível conectar ao WebDAV"
    echo "URL: $WEBDAV_URL"
    exit 1
fi

# Extrair pastas YYYY-MM do XML
FOLDERS=$(echo "$XML" | grep -oP '\d{4}-\d{2}' | sort -u -r)

if [ -z "$FOLDERS" ]; then
    echo "Nenhuma pasta YYYY-MM encontrada no WebDAV"
    echo ""
    echo "Verifique manualmente em:"
    echo "  https://arquivos.receitafederal.gov.br/index.php/s/${SHARE_TOKEN}"
    exit 1
fi

echo "Versões disponíveis (mais recente primeiro):"
echo ""
for ym in $FOLDERS; do
    echo "  $ym"
done

LATEST=$(echo "$FOLDERS" | head -1)

echo ""
echo "============================================"
echo "Última versão disponível: $LATEST"
echo "============================================"
echo ""
echo "Modo ZIP (individual por mês):"
echo "  bcd download --ym $LATEST --workdir /tmp/cnpj_rf"
echo "  bcd download --workdir /tmp/cnpj_rf            # auto-detecta"
echo "  bcd extract --workdir /tmp/cnpj_rf"
echo "  bcd load --workdir /tmp/cnpj_rf --out ./cnpj.sqlite"
echo ""
echo "Modo TAR (arquivo único ~60GB com a última versão):"
echo "  bcd download --mode tar --workdir /tmp/cnpj_rf"
echo "  bcd extract --mode tar --workdir /tmp/cnpj_rf"
echo "  bcd load --workdir /tmp/cnpj_rf --out ./cnpj.sqlite"
echo ""
