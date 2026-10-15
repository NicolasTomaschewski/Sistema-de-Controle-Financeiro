package category

import (
	"errors"
	"net/http"

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
	mux.HandleFunc("GET /api/v1/categories", h.list)
	mux.HandleFunc("POST /api/v1/categories", h.create)
	mux.HandleFunc("PATCH /api/v1/categories/{id}", h.update)
	mux.HandleFunc("DELETE /api/v1/categories/{id}", h.delete)
}

type categoryResponse struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Kind     string `json:"kind"`
	IsSystem bool   `json:"is_system"`
	Archived bool   `json:"archived"`
}

func toResponse(c *Category) categoryResponse {
	return categoryResponse{ID: c.ID, Name: c.Name, Kind: c.Kind, IsSystem: c.IsSystem(), Archived: c.ArchivedAt != nil}
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	userID, _ := authctx.UserID(r.Context())
	includeArchived := r.URL.Query().Get("include_archived") == "true"

	items, err := h.service.List(r.Context(), userID, includeArchived)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "Não foi possível listar as categorias.")
		return
	}
	resp := make([]categoryResponse, 0, len(items))
	for _, c := range items {
		resp = append(resp, toResponse(c))
	}
	httpx.JSON(w, http.StatusOK, resp)
}

type createRequest struct {
	Name string `json:"name"`
	Kind string `json:"kind"`
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	userID, _ := authctx.UserID(r.Context())

	var req createRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid_body", "Corpo da requisição inválido.")
		return
	}

	c, err := h.service.Create(r.Context(), userID, req.Name, req.Kind)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, toResponse(c))
}

type updateRequest struct {
	Name string `json:"name"`
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	userID, _ := authctx.UserID(r.Context())
	id := r.PathValue("id")

	var req updateRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid_body", "Corpo da requisição inválido.")
		return
	}

	c, err := h.service.Update(r.Context(), userID, id, req.Name)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, toResponse(c))
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
		httpx.JSON(w, http.StatusOK, map[string]string{"message": "Categoria possui lançamentos vinculados e foi arquivada em vez de excluída."})
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"message": "Categoria excluída."})
}

func writeDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		httpx.NotFoundOrForbidden(w)
	case errors.Is(err, ErrSystemCategory):
		httpx.Error(w, http.StatusForbidden, "system_category", "Categorias do sistema não podem ser alteradas.")
	case errors.Is(err, ErrInvalidName):
		httpx.ValidationError(w, map[string]string{"name": "Nome inválido."})
	case errors.Is(err, ErrInvalidKind):
		httpx.ValidationError(w, map[string]string{"kind": "Tipo inválido (use income ou expense)."})
	case errors.Is(err, ErrNameTaken):
		httpx.Error(w, http.StatusConflict, "name_taken", "Já existe uma categoria com este nome e tipo.")
	default:
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "Não foi possível concluir a operação.")
	}
}
