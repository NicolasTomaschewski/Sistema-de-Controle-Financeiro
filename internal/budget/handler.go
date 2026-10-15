package budget

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/shopspring/decimal"

	"github.com/nicolastomaschewski/controle-financeiro/internal/authctx"
	"github.com/nicolastomaschewski/controle-financeiro/internal/category"
	"github.com/nicolastomaschewski/controle-financeiro/internal/httpx"
	"github.com/nicolastomaschewski/controle-financeiro/internal/transaction"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/budgets", h.list)
	mux.HandleFunc("POST /api/v1/budgets", h.create)
	mux.HandleFunc("PATCH /api/v1/budgets/{id}", h.update)
	mux.HandleFunc("DELETE /api/v1/budgets/{id}", h.delete)
}

type budgetResponse struct {
	ID            string `json:"id"`
	CategoryID    string `json:"category_id"`
	Year          int    `json:"year"`
	Month         int    `json:"month"`
	PlannedAmount string `json:"planned_amount"`
}

func toResponse(b *Budget) budgetResponse {
	return budgetResponse{ID: b.ID, CategoryID: b.CategoryID, Year: b.Year, Month: b.Month, PlannedAmount: b.PlannedAmount.StringFixed(2)}
}

type summaryResponse struct {
	CategoryID      string `json:"category_id"`
	CategoryName    string `json:"category_name"`
	PlannedAmount   string `json:"planned_amount"`
	SpentAmount     string `json:"spent_amount"`
	AvailableAmount string `json:"available_amount"`
	PercentUsed     string `json:"percent_used"`
	OverLimit       bool   `json:"over_limit"`
}

func toSummaryResponse(s Summary) summaryResponse {
	return summaryResponse{
		CategoryID: s.CategoryID, CategoryName: s.CategoryName,
		PlannedAmount: s.PlannedAmount.StringFixed(2), SpentAmount: s.SpentAmount.StringFixed(2),
		AvailableAmount: s.AvailableAmount.StringFixed(2), PercentUsed: s.PercentUsed.StringFixed(1),
		OverLimit: s.OverLimit,
	}
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	userID, _ := authctx.UserID(r.Context())
	q := r.URL.Query()

	year, month, err := parseYearMonth(q)
	if err != nil {
		httpx.ValidationError(w, map[string]string{"year_month": "Informe year e month válidos."})
		return
	}

	summaries, err := h.service.SummaryByPeriod(r.Context(), userID, year, month)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "Não foi possível calcular o orçamento.")
		return
	}
	resp := make([]summaryResponse, 0, len(summaries))
	for _, s := range summaries {
		resp = append(resp, toSummaryResponse(s))
	}
	httpx.JSON(w, http.StatusOK, resp)
}

type createRequest struct {
	CategoryID    string `json:"category_id"`
	Year          int    `json:"year"`
	Month         int    `json:"month"`
	PlannedAmount string `json:"planned_amount"`
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	userID, _ := authctx.UserID(r.Context())

	var req createRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid_body", "Corpo da requisição inválido.")
		return
	}
	amount, err := decimal.NewFromString(req.PlannedAmount)
	if err != nil {
		httpx.ValidationError(w, map[string]string{"planned_amount": "Valor inválido."})
		return
	}

	b, err := h.service.Create(r.Context(), userID, Input{
		CategoryID: req.CategoryID, Year: req.Year, Month: req.Month, PlannedAmount: amount,
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, toResponse(b))
}

type updateRequest struct {
	PlannedAmount string `json:"planned_amount"`
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	userID, _ := authctx.UserID(r.Context())

	var req updateRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid_body", "Corpo da requisição inválido.")
		return
	}
	amount, err := decimal.NewFromString(req.PlannedAmount)
	if err != nil {
		httpx.ValidationError(w, map[string]string{"planned_amount": "Valor inválido."})
		return
	}

	b, err := h.service.Update(r.Context(), userID, r.PathValue("id"), amount)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, toResponse(b))
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	userID, _ := authctx.UserID(r.Context())
	if err := h.service.Delete(r.Context(), userID, r.PathValue("id")); err != nil {
		writeDomainError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"message": "Orçamento excluído."})
}

func parseYearMonth(q interface{ Get(string) string }) (int, int, error) {
	now := time.Now().UTC()
	year := now.Year()
	month := int(now.Month())
	if v := q.Get("year"); v != "" {
		y, err := strconv.Atoi(v)
		if err != nil {
			return 0, 0, err
		}
		year = y
	}
	if v := q.Get("month"); v != "" {
		m, err := strconv.Atoi(v)
		if err != nil || m < 1 || m > 12 {
			return 0, 0, errors.New("mês inválido")
		}
		month = m
	}
	return year, month, nil
}

func writeDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		httpx.NotFoundOrForbidden(w)
	case errors.Is(err, ErrInvalidAmount):
		httpx.ValidationError(w, map[string]string{"planned_amount": "Valor deve ser positivo."})
	case errors.Is(err, ErrInvalidPeriod):
		httpx.ValidationError(w, map[string]string{"year_month": "Período inválido."})
	case errors.Is(err, ErrAlreadyExists):
		httpx.Error(w, http.StatusConflict, "budget_exists", "Já existe um orçamento para esta categoria e período.")
	case errors.Is(err, category.ErrNotFound), errors.Is(err, transaction.ErrCategoryNotVisible):
		httpx.ValidationError(w, map[string]string{"category_id": "Categoria não encontrada."})
	case errors.Is(err, transaction.ErrCategoryKindMismatch):
		httpx.ValidationError(w, map[string]string{"category_id": "Orçamentos só podem ser definidos para categorias de despesa."})
	default:
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "Não foi possível concluir a operação.")
	}
}
