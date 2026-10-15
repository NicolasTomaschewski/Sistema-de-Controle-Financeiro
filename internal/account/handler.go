package account

import (
	"errors"
	"net/http"
	"time"

	"github.com/shopspring/decimal"

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
	mux.HandleFunc("GET /api/v1/accounts", h.list)
	mux.HandleFunc("POST /api/v1/accounts", h.create)
	mux.HandleFunc("GET /api/v1/accounts/{id}", h.get)
	mux.HandleFunc("PATCH /api/v1/accounts/{id}", h.update)
	mux.HandleFunc("DELETE /api/v1/accounts/{id}", h.delete)
}

type accountResponse struct {
	ID                 string  `json:"id"`
	Name               string  `json:"name"`
	Type               string  `json:"type"`
	Institution        *string `json:"institution"`
	Currency           string  `json:"currency"`
	InitialBalance     string  `json:"initial_balance"`
	InitialBalanceDate string  `json:"initial_balance_date"`
	Status             string  `json:"status"`
	CurrentBalance     string  `json:"current_balance"`
}

func toResponse(a *Account, balance decimal.Decimal) accountResponse {
	return accountResponse{
		ID:                 a.ID,
		Name:               a.Name,
		Type:               a.Type,
		Institution:        a.Institution,
		Currency:           a.Currency,
		InitialBalance:     a.InitialBalance.StringFixed(2),
		InitialBalanceDate: a.InitialBalanceDate.Format("2006-01-02"),
		Status:             a.Status,
		CurrentBalance:     balance.StringFixed(2),
	}
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	userID, _ := authctx.UserID(r.Context())
	includeArchived := r.URL.Query().Get("include_archived") == "true"

	items, err := h.service.List(r.Context(), userID, includeArchived)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "Não foi possível listar as contas.")
		return
	}

	resp := make([]accountResponse, 0, len(items))
	for _, it := range items {
		resp = append(resp, toResponse(it.Account, it.Balance))
	}
	httpx.JSON(w, http.StatusOK, resp)
}

type createRequest struct {
	Name               string  `json:"name"`
	Type               string  `json:"type"`
	Institution        *string `json:"institution"`
	Currency           string  `json:"currency"`
	InitialBalance     string  `json:"initial_balance"`
	InitialBalanceDate string  `json:"initial_balance_date"`
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	userID, _ := authctx.UserID(r.Context())

	var req createRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid_body", "Corpo da requisição inválido.")
		return
	}

	balance, err := parseDecimal(req.InitialBalance)
	if err != nil {
		httpx.ValidationError(w, map[string]string{"initial_balance": "Valor inválido."})
		return
	}
	date, err := parseDate(req.InitialBalanceDate)
	if err != nil {
		httpx.ValidationError(w, map[string]string{"initial_balance_date": "Data inválida (use AAAA-MM-DD)."})
		return
	}

	a, err := h.service.Create(r.Context(), userID, CreateInput{
		Name: req.Name, Type: req.Type, Institution: req.Institution,
		Currency: req.Currency, InitialBalance: balance, InitialBalanceDate: date,
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}

	httpx.JSON(w, http.StatusCreated, toResponse(a, a.InitialBalance))
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	userID, _ := authctx.UserID(r.Context())
	id := r.PathValue("id")

	a, balance, err := h.service.Get(r.Context(), userID, id)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, toResponse(a, balance))
}

type updateRequest struct {
	Name        string  `json:"name"`
	Type        string  `json:"type"`
	Institution *string `json:"institution"`
	Status      string  `json:"status"`
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	userID, _ := authctx.UserID(r.Context())
	id := r.PathValue("id")

	var req updateRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid_body", "Corpo da requisição inválido.")
		return
	}

	a, err := h.service.Update(r.Context(), userID, id, UpdateInput{
		Name: req.Name, Type: req.Type, Institution: req.Institution, Status: req.Status,
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}

	balance, err := h.service.Balance(r.Context(), userID, id)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, toResponse(a, balance))
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	userID, _ := authctx.UserID(r.Context())
	id := r.PathValue("id")

	archived, err := h.service.Delete(r.Context(), userID, id)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	if archived {
		httpx.JSON(w, http.StatusOK, map[string]string{"message": "Conta possui histórico e foi arquivada em vez de excluída."})
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"message": "Conta excluída."})
}

func writeDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		httpx.NotFoundOrForbidden(w)
	case errors.Is(err, ErrInvalidName):
		httpx.ValidationError(w, map[string]string{"name": "Nome inválido."})
	case errors.Is(err, ErrInvalidType):
		httpx.ValidationError(w, map[string]string{"type": "Tipo inválido."})
	case errors.Is(err, ErrInvalidCurrency):
		httpx.ValidationError(w, map[string]string{"currency": "Moeda inválida."})
	default:
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "Não foi possível concluir a operação.")
	}
}

func parseDecimal(s string) (decimal.Decimal, error) {
	if s == "" {
		return decimal.Zero, nil
	}
	return decimal.NewFromString(s)
}

func parseDate(s string) (time.Time, error) {
	if s == "" {
		return time.Now().UTC().Truncate(24 * time.Hour), nil
	}
	return time.Parse("2006-01-02", s)
}
