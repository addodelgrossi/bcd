# Guia de Portabilidade

Este documento explica a portabilidade do BCD e dos arquivos `.sqlite` gerados.

---

## 🎯 Resposta Rápida

### Arquivo .sqlite
✅ **100% PORTÁVEL** entre todas as plataformas

### Executável bcd
❌ **Específico por plataforma** (precisa compilar ou baixar versão correta)

---

## 📊 Matriz de Portabilidade

### Arquivo cnpj.sqlite

| Gerado em | Funciona em Linux x86? | Linux ARM? | Windows? | macOS? |
|-----------|----------------------|------------|----------|--------|
| Linux x86_64 | ✅ | ✅ | ✅ | ✅ |
| Linux ARM64 | ✅ | ✅ | ✅ | ✅ |
| Windows x64 | ✅ | ✅ | ✅ | ✅ |
| macOS Intel | ✅ | ✅ | ✅ | ✅ |
| macOS ARM | ✅ | ✅ | ✅ | ✅ |

**Conclusão:** O arquivo `.sqlite` funciona em **QUALQUER** plataforma! 🎉

### Executável bcd

| Binário | Linux x86? | Linux ARM? | Windows? | macOS Intel? | macOS ARM? |
|---------|-----------|------------|----------|--------------|------------|
| bcd-linux-amd64 | ✅ | ❌ | ❌ | ❌ | ❌ |
| bcd-linux-arm64 | ❌ | ✅ | ❌ | ❌ | ❌ |
| bcd-windows-amd64.exe | ❌ | ❌ | ✅ | ❌ | ❌ |
| bcd-darwin-amd64 | ❌ | ❌ | ❌ | ✅ | ❌ |
| bcd-darwin-arm64 | ❌ | ❌ | ❌ | ❌ | ✅ |

**Conclusão:** Cada binário funciona **apenas** na plataforma para qual foi compilado.

---

## 🔬 Por Que SQLite é Portável?

### 1. Formato Binário Padronizado
```
SQLite Database File Format Specification:
┌────────────────────────────────────────┐
│ Header (100 bytes)                     │
│ - Magic: "SQLite format 3\000"        │
│ - Page size: 4096 bytes (padrão)      │
│ - Byte order: Little-endian (sempre)  │
└────────────────────────────────────────┘
│ Page 1 (Schema)                        │
│ Page 2 (Data)                          │
│ ...                                    │
```

- **Byte order fixo:** Sempre little-endian, independente da CPU
- **Page size consistente:** Mesmo layout em todas as plataformas
- **Formato documentado:** Especificação pública e estável

### 2. Compatibilidade de Versão
```
SQLite 3.x (2004 - presente):
├─ 3.0.0 (2004) → formato base
├─ 3.7.0 (2010) → melhorias
├─ 3.35.0 (2021) → otimizações
└─ 3.40.0 (2023) → versão atual

Retrocompatibilidade: ✅
- Arquivo de 3.40 funciona em 3.35+
- Arquivo de 3.35 funciona em 3.30+
```

### 3. Zero Dependências Externas
- Não usa bibliotecas do sistema
- Não depende de locale/charset do OS
- Self-contained format

---

## 💻 Cenários Práticos

### Cenário 1: Servidor Potente + Raspberry Pi

**Objetivo:** Gerar banco em servidor x86_64, usar em Raspberry Pi (ARM)

```bash
# 1️⃣ Servidor Linux x86_64 (potente, rápido)
./bcd-linux-amd64 download --ym 2026-01 --workdir /tmp/cnpj
./bcd-linux-amd64 extract --ym 2026-01 --workdir /tmp/cnpj
./bcd-linux-amd64 load --ym 2026-01 --out cnpj.sqlite
# Tempo: ~60 minutos em servidor potente

# 2️⃣ Upload para S3 (ou transferência direta)
aws s3 cp cnpj.sqlite s3://meu-bucket/cnpj/2026-01/

# 3️⃣ Raspberry Pi (ARM)
aws s3 cp s3://meu-bucket/cnpj/2026-01/cnpj.sqlite /data/
# OU transferência direta:
# scp server:/path/cnpj.sqlite /data/

# 4️⃣ Usar na API (Raspberry Pi)
./api-arm64 --db=/data/cnpj.sqlite
# ✅ Funciona perfeitamente!
```

**Vantagens:**
- ⚡ Geração rápida (servidor x86_64)
- 💰 Raspberry Pi só lê (economia de CPU)
- ✅ .sqlite portável

---

### Cenário 2: Desenvolvimento Windows + Produção Linux

**Objetivo:** Desenvolver/testar no Windows, rodar em produção no Linux

```powershell
# 1️⃣ Windows (desenvolvimento)
.\bcd-windows-amd64.exe download --ym 2026-01 --workdir C:\temp\cnpj
.\bcd-windows-amd64.exe extract --ym 2026-01 --workdir C:\temp\cnpj
.\bcd-windows-amd64.exe load --ym 2026-01 --out cnpj.sqlite

# 2️⃣ Testar localmente no Windows
.\api.exe --db=cnpj.sqlite
# http://localhost:8080

# 3️⃣ Deploy para produção (Linux)
scp cnpj.sqlite user@linux-server:/data/

# 4️⃣ Produção Linux
./api --db=/data/cnpj.sqlite
# ✅ Funciona perfeitamente!
```

---

### Cenário 3: macOS M1 (ARM) + AWS EC2 (x86_64)

**Objetivo:** Gerar no Mac local, rodar em AWS

