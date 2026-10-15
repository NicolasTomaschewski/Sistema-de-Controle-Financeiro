package middleware

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"net/http"
)

const (
	CSRFCookieName = "csrf_token"
	CSRFHeaderName = "X-CSRF-Token"
)

// NewCSRFToken gera um novo valor de token CSRF para o padrão double-submit
// cookie. O valor não é secreto por si só (fica em um cookie legível por
// JavaScript), mas a política de mesma origem impede que um site de terceiros
// o leia para reproduzi-lo no cabeçalho exigido.
func NewCSRFToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func SetCSRFCookie(w http.ResponseWriter, token string, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     CSRFCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: false,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

func ClearCSRFCookie(w http.ResponseWriter, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     CSRFCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: false,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

// CSRFProtect exige, para métodos que alteram estado (POST, PATCH, PUT,
// DELETE), que o cabeçalho X-CSRF-Token coincida com o cookie csrf_token.
// Métodos de leitura (GET, HEAD, OPTIONS) não são verificados.
func CSRFProtect(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			next.ServeHTTP(w, r)
			return
		}

		cookie, err := r.Cookie(CSRFCookieName)
		if err != nil || cookie.Value == "" {
			http.Error(w, `{"error":{"code":"csrf_missing","message":"Token CSRF ausente."}}`, http.StatusForbidden)
			return
		}

		header := r.Header.Get(CSRFHeaderName)
		if header == "" || subtle.ConstantTimeCompare([]byte(header), []byte(cookie.Value)) != 1 {
			http.Error(w, `{"error":{"code":"csrf_invalid","message":"Token CSRF inválido."}}`, http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}
