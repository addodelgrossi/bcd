# Scripts Utilitários

Esta pasta contém scripts para facilitar o uso do BCD (Brazil Companies Database).

## 📋 Scripts Disponíveis

### 1. `build_and_publish.sh` - Pipeline Completo ⭐ **RECOMENDADO**

Pipeline automatizado completo: Download → Extract → Load → Upload S3.

**Uso básico:**
```bash
# Com upload para S3
S3_BUCKET=meu-bucket ./scripts/build_and_publish.sh

# Sem upload para S3 (apenas gera o .sqlite localmente)
./scripts/build_and_publish.sh
```

**Uso avançado:**
```bash
YEAR_MONTH=2026-01 \
S3_BUCKET=meu-bucket \
S3_PREFIX=cnpj \
WORKDIR=/tmp/cnpj \
OUTPUT=./cnpj.sqlite \
SKIP_CONFIRM=1 \
./scripts/build_and_publish.sh
```

**O que faz:**
1. ✅ Baixa automaticamente a última versão (ou versão específica)
2. ✅ Extrai todos os ZIPs
3. ✅ Carrega no SQLite com todas as otimizações
4. ✅ Faz upload para S3 (opcional)
5. ✅ Cria link "latest" no S3
6. ✅ Mostra resumo com métricas

**Tempo estimado:** 60-120 minutos

---

### 2. `check_latest.sh` - Verificar Última Versão

Verifica qual é a última versão disponível do CNPJ publicada pela Receita Federal **sem fazer download**.

**Uso:**
```bash
./scripts/check_latest.sh
```

**Saída exemplo:**
```
🔍 Verificando última versão disponível do CNPJ...

Verificando disponibilidade dos últimos 7 meses:

  2025-01: ✅ DISPONÍVEL
  2024-12: ✅ DISPONÍVEL
  2024-11: ❌ não disponível
  2024-10: ❌ não disponível
  ...

============================================
📅 Última versão disponível: 2025-01
============================================

Para baixar e processar, execute:

  ./bcd download --ym 2025-01 --workdir /tmp/cnpj_rf
  ./bcd extract --ym 2025-01 --workdir /tmp/cnpj_rf
  ./bcd load --ym 2025-01 --workdir /tmp/cnpj_rf --out ./cnpj.sqlite
```

---

### 3. `upload_to_s3.sh` - Upload para S3

Faz upload do banco SQLite gerado para Amazon S3.

**Pré-requisitos:**
```bash
# Instalar AWS CLI
brew install awscli  # macOS
# ou
sudo apt install awscli  # Ubuntu

# Configurar credenciais
aws configure
```

**Uso básico:**
```bash
S3_BUCKET=meu-bucket ./scripts/upload_to_s3.sh
```

**Uso avançado:**
```bash
S3_BUCKET=meu-bucket \
S3_PREFIX=cnpj \
SQLITE_FILE=./cnpj_2026-01.sqlite \
VERSION=2026-01 \
STORAGE_CLASS=INTELLIGENT_TIERING \
CREATE_LATEST_LINK=1 \
CLEANUP_OLD_VERSIONS=1 \
KEEP_VERSIONS=3 \
./scripts/upload_to_s3.sh
```

**Variáveis disponíveis:**
- `S3_BUCKET` (obrigatório) - Bucket S3 de destino
- `S3_PREFIX` - Prefixo/pasta no bucket (padrão: `cnpj`)
- `SQLITE_FILE` - Arquivo local (padrão: `./cnpj.sqlite`)
- `VERSION` - Versão para metadata (auto-detecta do nome do arquivo)
- `STORAGE_CLASS` - Classe de armazenamento (padrão: `INTELLIGENT_TIERING`)
- `CREATE_LATEST_LINK` - Criar link `latest/` (padrão: `1`)
- `CLEANUP_OLD_VERSIONS` - Limpar versões antigas (padrão: `0`)
- `KEEP_VERSIONS` - Quantas versões manter (padrão: `3`)
- `SKIP_CONFIRM` - Pular confirmação (padrão: `0`)

**Storage Classes recomendadas:**
- `INTELLIGENT_TIERING` - Otimiza custo automaticamente (recomendado)
- `STANDARD` - Acesso frequente
- `STANDARD_IA` - Acesso infrequente (mensal)
- `GLACIER_INSTANT_RETRIEVAL` - Arquivamento com acesso rápido

---

### 4. `download_latest.sh` - Download Automatizado

Baixa, extrai e processa automaticamente a última versão disponível do CNPJ.

**Uso básico:**
```bash
./scripts/download_latest.sh
```

**Uso avançado com variáveis de ambiente:**
```bash
# Customizar diretório de trabalho e arquivo de saída
WORKDIR=/data/cnpj_temp OUTPUT=/data/cnpj.sqlite ./scripts/download_latest.sh

# Usar binário do bcd em outro local
BCD=/usr/local/bin/bcd ./scripts/download_latest.sh
```

**Variáveis de ambiente:**
- `WORKDIR` - Diretório de trabalho temporário (padrão: `/tmp/cnpj_rf`)
- `OUTPUT` - Caminho do arquivo SQLite de saída (padrão: `./cnpj.sqlite`)
- `BCD` - Caminho do binário bcd (padrão: `./bcd`)

