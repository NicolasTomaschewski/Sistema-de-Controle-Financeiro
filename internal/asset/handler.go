package asset

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
	mux.HandleFunc("GET /api/v1/assets", h.list)
	mux.HandleFunc("POST /api/v1/assets", h.create)
	mux.HandleFunc("GET /api/v1/assets/{id}", h.get)
	mux.HandleFunc("PATCH /api/v1/assets/{id}", h.update)
	mux.HandleFunc("DELETE /api/v1/assets/{id}", h.delete)
	mux.HandleFunc("GET /api/v1/assets/{id}/valuations", h.listValuations)
	mux.HandleFunc("POST /api/v1/assets/{id}/valuations", h.addValuation)
}

type assetResponse struct {
	ID            string  `json:"id"`
	Name          string  `json:"name"`
	Type          string  `json:"type"`
	CurrentValue  string  `json:"current_value"`
	ValuationDate string  `json:"valuation_date"`
	Notes         *string `json:"notes"`
	Status        string  `json:"status"`
	IsEstimate    bool    `json:"is_estimate"`
}

func toResponse(a *Asset) assetResponse {
	return assetResponse{
		ID: a.ID, Name: a.Name, Type: a.Type, CurrentValue: a.CurrentValue.StringFixed(2),
		ValuationDate: a.ValuationDate.Format("2006-01-02"), Notes: a.Notes, Status: a.Status,
		// Sem integração externa de mercado configurada, todo valor é uma
		// estimativa informada manualmente pelo usuário.
		IsEstimate: true,
	}
}

type valuationResponse struct {
	ID            string  `json:"id"`
	Value         string  `json:"value"`
	ValuationDate string  `json:"valuation_date"`
	Notes         *string `json:"notes"`
}

func toValuationResponse(v *Valuation) valuationResponse {
	return valuationResponse{ID: v.ID, Value: v.Value.StringFixed(2), ValuationDate: v.ValuationDate.Format("2006-01-02"), Notes: v.Notes}
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	userID, _ := authctx.UserID(r.Context())
	includeArchived := r.URL.Query().Get("include_archived") == "true"

	items, err := h.service.List(r.Context(), userID, includeArchived)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "Não foi possível listar os ativos.")
		return
	}
	resp := make([]assetResponse, 0, len(items))
	for _, a := range items {
		resp = append(resp, toResponse(a))
	}
	httpx.JSON(w, http.StatusOK, resp)
}

type createRequest struct {
	Name          string  `json:"name"`
	Type          string  `json:"type"`
	CurrentValue  string  `json:"current_value"`
	ValuationDate string  `json:"valuation_date"`
	Notes         *string `json:"notes"`
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	userID, _ := authctx.UserID(r.Context())

	var req createRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid_body", "Corpo da requisição inválido.")
		return
	}
	value, err := decimal.NewFromString(req.CurrentValue)
	if err != nil {
		httpx.ValidationError(w, map[string]string{"current_value": "Valor inválido."})
		return
	}
	date, err := time.Parse("2006-01-02", req.ValuationDate)
	if err != nil {
		httpx.ValidationError(w, map[string]string{"valuation_date": "Data inválida (use AAAA-MM-DD)."})
		return
	}

	a, err := h.service.Create(r.Context(), userID, CreateInput{
		Name: req.Name, Type: req.Type, CurrentValue: value, ValuationDate: date, Notes: req.Notes,
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, toResponse(a))
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	userID, _ := authctx.UserID(r.Context())
	a, err := h.service.Get(r.Context(), userID, r.PathValue("id"))
	if err != nil {
		writeDomainError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, toResponse(a))
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

	a, err := h.service.Update(r.Context(), userID, r.PathValue("id"), UpdateInput{
		Name: req.Name, Type: req.Type, Notes: req.Notes, Status: req.Status,
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, toResponse(a))
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	userID, _ := authctx.UserID(r.Context())
	if err := h.service.Delete(r.Context(), userID, r.PathValue("id")); err != nil {
		writeDomainError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"message": "Ativo excluído."})
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
	Value         string  `json:"value"`
	ValuationDate string  `json:"valuation_date"`
	Notes         *string `json:"notes"`
}

func (h *Handler) addValuation(w http.ResponseWriter, r *http.Request) {
	userID, _ := authctx.UserID(r.Context())

	var req addValuationRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid_body", "Corpo da requisição inválido.")
		return
	}
	value, err := decimal.NewFromString(req.Value)
	if err != nil {
		httpx.ValidationError(w, map[string]string{"value": "Valor inválido."})
		return
	}
	date, err := time.Parse("2006-01-02", req.ValuationDate)
	if err != nil {
		httpx.ValidationError(w, map[string]string{"valuation_date": "Data inválida (use AAAA-MM-DD)."})
		return
	}

	v, err := h.service.AddValuation(r.Context(), userID, r.PathValue("id"), value, date, req.Notes)
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
	case errors.Is(err, ErrInvalidValue):
		httpx.ValidationError(w, map[string]string{"current_value": "Valor não pode ser negativo."})
	case errors.Is(err, ErrValuationExists):
		httpx.Error(w, http.StatusConflict, "valuation_exists", "Já existe uma avaliação registrada para esta data.")
	default:
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "Não foi possível concluir a operação.")
	}
}
