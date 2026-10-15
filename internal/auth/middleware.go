package auth

import (
	"net/http"

	"github.com/nicolastomaschewski/controle-financeiro/internal/authctx"
	"github.com/nicolastomaschewski/controle-financeiro/internal/httpx"
)

// RequireAuth exige uma sessão válida para acessar a rota, carregando a
// identidade exclusivamente a partir do cookie de sessão validado contra o
// banco de dados — nunca a partir de qualquer dado enviado pelo cliente.
func RequireAuth(sessions *SessionRepository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, ok := ReadSessionToken(r)
			if !ok {
				httpx.Error(w, http.StatusUnauthorized, "unauthorized", "Autenticação necessária.")
				return
			}

			userID, err := sessions.ValidateToken(r.Context(), token)
			if err != nil {
				httpx.Error(w, http.StatusUnauthorized, "unauthorized", "Sessão inválida ou expirada.")
				return
			}

			ctx := authctx.WithUserID(r.Context(), userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
