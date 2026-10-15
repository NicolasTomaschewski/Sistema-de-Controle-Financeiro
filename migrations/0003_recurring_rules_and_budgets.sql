-- Etapa 4: despesas recorrentes e orçamento mensal por categoria.

CREATE TABLE recurring_rules (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    account_id      UUID,
    category_id     UUID REFERENCES categories (id),
    description     TEXT NOT NULL,
    planned_amount  NUMERIC(18,2) NOT NULL,
    periodicity     TEXT NOT NULL DEFAULT 'monthly',
    start_date      DATE NOT NULL,
    end_date        DATE,
    due_day         SMALLINT,
    active          BOOLEAN NOT NULL DEFAULT true,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT recurring_rules_description_not_blank CHECK (length(btrim(description)) > 0),
    CONSTRAINT recurring_rules_amount_positive CHECK (planned_amount > 0),
    CONSTRAINT recurring_rules_periodicity_valid CHECK (periodicity IN ('monthly', 'weekly', 'yearly')),
    CONSTRAINT recurring_rules_due_day_valid CHECK (due_day IS NULL OR (due_day BETWEEN 1 AND 31)),
    CONSTRAINT recurring_rules_end_after_start CHECK (end_date IS NULL OR end_date >= start_date),
    CONSTRAINT recurring_rules_account_fk FOREIGN KEY (account_id, user_id) REFERENCES financial_accounts (id, user_id)
);

CREATE INDEX idx_recurring_rules_user_id ON recurring_rules (user_id);

CREATE TABLE budgets (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    category_id     UUID NOT NULL REFERENCES categories (id),
    year            INTEGER NOT NULL,
    month           INTEGER NOT NULL,
    planned_amount  NUMERIC(18,2) NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT budgets_amount_positive CHECK (planned_amount > 0),
    CONSTRAINT budgets_month_valid CHECK (month BETWEEN 1 AND 12),
    CONSTRAINT budgets_year_valid CHECK (year BETWEEN 2000 AND 2100),
    CONSTRAINT budgets_unique_period UNIQUE (user_id, category_id, year, month)
);

CREATE INDEX idx_budgets_user_period ON budgets (user_id, year, month);

-- Vincula uma movimentação gerada à regra de recorrência que a originou,
-- permitindo distinguir ocorrências previstas de lançamentos avulsos e
-- evitar duplicação de geração para o mesmo período.
ALTER TABLE transactions ADD COLUMN recurring_rule_id UUID REFERENCES recurring_rules (id);

CREATE UNIQUE INDEX idx_transactions_recurring_occurrence
    ON transactions (recurring_rule_id, occurred_on)
    WHERE recurring_rule_id IS NOT NULL;

CREATE INDEX idx_transactions_recurring_rule_id ON transactions (recurring_rule_id);
