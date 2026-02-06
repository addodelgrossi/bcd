# Novo Layout para os DADOS ABERTOS do CNPJ

## EMPRESAS

| Campo | Descrição |
|---|---|
| **CNPJ BÁSICO** | Número base de inscrição no CNPJ (oito primeiros dígitos do CNPJ). |
| **RAZÃO SOCIAL / NOME EMPRESARIAL** | Nome empresarial da pessoa jurídica. |
| **NATUREZA JURÍDICA** | Código da natureza jurídica. |
| **QUALIFICAÇÃO DO RESPONSÁVEL** | Qualificação da pessoa física responsável pela empresa. |
| **CAPITAL SOCIAL DA EMPRESA** | Capital social da empresa. |
| **PORTE DA EMPRESA** | Código do porte da empresa: `00` – Não informado; `01` – Micro empresa; `03` – Empresa de pequeno porte; `05` – Demais. |
| **ENTE FEDERATIVO RESPONSÁVEL** | Preenchido para os casos de órgãos e entidades do grupo de natureza jurídica **1XXX**. Para as demais naturezas, fica em branco. |

---

## ESTABELECIMENTOS

| Campo | Descrição |
|---|---|
| **CNPJ BÁSICO** | Número base de inscrição no CNPJ (oito primeiros dígitos do CNPJ). |
| **CNPJ ORDEM** | Número do estabelecimento de inscrição no CNPJ (do nono até o décimo segundo dígito do CNPJ). |
| **CNPJ DV** | Dígito verificador do número de inscrição no CNPJ (dois últimos dígitos do CNPJ). |
| **IDENTIFICADOR MATRIZ/FILIAL** | Código do identificador matriz/filial: `1` – Matriz; `2` – Filial. |
| **NOME FANTASIA** | Corresponde ao nome fantasia. |
| **SITUAÇÃO CADASTRAL** | Código da situação cadastral: `01` – Nula; `2` – Ativa; `3` – Suspensa; `4` – Inapta; `08` – Baixada. |
| **DATA SITUAÇÃO CADASTRAL** | Data do evento da situação cadastral. |
| **MOTIVO SITUAÇÃO CADASTRAL** | Código do motivo da situação cadastral. |
| **NOME DA CIDADE NO EXTERIOR** | Nome da cidade no exterior. |
| **PAIS** | Código do país. |
| **DATA DE INÍCIO ATIVIDADE** | Data de início da atividade. |
| **CNAE FISCAL PRINCIPAL** | Código da atividade econômica principal do estabelecimento. |
| **CNAE FISCAL SECUNDÁRIA** | Código da(s) atividade(s) econômica(s) secundária(s) do estabelecimento. |
| **TIPO DE LOGRADOURO** | Descrição do tipo de logradouro. |
| **LOGRADOURO** | Nome do logradouro onde se localiza o estabelecimento. |
| **NÚMERO** | Número onde se localiza o estabelecimento. Quando não houver preenchimento do número, haverá `S/N`. |
| **COMPLEMENTO** | Complemento para o endereço de localização do estabelecimento. |
| **BAIRRO** | Bairro onde se localiza o estabelecimento. |
| **CEP** | Código de endereçamento postal referente ao logradouro no qual o estabelecimento está localizado. |
| **UF** | Sigla da unidade da federação em que se encontra o estabelecimento. |
| **MUNICÍPIO** | Código do município de jurisdição onde se encontra o estabelecimento. |
| **DDD 1** | Contém o DDD 1. |
| **TELEFONE 1** | Contém o número do telefone 1. |
| **DDD 2** | Contém o DDD 2. |
| **TELEFONE 2** | Contém o número do telefone 2. |
| **DDD DO FAX** | Contém o DDD do fax. |
| **FAX** | Contém o número do fax. |
| **CORREIO ELETRÔNICO** | Contém o e-mail do contribuinte. |
| **SITUAÇÃO ESPECIAL** | Situação especial da empresa. |
| **DATA DA SITUAÇÃO ESPECIAL** | Data em que a empresa entrou em situação especial. |

---

## DADOS DO SIMPLES

| Campo | Descrição |
|---|---|
| **CNPJ BÁSICO** | Número base de inscrição no CNPJ (oito primeiros dígitos do CNPJ). |
| **OPÇÃO PELO SIMPLES** | Indicador da existência da opção pelo Simples: `S` – Sim; `N` – Não; em branco – Outros. |
| **DATA DE OPÇÃO PELO SIMPLES** | Data de opção pelo Simples. |
| **DATA DE EXCLUSÃO DO SIMPLES** | Data de exclusão do Simples. |
| **OPÇÃO PELO MEI** | Indicador da existência da opção pelo MEI: `S` – Sim; `N` – Não; em branco – Outros. |
| **DATA DE OPÇÃO PELO MEI** | Data de opção pelo MEI. |
| **DATA DE EXCLUSÃO DO MEI** | Data de exclusão do MEI. |

