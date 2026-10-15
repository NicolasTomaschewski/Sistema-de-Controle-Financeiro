-- Etapa 2: identidade do usuário, sessões e controle de tentativas de login.
-- gen_random_uuid() é nativo a partir do PostgreSQL 13, sem necessidade de extensão.

CREATE TABLE users (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name                TEXT NOT NULL,
    email               TEXT NOT NULL,
    email_normalized    TEXT NOT NULL,
    password_hash       TEXT NOT NULL,
    email_verified_at   TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT users_email_normalized_unique UNIQUE (email_normalized),
    CONSTRAINT users_name_not_blank CHECK (length(btrim(name)) > 0)
);

-- email_normalized guarda o e-mail em minúsculas para garantir unicidade
-- independente de caixa, sem depender da extensão citext.
CREATE INDEX idx_users_email_normalized ON users (email_normalized);

CREATE TABLE sessions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    token_hash      TEXT NOT NULL,
    user_agent      TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at      TIMESTAMPTZ NOT NULL,
    last_used_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    revoked_at      TIMESTAMPTZ,
    CONSTRAINT sessions_token_hash_unique UNIQUE (token_hash)
);

CREATE INDEX idx_sessions_user_id ON sessions (user_id);
CREATE INDEX idx_sessions_expires_at ON sessions (expires_at);

-- Registra tentativas de login (sucesso e falha) para permitir limitar
-- tentativas excessivas de autenticação de forma persistente, sobrevivendo a
-- reinicializações do processo.
CREATE TABLE login_attempts (
    id              BIGSERIAL PRIMARY KEY,
    email_normalized TEXT NOT NULL,
    ip_address      TEXT NOT NULL,
    success         BOOLEAN NOT NULL,
    attempted_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_login_attempts_lookup ON login_attempts (email_normalized, attempted_at);
CREATE INDEX idx_login_attempts_ip_lookup ON login_attempts (ip_address, attempted_at);

-- Eventos de auditoria para ações relevantes de segurança e de negócio.
CREATE TABLE audit_events (
    id          BIGSERIAL PRIMARY KEY,
    user_id     UUID REFERENCES users (id) ON DELETE SET NULL,
    action      TEXT NOT NULL,
    entity_type TEXT NOT NULL,
    entity_id   TEXT,
    metadata    JSONB,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_audit_events_user_id ON audit_events (user_id, created_at);
