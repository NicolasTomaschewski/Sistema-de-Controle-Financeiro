// Tipos espelhando os DTOs JSON retornados pela API Go. Valores monetários
// chegam sempre como string decimal (ex.: "1234.56") para não perder
// precisão em operações numéricas no cliente — nunca convertidos para
// number para fins de cálculo, apenas para exibição formatada.

export type Profile = {
  id: string
  name: string
  email: string
  email_verified_at: string | null
}

export type Account = {
  id: string
  name: string
  type: string
  institution: string | null
  currency: string
  initial_balance: string
  initial_balance_date: string
  status: 'active' | 'archived'
  current_balance: string
}

export type Category = {
  id: string
  name: string
  kind: 'income' | 'expense'
  is_system: boolean
  archived: boolean
}

export type TransactionStatus = 'planned' | 'pending' | 'paid' | 'cancelled'
export type TransactionType = 'income' | 'expense' | 'transfer_in' | 'transfer_out'

export type Transaction = {
  id: string
  account_id: string
  category_id: string | null
  transfer_id: string | null
  investment_transaction_id: string | null
  type: TransactionType
  status: TransactionStatus
  amount: string
  description: string
  occurred_on: string
  notes: string | null
}

export type Transfer = {
  id: string
  from_account_id: string
  to_account_id: string
  amount: string
  description: string
  occurred_on: string
}

export type CategoryBreakdown = { category_id: string | null; category_name: string; total: string }

export type AnnualSummary = {
  year: number
  total_income: string
  total_expense: string
  result: string
  expenses_by_category: CategoryBreakdown[]
  income_by_category: CategoryBreakdown[]
}

export type MonthlySummary = {
  year: number
  month: number
  total_income: string
  total_expense: string
  result: string
  expenses_by_category: CategoryBreakdown[]
  income_by_category: CategoryBreakdown[]
  previous_month?: { total_income: string; total_expense: string; result: string }
}

export type RecurringRule = {
  id: string
  account_id: string | null
  category_id: string | null
  description: string
  planned_amount: string
  periodicity: 'monthly' | 'weekly' | 'yearly'
  start_date: string
  end_date: string | null
  due_day: number | null
  due_month: number | null
  active: boolean
  annual_cost: string
  monthly_equivalent: string
}

export type CostSummary = {
  monthly_rules_total: string
  yearly_rules_total: string
  annual_total: string
  monthly_equivalent: string
  by_category: { category_id: string | null; annual_total: string; monthly_equivalent: string }[]
}

export type BudgetSummary = {
  category_id: string
  category_name: string
  planned_amount: string
  spent_amount: string
  available_amount: string
  percent_used: string
  over_limit: boolean
}

export type Asset = {
  id: string
  name: string
  type: string
  current_value: string
  valuation_date: string
  notes: string | null
  status: 'active' | 'archived'
  is_estimate: boolean
}

export type Valuation = {
  id: string
  value?: string
  market_value?: string
  balance?: string
  valuation_date?: string
  position_date?: string
  reference_date?: string
  notes: string | null
}

export type Liability = {
  id: string
  name: string
  type: string
  principal_amount: string
  current_balance: string
  reference_date: string
  notes: string | null
  status: 'active' | 'settled'
}

export type Investment = {
  id: string
  name: string
  asset_type: string
  institution: string | null
  ticker: string | null
  quantity: string | null
  average_price: string | null
  invested_amount: string
  market_value: string
  unrealized_gain: string
  simple_return_percent: string | null
  position_date: string
  acquisition_date: string | null
  maturity_date: string | null
  include_in_net_worth: boolean
  notes: string | null
  status: 'active' | 'closed'
  is_estimate: boolean
}

export type InvestmentMovement = {
  id: string
  account_id: string | null
  type: 'contribution' | 'redemption' | 'income' | 'fee'
  amount: string
  quantity: string | null
  fees: string
  realized_gain: string | null
  occurred_on: string
  notes: string | null
}

export type InvestmentSummary = {
  total_invested: string
  total_market_value: string
  unrealized_gain: string
  total_realized_result: string
  by_asset_type: Record<string, string>
}

export type NetWorth = {
  accounts_balance: string
  declared_assets_value: string
  investments_value: string
  gross_worth: string
  total_liabilities: string
  net_worth: string
  as_of: string
}

export type DashboardSummary = {
  accounts_balance: string
  monthly_income: string
  monthly_expense: string
  monthly_result: string
  net_worth: NetWorth
  total_invested: string
  total_market_value: string
  recent_transactions: { id: string; description: string; type: TransactionType; status: TransactionStatus; amount: string; occurred_on: string }[]
}

export type MonthlyFlowPoint = {
  year: number
  month: number
  total_income: string
  total_expense: string
  result: string
}

export type NetWorthComposition = {
  assets_value: string
  investments_value: string
  total: string
  slices: { group: 'asset' | 'investment'; type: string; value: string; share_percent: string }[]
  as_of: string
}