```bash
# 1️⃣ macOS M1 (local)
./bcd-darwin-arm64 download --ym 2026-01 --workdir /tmp/cnpj
./bcd-darwin-arm64 extract --ym 2026-01 --workdir /tmp/cnpj
./bcd-darwin-arm64 load --ym 2026-01 --out cnpj.sqlite

# 2️⃣ Upload para S3
aws s3 cp cnpj.sqlite s3://meu-bucket/cnpj/2026-01/

# 3️⃣ EC2 Linux x86_64 (AWS)
aws s3 cp s3://meu-bucket/cnpj/2026-01/cnpj.sqlite /data/
./api --db=/data/cnpj.sqlite
# ✅ Funciona perfeitamente!
```

---

## ⚙️ Compilação Cross-Platform

### Compilar para Outras Plataformas

```bash
# No seu ambiente de desenvolvimento (qualquer plataforma)

# Linux AMD64
GOOS=linux GOARCH=amd64 go build -o bcd-linux-amd64

# Linux ARM64 (Raspberry Pi, AWS Graviton)
GOOS=linux GOARCH=arm64 go build -o bcd-linux-arm64

# macOS Intel
GOOS=darwin GOARCH=amd64 go build -o bcd-darwin-amd64

# macOS Apple Silicon (M1/M2/M3)
GOOS=darwin GOARCH=arm64 go build -o bcd-darwin-arm64

# Windows x64
GOOS=windows GOARCH=amd64 go build -o bcd-windows-amd64.exe

# Windows ARM64 (Surface Pro X)
GOOS=windows GOARCH=arm64 go build -o bcd-windows-arm64.exe
```

### Script de Build Multi-Plataforma

```bash
#!/bin/bash
# build-all.sh

VERSION="v1.0.0"
PLATFORMS=(
  "linux/amd64"
  "linux/arm64"
  "darwin/amd64"
  "darwin/arm64"
  "windows/amd64"
)

for platform in "${PLATFORMS[@]}"; do
  GOOS=${platform%/*}
  GOARCH=${platform#*/}
  output="bcd-${GOOS}-${GOARCH}"

  if [ "$GOOS" = "windows" ]; then
    output+=".exe"
  fi

  echo "Building $output..."
  GOOS=$GOOS GOARCH=$GOARCH go build \
    -ldflags="-s -w -X github.com/addodelgrossi/bcd/cmd.Version=$VERSION" \
    -o "dist/$output"
done

echo "✅ Build concluído! Arquivos em dist/"
```

---

## 🧪 Testes de Portabilidade

### Teste 1: Verificar Formato SQLite

```bash
# Em qualquer plataforma
file cnpj.sqlite

# Output esperado:
# cnpj.sqlite: SQLite 3.x database, last written using SQLite version 3040000
```

### Teste 2: Validar Integridade

```bash
# Gerar MD5 na origem
md5sum cnpj.sqlite > cnpj.sqlite.md5

# Transferir ambos os arquivos
scp cnpj.sqlite cnpj.sqlite.md5 user@remote:/data/

# Verificar na destinação
md5sum -c cnpj.sqlite.md5
# Output: cnpj.sqlite: OK ✅
```

### Teste 3: Query Cross-Platform

```bash
# Mesmo banco em diferentes plataformas deve retornar mesmos dados

# Linux x86_64
sqlite3 cnpj.sqlite "SELECT COUNT(*) FROM estabelecimentos;"
# 50000000

# Transferir para ARM
# ...

# Linux ARM
sqlite3 cnpj.sqlite "SELECT COUNT(*) FROM estabelecimentos;"
# 50000000 ✅ (mesmo resultado)
```

---

## ⚠️ Limitações Conhecidas

### 1. Versão Mínima do SQLite

O banco gerado requer SQLite **3.35+** (2021 ou mais recente).

```bash
# Verificar versão instalada
sqlite3 --version

# Se versão < 3.35, pode ter problemas
# Solução: atualizar SQLite
```

### 2. File System Differences

Em **Windows**, use barras invertidas ou forward slashes:
```powershell
# ✅ OK
.\bcd.exe load --out C:\data\cnpj.sqlite
.\bcd.exe load --out C:/data/cnpj.sqlite

# ❌ Evite misturar
.\bcd.exe load --out C:\data/cnpj.sqlite
```

### 3. Permissions

Após transferir para Linux, ajuste permissões:
```bash
chmod 644 cnpj.sqlite  # Read-write para owner, read para grupo/outros
chown api-user:api-group cnpj.sqlite
```

---

## 📋 Checklist de Transferência

### Antes de Transferir

- [ ] Verificar tamanho do arquivo (deve ser ~20GB)
- [ ] Gerar MD5/SHA256 checksum
- [ ] Confirmar integridade local (`sqlite3 cnpj.sqlite "PRAGMA integrity_check;"`)

### Durante Transferência

- [ ] Usar SCP, rsync ou S3 (evite FTP que pode corromper binários)
- [ ] Transferir em modo binário
- [ ] Verificar se transferência completou (tamanho correto)

### Após Transferência

- [ ] Verificar checksum
- [ ] Testar query simples
- [ ] Verificar permissões de arquivo
- [ ] Confirmar performance (índices funcionando)

---

## 🎯 Recomendações Finais

### Para Produção

1. **Gere o banco em servidor potente** (x86_64)
2. **Upload para S3** (versionamento + backup)
3. **Download em cada ambiente** conforme necessário
4. **Use read-only** sempre que possível

### Para Desenvolvimento

1. **Gere uma vez** em ambiente de dev
2. **Compartilhe o .sqlite** via S3 ou rede interna
3. **Desenvolvedores baixam** ao invés de gerar localmente
4. **Economiza tempo** (60-90 min por dev)

---

## 🔗 Recursos

- [SQLite File Format](https://www.sqlite.org/fileformat.html)
- [SQLite Portability](https://www.sqlite.org/different.html)
- [Go Cross Compilation](https://go.dev/doc/install/source#environment)

---

**Última atualização:** Janeiro 2026
