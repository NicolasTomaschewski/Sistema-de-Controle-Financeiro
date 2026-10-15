import { api, getWithMeta } from './client'
import type {
  Account, AnnualSummary, Asset, CostSummary, NetWorthComposition, Transaction,
  BudgetSummary,
  Category,
  DashboardSummary,
  Investment,
  InvestmentMovement,
  InvestmentSummary,
  Liability,
  MonthlyFlowPoint,
  MonthlySummary,
  NetWorth,
  Profile,
  RecurringRule,
  Transfer,
  Valuation,
} from './types'

// Autenticação
export const authApi = {
  register: (input: { name: string; email: string; password: string }) =>
    api.post<{ id: string; name: string; email: string }>('/api/v1/auth/register', input),
  login: (input: { email: string; password: string }) => api.post<{ message: string }>('/api/v1/auth/login', input),
  logout: () => api.post<{ message: string }>('/api/v1/auth/logout'),
}

// Perfil
export const profileApi = {
  get: () => api.get<Profile>('/api/v1/me'),
  update: (input: { name: string; email: string }) => api.patch<Profile>('/api/v1/me', input),
  changePassword: (input: { current_password: string; new_password: string }) =>
    api.patch<{ message: string }>('/api/v1/me/password', input),
}

// Contas financeiras
export const accountsApi = {
  list: (includeArchived = false) => api.get<Account[]>('/api/v1/accounts', { include_archived: includeArchived }),
  get: (id: string) => api.get<Account>(`/api/v1/accounts/${id}`),
  create: (input: Record<string, unknown>) => api.post<Account>('/api/v1/accounts', input),
  update: (id: string, input: Record<string, unknown>) => api.patch<Account>(`/api/v1/accounts/${id}`, input),
  remove: (id: string) => api.delete<{ message: string }>(`/api/v1/accounts/${id}`),
}

// Categorias
export const categoriesApi = {
  list: (includeArchived = false) => api.get<Category[]>('/api/v1/categories', { include_archived: includeArchived }),
  create: (input: { name: string; kind: 'income' | 'expense' }) => api.post<Category>('/api/v1/categories', input),
  update: (id: string, input: { name: string }) => api.patch<Category>(`/api/v1/categories/${id}`, input),
  remove: (id: string) => api.delete<{ message: string }>(`/api/v1/categories/${id}`),
}

// Lançamentos (receitas e despesas) e transferências
export const transactionsApi = {
  list: (filter: Record<string, string | number | undefined>) => getWithMeta<Transaction[]>('/api/v1/transactions', filter),
  get: (id: string) => api.get<Transaction>(`/api/v1/transactions/${id}`),
  create: (input: Record<string, unknown>) => api.post<Transaction>('/api/v1/transactions', input),
  update: (id: string, input: Record<string, unknown>) => api.patch<Transaction>(`/api/v1/transactions/${id}`, input),
  remove: (id: string) => api.delete<{ message: string }>(`/api/v1/transactions/${id}`),
  monthlySummary: (year: number, month: number, compare = false) =>
    api.get<MonthlySummary>('/api/v1/transactions/monthly-summary', { year, month, compare }),
  annualSummary: (year: number) => api.get<AnnualSummary>('/api/v1/transactions/annual-summary', { year }),
  listTransfers: () => api.get<Transfer[]>('/api/v1/transfers'),
  createTransfer: (input: Record<string, unknown>) => api.post<Transfer>('/api/v1/transfers', input),
}

// Despesas recorrentes
export const recurringApi = {
  list: (activeOnly = false) => api.get<RecurringRule[]>('/api/v1/recurring-rules', { active_only: activeOnly }),
  summary: () => api.get<CostSummary>('/api/v1/recurring-rules/summary'),
  create: (input: Record<string, unknown>) => api.post<RecurringRule>('/api/v1/recurring-rules', input),
  update: (id: string, input: Record<string, unknown>) => api.patch<RecurringRule>(`/api/v1/recurring-rules/${id}`, input),
  remove: (id: string) => api.delete<{ message: string }>(`/api/v1/recurring-rules/${id}`),
  generate: (id: string, year: number, month: number) =>
    api.post<{ transaction_id: string; status: string; occurred_on: string; created: boolean }>(
      `/api/v1/recurring-rules/${id}/generate?year=${year}&month=${month}`,
    ),
}

