# Controle Financeiro

Sistema de controle financeiro pessoal e doméstico, multiusuário, com backend em Go e PostgreSQL como
fonte de verdade dos dados, e frontend em React/TypeScript. Interface em português brasileiro, valores em
Real (BRL) e datas no formato brasileiro (DD/MM/AAAA).

## Objetivo

Permitir que cada usuário controle, de forma isolada e segura, suas contas financeiras, receitas e
despesas mensais, orçamentos, despesas recorrentes, patrimônio (ativos e passivos) e investimentos, com um
dashboard consolidado e relatórios.

## Tecnologias

**Backend**
- Go 1.26 (gerenciado via `go.mod`/toolchain automático)
- `net/http` com o roteador nativo (padrões `MÉTODO /caminho` do Go 1.22+), sem framework HTTP externo
- `database/sql` com driver [`pgx`](https://github.com/jackc/pgx)
- PostgreSQL 16
- [`shopspring/decimal`](https://github.com/shopspring/decimal) para valores monetários (nunca `float64`)
- Argon2id (`golang.org/x/crypto`) para hashing de senhas
- Logs estruturados via `log/slog`

**Frontend**
- React 19 + TypeScript, construído com Vite
- `react-router-dom` para navegação
- Sem framework de UI ou biblioteca de gráficos externa — CSS próprio e gráficos em SVG/HTML
- Tema escuro único, com todas as cores centralizadas em variáveis CSS (`web/src/styles/global.css`)
- Gráficos próprios: pizza (rosca) da composição do patrimônio e barras horizontais de receitas por fonte
  e custo de vida por categoria, sempre com os valores calculados pela API

**Banco de dados**: PostgreSQL (único suportado; `database/sql` + `pgx`).

## Estrutura do projeto

```
cmd/api/              ponto de entrada da API (main.go)
cmd/demoseed/         gerador dos dados fictícios da conta de demonstração (usado pelo dev.sh)
internal/
  auth/               sessões, login/registro/logout, rate limiting, CSRF
  user/                perfil, senha, validação
  authctx/             propagação da identidade autenticada via contexto
  account/             contas financeiras e saldo
  category/            categorias globais e privadas
  transaction/         lançamentos, transferências, resumo mensal
  recurring/           despesas recorrentes
  budget/              orçamento mensal por categoria
  asset/ liability/     patrimônio (ativos e passivos) e histórico de avaliações
  investment/          investimentos, movimentações, métricas
  networth/            cálculo consolidado do patrimônio líquido
  dashboard/           indicadores consolidados e evolução mensal
  database/            conexão e runner de migrações
  middleware/          logging, recovery, CORS, CSRF, cabeçalhos de segurança
  httpx/               respostas JSON padronizadas
  audit/               eventos de auditoria
  server/              montagem do roteador HTTP completo (usado por main.go e pelos testes)
migrations/            migrações SQL versionadas, aplicadas em ordem na inicialização
tests/integration/     testes de integração (HTTP real + PostgreSQL real)
web/                   frontend React/TypeScript (Vite)
docker-compose.yml     PostgreSQL local para desenvolvimento
```

## Pré-requisitos

- Go 1.23+ (o `go.mod` usa o mecanismo de toolchain automático do Go; qualquer Go 1.21+ já baixa a
  versão correta na primeira execução, desde que haja acesso à internet)
- Node.js 20+ e npm
- Docker (para o PostgreSQL local via `docker-compose.yml`) — ou um PostgreSQL 14+ já disponível

## Início rápido (um comando)

```
./dev.sh
```

O script, a partir da raiz do projeto:
1. verifica os pré-requisitos (`go`, `node`, `npm`, `docker`, `openssl`, `curl`);
2. cria o `.env` a partir do `.env.example` na primeira execução, já com um `SESSION_SECRET` aleatório;
3. sobe o PostgreSQL com `docker compose` e aguarda o healthcheck;
4. instala as dependências do frontend (`npm ci`) se `web/node_modules` não existir;
5. compila e inicia a API (as migrações rodam na inicialização) e aguarda `/healthz`;
6. cria (ou reaproveita) a conta de demonstração e, se ela estiver vazia, preenche-a com dados fictícios;
7. inicia o frontend em `http://localhost:5173`.

Depois, abra `http://localhost:5173` e entre com:

| E-mail | Senha |
|---|---|
| `demo@finctrl.local` | `demo12345` |

`Ctrl+C` encerra a API e o frontend. O PostgreSQL continua em segundo plano; pare-o com
`docker compose down` (os dados permanecem no volume `finctrl_pgdata`).

Os passos manuais equivalentes estão descritos abaixo.

## Credenciais de acesso (ambiente local)

> Valores exclusivos para desenvolvimento local, definidos em `docker-compose.yml` e `.env.example`.
> Não são credenciais reais e nunca devem ser usados em produção.

### Aplicação

| Item | Valor |
|---|---|
| Frontend | `http://localhost:5173` |
| API | `http://localhost:8080` (verificação: `/healthz`) |
| **E-mail** | `demo@finctrl.local` |
| **Senha** | `demo12345` |

A conta de demonstração acima é criada automaticamente pelo `./dev.sh` (somente com `APP_ENV=development`),
usando o fluxo normal de cadastro da API, e é a **única** conta que recebe dados fictícios.

**Dados de demonstração.** Logo após criar a conta, o `./dev.sh` executa `cmd/demoseed`, que preenche os
últimos 12 meses (até a data atual) com:
- salário, aluguel recebido, trabalhos avulsos, dividendos e um reembolso pendente;
- 8 custos fixos mensais (aluguel, condomínio, energia, água, internet, streaming, academia, curso) e
  3 anuais (IPVA, IPTU, seguro), já lançados e pagos nos meses passados, com valores reais de consumo variando;
- despesas do dia a dia (mercado, combustível, restaurante, farmácia etc.), uma despesa planejada e
  orçamentos do mês atual e do anterior, com Lazer propositalmente acima do limite;
- 3 bens (apartamento, sala comercial, carro) com histórico de avaliações;
- 6 investimentos com aportes, resgate, rendimentos, taxa e uma reavaliação atual (com ganhos e uma perda).

Tudo é gravado **pela API**, com as mesmas validações e regras da interface — nada é inserido direto no
banco. Os valores são escolhidos de forma determinística, sem `float`. O gerador é idempotente: se a conta
demo já tiver lançamentos, ele não faz nada (assim, o que você alterar não é sobrescrito). Bens ou
investimentos cadastrados à mão antes disso não impedem a geração e são mantidos. Para recomeçar do zero, apague os dados com `docker compose down -v` e rode o `./dev.sh`
de novo. A conta demo, se já existir, é apenas reaproveitada. Quem sobe o backend manualmente (sem o `./dev.sh`) não terá essa conta e deve
criá-la pela tela **Cadastro** ou pelo comando abaixo.

Também é possível criar contas adicionais pela tela **Cadastro**; os dados de cada conta são isolados das
demais. Requisitos: nome com 2 a 150 caracteres, e-mail válido e senha com 8 a 128 caracteres.

Para criar uma conta pela linha de comando (com a API rodando):
```
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"name":"Usuário Teste","email":"teste@example.com","password":"senha-teste-123"}'
```

### Banco de dados (PostgreSQL)

| Item | Valor |
|---|---|
| Host | `localhost` |
| Porta | `5432` |
| Banco | `finctrl` |
| Usuário | `finctrl` |
| Senha | `finctrl_local_dev` |
| URL de conexão | `postgres://finctrl:finctrl_local_dev@localhost:5432/finctrl?sslmode=disable` |

Para abrir um terminal SQL no banco do contêiner:
```
docker compose exec postgres psql -U finctrl -d finctrl
```

O `SESSION_SECRET` não é uma credencial de acesso: é gerado aleatoriamente no `.env` de cada máquina
(pelo `./dev.sh` ou manualmente) e não deve ser compartilhado nem versionado.

## Configuração local

1. Copie `.env.example` para `.env` e ajuste os valores (nenhum valor de exemplo é uma credencial real):
   ```
   cp .env.example .env
   ```
2. Gere um `SESSION_SECRET` aleatório com pelo menos 32 caracteres, por exemplo:
   ```
   openssl rand -hex 32
   ```

### Banco de dados

Suba o PostgreSQL local:
```
docker compose up -d postgres
```
Isso cria o banco `finctrl` com usuário/senha `finctrl` / `finctrl_local_dev` (apenas para desenvolvimento
local, nunca usar em produção).

### Migrações

As migrações em `migrations/*.sql` são aplicadas automaticamente, em ordem, sempre que o backend inicia —
não é necessário um comando separado. O processo é idempotente (seguro executar repetidamente) e
reproduzível em um banco vazio. Elas também podem ser inspecionadas e aplicadas manualmente com qualquer
cliente psql, se preferido.

### Backend

```
export $(grep -v '^#' .env | xargs)   # carrega as variáveis do .env no shell
go run ./cmd/api
```
O servidor inicia em `http://localhost:8080` (porta configurável via `HTTP_PORT`). Teste com:
```
curl http://localhost:8080/healthz
```

### Frontend

```
cd web
npm install
npm run dev
```
O Vite inicia em `http://localhost:5173` e faz proxy de `/api/*` e `/healthz` para `http://localhost:8080`
— frontend e backend ficam na mesma origem durante o desenvolvimento, então o cookie de sessão funciona
normalmente, sem necessidade de configurar CORS entre domínios diferentes.

Abra `http://localhost:5173` no navegador, crie uma conta e explore a aplicação.

## Testes

**Backend** (requer o PostgreSQL de `docker compose up -d postgres` rodando):
```
export DATABASE_URL="postgres://finctrl:finctrl_local_dev@localhost:5432/finctrl?sslmode=disable"
go test ./...
```
Os testes de integração (`tests/integration`) criam um schema PostgreSQL isolado e descartável por teste,
aplicam as migrações reais e exercitam a pilha HTTP completa (sem mocks) — cobrindo autenticação,
isolamento entre usuários, saldo de contas, transferências atômicas, orçamento, recorrências, patrimônio e
investimentos.

**Frontend**:
```
cd web
npx tsc -b        # verificação de tipos
npm run build     # build de produção
```

## Variáveis de ambiente

Veja `.env.example` para a lista completa com descrições. As obrigatórias são `DATABASE_URL` e
`SESSION_SECRET` — a aplicação recusa iniciar sem elas, com uma mensagem de erro clara.

## Endpoints principais

Todas as rotas sob `/api/v1/` (exceto `/api/v1/auth/*`) exigem sessão autenticada (cookie `session_token`)
e, para métodos que alteram estado, o cabeçalho `X-CSRF-Token` correspondente ao cookie `csrf_token`
emitido no login.

| Domínio | Rotas |
|---|---|
| Autenticação | `POST /api/v1/auth/register`, `/login`, `/logout` |
| Perfil | `GET/PATCH /api/v1/me`, `PATCH /api/v1/me/password` |
| Contas | `GET/POST /api/v1/accounts`, `GET/PATCH/DELETE /api/v1/accounts/{id}` |
| Categorias | `GET/POST /api/v1/categories`, `PATCH/DELETE /api/v1/categories/{id}` |
| Lançamentos | `GET/POST /api/v1/transactions`, `GET/PATCH/DELETE /api/v1/transactions/{id}`, `GET /api/v1/transactions/monthly-summary`, `GET /api/v1/transactions/annual-summary?year=` |
| Transferências | `GET/POST /api/v1/transfers` |
| Custos fixos (recorrências) | `GET/POST /api/v1/recurring-rules`, `GET /api/v1/recurring-rules/summary`, `.../{id}`, `POST .../{id}/generate?year=&month=` |
| Orçamento | `GET/POST /api/v1/budgets`, `PATCH/DELETE /api/v1/budgets/{id}` |
| Patrimônio | `GET/POST /api/v1/assets`, `/liabilities`, `.../{id}/valuations`, `GET /api/v1/net-worth`, `GET /api/v1/net-worth/composition` |
| Investimentos | `GET/POST /api/v1/investments`, `.../{id}/transactions`, `.../{id}/valuations`, `GET /api/v1/investments/summary` |
| Dashboard | `GET /api/v1/dashboard/summary`, `/monthly-flow`, `/expenses-by-category` |

## Decisões financeiras e convenções

- **Valores monetários**: `NUMERIC(18,2)` no banco, `decimal.Decimal` no Go — nunca `float64`. Transportados
  como string decimal no JSON.
- **Saldo de conta** = saldo inicial + soma das movimentações com status `paid` (pendentes, planejadas e
  canceladas não afetam o saldo).
- **Transferências** entre contas próprias geram duas movimentações vinculadas (`transfer_out`/`transfer_in`),
  atômicas (uma única transação SQL), e nunca entram no consolidado de receitas/despesas.
- **Aportes/resgates em investimentos** com conta vinculada reaproveitam o mesmo mecanismo de
  `transfer_out`/`transfer_in`, também excluídos do consolidado de receitas/despesas — nunca tratados como
  despesa de consumo.
- **Preço médio de investimentos** é recalculado por média ponderada a cada aporte com quantidade; resgates
  reduzem o valor investido proporcionalmente à quantidade resgatada, gerando um ganho/perda realizado
  explícito. Quando não há dados de quantidade/preço médio suficientes, o ganho realizado não é inventado
  (fica `null`).
- **Valor de mercado após movimentações**: um aporte soma o valor aplicado ao valor de mercado e um resgate
  subtrai o valor sacado (sem ficar negativo), na mesma transação SQL que atualiza o valor investido. Assim,
  aportes e resgates nunca aparecem como ganho ou perda; só uma nova avaliação informada pelo usuário muda a
  rentabilidade. Rendimentos pagos e taxas não alteram a posição.
- **Patrimônio líquido** = (saldo das contas + valor dos ativos declarados + valor de mercado dos
  investimentos marcados como `include_in_net_worth`) − saldo devedor dos passivos ativos. O campo
  `include_in_net_worth` existe para o usuário excluir uma posição já representada em outro lugar e evitar
  contagem duplicada.
- **Páginas e o que cada uma soma**:
  - **Receitas**: tudo o que o usuário ganha. Totais do mês e do ano (1º/jan a 31/dez) e "fontes de renda"
    (receitas agrupadas por categoria) consideram apenas receitas com status `paid` (recebidas).
  - **Despesas**: custo de vida. Custos fixos são regras de recorrência **mensais** (vencem todo mês, no dia
    informado) ou **anuais** (vencem uma vez por ano, no dia e mês informados — `due_month` é obrigatório para
    regras anuais e proibido nas demais). Fórmulas, usando apenas regras ativas e não encerradas:
    custo anual da regra = valor × 12 (mensal) | valor × 1 (anual) | valor × 52 (semanal, legado);
    **custo de vida por ano** = soma dos custos anuais; **custo de vida por mês** = custo de vida por ano ÷ 12,
    arredondado a 2 casas (meio para cima) a partir do total exato, não da soma de valores já arredondados.
    "Lançar" um custo fixo cria o lançamento do mês como `planned`/`pending` — nunca como pago — e é
    idempotente; um custo anual só pode ser lançado no seu mês de vencimento.
  - **Patrimônio**: bens declarados + investimentos com `include_in_net_worth`, sem saldo de contas e sem
    passivos (`GET /api/v1/net-worth/composition`). Os totais coincidem com `declared_assets_value` e
    `investments_value` de `/net-worth`. A participação de cada fatia é calculada no backend (1 casa decimal).
    Os passivos continuam existindo na API e entram no **patrimônio líquido** do Dashboard; a página
    Patrimônio apenas não os exibe. É também na página Patrimônio que os investimentos são cadastrados,
    movimentados (aporte, resgate, rendimento, taxa), reavaliados e excluídos — não há página separada de
    investimentos (o endereço antigo `/investimentos` redireciona para `/patrimonio`).
- **Histórico de avaliações**: editar o valor atual de um ativo, o saldo de um passivo ou o valor de
  mercado de um investimento sempre adiciona um registro de histórico, nunca sobrescreve ou apaga os
  anteriores.
- **Categorias**: categorias globais do sistema (pré-cadastradas) não podem ser alteradas/excluídas pelo
  usuário. Categorias privadas em uso por lançamentos são arquivadas em vez de excluídas. A interface não tem
  mais página de categorias: os formulários usam as categorias do sistema (e as privadas já existentes).
- **Conta padrão (Carteira)**: a interface não tem mais página de contas. Lançamentos e custos fixos criados
  sem `account_id` vão para a conta padrão do usuário — a conta ativa mais antiga ou, se não houver nenhuma,
  uma conta "Carteira" (saldo inicial 0) criada automaticamente. A criação é serializada por usuário
  (`pg_advisory_xact_lock`), então requisições simultâneas nunca geram duas Carteiras. Na edição, o
  lançamento permanece na conta em que já estava. O Dashboard mostra o saldo dessas contas como
  "Saldo disponível".

## Limitações conhecidas

- **Recuperação de senha e verificação de e-mail não estão implementadas.** Ambas exigem um serviço de
  envio de e-mail, que não está configurado neste ambiente. Implementar apenas a interface sem o envio
  real simularia uma funcionalidade inexistente — por isso essas rotas não existem na API nem na interface.
  A troca de senha estando autenticado funciona normalmente.
- **Geração de ocorrências de custos fixos** é implementada para periodicidades mensal e anual; regras
  semanais (que a interface não oferece mais para novos cadastros) retornam `501 Not Implemented`.
- **Contas e categorias não têm mais tela própria.** Contas extras, transferências entre contas, saldo
  inicial e categorias privadas continuam disponíveis na API (`/api/v1/accounts`, `/api/v1/transfers`,
  `/api/v1/categories`), mas não pela interface; os endereços antigos `/contas` e `/categorias`
  redirecionam para o Dashboard. Usuários que já tinham várias contas mantêm os lançamentos nelas, e novos
  lançamentos vão para a mais antiga.
- **Passivos não têm mais tela própria.** A página Patrimônio mostra apenas bens e investimentos, por
  decisão de produto. Passivos já cadastrados continuam no banco, acessíveis pela API e descontados no
  patrimônio líquido do Dashboard, mas não podem ser editados pela interface.
- **Evolução do patrimônio líquido ao longo do tempo** (série histórica) não é exposta como um único
  relatório consolidado — o que existe é o histórico de avaliação de cada ativo/passivo/investimento
  individualmente, mais a fotografia atual da composição patrimonial. Calcular corretamente o patrimônio
  "como era" em uma data passada exigiria reconstruir o saldo histórico de cada conta, o que não foi
  implementado para evitar apresentar um número impreciso.
- **Sem integração externa de cotação/mercado.** Valores de ativos, passivos e investimentos são sempre
  estimativas informadas manualmente pelo usuário, com data de referência explícita — a API nunca afirma
  ter uma cotação de mercado em tempo real.
- **Frontend testado via requisições HTTP diretas e verificação de tipos/build**, não através de um
  navegador real neste ambiente (extensão de automação de navegador não disponível). A lógica de
  autenticação, proxy e chamadas de API foi verificada ponta a ponta via `curl`; a renderização visual e a
  responsividade devem ser confirmadas manualmente em um navegador.
# Sistema-de-Controle-Financeiro
