-- Etapa 5: patrimônio — ativos, passivos, histórico de avaliações e
-- cálculo de patrimônio líquido.

CREATE TABLE assets (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    name            TEXT NOT NULL,
    type            TEXT NOT NULL,
    current_value   NUMERIC(18,2) NOT NULL,
    valuation_date  DATE NOT NULL,
    notes           TEXT,
    status          TEXT NOT NULL DEFAULT 'active',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT assets_name_not_blank CHECK (length(btrim(name)) > 0),
    CONSTRAINT assets_type_not_blank CHECK (length(btrim(type)) > 0),
    CONSTRAINT assets_value_not_negative CHECK (current_value >= 0),
    CONSTRAINT assets_status_valid CHECK (status IN ('active', 'archived')),
    CONSTRAINT assets_id_user_unique UNIQUE (id, user_id)
);

CREATE INDEX idx_assets_user_id ON assets (user_id);

-- Histórico de avaliações: editar o valor atual do ativo nunca apaga as
-- avaliações anteriores, permitindo acompanhar a evolução patrimonial.
CREATE TABLE asset_valuations (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    asset_id        UUID NOT NULL,
    user_id         UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    value           NUMERIC(18,2) NOT NULL,
    valuation_date  DATE NOT NULL,
    notes           TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT asset_valuations_value_not_negative CHECK (value >= 0),
    CONSTRAINT asset_valuations_unique_date UNIQUE (asset_id, valuation_date),
    CONSTRAINT asset_valuations_asset_fk FOREIGN KEY (asset_id, user_id) REFERENCES assets (id, user_id) ON DELETE CASCADE
);

CREATE INDEX idx_asset_valuations_asset_id ON asset_valuations (asset_id, valuation_date);

CREATE TABLE liabilities (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    name            TEXT NOT NULL,
    type            TEXT NOT NULL,
    principal_amount NUMERIC(18,2) NOT NULL,
    current_balance NUMERIC(18,2) NOT NULL,
    reference_date  DATE NOT NULL,
    notes           TEXT,
    status          TEXT NOT NULL DEFAULT 'active',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT liabilities_name_not_blank CHECK (length(btrim(name)) > 0),
    CONSTRAINT liabilities_type_not_blank CHECK (length(btrim(type)) > 0),
    CONSTRAINT liabilities_principal_not_negative CHECK (principal_amount >= 0),
    CONSTRAINT liabilities_balance_not_negative CHECK (current_balance >= 0),
    CONSTRAINT liabilities_status_valid CHECK (status IN ('active', 'settled')),
    CONSTRAINT liabilities_id_user_unique UNIQUE (id, user_id)
);

CREATE INDEX idx_liabilities_user_id ON liabilities (user_id);

-- Histórico de saldo devedor: cada baixa/atualização de saldo preserva o
-- registro anterior em vez de sobrescrevê-lo silenciosamente.
CREATE TABLE liability_valuations (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    liability_id    UUID NOT NULL,
    user_id         UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    balance         NUMERIC(18,2) NOT NULL,
    reference_date  DATE NOT NULL,
    notes           TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT liability_valuations_balance_not_negative CHECK (balance >= 0),
    CONSTRAINT liability_valuations_unique_date UNIQUE (liability_id, reference_date),
    CONSTRAINT liability_valuations_fk FOREIGN KEY (liability_id, user_id) REFERENCES liabilities (id, user_id) ON DELETE CASCADE
);

CREATE INDEX idx_liability_valuations_liability_id ON liability_valuations (liability_id, reference_date);
