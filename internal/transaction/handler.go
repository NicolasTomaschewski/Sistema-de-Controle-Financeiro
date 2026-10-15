package transaction

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
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/transactions", h.list)
	mux.HandleFunc("POST /api/v1/transactions", h.create)
	mux.HandleFunc("GET /api/v1/transactions/{id}", h.get)
	mux.HandleFunc("PATCH /api/v1/transactions/{id}", h.update)
	mux.HandleFunc("DELETE /api/v1/transactions/{id}", h.delete)

	mux.HandleFunc("GET /api/v1/transfers", h.listTransfers)
	mux.HandleFunc("POST /api/v1/transfers", h.createTransfer)

	mux.HandleFunc("GET /api/v1/transactions/monthly-summary", h.monthlySummary)
	mux.HandleFunc("GET /api/v1/transactions/annual-summary", h.annualSummary)
}

type transactionResponse struct {
	ID                      string  `json:"id"`
	AccountID               string  `json:"account_id"`
	CategoryID              *string `json:"category_id"`
	TransferID              *string `json:"transfer_id"`
	InvestmentTransactionID *string `json:"investment_transaction_id"`
	Type                    string  `json:"type"`
	Status                  string  `json:"status"`
	Amount                  string  `json:"amount"`
	Description             string  `json:"description"`
	OccurredOn              string  `json:"occurred_on"`
	Notes                   *string `json:"notes"`
}

func toResponse(t *Transaction) transactionResponse {
	return transactionResponse{
		ID: t.ID, AccountID: t.AccountID, CategoryID: t.CategoryID, TransferID: t.TransferID,
		InvestmentTransactionID: t.InvestmentTransactionID,
		Type:                    t.Type, Status: t.Status, Amount: t.Amount.StringFixed(2),
		Description: t.Description, OccurredOn: t.OccurredOn.Format("2006-01-02"), Notes: t.Notes,
	}
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	userID, _ := authctx.UserID(r.Context())
	q := r.URL.Query()

	f := Filter{
		AccountID:  q.Get("account_id"),
		CategoryID: q.Get("category_id"),
		Type:       q.Get("type"),
		Status:     q.Get("status"),
		Limit:      atoiOr(q.Get("limit"), 50),
		Offset:     atoiOr(q.Get("offset"), 0),
	}
	if v := q.Get("from"); v != "" {
		if d, err := time.Parse("2006-01-02", v); err == nil {
			f.From = &d
		}
	}
	if v := q.Get("to"); v != "" {
		if d, err := time.Parse("2006-01-02", v); err == nil {
			f.To = &d
		}
	}

	items, total, err := h.service.List(r.Context(), userID, f)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "Não foi possível listar os lançamentos.")
		return
	}
	resp := make([]transactionResponse, 0, len(items))
	for _, t := range items {
		resp = append(resp, toResponse(t))
	}
	httpx.JSONWithMeta(w, http.StatusOK, resp, map[string]int{"total": total, "limit": f.Limit, "offset": f.Offset})
}

type createRequest struct {
	AccountID   string  `json:"account_id"`
	CategoryID  *string `json:"category_id"`
	Type        string  `json:"type"`
	Status      string  `json:"status"`
	Amount      string  `json:"amount"`
	Description string  `json:"description"`
	OccurredOn  string  `json:"occurred_on"`
	Notes       *string `json:"notes"`
}

func (req createRequest) toInput() (Input, error) {
	amount, err := decimal.NewFromString(req.Amount)
	if err != nil {
		return Input{}, errors.New("valor inválido")
	}
	occurredOn, err := time.Parse("2006-01-02", req.OccurredOn)
	if err != nil {
		return Input{}, errors.New("data inválida")
	}
	status := req.Status
	if status == "" {
		status = StatusPaid
	}
	return Input{
		AccountID: req.AccountID, CategoryID: req.CategoryID, Type: req.Type, Status: status,
		Amount: amount, Description: req.Description, OccurredOn: occurredOn, Notes: req.Notes,
	}, nil
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	userID, _ := authctx.UserID(r.Context())

	var req createRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid_body", "Corpo da requisição inválido.")
		return
	}
	in, err := req.toInput()
	if err != nil {
		httpx.ValidationError(w, map[string]string{"amount_or_date": err.Error()})
		return
	}

	t, err := h.service.Create(r.Context(), userID, in)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, toResponse(t))
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	userID, _ := authctx.UserID(r.Context())
	t, err := h.service.Get(r.Context(), userID, r.PathValue("id"))
	if err != nil {
		writeDomainError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, toResponse(t))
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	userID, _ := authctx.UserID(r.Context())

	var req createRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid_body", "Corpo da requisição inválido.")
		return
	}
	in, err := req.toInput()
	if err != nil {
		httpx.ValidationError(w, map[string]string{"amount_or_date": err.Error()})
		return
	}

	t, err := h.service.Update(r.Context(), userID, r.PathValue("id"), in)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, toResponse(t))
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	userID, _ := authctx.UserID(r.Context())
	if err := h.service.Delete(r.Context(), userID, r.PathValue("id")); err != nil {
		writeDomainError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"message": "Lançamento excluído."})
}