---

## SÓCIOS

| Campo | Descrição |
|---|---|
| **CNPJ BÁSICO** | Número base de inscrição no CNPJ (Cadastro Nacional da Pessoa Jurídica). |
| **IDENTIFICADOR DE SÓCIO** | Código do identificador de sócio: `1` – Pessoa jurídica; `2` – Pessoa física; `3` – Estrangeiro. |
| **NOME DO SÓCIO (PF) OU RAZÃO SOCIAL (PJ)** | Nome do sócio pessoa física **ou** a razão social e/ou nome empresarial da pessoa jurídica **e/ou** nome do sócio/razão social do sócio estrangeiro. |
| **CNPJ/CPF DO SÓCIO** | CPF ou CNPJ do sócio (sócio estrangeiro não tem esta informação). |
| **QUALIFICAÇÃO DO SÓCIO** | Código da qualificação do sócio. |
| **DATA DE ENTRADA SOCIEDADE** | Data de entrada na sociedade. |
| **PAIS** | Código país do sócio estrangeiro. |
| **REPRESENTANTE LEGAL** | Número do CPF do representante legal. |
| **NOME DO REPRESENTANTE** | Nome do representante legal. |
| **QUALIFICAÇÃO DO REPRESENTANTE LEGAL** | Código da qualificação do representante legal. |
| **FAIXA ETÁRIA** | Código correspondente à faixa etária do sócio. |

---

# Tabelas de domínio (arquivos separados)

> “Será gerado um arquivo para cada tabela de domínio listado abaixo:”

## PAÍSES

| Campo | Descrição |
|---|---|
| **CÓDIGO** | Código do país |
| **DESCRIÇÃO** | Nome do país |

## MUNICÍPIOS

| Campo | Descrição |
|---|---|
| **CÓDIGO** | Código do município |
| **DESCRIÇÃO** | Nome do município |

## QUALIFICAÇÕES DE SÓCIOS

| Campo | Descrição |
|---|---|
| **CÓDIGO** | Código da qualificação do sócio |
| **DESCRIÇÃO** | Nome da qualificação do sócio |

## NATUREZAS JURÍDICAS

| Campo | Descrição |
|---|---|
| **CÓDIGO** | Código da natureza jurídica |
| **DESCRIÇÃO** | Nome da natureza jurídica |

## CNAEs

| Campo | Descrição |
|---|---|
| **CÓDIGO** | Código da atividade econômica |
| **DESCRIÇÃO** | Nome da atividade econômica |

---

# Regras / Observações do Layout

## 1) Formato do arquivo
- Padrão de carga automática em bancos de dados relacionais (RDBMS).
- Separador de atributos: **ponto e vírgula (`;`)**.

## 2) Descaracterização de CPF (layout Sócios)
- Campos:
  - **169 (CNPJ/CPF DO SÓCIO)**
  - **271 (CNPJ/CPF DO REPRESENTANTE)**
- Regra: ocultar os **3 primeiros dígitos** e os **2 dígitos verificadores** (conforme orientação citada no PDF).

## 3) Ente Federativo Responsável (EFR)
- No layout principal (dados cadastrais): preenchido para órgãos e entidades do grupo de natureza jurídica **1XX**.
- Para demais naturezas: em branco.
- Exemplos de textos no arquivo final:
  - `UNIÃO`, `DISTRITO FEDERAL`, `BAHIA`
  - Para municípios: exibir também a UF, por exemplo `SÃO PAULO – SP`, `BELO HORIZONTE – MG`.

## 4) Faixa Etária (layout Sócios)
Baseada na data de nascimento do CPF do sócio:

| Código | Intervalo |
|---|---|
| 1 | 0 a 12 anos |
| 2 | 13 a 20 anos |
| 3 | 21 a 30 anos |
| 4 | 31 a 40 anos |
| 5 | 41 a 50 anos |
| 6 | 51 a 60 anos |
| 7 | 61 a 70 anos |
| 8 | 71 a 80 anos |
| 9 | > 80 anos |
| 0 | Não se aplica |

## 5) CNAE FISCAL SECUNDÁRIA (layout Estabelecimentos)
- Quando houver várias ocorrências, preencher separando por **vírgula**.