package integration

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/nicolastomaschewski/controle-financeiro/internal/config"
	"github.com/nicolastomaschewski/controle-financeiro/internal/server"
)

// newTestServer monta a aplicação HTTP completa (rotas, middlewares,
// autenticação, CSRF) contra um schema de PostgreSQL de teste isolado e
// já migrado.
func newTestServer(t *testing.T) (*httptest.Server, *http.Client) {
	t.Helper()
	db := newMigratedTestDB(t)

	cfg := &config.Config{
		CookieSecure: false,
		CORSOrigins:  []string{},
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	handler := server.New(db, cfg, logger)
	ts := httptest.NewServer(handler)
	t.Cleanup(ts.Close)

	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("criar cookie jar: %v", err)
	}
	client := &http.Client{Jar: jar, Timeout: 5 * time.Second}

	return ts, client
}

func doJSON(t *testing.T, client *http.Client, method, url string, body any, csrfToken string) *http.Response {
	t.Helper()
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("serializar corpo: %v", err)
		}
		reader = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, url, reader)
	if err != nil {
		t.Fatalf("criar requisição: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if csrfToken != "" {
		req.Header.Set("X-CSRF-Token", csrfToken)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("executar requisição %s %s: %v", method, url, err)
	}
	return resp
}

func csrfTokenFromJar(client *http.Client, rawURL string) string {
	u, _ := http.NewRequest(http.MethodGet, rawURL, nil)
	for _, c := range client.Jar.Cookies(u.URL) {
		if c.Name == "csrf_token" {
			return c.Value
		}
	}
	return ""
}

func registerUser(t *testing.T, client *http.Client, baseURL, name, email, password string) {
	t.Helper()
	resp := doJSON(t, client, http.MethodPost, baseURL+"/api/v1/auth/register", map[string]string{
		"name": name, "email": email, "password": password,
	}, "")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("esperava 201 no registro, obteve %d: %s", resp.StatusCode, b)
	}
}

func loginUser(t *testing.T, client *http.Client, baseURL, email, password string) {
	t.Helper()
	resp := doJSON(t, client, http.MethodPost, baseURL+"/api/v1/auth/login", map[string]string{
		"email": email, "password": password,
	}, "")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("esperava 200 no login, obteve %d: %s", resp.StatusCode, b)
	}
}