type transferResponse struct {
	ID            string `json:"id"`
	FromAccountID string `json:"from_account_id"`
	ToAccountID   string `json:"to_account_id"`
	Amount        string `json:"amount"`
	Description   string `json:"description"`
	OccurredOn    string `json:"occurred_on"`
}

func toTransferResponse(t *Transfer) transferResponse {
	return transferResponse{
		ID: t.ID, FromAccountID: t.FromAccountID, ToAccountID: t.ToAccountID,
		Amount: t.Amount.StringFixed(2), Description: t.Description, OccurredOn: t.OccurredOn.Format("2006-01-02"),
	}
}

type createTransferRequest struct {
	FromAccountID string `json:"from_account_id"`
	ToAccountID   string `json:"to_account_id"`
	Amount        string `json:"amount"`
	Description   string `json:"description"`
	OccurredOn    string `json:"occurred_on"`
}

func (h *Handler) createTransfer(w http.ResponseWriter, r *http.Request) {
	userID, _ := authctx.UserID(r.Context())

	var req createTransferRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid_body", "Corpo da requisição inválido.")
		return
	}
	amount, err := decimal.NewFromString(req.Amount)
	if err != nil {
		httpx.ValidationError(w, map[string]string{"amount": "Valor inválido."})
		return
	}
	occurredOn, err := time.Parse("2006-01-02", req.OccurredOn)
	if err != nil {
		httpx.ValidationError(w, map[string]string{"occurred_on": "Data inválida (use AAAA-MM-DD)."})
		return
	}

	transfer, err := h.service.CreateTransfer(r.Context(), userID, TransferInput{
		FromAccountID: req.FromAccountID, ToAccountID: req.ToAccountID,
		Amount: amount, OccurredOn: occurredOn, Description: req.Description,
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, toTransferResponse(transfer))
}

func (h *Handler) listTransfers(w http.ResponseWriter, r *http.Request) {
	userID, _ := authctx.UserID(r.Context())
	q := r.URL.Query()

	items, err := h.service.ListTransfers(r.Context(), userID, atoiOr(q.Get("limit"), 50), atoiOr(q.Get("offset"), 0))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "Não foi possível listar as transferências.")
		return
	}
	resp := make([]transferResponse, 0, len(items))
	for _, t := range items {
		resp = append(resp, toTransferResponse(t))
	}
	httpx.JSON(w, http.StatusOK, resp)
}

type monthlySummaryResponse struct {
	Year               int                        `json:"year"`
	Month              int                        `json:"month"`
	TotalIncome        string                     `json:"total_income"`
	TotalExpense       string                     `json:"total_expense"`
	Result             string                     `json:"result"`
	ExpensesByCategory []categoryBreakdownItemDTO `json:"expenses_by_category"`
	IncomeByCategory   []categoryBreakdownItemDTO `json:"income_by_category"`
	PreviousMonth      *monthlySummaryTotalsDTO   `json:"previous_month,omitempty"`
}

type annualSummaryResponse struct {
	Year               int                        `json:"year"`
	TotalIncome        string                     `json:"total_income"`
	TotalExpense       string                     `json:"total_expense"`
	Result             string                     `json:"result"`
	ExpensesByCategory []categoryBreakdownItemDTO `json:"expenses_by_category"`
	IncomeByCategory   []categoryBreakdownItemDTO `json:"income_by_category"`
}

func toBreakdownDTO(items []CategoryBreakdownItem) []categoryBreakdownItemDTO {
	out := make([]categoryBreakdownItemDTO, 0, len(items))
	for _, item := range items {
		out = append(out, categoryBreakdownItemDTO{CategoryID: item.CategoryID, CategoryName: item.CategoryName, Total: item.Total.StringFixed(2)})
	}
	return out
}

