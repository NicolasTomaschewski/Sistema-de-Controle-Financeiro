#!/usr/bin/env bash
# Sobe o ambiente local completo (PostgreSQL + API + frontend) com um único comando:
#   ./dev.sh
# Ctrl+C encerra a API e o frontend. O PostgreSQL continua rodando em segundo plano
# (pare com `docker compose down`; os dados ficam no volume finctrl_pgdata).
set -euo pipefail

cd "$(dirname "$0")"

log() { printf '\033[1;34m[dev]\033[0m %s\n' "$*"; }
fail() { printf '\033[1;31m[dev] erro:\033[0m %s\n' "$*" >&2; exit 1; }

for cmd in go node npm docker openssl curl; do
  command -v "$cmd" >/dev/null 2>&1 || fail "comando '$cmd' não encontrado. Veja os pré-requisitos no README.md."
done
docker compose version >/dev/null 2>&1 || fail "o plugin 'docker compose' não está disponível."

# 1. .env: criado a partir do exemplo na primeira execução, com SESSION_SECRET aleatório.
if [[ ! -f .env ]]; then
  log "criando .env a partir de .env.example com um SESSION_SECRET aleatório"
  secret="$(openssl rand -hex 32)"
  sed "s|^SESSION_SECRET=.*|SESSION_SECRET=${secret}|" .env.example > .env
fi
set -a
# shellcheck disable=SC1091
source .env
set +a

# 2. PostgreSQL: aguarda o healthcheck do docker-compose.yml.
log "iniciando PostgreSQL (docker compose)"
docker compose up -d --wait postgres

# 3. Dependências do frontend, apenas se ainda não instaladas.
if [[ ! -d web/node_modules ]]; then
  log "instalando dependências do frontend (npm ci)"
  (cd web && npm ci)
fi

# Ao sair (Ctrl+C ou falha de um dos processos), encerra todos os processos filhos.
bin_dir="$(mktemp -d)"
# Encerra um processo e todos os seus descendentes (ex.: npm -> sh -> node/vite).
kill_tree() {
  local child
  for child in $(pgrep -P "$1" 2>/dev/null); do kill_tree "$child"; done
  kill "$1" 2>/dev/null || true
}
cleanup() {
  trap - EXIT INT TERM
  log "encerrando API e frontend"
  local pid
  for pid in $(jobs -p); do kill_tree "$pid"; done
  wait 2>/dev/null || true
  rm -rf "$bin_dir"
}
trap cleanup EXIT INT TERM

# 4. API: compila e executa o binário (as migrações rodam automaticamente na inicialização).
log "compilando a API"
go build -o "$bin_dir/api" ./cmd/api
log "iniciando a API em http://localhost:${HTTP_PORT:-8080}"
"$bin_dir/api" &
api_pid=$!

for _ in $(seq 1 60); do
  curl -fsS "http://localhost:${HTTP_PORT:-8080}/healthz" >/dev/null 2>&1 && break
  kill -0 "$api_pid" 2>/dev/null || fail "a API encerrou durante a inicialização (veja o log acima)."
  sleep 0.5
done
curl -fsS "http://localhost:${HTTP_PORT:-8080}/healthz" >/dev/null 2>&1 || fail "a API não respondeu em /healthz a tempo."

# Conta de demonstração, apenas para desenvolvimento local. É criada pelo fluxo normal
# de cadastro da API (sem atalhos no banco) e não existe fora deste script.
DEMO_EMAIL="demo@finctrl.local"
DEMO_PASSWORD="demo12345"
if [[ "${APP_ENV:-development}" == "development" ]]; then
  status="$(curl -s -o /dev/null -w '%{http_code}' -X POST \
    "http://localhost:${HTTP_PORT:-8080}/api/v1/auth/register" \
    -H 'Content-Type: application/json' \
    -d "{\"name\":\"Usuário Demo\",\"email\":\"${DEMO_EMAIL}\",\"password\":\"${DEMO_PASSWORD}\"}")"
  case "$status" in
    201) log "conta de demonstração criada: ${DEMO_EMAIL} / ${DEMO_PASSWORD}" ;;
    409) log "conta de demonstração: ${DEMO_EMAIL} / ${DEMO_PASSWORD}" ;;
    *)   log "aviso: não foi possível criar a conta de demonstração (HTTP ${status})" ;;
  esac
  # Dados fictícios da conta demo (um ano de receitas, despesas, bens e investimentos),
  # gravados pela API. Só roda se a conta ainda estiver vazia.
  if [[ "$status" == "201" || "$status" == "409" ]]; then
    go build -o "$bin_dir/demoseed" ./cmd/demoseed
    "$bin_dir/demoseed" -api "http://localhost:${HTTP_PORT:-8080}" -email "$DEMO_EMAIL" -password "$DEMO_PASSWORD" \
      | while read -r line; do log "demo: $line"; done \
      || log "aviso: não foi possível gerar os dados de demonstração"
  fi
fi

# 5. Frontend (Vite), com proxy de /api para a API.
log "iniciando o frontend em http://localhost:5173"
(cd web && exec npm run dev) &

log "pronto — abra http://localhost:5173 (Ctrl+C para encerrar)"
if [[ "${APP_ENV:-development}" == "development" ]]; then
  log "login de demonstração → e-mail: ${DEMO_EMAIL}  senha: ${DEMO_PASSWORD}"
fi
# Se qualquer um dos processos terminar, encerra o outro também.
wait -n