// Orçamento mensal
export const budgetsApi = {
  summary: (year: number, month: number) => api.get<BudgetSummary[]>('/api/v1/budgets', { year, month }),
  create: (input: Record<string, unknown>) => api.post<unknown>('/api/v1/budgets', input),
  update: (id: string, plannedAmount: string) => api.patch<unknown>(`/api/v1/budgets/${id}`, { planned_amount: plannedAmount }),
  remove: (id: string) => api.delete<{ message: string }>(`/api/v1/budgets/${id}`),
}

// Patrimônio: ativos e passivos
export const assetsApi = {
  list: (includeArchived = false) => api.get<Asset[]>('/api/v1/assets', { include_archived: includeArchived }),
  create: (input: Record<string, unknown>) => api.post<Asset>('/api/v1/assets', input),
  update: (id: string, input: Record<string, unknown>) => api.patch<Asset>(`/api/v1/assets/${id}`, input),
  remove: (id: string) => api.delete<{ message: string }>(`/api/v1/assets/${id}`),
  listValuations: (id: string) => api.get<Valuation[]>(`/api/v1/assets/${id}/valuations`),
  addValuation: (id: string, input: Record<string, unknown>) => api.post<Valuation>(`/api/v1/assets/${id}/valuations`, input),
}

export const liabilitiesApi = {
  list: (includeSettled = false) => api.get<Liability[]>('/api/v1/liabilities', { include_settled: includeSettled }),
  create: (input: Record<string, unknown>) => api.post<Liability>('/api/v1/liabilities', input),
  update: (id: string, input: Record<string, unknown>) => api.patch<Liability>(`/api/v1/liabilities/${id}`, input),
  remove: (id: string) => api.delete<{ message: string }>(`/api/v1/liabilities/${id}`),
  listValuations: (id: string) => api.get<Valuation[]>(`/api/v1/liabilities/${id}/valuations`),
  addValuation: (id: string, input: Record<string, unknown>) => api.post<Valuation>(`/api/v1/liabilities/${id}/valuations`, input),
}

export const netWorthApi = {
  get: () => api.get<NetWorth>('/api/v1/net-worth'),
  composition: () => api.get<NetWorthComposition>('/api/v1/net-worth/composition'),
}

// Investimentos
export const investmentsApi = {
  list: (includeClosed = false) => api.get<Investment[]>('/api/v1/investments', { include_closed: includeClosed }),
  get: (id: string) => api.get<Investment>(`/api/v1/investments/${id}`),
  create: (input: Record<string, unknown>) => api.post<Investment>('/api/v1/investments', input),
  update: (id: string, input: Record<string, unknown>) => api.patch<Investment>(`/api/v1/investments/${id}`, input),
  remove: (id: string) => api.delete<{ message: string }>(`/api/v1/investments/${id}`),
  summary: () => api.get<InvestmentSummary>('/api/v1/investments/summary'),
  listMovements: (id: string) => api.get<InvestmentMovement[]>(`/api/v1/investments/${id}/transactions`),
  recordMovement: (id: string, input: Record<string, unknown>) => api.post<InvestmentMovement>(`/api/v1/investments/${id}/transactions`, input),
  listValuations: (id: string) => api.get<Valuation[]>(`/api/v1/investments/${id}/valuations`),
  addValuation: (id: string, input: Record<string, unknown>) => api.post<Valuation>(`/api/v1/investments/${id}/valuations`, input),
}

// Dashboard
export const dashboardApi = {
  summary: (year: number, month: number) => api.get<DashboardSummary>('/api/v1/dashboard/summary', { year, month }),
  monthlyFlow: (year: number, month: number, months = 6) =>
    api.get<MonthlyFlowPoint[]>('/api/v1/dashboard/monthly-flow', { year, month, months }),
  expensesByCategory: (year: number, month: number) =>
    api.get<{ category_id: string | null; category_name: string; total: string }[]>('/api/v1/dashboard/expenses-by-category', { year, month }),
}
