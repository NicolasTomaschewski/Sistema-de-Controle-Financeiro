package user

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
	mux.HandleFunc("GET /api/v1/me", h.getProfile)
	mux.HandleFunc("PATCH /api/v1/me", h.updateProfile)
	mux.HandleFunc("PATCH /api/v1/me/password", h.changePassword)
}

type profileResponse struct {
	ID              string  `json:"id"`
	Name            string  `json:"name"`
	Email           string  `json:"email"`
	EmailVerifiedAt *string `json:"email_verified_at"`
}

func toProfileResponse(u *User) profileResponse {
	resp := profileResponse{ID: u.ID, Name: u.Name, Email: u.Email}
	if u.EmailVerifiedAt != nil {
		s := u.EmailVerifiedAt.Format("2006-01-02T15:04:05Z07:00")
		resp.EmailVerifiedAt = &s
	}
	return resp
}

func (h *Handler) getProfile(w http.ResponseWriter, r *http.Request) {
	userID, ok := authctx.UserID(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized", "Autenticação necessária.")
		return
	}

	u, err := h.service.GetProfile(r.Context(), userID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "Não foi possível carregar o perfil.")
		return
	}

	httpx.JSON(w, http.StatusOK, toProfileResponse(u))
}

type updateProfileRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

func (h *Handler) updateProfile(w http.ResponseWriter, r *http.Request) {
	userID, ok := authctx.UserID(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized", "Autenticação necessária.")
		return
	}

	var req updateProfileRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid_body", "Corpo da requisição inválido.")
		return
	}

	u, err := h.service.UpdateProfile(r.Context(), userID, req.Name, req.Email)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidName):
			httpx.ValidationError(w, map[string]string{"name": "Nome inválido."})
		case errors.Is(err, ErrInvalidEmail):
			httpx.ValidationError(w, map[string]string{"email": "E-mail inválido."})
		case errors.Is(err, ErrEmailTaken):
			httpx.Error(w, http.StatusConflict, "email_taken", "Este e-mail já está em uso.")
		default:
			httpx.Error(w, http.StatusInternalServerError, "internal_error", "Não foi possível atualizar o perfil.")
		}
		return
	}

	httpx.JSON(w, http.StatusOK, toProfileResponse(u))
}

type changePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

func (h *Handler) changePassword(w http.ResponseWriter, r *http.Request) {
	userID, ok := authctx.UserID(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized", "Autenticação necessária.")
		return
	}

	var req changePasswordRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid_body", "Corpo da requisição inválido.")
		return
	}

	err := h.service.ChangePassword(r.Context(), userID, req.CurrentPassword, req.NewPassword)
	if err != nil {
		switch {
		case errors.Is(err, ErrWrongPassword):
			httpx.Error(w, http.StatusUnauthorized, "wrong_password", "Senha atual incorreta.")
		case errors.Is(err, ErrInvalidPassword):
			httpx.ValidationError(w, map[string]string{"new_password": "A nova senha deve ter entre 8 e 128 caracteres."})
		default:
			httpx.Error(w, http.StatusInternalServerError, "internal_error", "Não foi possível alterar a senha.")
		}
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]string{"message": "Senha alterada com sucesso."})
}
