package dashboard

import (
	"net/http"
	"strconv"
	"time"

	"github.com/nicolastomaschewski/controle-financeiro/internal/authctx"
	"github.com/nicolastomaschewski/controle-financeiro/internal/httpx"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/dashboard/summary", h.summary)
	mux.HandleFunc("GET /api/v1/dashboard/monthly-flow", h.monthlyFlow)
	mux.HandleFunc("GET /api/v1/dashboard/expenses-by-category", h.expensesByCategory)
}

func yearMonthFromQuery(q interface{ Get(string) string }) (int, time.Month) {
	now := time.Now().UTC()
	year := now.Year()
	month := now.Month()
	if v := q.Get("year"); v != "" {
		if y, err := strconv.Atoi(v); err == nil {
			year = y
		}
	}
	if v := q.Get("month"); v != "" {
		if m, err := strconv.Atoi(v); err == nil && m >= 1 && m <= 12 {
			month = time.Month(m)
		}
	}
	return year, month
}

type summaryResponse struct {
	AccountsBalance    string                 `json:"accounts_balance"`
	MonthlyIncome      string                 `json:"monthly_income"`
	MonthlyExpense     string                 `json:"monthly_expense"`
	MonthlyResult      string                 `json:"monthly_result"`
	NetWorth           netWorthDTO            `json:"net_worth"`
	TotalInvested      string                 `json:"total_invested"`
	TotalMarketValue   string                 `json:"total_market_value"`
	RecentTransactions []recentTransactionDTO `json:"recent_transactions"`
}

type netWorthDTO struct {
	AccountsBalance     string `json:"accounts_balance"`
	DeclaredAssetsValue string `json:"declared_assets_value"`
	InvestmentsValue    string `json:"investments_value"`
	GrossWorth          string `json:"gross_worth"`
	TotalLiabilities    string `json:"total_liabilities"`
	NetWorth            string `json:"net_worth"`
	AsOf                string `json:"as_of"`
}

type recentTransactionDTO struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	Type        string `json:"type"`
	Status      string `json:"status"`
	Amount      string `json:"amount"`
	OccurredOn  string `json:"occurred_on"`
}

func (h *Handler) summary(w http.ResponseWriter, r *http.Request) {
	userID, _ := authctx.UserID(r.Context())
	year, month := yearMonthFromQuery(r.URL.Query())

	s, err := h.service.Summary(r.Context(), userID, year, month)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "Não foi possível calcular o resumo do dashboard.")
		return
	}

	recent := make([]recentTransactionDTO, 0, len(s.RecentTransactions))
	for _, t := range s.RecentTransactions {
		recent = append(recent, recentTransactionDTO{
			ID: t.ID, Description: t.Description, Type: t.Type, Status: t.Status,
			Amount: t.Amount.StringFixed(2), OccurredOn: t.OccurredOn.Format("2006-01-02"),
		})
	}

	httpx.JSON(w, http.StatusOK, summaryResponse{
		AccountsBalance: s.AccountsBalance.StringFixed(2),
		MonthlyIncome:   s.MonthlyIncome.StringFixed(2),
		MonthlyExpense:  s.MonthlyExpense.StringFixed(2),
		MonthlyResult:   s.MonthlyResult.StringFixed(2),
		NetWorth: netWorthDTO{
			AccountsBalance:     s.NetWorth.AccountsBalance.StringFixed(2),
			DeclaredAssetsValue: s.NetWorth.DeclaredAssetsValue.StringFixed(2),
			InvestmentsValue:    s.NetWorth.InvestmentsValue.StringFixed(2),
			GrossWorth:          s.NetWorth.GrossWorth.StringFixed(2),
			TotalLiabilities:    s.NetWorth.TotalLiabilities.StringFixed(2),
			NetWorth:            s.NetWorth.NetWorth.StringFixed(2),
			AsOf:                s.NetWorth.AsOf.Format("2006-01-02T15:04:05Z07:00"),
		},
		TotalInvested:      s.TotalInvested.StringFixed(2),
		TotalMarketValue:   s.TotalMarketValue.StringFixed(2),
		RecentTransactions: recent,
	})
}

type monthlyFlowPointDTO struct {
	Year         int    `json:"year"`
	Month        int    `json:"month"`
	TotalIncome  string `json:"total_income"`
	TotalExpense string `json:"total_expense"`
	Result       string `json:"result"`
}

func (h *Handler) monthlyFlow(w http.ResponseWriter, r *http.Request) {
	userID, _ := authctx.UserID(r.Context())
	q := r.URL.Query()
	year, month := yearMonthFromQuery(q)

	months := 6
	if v := q.Get("months"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			months = n
		}
	}

	points, err := h.service.MonthlyFlow(r.Context(), userID, year, month, months)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "Não foi possível calcular a evolução mensal.")
		return
	}

	resp := make([]monthlyFlowPointDTO, 0, len(points))
	for _, p := range points {
		resp = append(resp, monthlyFlowPointDTO{
			Year: p.Year, Month: p.Month,
			TotalIncome: p.TotalIncome.StringFixed(2), TotalExpense: p.TotalExpense.StringFixed(2), Result: p.Result.StringFixed(2),
		})
	}
	httpx.JSON(w, http.StatusOK, resp)
}

type categoryBreakdownDTO struct {
	CategoryID   *string `json:"category_id"`
	CategoryName string  `json:"category_name"`
	Total        string  `json:"total"`
}

func (h *Handler) expensesByCategory(w http.ResponseWriter, r *http.Request) {
	userID, _ := authctx.UserID(r.Context())
	year, month := yearMonthFromQuery(r.URL.Query())

	items, err := h.service.ExpensesByCategory(r.Context(), userID, year, month)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "Não foi possível calcular a distribuição por categoria.")
		return
	}

	resp := make([]categoryBreakdownDTO, 0, len(items))
	for _, it := range items {
		resp = append(resp, categoryBreakdownDTO{CategoryID: it.CategoryID, CategoryName: it.CategoryName, Total: it.Total.StringFixed(2)})
	}
	httpx.JSON(w, http.StatusOK, resp)
}
