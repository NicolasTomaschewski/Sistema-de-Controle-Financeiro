package recurring

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/shopspring/decimal"

	"github.com/nicolastomaschewski/controle-financeiro/internal/account"
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
	mux.HandleFunc("GET /api/v1/recurring-rules", h.list)
	mux.HandleFunc("GET /api/v1/recurring-rules/summary", h.summary)
	mux.HandleFunc("POST /api/v1/recurring-rules", h.create)
	mux.HandleFunc("GET /api/v1/recurring-rules/{id}", h.get)
	mux.HandleFunc("PATCH /api/v1/recurring-rules/{id}", h.update)
	mux.HandleFunc("DELETE /api/v1/recurring-rules/{id}", h.delete)
	mux.HandleFunc("POST /api/v1/recurring-rules/{id}/generate", h.generate)
}

type ruleResponse struct {
	ID            string  `json:"id"`
	AccountID     *string `json:"account_id"`
	CategoryID    *string `json:"category_id"`
	Description   string  `json:"description"`
	PlannedAmount string  `json:"planned_amount"`
	Periodicity   string  `json:"periodicity"`
	StartDate     string  `json:"start_date"`
	EndDate       *string `json:"end_date"`
	DueDay        *int    `json:"due_day"`
	DueMonth      *int    `json:"due_month"`
	Active        bool    `json:"active"`
	// Derivados da regra: custo de um ano completo e sua distribuição mensal.
	AnnualCost        string `json:"annual_cost"`
	MonthlyEquivalent string `json:"monthly_equivalent"`
}

func toResponse(r *Rule) ruleResponse {
	resp := ruleResponse{
		ID: r.ID, AccountID: r.AccountID, CategoryID: r.CategoryID, Description: r.Description,
		PlannedAmount: r.PlannedAmount.StringFixed(2), Periodicity: r.Periodicity,
		StartDate: r.StartDate.Format("2006-01-02"), DueDay: r.DueDay, DueMonth: r.DueMonth, Active: r.Active,
		AnnualCost: r.AnnualCost().StringFixed(2), MonthlyEquivalent: r.MonthlyEquivalent().StringFixed(2),
	}
	if r.EndDate != nil {
		s := r.EndDate.Format("2006-01-02")
		resp.EndDate = &s
	}
	return resp
}

type ruleRequest struct {
	AccountID     *string `json:"account_id"`
	CategoryID    *string `json:"category_id"`
	Description   string  `json:"description"`
	PlannedAmount string  `json:"planned_amount"`
	Periodicity   string  `json:"periodicity"`
	StartDate     string  `json:"start_date"`
	EndDate       *string `json:"end_date"`
	DueDay        *int    `json:"due_day"`
	DueMonth      *int    `json:"due_month"`
	Active        *bool   `json:"active"`
}

func (req ruleRequest) toInput() (Input, error) {
	amount, err := decimal.NewFromString(req.PlannedAmount)
	if err != nil {
		return Input{}, errors.New("valor inválido")
	}
	start, err := time.Parse("2006-01-02", req.StartDate)
	if err != nil {
		return Input{}, errors.New("data inicial inválida")
	}
	var end *time.Time
	if req.EndDate != nil && *req.EndDate != "" {
		d, err := time.Parse("2006-01-02", *req.EndDate)
		if err != nil {
			return Input{}, errors.New("data final inválida")
		}
		end = &d
	}
	active := true
	if req.Active != nil {
		active = *req.Active
	}
	periodicity := req.Periodicity
	if periodicity == "" {
		periodicity = PeriodicityMonthly
	}
	return Input{
		AccountID: req.AccountID, CategoryID: req.CategoryID, Description: req.Description,
		PlannedAmount: amount, Periodicity: periodicity, StartDate: start, EndDate: end,
		DueDay: req.DueDay, DueMonth: req.DueMonth, Active: active,
	}, nil
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	userID, _ := authctx.UserID(r.Context())
	activeOnly := r.URL.Query().Get("active_only") == "true"

	rules, err := h.service.List(r.Context(), userID, activeOnly)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "Não foi possível listar as regras de recorrência.")
		return
	}
	resp := make([]ruleResponse, 0, len(rules))
	for _, rule := range rules {
		resp = append(resp, toResponse(rule))
	}
	httpx.JSON(w, http.StatusOK, resp)
}

type categoryCostDTO struct {
	CategoryID        *string `json:"category_id"`
	AnnualTotal       string  `json:"annual_total"`
	MonthlyEquivalent string  `json:"monthly_equivalent"`
}

type summaryResponse struct {
	MonthlyRulesTotal string            `json:"monthly_rules_total"`
	YearlyRulesTotal  string            `json:"yearly_rules_total"`
	AnnualTotal       string            `json:"annual_total"`
	MonthlyEquivalent string            `json:"monthly_equivalent"`
	ByCategory        []categoryCostDTO `json:"by_category"`
}

