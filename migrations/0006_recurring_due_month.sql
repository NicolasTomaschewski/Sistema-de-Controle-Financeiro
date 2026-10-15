-- Custos fixos anuais: mês de vencimento explícito para regras anuais.
-- Regras anuais já existentes recebem o mês da data inicial, preservando a
-- semântica implícita anterior antes de a restrição passar a valer.

ALTER TABLE recurring_rules ADD COLUMN due_month SMALLINT;

UPDATE recurring_rules
SET due_month = EXTRACT(MONTH FROM start_date)::SMALLINT
WHERE periodicity = 'yearly' AND due_month IS NULL;

ALTER TABLE recurring_rules
    ADD CONSTRAINT recurring_rules_due_month_valid CHECK (due_month IS NULL OR (due_month BETWEEN 1 AND 12)),
    ADD CONSTRAINT recurring_rules_yearly_has_due_month CHECK (periodicity <> 'yearly' OR due_month IS NOT NULL);
