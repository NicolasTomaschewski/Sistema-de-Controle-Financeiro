-- Etapa 3: contas financeiras, categorias, transações e transferências.

CREATE TABLE financial_accounts (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id                 UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    name                    TEXT NOT NULL,
    type                    TEXT NOT NULL,
    institution             TEXT,
    currency                CHAR(3) NOT NULL DEFAULT 'BRL',
    initial_balance         NUMERIC(18,2) NOT NULL DEFAULT 0,
    initial_balance_date    DATE NOT NULL,
    status                  TEXT NOT NULL DEFAULT 'active',
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT financial_accounts_name_not_blank CHECK (length(btrim(name)) > 0),
    CONSTRAINT financial_accounts_type_not_blank CHECK (length(btrim(type)) > 0),
    CONSTRAINT financial_accounts_status_valid CHECK (status IN ('active', 'archived')),
    -- Permite que transações referenciem (account_id, user_id) via chave
    -- estrangeira composta, garantindo no próprio banco que uma transação
    -- nunca aponte para uma conta de outro usuário.
    CONSTRAINT financial_accounts_id_user_unique UNIQUE (id, user_id)
);

CREATE INDEX idx_financial_accounts_user_id ON financial_accounts (user_id);

CREATE TABLE categories (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    -- user_id NULL identifica uma categoria global do sistema, visível a
    -- todos os usuários e não editável por eles.
    user_id         UUID REFERENCES users (id) ON DELETE CASCADE,
    name            TEXT NOT NULL,
    kind            TEXT NOT NULL,
    archived_at     TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT categories_name_not_blank CHECK (length(btrim(name)) > 0),
    CONSTRAINT categories_kind_valid CHECK (kind IN ('income', 'expense'))
);

-- Unicidade de nome por tipo, tratando categorias globais e privadas em
-- índices parciais separados (NULL não garante unicidade em um UNIQUE comum).
CREATE UNIQUE INDEX idx_categories_global_unique ON categories (lower(name), kind) WHERE user_id IS NULL;
CREATE UNIQUE INDEX idx_categories_user_unique ON categories (user_id, lower(name), kind) WHERE user_id IS NOT NULL;
CREATE INDEX idx_categories_user_id ON categories (user_id);

CREATE TABLE transfers (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    from_account_id UUID NOT NULL,
    to_account_id   UUID NOT NULL,
    amount          NUMERIC(18,2) NOT NULL,
    occurred_on     DATE NOT NULL,
    description     TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT transfers_amount_positive CHECK (amount > 0),
    CONSTRAINT transfers_different_accounts CHECK (from_account_id <> to_account_id),
    CONSTRAINT transfers_from_account_fk FOREIGN KEY (from_account_id, user_id) REFERENCES financial_accounts (id, user_id),
    CONSTRAINT transfers_to_account_fk FOREIGN KEY (to_account_id, user_id) REFERENCES financial_accounts (id, user_id)
);

CREATE INDEX idx_transfers_user_id ON transfers (user_id);

CREATE TABLE transactions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    account_id      UUID NOT NULL,
    category_id     UUID,
    transfer_id     UUID REFERENCES transfers (id) ON DELETE CASCADE,
    type            TEXT NOT NULL,
    status          TEXT NOT NULL DEFAULT 'paid',
    amount          NUMERIC(18,2) NOT NULL,
    description     TEXT NOT NULL,
    occurred_on     DATE NOT NULL,
    notes           TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT transactions_amount_positive CHECK (amount > 0),
    CONSTRAINT transactions_description_not_blank CHECK (length(btrim(description)) > 0),
    CONSTRAINT transactions_type_valid CHECK (type IN ('income', 'expense', 'transfer_in', 'transfer_out')),
    CONSTRAINT transactions_status_valid CHECK (status IN ('planned', 'pending', 'paid', 'cancelled')),
    -- Movimentações de transferência não possuem categoria e sempre têm um
    -- transfer_id; receitas/despesas nunca têm transfer_id.
    CONSTRAINT transactions_transfer_consistency CHECK (
        (type IN ('transfer_in', 'transfer_out') AND transfer_id IS NOT NULL AND category_id IS NULL)
        OR
        (type IN ('income', 'expense') AND transfer_id IS NULL)
    ),
    CONSTRAINT transactions_account_fk FOREIGN KEY (account_id, user_id) REFERENCES financial_accounts (id, user_id),
    -- A propriedade/visibilidade da categoria (privada do próprio usuário ou
    -- global do sistema) e a compatibilidade entre category.kind e
    -- transaction.type são validadas na camada de serviço, pois uma
    -- categoria global tem user_id NULL e não pode ser expressa em uma
    -- chave estrangeira composta com transactions.user_id.
    CONSTRAINT transactions_category_fk FOREIGN KEY (category_id) REFERENCES categories (id)
);

CREATE INDEX idx_transactions_user_period ON transactions (user_id, occurred_on);
CREATE INDEX idx_transactions_account_id ON transactions (account_id);
CREATE INDEX idx_transactions_category_id ON transactions (category_id);
CREATE INDEX idx_transactions_transfer_id ON transactions (transfer_id);

-- Categorias padrão do sistema (globais), conforme seção 10.3 da especificação.
INSERT INTO categories (name, kind) VALUES
    ('Moradia', 'expense'),
    ('Água e energia', 'expense'),
    ('Internet e telefone', 'expense'),
    ('Alimentação', 'expense'),
    ('Transporte', 'expense'),
    ('Saúde', 'expense'),
    ('Educação', 'expense'),
    ('Lazer', 'expense'),
    ('Assinaturas', 'expense'),
    ('Compras pessoais', 'expense'),
    ('Impostos', 'expense'),
    ('Outros', 'expense'),
    ('Salário', 'income'),
    ('Trabalho autônomo', 'income'),
    ('Rendimentos', 'income'),
    ('Aluguel recebido', 'income'),
    ('Outras receitas', 'income');
