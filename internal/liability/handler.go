package liability

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
	mux.HandleFunc("GET /api/v1/liabilities", h.list)
	mux.HandleFunc("POST /api/v1/liabilities", h.create)
	mux.HandleFunc("GET /api/v1/liabilities/{id}", h.get)
	mux.HandleFunc("PATCH /api/v1/liabilities/{id}", h.update)
	mux.HandleFunc("DELETE /api/v1/liabilities/{id}", h.delete)
	mux.HandleFunc("GET /api/v1/liabilities/{id}/valuations", h.listValuations)
	mux.HandleFunc("POST /api/v1/liabilities/{id}/valuations", h.addValuation)
}

type liabilityResponse struct {
	ID              string  `json:"id"`
	Name            string  `json:"name"`
	Type            string  `json:"type"`
	PrincipalAmount string  `json:"principal_amount"`
	CurrentBalance  string  `json:"current_balance"`
	ReferenceDate   string  `json:"reference_date"`
	Notes           *string `json:"notes"`
	Status          string  `json:"status"`
}

func toResponse(l *Liability) liabilityResponse {
	return liabilityResponse{
		ID: l.ID, Name: l.Name, Type: l.Type, PrincipalAmount: l.PrincipalAmount.StringFixed(2),
		CurrentBalance: l.CurrentBalance.StringFixed(2), ReferenceDate: l.ReferenceDate.Format("2006-01-02"),
		Notes: l.Notes, Status: l.Status,
	}
}

type valuationResponse struct {
	ID            string  `json:"id"`
	Balance       string  `json:"balance"`
	ReferenceDate string  `json:"reference_date"`
	Notes         *string `json:"notes"`
}

func toValuationResponse(v *Valuation) valuationResponse {
	return valuationResponse{ID: v.ID, Balance: v.Balance.StringFixed(2), ReferenceDate: v.ReferenceDate.Format("2006-01-02"), Notes: v.Notes}
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	userID, _ := authctx.UserID(r.Context())
	includeSettled := r.URL.Query().Get("include_settled") == "true"

	items, err := h.service.List(r.Context(), userID, includeSettled)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "Não foi possível listar os passivos.")
		return
	}
	resp := make([]liabilityResponse, 0, len(items))
	for _, l := range items {
		resp = append(resp, toResponse(l))
	}
	httpx.JSON(w, http.StatusOK, resp)
}

type createRequest struct {
	Name            string  `json:"name"`
	Type            string  `json:"type"`
	PrincipalAmount string  `json:"principal_amount"`
	CurrentBalance  string  `json:"current_balance"`
	ReferenceDate   string  `json:"reference_date"`
	Notes           *string `json:"notes"`
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	userID, _ := authctx.UserID(r.Context())

	var req createRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid_body", "Corpo da requisição inválido.")
		return
	}
	principal, err := decimal.NewFromString(req.PrincipalAmount)
	if err != nil {
		httpx.ValidationError(w, map[string]string{"principal_amount": "Valor inválido."})
		return
	}
	balance, err := decimal.NewFromString(req.CurrentBalance)
	if err != nil {
		httpx.ValidationError(w, map[string]string{"current_balance": "Valor inválido."})
		return
	}
	date, err := time.Parse("2006-01-02", req.ReferenceDate)
	if err != nil {
		httpx.ValidationError(w, map[string]string{"reference_date": "Data inválida (use AAAA-MM-DD)."})
		return
	}

	l, err := h.service.Create(r.Context(), userID, CreateInput{
		Name: req.Name, Type: req.Type, PrincipalAmount: principal, CurrentBalance: balance,
		ReferenceDate: date, Notes: req.Notes,
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, toResponse(l))
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	userID, _ := authctx.UserID(r.Context())
	l, err := h.service.Get(r.Context(), userID, r.PathValue("id"))
	if err != nil {
		writeDomainError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, toResponse(l))
}

type updateRequest struct {
	Name   string  `json:"name"`
	Type   string  `json:"type"`
	Notes  *string `json:"notes"`
	Status string  `json:"status"`
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	userID, _ := authctx.UserID(r.Context())

	var req updateRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid_body", "Corpo da requisição inválido.")
		return
	}

	l, err := h.service.Update(r.Context(), userID, r.PathValue("id"), UpdateInput{
		Name: req.Name, Type: req.Type, Notes: req.Notes, Status: req.Status,
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, toResponse(l))
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	userID, _ := authctx.UserID(r.Context())
	if err := h.service.Delete(r.Context(), userID, r.PathValue("id")); err != nil {
		writeDomainError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"message": "Passivo excluído."})
}

func (h *Handler) listValuations(w http.ResponseWriter, r *http.Request) {
	userID, _ := authctx.UserID(r.Context())
	items, err := h.service.ListValuations(r.Context(), userID, r.PathValue("id"))
	if err != nil {
		writeDomainError(w, err)
		return
	}
	resp := make([]valuationResponse, 0, len(items))
	for _, v := range items {
		resp = append(resp, toValuationResponse(v))
	}
	httpx.JSON(w, http.StatusOK, resp)
}

type addValuationRequest struct {
	Balance       string  `json:"balance"`
	ReferenceDate string  `json:"reference_date"`
	Notes         *string `json:"notes"`
}

func (h *Handler) addValuation(w http.ResponseWriter, r *http.Request) {
	userID, _ := authctx.UserID(r.Context())

	var req addValuationRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid_body", "Corpo da requisição inválido.")
		return
	}
	balance, err := decimal.NewFromString(req.Balance)
	if err != nil {
		httpx.ValidationError(w, map[string]string{"balance": "Valor inválido."})
		return
	}
	date, err := time.Parse("2006-01-02", req.ReferenceDate)
	if err != nil {
		httpx.ValidationError(w, map[string]string{"reference_date": "Data inválida (use AAAA-MM-DD)."})
		return
	}

	v, err := h.service.AddValuation(r.Context(), userID, r.PathValue("id"), balance, date, req.Notes)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, toValuationResponse(v))
}

func writeDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		httpx.NotFoundOrForbidden(w)
	case errors.Is(err, ErrInvalidName):
		httpx.ValidationError(w, map[string]string{"name": "Nome inválido."})
	case errors.Is(err, ErrInvalidType):
		httpx.ValidationError(w, map[string]string{"type": "Tipo inválido."})
	case errors.Is(err, ErrInvalidAmount):
		httpx.ValidationError(w, map[string]string{"amount": "Valor não pode ser negativo."})
	case errors.Is(err, ErrValuationExists):
		httpx.Error(w, http.StatusConflict, "valuation_exists", "Já existe um registro de saldo para esta data.")
	default:
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "Não foi possível concluir a operação.")
	}
}