func TestAuth_RegisterLoginLogoutFlow(t *testing.T) {
	ts, client := newTestServer(t)

	registerUser(t, client, ts.URL, "Ana Silva", "ana@example.com", "senhaForte123")
	loginUser(t, client, ts.URL, "ana@example.com", "senhaForte123")

	// Acesso ao próprio perfil deve funcionar após login.
	resp := doJSON(t, client, http.MethodGet, ts.URL+"/api/v1/me", nil, "")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("esperava 200 ao consultar /me, obteve %d", resp.StatusCode)
	}
	var profile struct {
		Data struct {
			Email string `json:"email"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&profile); err != nil {
		t.Fatalf("decodificar perfil: %v", err)
	}
	if profile.Data.Email != "ana@example.com" {
		t.Fatalf("e-mail do perfil incorreto: %s", profile.Data.Email)
	}

	// Logout deve revogar a sessão.
	logoutResp := doJSON(t, client, http.MethodPost, ts.URL+"/api/v1/auth/logout", nil, "")
	logoutResp.Body.Close()
	if logoutResp.StatusCode != http.StatusOK {
		t.Fatalf("esperava 200 no logout, obteve %d", logoutResp.StatusCode)
	}

	afterLogout := doJSON(t, client, http.MethodGet, ts.URL+"/api/v1/me", nil, "")
	defer afterLogout.Body.Close()
	if afterLogout.StatusCode != http.StatusUnauthorized {
		t.Fatalf("esperava 401 após logout, obteve %d", afterLogout.StatusCode)
	}
}

func TestAuth_DuplicateEmailIsRejected(t *testing.T) {
	ts, client := newTestServer(t)
	registerUser(t, client, ts.URL, "Bruno", "bruno@example.com", "senhaForte123")

	resp := doJSON(t, client, http.MethodPost, ts.URL+"/api/v1/auth/register", map[string]string{
		"name": "Bruno 2", "email": "bruno@example.com", "password": "outraSenha123",
	}, "")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("esperava 409 para e-mail duplicado, obteve %d", resp.StatusCode)
	}
}

func TestAuth_WrongPasswordIsRejected(t *testing.T) {
	ts, client := newTestServer(t)
	registerUser(t, client, ts.URL, "Carla", "carla@example.com", "senhaCorreta123")

	resp := doJSON(t, client, http.MethodPost, ts.URL+"/api/v1/auth/login", map[string]string{
		"email": "carla@example.com", "password": "senhaErrada123",
	}, "")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("esperava 401 para senha incorreta, obteve %d", resp.StatusCode)
	}
}

func TestAuth_UnauthenticatedAccessIsRejected(t *testing.T) {
	ts, client := newTestServer(t)

	resp := doJSON(t, client, http.MethodGet, ts.URL+"/api/v1/me", nil, "")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("esperava 401 sem autenticação, obteve %d", resp.StatusCode)
	}
}

func TestAuth_LoginRateLimitingPersists(t *testing.T) {
	ts, client := newTestServer(t)
	registerUser(t, client, ts.URL, "Diego", "diego@example.com", "senhaCorreta123")

	var lastStatus int
	for i := 0; i < 6; i++ {
		resp := doJSON(t, client, http.MethodPost, ts.URL+"/api/v1/auth/login", map[string]string{
			"email": "diego@example.com", "password": "senhaErrada123",
		}, "")
		lastStatus = resp.StatusCode
		resp.Body.Close()
	}

	if lastStatus != http.StatusTooManyRequests {
		t.Fatalf("esperava 429 após exceder tentativas de login, obteve %d", lastStatus)
	}
}

func TestAuth_CSRFProtectionBlocksMutationWithoutToken(t *testing.T) {
	ts, client := newTestServer(t)
	registerUser(t, client, ts.URL, "Elaine", "elaine@example.com", "senhaForte123")
	loginUser(t, client, ts.URL, "elaine@example.com", "senhaForte123")

	// Sem o cabeçalho X-CSRF-Token, a mutação deve ser bloqueada mesmo com
	// sessão válida.
	resp := doJSON(t, client, http.MethodPatch, ts.URL+"/api/v1/me", map[string]string{
		"name": "Elaine Souza", "email": "elaine@example.com",
	}, "")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("esperava 403 sem token CSRF, obteve %d", resp.StatusCode)
	}

	token := csrfTokenFromJar(client, ts.URL)
	if token == "" {
		t.Fatal("esperava cookie csrf_token presente após login")
	}

	okResp := doJSON(t, client, http.MethodPatch, ts.URL+"/api/v1/me", map[string]string{
		"name": "Elaine Souza", "email": "elaine@example.com",
	}, token)
	defer okResp.Body.Close()
	if okResp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(okResp.Body)
		t.Fatalf("esperava 200 com token CSRF correto, obteve %d: %s", okResp.StatusCode, b)
	}
}

func TestAuth_TwoUsersHaveIndependentProfiles(t *testing.T) {
	ts, clientA := newTestServer(t) // compartilha o mesmo servidor/DB para o segundo cliente
	jarB, _ := cookiejar.New(nil)
	clientB := &http.Client{Jar: jarB, Timeout: 5 * time.Second}

	registerUser(t, clientA, ts.URL, "Usuario A", "user.a@example.com", "senhaForte123")
	registerUser(t, clientB, ts.URL, "Usuario B", "user.b@example.com", "senhaForte123")

	loginUser(t, clientA, ts.URL, "user.a@example.com", "senhaForte123")
	loginUser(t, clientB, ts.URL, "user.b@example.com", "senhaForte123")

	respA := doJSON(t, clientA, http.MethodGet, ts.URL+"/api/v1/me", nil, "")
	defer respA.Body.Close()
	respB := doJSON(t, clientB, http.MethodGet, ts.URL+"/api/v1/me", nil, "")
	defer respB.Body.Close()

	var profileA, profileB struct {
		Data struct {
			Email string `json:"email"`
		} `json:"data"`
	}
	json.NewDecoder(respA.Body).Decode(&profileA)
	json.NewDecoder(respB.Body).Decode(&profileB)

	if profileA.Data.Email != "user.a@example.com" {
		t.Fatalf("sessão do usuário A retornou dados incorretos: %+v", profileA)
	}
	if profileB.Data.Email != "user.b@example.com" {
		t.Fatalf("sessão do usuário B retornou dados incorretos: %+v", profileB)
	}
	if profileA.Data.Email == profileB.Data.Email {
		t.Fatal("usuários distintos não deveriam compartilhar a mesma sessão/perfil")
	}
}

func TestAuth_ForgedSessionTokenIsRejected(t *testing.T) {
	ts, client := newTestServer(t)
	registerUser(t, client, ts.URL, "Fabio", "fabio@example.com", "senhaForte123")

	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/api/v1/me", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: "token-forjado-que-nao-existe-no-banco"})
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("executar requisição: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("esperava 401 para token de sessão forjado, obteve %d", resp.StatusCode)
	}
}