**O que o script faz:**
1. ✅ Tenta baixar o mês atual
2. ✅ Se falhar, tenta os 3 meses anteriores automaticamente
3. ✅ Extrai todos os arquivos ZIP
4. ✅ Carrega no SQLite com todas as otimizações
5. ✅ Mostra estatísticas e próximos passos

**Tempo estimado:** 1-2 horas (dependendo da conexão e hardware)

---

## 🔄 Fluxo de Atualização Mensal Recomendado

### Opção 1: Simples (um comando)

```bash
# Executar no início de cada mês
./scripts/download_latest.sh
```

### Opção 2: Pipeline Completo (mais controle)

```bash
#!/bin/bash
# monthly_update.sh

set -e

# 1. Verificar se há nova versão
echo "Verificando versões disponíveis..."
./scripts/check_latest.sh

# 2. Confirmar com usuário
read -p "Deseja prosseguir com o download? (y/n) " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    exit 0
fi

# 3. Download e processamento
WORKDIR=/tmp/cnpj_new OUTPUT=/data/cnpj_v2.sqlite ./scripts/download_latest.sh

# 4. Validar
echo "Validando banco gerado..."
sqlite3 /data/cnpj_v2.sqlite "SELECT COUNT(*) FROM estabelecimentos;"

# 5. Backup do banco anterior
if [ -f /data/cnpj.sqlite ]; then
    cp /data/cnpj.sqlite /data/cnpj_backup_$(date +%Y%m%d).sqlite
fi

# 6. Swap atômico
ln -sfn /data/cnpj_v2.sqlite /data/cnpj.sqlite

# 7. Restart da API (exemplo com Docker)
docker restart cnpj-api

echo "✅ Atualização concluída!"
```

---

## 📅 Quando Executar?

A Receita Federal geralmente publica os dados nos **primeiros dias de cada mês** com os dados do **mês anterior**.

**Calendário recomendado:**
- **Dia 1-5 de cada mês**: Executar `check_latest.sh` para verificar disponibilidade
- **Quando disponível**: Executar `download_latest.sh` em horário de baixo tráfego

---

## 🐛 Troubleshooting

### Erro: "Nenhuma versão disponível encontrada"

**Possíveis causas:**
1. A Receita Federal ainda não publicou os dados do mês
2. Problema de conectividade
3. URL da Receita Federal mudou

**Solução:**
```bash
# Verificar manualmente no portal
open https://dados.gov.br/dados/conjuntos-dados/cadastro-nacional-da-pessoa-juridica---cnpj

# Ou testar um mês específico
./bcd download --ym 2024-12 --workdir /tmp/test
```

---

### Erro: "command not found: date -d"

Isso acontece no macOS, que usa BSD date ao invés de GNU date.

**Solução:**
```bash
# Instalar GNU coreutils (opcional)
brew install coreutils

# Ou usar gdate no lugar de date
alias date=gdate
```

---

### Script muito lento

O download e processamento é realmente demorado (60-120 min).

**Dicas para acelerar:**
1. Use SSD/NVMe para `WORKDIR`
2. Aumente a RAM disponível
3. Execute em horário de menor carga do servidor
4. Considere usar screen/tmux para não perder progresso:

```bash
# Iniciar sessão tmux
tmux new -s cnpj_update

# Executar script
./scripts/download_latest.sh

# Desconectar: Ctrl+B, depois D
# Reconectar: tmux attach -t cnpj_update
```

---

## 📝 Automação com Cron

Para executar automaticamente todo mês:

```bash
# Editar crontab
crontab -e

# Adicionar linha (executa dia 5 de cada mês às 2h da manhã)
0 2 5 * * cd /path/to/bcd && ./scripts/download_latest.sh >> /var/log/cnpj_update.log 2>&1
```

**Recomendação:** Use um sistema de orquestração mais robusto para produção (Airflow, Jenkins, GitHub Actions, etc).

---

## 🔐 Permissões

Os scripts precisam de:
- ✅ Permissão de execução (`chmod +x`)
- ✅ Acesso de escrita em `WORKDIR` e `OUTPUT`
- ✅ Acesso de leitura ao binário `bcd`

```bash
# Dar permissão de execução
chmod +x scripts/*.sh

# Verificar permissões
ls -la scripts/
```

---

## 📊 Logs e Monitoramento

Para monitorar o progresso do download:

```bash
# Executar com output para arquivo
./scripts/download_latest.sh 2>&1 | tee /var/log/cnpj_$(date +%Y%m%d).log

# Em outro terminal, acompanhar
tail -f /var/log/cnpj_*.log
```

---

## ✨ Contribuindo

Sinta-se à vontade para melhorar os scripts! Sugestões:

- [ ] Adicionar notificações (email, Slack, etc)
- [ ] Implementar retry automático em caso de falha
- [ ] Adicionar checksum validation
- [ ] Integração com S3/Cloud Storage
- [ ] Métricas e dashboard de monitoramento

---

## 📚 Mais Informações

- [README principal](../README.md)
- [Guia de Performance](../PERFORMANCE.md)
- [Estratégia de Otimização](../OPTIMIZATION_STRATEGY.md)