func (h *Handler) summary(w http.ResponseWriter, r *http.Request) {
	userID, _ := authctx.UserID(r.Context())

	sum, err := h.service.Summary(r.Context(), userID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "Não foi possível calcular o resumo dos custos fixos.")
		return
	}
	resp := summaryResponse{
		MonthlyRulesTotal: sum.MonthlyRulesTotal.StringFixed(2),
		YearlyRulesTotal:  sum.YearlyRulesTotal.StringFixed(2),
		AnnualTotal:       sum.AnnualTotal.StringFixed(2),
		MonthlyEquivalent: sum.MonthlyEquivalent.StringFixed(2),
		ByCategory:        make([]categoryCostDTO, 0, len(sum.ByCategory)),
	}
	for _, c := range sum.ByCategory {
		resp.ByCategory = append(resp.ByCategory, categoryCostDTO{
			CategoryID: c.CategoryID, AnnualTotal: c.AnnualTotal.StringFixed(2), MonthlyEquivalent: c.MonthlyEquivalent.StringFixed(2),
		})
	}
	httpx.JSON(w, http.StatusOK, resp)
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	userID, _ := authctx.UserID(r.Context())

	var req ruleRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid_body", "Corpo da requisição inválido.")
		return
	}
	in, err := req.toInput()
	if err != nil {
		httpx.ValidationError(w, map[string]string{"input": err.Error()})
		return
	}

	rule, err := h.service.Create(r.Context(), userID, in)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, toResponse(rule))
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	userID, _ := authctx.UserID(r.Context())
	rule, err := h.service.Get(r.Context(), userID, r.PathValue("id"))
	if err != nil {
		writeDomainError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, toResponse(rule))
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	userID, _ := authctx.UserID(r.Context())

	var req ruleRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid_body", "Corpo da requisição inválido.")
		return
	}
	in, err := req.toInput()
	if err != nil {
		httpx.ValidationError(w, map[string]string{"input": err.Error()})
		return
	}

	rule, err := h.service.Update(r.Context(), userID, r.PathValue("id"), in)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, toResponse(rule))
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	userID, _ := authctx.UserID(r.Context())
	if err := h.service.Delete(r.Context(), userID, r.PathValue("id")); err != nil {
		writeDomainError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"message": "Regra de recorrência excluída."})
}

func (h *Handler) generate(w http.ResponseWriter, r *http.Request) {
	userID, _ := authctx.UserID(r.Context())

	year, err1 := strconv.Atoi(r.URL.Query().Get("year"))
	month, err2 := strconv.Atoi(r.URL.Query().Get("month"))
	if err1 != nil || err2 != nil || month < 1 || month > 12 {
		httpx.ValidationError(w, map[string]string{"year_month": "Informe year e month válidos como parâmetros de consulta."})
		return
	}

	t, created, err := h.service.GenerateOccurrence(r.Context(), userID, r.PathValue("id"), year, time.Month(month))
	if err != nil {
		writeDomainError(w, err)
		return
	}

	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}
	httpx.JSON(w, status, map[string]any{
		"transaction_id": t.ID,
		"status":         t.Status,
		"occurred_on":    t.OccurredOn.Format("2006-01-02"),
		"created":        created,
	})
}

func writeDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		httpx.NotFoundOrForbidden(w)
	case errors.Is(err, ErrInvalidDesc):
		httpx.ValidationError(w, map[string]string{"description": "Descrição inválida."})
	case errors.Is(err, ErrInvalidAmount):
		httpx.ValidationError(w, map[string]string{"planned_amount": "Valor deve ser positivo."})
	case errors.Is(err, ErrInvalidPeriod):
		httpx.ValidationError(w, map[string]string{"periodicity": "Periodicidade inválida."})
	case errors.Is(err, ErrInvalidDueDay):
		httpx.ValidationError(w, map[string]string{"due_day": "Dia de vencimento deve estar entre 1 e 31."})
	case errors.Is(err, ErrInvalidDueMonth):
		httpx.ValidationError(w, map[string]string{"due_month": "Informe o mês de vencimento (1 a 12) apenas para custos anuais."})
	case errors.Is(err, ErrNotDueInMonth):
		httpx.ValidationError(w, map[string]string{"year_month": "Este custo anual não vence no mês selecionado."})
	case errors.Is(err, account.ErrNotFound), errors.Is(err, transaction.ErrAccountNotOwned):
		httpx.ValidationError(w, map[string]string{"account_id": "Conta não encontrada ou não pertence ao usuário."})
	case errors.Is(err, category.ErrNotFound), errors.Is(err, transaction.ErrCategoryNotVisible):
		httpx.ValidationError(w, map[string]string{"category_id": "Categoria não encontrada."})
	case errors.Is(err, transaction.ErrCategoryKindMismatch):
		httpx.ValidationError(w, map[string]string{"category_id": "Categorias de despesas recorrentes devem ser do tipo despesa."})
	case errors.Is(err, ErrRuleInactive):
		httpx.Error(w, http.StatusConflict, "rule_inactive", "A regra está inativa e não pode gerar novas ocorrências.")
	case errors.Is(err, ErrOutsideValidity):
		httpx.ValidationError(w, map[string]string{"year_month": "Período fora da vigência da regra."})
	case errors.Is(err, ErrUnsupportedPeriodicityForGeneration):
		httpx.Error(w, http.StatusNotImplemented, "unsupported_periodicity", err.Error())
	default:
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "Não foi possível concluir a operação.")
	}
}
