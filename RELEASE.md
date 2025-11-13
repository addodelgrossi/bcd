# Guia de Release

Este documento descreve como criar uma nova versão do **bcd** (Brazil Companies Database).

## Processo Automatizado

O projeto usa GitHub Actions para automaticamente compilar binários para múltiplas plataformas quando uma tag de versão é criada.

### Plataformas Suportadas

Os binários são automaticamente compilados para:

- **Linux**: AMD64, ARM64
- **macOS**: Intel (AMD64), Apple Silicon (ARM64)
- **Windows**: AMD64, ARM64

### Criando um Release

1. **Certifique-se de que todas as mudanças estão commitadas:**
   ```bash
   git status
   ```

2. **Atualize o CHANGELOG (se existir) ou prepare as notas de release**

3. **Crie e push a tag de versão:**
   ```bash
   # Formato: v<MAJOR>.<MINOR>.<PATCH>
   git tag -a v1.0.0 -m "Release v1.0.0"
   git push origin v1.0.0
   ```

4. **O GitHub Actions automaticamente:**
   - Compila binários para todas as plataformas em paralelo
   - Cria arquivos `.tar.gz` (Linux/macOS) e `.zip` (Windows)
   - Gera checksums SHA256 para cada arquivo
   - Cria um GitHub Release com todos os binários anexados
   - Gera release notes automáticas

5. **Após o workflow completar (~2-5 minutos):**
   - Acesse: https://github.com/addodelgrossi/bcd/releases
   - Edite as release notes se necessário
   - Adicione informações sobre mudanças importantes
   - Publique o release (se deixou como draft)

## Testando Localmente

Antes de criar um release, teste o build localmente:

```bash
# Build padrão
go build -o bcd

# Build com versão injetada (simula o CI)
go build -ldflags="-s -w -X github.com/addodelgrossi/bcd/cmd.Version=1.0.0" -o bcd

# Verificar versão
./bcd --version
```

## Testando Cross-Compilation

```bash
# Linux AMD64
GOOS=linux GOARCH=amd64 go build -o bcd-linux-amd64

# macOS ARM64 (Apple Silicon)
GOOS=darwin GOARCH=arm64 go build -o bcd-darwin-arm64

# Windows AMD64
GOOS=windows GOARCH=amd64 go build -o bcd-windows-amd64.exe
```

## Versionamento Semântico

Seguimos [Semantic Versioning](https://semver.org/):

- **MAJOR**: Mudanças incompatíveis na API
- **MINOR**: Novas funcionalidades mantendo compatibilidade
- **PATCH**: Correções de bugs

Exemplos:
- `v1.0.0` - Primeiro release estável
- `v1.1.0` - Adiciona nova funcionalidade
- `v1.1.1` - Correção de bug
- `v2.0.0` - Breaking change

## Pre-releases

Para versões beta ou release candidates:

```bash
git tag -a v1.0.0-beta.1 -m "Beta release v1.0.0-beta.1"
git push origin v1.0.0-beta.1
```

O workflow marcará automaticamente como "pre-release" no GitHub.

## Troubleshooting

### O workflow falhou

1. Vá para: https://github.com/addodelgrossi/bcd/actions
2. Clique no workflow que falhou
3. Verifique os logs de cada job
4. Corrija o problema
5. Delete a tag e recrie:
   ```bash
   git tag -d v1.0.0
   git push origin :refs/tags/v1.0.0
   # Faça as correções necessárias
   git tag -a v1.0.0 -m "Release v1.0.0"
   git push origin v1.0.0
   ```

### Build local falha

Certifique-se de que:
- Go 1.21+ está instalado
- As dependências estão atualizadas: `go mod tidy`
- O código compila sem erros: `go build`

### Binário não mostra versão correta

Verifique se usou `-ldflags` corretamente:
```bash
go build -ldflags="-X github.com/addodelgrossi/bcd/cmd.Version=1.0.0" -o bcd
./bcd --version
```

## Checklist de Release

- [ ] Todas as mudanças estão commitadas
- [ ] Testes passando (se houver)
- [ ] Build local funciona
- [ ] CHANGELOG atualizado (se existir)
- [ ] Versão segue Semantic Versioning
- [ ] Tag criada e pushed
- [ ] GitHub Actions completou com sucesso
- [ ] Binários testados (pelo menos um)
- [ ] Release notes revisadas e publicadas