func (h *Handler) annualSummary(w http.ResponseWriter, r *http.Request) {
	userID, _ := authctx.UserID(r.Context())
	year := atoiOr(r.URL.Query().Get("year"), time.Now().UTC().Year())
	if year < 2000 || year > 2100 {
		httpx.ValidationError(w, map[string]string{"year": "Ano inválido (use 2000 a 2100)."})
		return
	}

	report, err := h.service.AnnualReport(r.Context(), userID, year)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "Não foi possível calcular o resumo anual.")
		return
	}
	httpx.JSON(w, http.StatusOK, annualSummaryResponse{
		Year:               year,
		TotalIncome:        report.Summary.TotalIncome.StringFixed(2),
		TotalExpense:       report.Summary.TotalExpense.StringFixed(2),
		Result:             report.Summary.Result.StringFixed(2),
		ExpensesByCategory: toBreakdownDTO(report.ExpensesByCategory),
		IncomeByCategory:   toBreakdownDTO(report.IncomeByCategory),
	})
}

type categoryBreakdownItemDTO struct {
	CategoryID   *string `json:"category_id"`
	CategoryName string  `json:"category_name"`
	Total        string  `json:"total"`
}

type monthlySummaryTotalsDTO struct {
	TotalIncome  string `json:"total_income"`
	TotalExpense string `json:"total_expense"`
	Result       string `json:"result"`
}

func (h *Handler) monthlySummary(w http.ResponseWriter, r *http.Request) {
	userID, _ := authctx.UserID(r.Context())
	q := r.URL.Query()

	now := time.Now().UTC()
	year := atoiOr(q.Get("year"), now.Year())
	month := atoiOr(q.Get("month"), int(now.Month()))
	if month < 1 || month > 12 {
		httpx.ValidationError(w, map[string]string{"month": "Mês inválido (use 1 a 12)."})
		return
	}
	compare := q.Get("compare") == "true"

	report, err := h.service.MonthlyReport(r.Context(), userID, year, time.Month(month), compare)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "Não foi possível calcular o resumo mensal.")
		return
	}

	resp := monthlySummaryResponse{
		Year: year, Month: month,
		TotalIncome:  report.Summary.TotalIncome.StringFixed(2),
		TotalExpense: report.Summary.TotalExpense.StringFixed(2),
		Result:       report.Summary.Result.StringFixed(2),
	}
	resp.ExpensesByCategory = toBreakdownDTO(report.ExpensesByCategory)
	resp.IncomeByCategory = toBreakdownDTO(report.IncomeByCategory)
	if report.PreviousMonth != nil {
		resp.PreviousMonth = &monthlySummaryTotalsDTO{
			TotalIncome:  report.PreviousMonth.TotalIncome.StringFixed(2),
			TotalExpense: report.PreviousMonth.TotalExpense.StringFixed(2),
			Result:       report.PreviousMonth.Result.StringFixed(2),
		}
	}

	httpx.JSON(w, http.StatusOK, resp)
}

func writeDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		httpx.NotFoundOrForbidden(w)
	case errors.Is(err, account.ErrNotFound), errors.Is(err, ErrAccountNotOwned):
		httpx.ValidationError(w, map[string]string{"account_id": "Conta não encontrada ou não pertence ao usuário."})
	case errors.Is(err, category.ErrNotFound), errors.Is(err, ErrCategoryNotVisible):
		httpx.ValidationError(w, map[string]string{"category_id": "Categoria não encontrada."})
	case errors.Is(err, ErrCategoryKindMismatch):
		httpx.ValidationError(w, map[string]string{"category_id": "Categoria incompatível com o tipo do lançamento."})
	case errors.Is(err, ErrInvalidAmount):
		httpx.ValidationError(w, map[string]string{"amount": "Valor deve ser positivo."})
	case errors.Is(err, ErrInvalidDescription):
		httpx.ValidationError(w, map[string]string{"description": "Descrição inválida."})
	case errors.Is(err, ErrInvalidType):
		httpx.ValidationError(w, map[string]string{"type": "Tipo inválido (use income ou expense)."})
	case errors.Is(err, ErrInvalidStatus):
		httpx.ValidationError(w, map[string]string{"status": "Status inválido."})
	case errors.Is(err, ErrIsTransferLeg):
		httpx.Error(w, http.StatusForbidden, "transfer_leg", "Movimentações de transferência não podem ser editadas individualmente.")
	case errors.Is(err, ErrTransferSameAccount):
		httpx.ValidationError(w, map[string]string{"to_account_id": "A conta de destino deve ser diferente da origem."})
	default:
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "Não foi possível concluir a operação.")
	}
}

func atoiOr(s string, fallback int) int {
	if s == "" {
		return fallback
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return fallback
	}
	return n
}
