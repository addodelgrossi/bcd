# Guia de Armazenamento no S3

## 🪣 Por Que Usar S3?

Armazenar o banco SQLite no S3 traz vários benefícios:

### ✅ Vantagens

1. **Distribuição Fácil**
   - Compartilhe entre múltiplos servidores/containers
   - Download rápido de qualquer região AWS
   - Versionamento automático

2. **Backup Automático**
   - Durabilidade 99.999999999% (11 noves)
   - Replicação automática entre AZs
   - Lifecycle policies para arquivamento

3. **Custo-Benefício**
   - ~$0.023/GB/mês (INTELLIGENT_TIERING)
   - Banco de 20GB = ~$0.46/mês
   - Muito mais barato que EBS volumes replicados

4. **Integração com CI/CD**
   - Fácil automatizar deploys
   - CloudFront para CDN global
   - Lambda triggers para pós-processamento

---

## 💰 Estimativa de Custos (2026)

### Armazenamento

Para um banco de **20GB** atualizado **mensalmente**:

| Storage Class | Custo/GB/mês | Custo Total/mês | Quando Usar |
|---------------|--------------|-----------------|-------------|
| **INTELLIGENT_TIERING** | $0.023 | **$0.46** | ⭐ **Recomendado** (otimiza automaticamente) |
| STANDARD | $0.023 | $0.46 | Acesso muito frequente (>1x/dia) |
| STANDARD_IA | $0.0125 | $0.25 | Acesso infrequente (1x/mês) + $0.01/GB retrieval |
| GLACIER_INSTANT | $0.004 | $0.08 | Arquivamento com acesso rápido |

**Recomendação:** Use **INTELLIGENT_TIERING** - ele automaticamente move entre tiers economizando custo.

### Transferência de Dados

| Operação | Custo | Exemplo (20GB/mês) |
|----------|-------|-------------------|
| Upload (PUT) | **GRÁTIS** | $0.00 |
| Download (GET) - Internet | $0.09/GB | $1.80 (se baixar 1x fora da AWS) |
| Download (GET) - mesma região | **GRÁTIS** | $0.00 (EC2 na mesma região) |

**Dica:** Use EC2/ECS na **mesma região** do bucket S3 para evitar custos de transferência.

### Custo Total Estimado

```
Upload mensal (20GB):        GRÁTIS
Armazenamento (20GB):        $0.46/mês
Download para EC2 (1x):      GRÁTIS (mesma região)
────────────────────────────────────
TOTAL:                       ~$0.50/mês
```

**Muito barato!** Menos de $6/ano para banco completo + backups.

---

## 🏗️ Estrutura Recomendada no S3

```
s3://meu-bucket/
└── cnpj/
    ├── 2026-01/
    │   ├── cnpj.sqlite         (20GB)
    │   └── metadata.json       (opcional)
    ├── 2025-12/
    │   └── cnpj.sqlite
    ├── 2025-11/
    │   └── cnpj.sqlite
    └── latest/                 (link simbólico)
        └── cnpj.sqlite         → 2026-01/cnpj.sqlite
```

**Vantagens dessa estrutura:**
- ✅ Versionamento por mês
- ✅ `latest/` sempre aponta para versão mais recente
- ✅ Fácil rollback para mês anterior
- ✅ Lifecycle policies por prefixo

---

## 🚀 Workflows de Uso

### Workflow 1: Build Mensal com Upload

```bash
# Gerar banco e subir para S3 (tudo automatizado)
S3_BUCKET=meu-bucket ./scripts/build_and_publish.sh

# Resultado:
# - Local: ./cnpj_2026-01.sqlite
# - S3: s3://meu-bucket/cnpj/2026-01/cnpj.sqlite
# - S3: s3://meu-bucket/cnpj/latest/cnpj.sqlite (link)
```

---

### Workflow 2: Deploy de Produção

#### Opção A: Download Direto no Container

```dockerfile
FROM golang:1.24-alpine

# Instalar AWS CLI
RUN apk add --no-cache aws-cli

WORKDIR /app
COPY api .

# Download do S3 no startup
COPY entrypoint.sh .
RUN chmod +x entrypoint.sh

ENTRYPOINT ["./entrypoint.sh"]
```

```bash
# entrypoint.sh
#!/bin/sh
aws s3 cp s3://meu-bucket/cnpj/latest/cnpj.sqlite /data/cnpj.sqlite
exec ./api
```

**Vantagens:**
- ✅ Imagem Docker pequena (~15MB)
- ✅ Sempre usa versão mais recente
- ✅ Fácil atualizar (restart container)

**Desvantagens:**
- ❌ Startup demora ~30-60s (download de 20GB)
- ❌ Usa bandwidth de download (se fora da AWS)

---

#### Opção B: EBS Volume com Sync Periódico

```bash
# Uma vez: criar volume EBS e baixar
aws ec2 create-volume --size 30 --volume-type gp3
aws s3 cp s3://meu-bucket/cnpj/latest/cnpj.sqlite /mnt/data/

# Atualizar mensalmente via cron
0 2 1 * * aws s3 sync s3://meu-bucket/cnpj/latest/ /mnt/data/
```

**Vantagens:**
- ✅ Startup instantâneo
- ✅ Performance máxima

**Desvantagens:**
- ❌ Mais complexo
- ❌ Custo adicional do EBS (~$3/mês para 30GB gp3)

---

