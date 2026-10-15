package auth

import (
	"errors"
	"net"
	"net/http"

	"github.com/nicolastomaschewski/controle-financeiro/internal/httpx"
	"github.com/nicolastomaschewski/controle-financeiro/internal/middleware"
	"github.com/nicolastomaschewski/controle-financeiro/internal/user"
)

type Handler struct {
	service      *Service
	cookieSecure bool
}

func NewHandler(service *Service, cookieSecure bool) *Handler {
	return &Handler{service: service, cookieSecure: cookieSecure}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/auth/register", h.register)
	mux.HandleFunc("POST /api/v1/auth/login", h.login)
	mux.HandleFunc("POST /api/v1/auth/logout", h.logout)
}

type registerRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *Handler) register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid_body", "Corpo da requisição inválido.")
		return
	}

	u, err := h.service.Register(r.Context(), req.Name, req.Email, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, user.ErrInvalidName):
			httpx.ValidationError(w, map[string]string{"name": "Nome inválido."})
		case errors.Is(err, user.ErrInvalidEmail):
			httpx.ValidationError(w, map[string]string{"email": "E-mail inválido."})
		case errors.Is(err, user.ErrInvalidPassword):
			httpx.ValidationError(w, map[string]string{"password": "A senha deve ter entre 8 e 128 caracteres."})
		case errors.Is(err, user.ErrEmailTaken):
			httpx.Error(w, http.StatusConflict, "email_taken", "Este e-mail já está cadastrado.")
		default:
			httpx.Error(w, http.StatusInternalServerError, "internal_error", "Não foi possível concluir o cadastro.")
		}
		return
	}

	httpx.JSON(w, http.StatusCreated, map[string]string{"id": u.ID, "name": u.Name, "email": u.Email})
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid_body", "Corpo da requisição inválido.")
		return
	}

	ip := clientIP(r)
	token, expiresAt, err := h.service.Login(r.Context(), req.Email, req.Password, ip, r.UserAgent())
	if err != nil {
		switch {
		case errors.Is(err, ErrRateLimited):
			httpx.Error(w, http.StatusTooManyRequests, "rate_limited", "Muitas tentativas de login. Tente novamente mais tarde.")
		case errors.Is(err, ErrInvalidCredentials):
			httpx.Error(w, http.StatusUnauthorized, "invalid_credentials", "E-mail ou senha inválidos.")
		default:
			httpx.Error(w, http.StatusInternalServerError, "internal_error", "Não foi possível autenticar.")
		}
		return
	}

	SetSessionCookie(w, token, expiresAt, h.cookieSecure)

	csrfToken, err := middleware.NewCSRFToken()
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "Não foi possível iniciar a sessão.")
		return
	}
	middleware.SetCSRFCookie(w, csrfToken, h.cookieSecure)

	httpx.JSON(w, http.StatusOK, map[string]string{"message": "Login realizado com sucesso."})
}

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	token, ok := ReadSessionToken(r)
	if ok {
		_ = h.service.Logout(r.Context(), token)
	}
	ClearSessionCookie(w, h.cookieSecure)
	middleware.ClearCSRFCookie(w, h.cookieSecure)
	httpx.JSON(w, http.StatusOK, map[string]string{"message": "Logout realizado com sucesso."})
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
