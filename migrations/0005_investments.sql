-- Etapa 6: investimentos, movimentações e histórico de valor de mercado.

CREATE TABLE investments (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id               UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    name                  TEXT NOT NULL,
    asset_type            TEXT NOT NULL,
    institution           TEXT,
    ticker                TEXT,
    quantity              NUMERIC(24,8),
    average_price         NUMERIC(18,6),
    invested_amount       NUMERIC(18,2) NOT NULL DEFAULT 0,
    market_value          NUMERIC(18,2) NOT NULL DEFAULT 0,
    position_date         DATE NOT NULL,
    acquisition_date      DATE,
    maturity_date         DATE,
    -- Deixa explícito se esta posição deve ser somada ao patrimônio líquido
    -- geral, evitando contar o mesmo valor duas vezes quando o dinheiro já
    -- está representado em uma conta financeira do tipo "investimento".
    include_in_net_worth  BOOLEAN NOT NULL DEFAULT true,
    notes                 TEXT,
    status                TEXT NOT NULL DEFAULT 'active',
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT investments_name_not_blank CHECK (length(btrim(name)) > 0),
    CONSTRAINT investments_asset_type_not_blank CHECK (length(btrim(asset_type)) > 0),
    CONSTRAINT investments_invested_amount_not_negative CHECK (invested_amount >= 0),
    CONSTRAINT investments_market_value_not_negative CHECK (market_value >= 0),
    CONSTRAINT investments_quantity_not_negative CHECK (quantity IS NULL OR quantity >= 0),
    CONSTRAINT investments_status_valid CHECK (status IN ('active', 'closed')),
    CONSTRAINT investments_id_user_unique UNIQUE (id, user_id)
);

CREATE INDEX idx_investments_user_id ON investments (user_id);

CREATE TABLE investment_transactions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    investment_id   UUID NOT NULL,
    account_id      UUID,
    type            TEXT NOT NULL,
    amount          NUMERIC(18,2) NOT NULL,
    quantity        NUMERIC(24,8),
    fees            NUMERIC(18,2) NOT NULL DEFAULT 0,
    -- Ganho realizado da operação, calculado e armazenado pela camada de
    -- serviço apenas quando há dados suficientes (resgate de ativo com
    -- quantidade e preço médio conhecidos). Nunca inventado quando faltam dados.
    realized_gain   NUMERIC(18,2),
    occurred_on     DATE NOT NULL,
    notes           TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT investment_transactions_amount_positive CHECK (amount > 0),
    CONSTRAINT investment_transactions_fees_not_negative CHECK (fees >= 0),
    CONSTRAINT investment_transactions_quantity_not_negative CHECK (quantity IS NULL OR quantity >= 0),
    CONSTRAINT investment_transactions_type_valid CHECK (type IN ('contribution', 'redemption', 'income', 'fee')),
    CONSTRAINT investment_transactions_investment_fk FOREIGN KEY (investment_id, user_id) REFERENCES investments (id, user_id),
    CONSTRAINT investment_transactions_account_fk FOREIGN KEY (account_id, user_id) REFERENCES financial_accounts (id, user_id)
);

CREATE INDEX idx_investment_transactions_investment_id ON investment_transactions (investment_id, occurred_on);
CREATE INDEX idx_investment_transactions_user_id ON investment_transactions (user_id);

-- Histórico de valor de mercado, independente das movimentações de caixa.
CREATE TABLE investment_valuations (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    investment_id   UUID NOT NULL,
    user_id         UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    market_value    NUMERIC(18,2) NOT NULL,
    position_date   DATE NOT NULL,
    notes           TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT investment_valuations_value_not_negative CHECK (market_value >= 0),
    CONSTRAINT investment_valuations_unique_date UNIQUE (investment_id, position_date),
    CONSTRAINT investment_valuations_fk FOREIGN KEY (investment_id, user_id) REFERENCES investments (id, user_id) ON DELETE CASCADE
);

CREATE INDEX idx_investment_valuations_investment_id ON investment_valuations (investment_id, position_date);

-- Vincula uma movimentação de investimento a uma perna (transaction) em
-- conta financeira, reaproveitando o mesmo mecanismo de transfer_in/out já
-- usado para transferências entre contas — uma transferência para uma
-- posição de investimento é tratada como movimentação patrimonial, nunca
-- como despesa de consumo.
ALTER TABLE transactions ADD COLUMN investment_transaction_id UUID REFERENCES investment_transactions (id);

ALTER TABLE transactions DROP CONSTRAINT transactions_transfer_consistency;
ALTER TABLE transactions ADD CONSTRAINT transactions_transfer_consistency CHECK (
    (
        type IN ('transfer_in', 'transfer_out')
        AND category_id IS NULL
        AND (transfer_id IS NOT NULL OR investment_transaction_id IS NOT NULL)
        AND NOT (transfer_id IS NOT NULL AND investment_transaction_id IS NOT NULL)
    )
    OR
    (
        type IN ('income', 'expense')
        AND transfer_id IS NULL
        AND investment_transaction_id IS NULL
    )
);

CREATE INDEX idx_transactions_investment_transaction_id ON transactions (investment_transaction_id);