#### Opção C: S3 Mountpoint (FUSE)

```bash
# Instalar mountpoint-s3
wget https://s3.amazonaws.com/mountpoint-s3-release/latest/x86_64/mount-s3.deb
sudo apt install ./mount-s3.deb

# Montar bucket como filesystem
mount-s3 meu-bucket /mnt/s3
```

**Vantagens:**
- ✅ Acesso direto ao S3 como filesystem
- ✅ Não precisa baixar arquivo completo

**Desvantagens:**
- ❌ Latência maior que disco local
- ❌ Read-only por padrão
- ❌ SQLite pode ter problemas com FUSE

**❌ NÃO recomendado para SQLite em produção!**

---

### Workflow 3: Distribuição Multi-Região

Se você tem servidores em múltiplas regiões AWS:

```bash
# Bucket principal (us-east-1)
s3://cnpj-us-east-1/

# Replicação automática para outras regiões
aws s3api put-bucket-replication \
  --bucket cnpj-us-east-1 \
  --replication-configuration file://replication.json

# Cada região baixa do bucket local
# us-east-1: s3://cnpj-us-east-1/cnpj/latest/
# eu-west-1: s3://cnpj-eu-west-1/cnpj/latest/
# ap-south-1: s3://cnpj-ap-south-1/cnpj/latest/
```

**Custo adicional:** ~$0.46/região/mês (apenas storage, replicação é grátis entre regiões)

---

## 🔒 Segurança e Permissões

### IAM Policy para Upload (CI/CD)

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "s3:PutObject",
        "s3:PutObjectAcl"
      ],
      "Resource": "arn:aws:s3:::meu-bucket/cnpj/*"
    }
  ]
}
```

### IAM Policy para Download (Produção)

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "s3:GetObject",
        "s3:ListBucket"
      ],
      "Resource": [
        "arn:aws:s3:::meu-bucket",
        "arn:aws:s3:::meu-bucket/cnpj/*"
      ]
    }
  ]
}
```

**Princípio do menor privilégio:** Produção só precisa de leitura!

---

## 📊 Monitoramento e Métricas

### CloudWatch Metrics

Crie alarmes para:
- `BucketSizeBytes` - Tamanho do bucket
- `NumberOfObjects` - Quantidade de objetos
- `AllRequests` - Total de requests

```bash
# Criar alarme de custo
aws cloudwatch put-metric-alarm \
  --alarm-name cnpj-s3-cost-spike \
  --metric-name EstimatedCharges \
  --threshold 10 \
  --comparison-operator GreaterThanThreshold
```

### S3 Inventory

Configure inventário para auditoria:

```bash
aws s3api put-bucket-inventory-configuration \
  --bucket meu-bucket \
  --id cnpj-inventory \
  --inventory-configuration file://inventory.json
```

---

## 🗑️ Lifecycle Policies

Mover versões antigas para Glacier automaticamente:

```json
{
  "Rules": [
    {
      "Id": "archive-old-versions",
      "Status": "Enabled",
      "Prefix": "cnpj/",
      "Transitions": [
        {
          "Days": 90,
          "StorageClass": "GLACIER_INSTANT_RETRIEVAL"
        },
        {
          "Days": 365,
          "StorageClass": "DEEP_ARCHIVE"
        }
      ]
    }
  ]
}
```

**Economia:**
- Versões >3 meses: $0.004/GB (Glacier Instant)
- Versões >1 ano: $0.00099/GB (Deep Archive)

---

## ⚡ Performance Tips

### 1. Use S3 Transfer Acceleration

```bash
aws s3 cp cnpj.sqlite s3://meu-bucket/cnpj/latest/ \
  --endpoint-url https://meu-bucket.s3-accelerate.amazonaws.com
```

**Ganho:** 50-500% mais rápido para uploads/downloads globais

**Custo:** +$0.04/GB (vale a pena para uploads de 20GB)

---

### 2. Multipart Upload Automático

O script `upload_to_s3.sh` já usa multipart automaticamente via AWS CLI.

Configurar tamanho ideal:

```bash
aws configure set default.s3.multipart_threshold 64MB
aws configure set default.s3.multipart_chunksize 16MB
```

---

### 3. Cache com CloudFront

Para distribuição global do banco (read-only):

```bash
# Criar distribuição CloudFront
aws cloudfront create-distribution \
  --origin-domain-name meu-bucket.s3.amazonaws.com
```

**Benefícios:**
- Download 10x mais rápido globalmente
- Reduz custos de GET requests no S3
- SSL/HTTPS automático

---

## 🎯 Recomendação Final

### Para Maioria dos Casos:

```bash
# 1. Build mensal automatizado
S3_BUCKET=meu-bucket ./scripts/build_and_publish.sh

# 2. Deploy: download no startup do container
aws s3 cp s3://meu-bucket/cnpj/latest/cnpj.sqlite /data/
```

**Custo:** ~$0.50/mês
**Complexidade:** Baixa
**Performance:** Excelente

---

## 📚 Recursos

- [AWS S3 Pricing](https://aws.amazon.com/s3/pricing/)
- [S3 Transfer Acceleration](https://aws.amazon.com/s3/transfer-acceleration/)
- [Mountpoint for S3](https://github.com/awslabs/mountpoint-s3)
- [S3 Best Practices](https://docs.aws.amazon.com/AmazonS3/latest/userguide/optimizing-performance.html)
