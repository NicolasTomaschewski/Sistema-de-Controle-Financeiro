package integration

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/cookiejar"
	"testing"
	"time"
)

type testAccount struct {
	ID             string `json:"id"`
	CurrentBalance string `json:"current_balance"`
}

type testCategory struct {
	ID   string `json:"id"`
	Kind string `json:"kind"`
}

func mustCSRF(t *testing.T, client *http.Client, baseURL string) string {
	t.Helper()
	tok := csrfTokenFromJar(client, baseURL)
	if tok == "" {
		t.Fatal("token csrf ausente; o usuário precisa estar autenticado")
	}
	return tok
}

func createAccount(t *testing.T, client *http.Client, baseURL, csrf, name, initialBalance, date string) testAccount {
	t.Helper()
	resp := doJSON(t, client, http.MethodPost, baseURL+"/api/v1/accounts", map[string]string{
		"name": name, "type": "conta_corrente", "currency": "BRL",
		"initial_balance": initialBalance, "initial_balance_date": date,
	}, csrf)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("esperava 201 ao criar conta, obteve %d: %s", resp.StatusCode, b)
	}
	var out struct {
		Data testAccount `json:"data"`
	}
	json.NewDecoder(resp.Body).Decode(&out)
	return out.Data
}

func getAccount(t *testing.T, client *http.Client, baseURL, id string) (testAccount, int) {
	t.Helper()
	resp := doJSON(t, client, http.MethodGet, baseURL+"/api/v1/accounts/"+id, nil, "")
	defer resp.Body.Close()
	var out struct {
		Data testAccount `json:"data"`
	}
	json.NewDecoder(resp.Body).Decode(&out)
	return out.Data, resp.StatusCode
}

func listCategories(t *testing.T, client *http.Client, baseURL string) []testCategory {
	t.Helper()
	resp := doJSON(t, client, http.MethodGet, baseURL+"/api/v1/categories", nil, "")
	defer resp.Body.Close()
	var out struct {
		Data []testCategory `json:"data"`
	}
	json.NewDecoder(resp.Body).Decode(&out)
	return out.Data
}

func firstCategoryOfKind(cats []testCategory, kind string) string {
	for _, c := range cats {
		if c.Kind == kind {
			return c.ID
		}
	}
	return ""
}

func createTransaction(t *testing.T, client *http.Client, baseURL, csrf string, body map[string]any) *http.Response {
	t.Helper()
	return doJSON(t, client, http.MethodPost, baseURL+"/api/v1/transactions", body, csrf)
}

func TestFinance_IncomeAndExpenseAffectAccountBalance(t *testing.T) {
	ts, client := newTestServer(t)
	registerUser(t, client, ts.URL, "Gabriel", "gabriel@example.com", "senhaForte123")
	loginUser(t, client, ts.URL, "gabriel@example.com", "senhaForte123")
	csrf := mustCSRF(t, client, ts.URL)

	acc := createAccount(t, client, ts.URL, csrf, "Conta Corrente", "1000.00", "2026-01-01")
	cats := listCategories(t, client, ts.URL)
	incomeCat := firstCategoryOfKind(cats, "income")
	expenseCat := firstCategoryOfKind(cats, "expense")

	// Receita paga aumenta o saldo.
	resp := createTransaction(t, client, ts.URL, csrf, map[string]any{
		"account_id": acc.ID, "category_id": incomeCat, "type": "income", "status": "paid",
		"amount": "500.00", "description": "Salário", "occurred_on": "2026-01-05",
	})
	if resp.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("esperava 201 na receita, obteve %d: %s", resp.StatusCode, b)
	}
	resp.Body.Close()

	// Despesa paga diminui o saldo.
	resp = createTransaction(t, client, ts.URL, csrf, map[string]any{
		"account_id": acc.ID, "category_id": expenseCat, "type": "expense", "status": "paid",
		"amount": "200.00", "description": "Supermercado", "occurred_on": "2026-01-06",
	})
	if resp.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("esperava 201 na despesa, obteve %d: %s", resp.StatusCode, b)
	}
	resp.Body.Close()

	// Despesa pendente NÃO deve afetar o saldo.
	resp = createTransaction(t, client, ts.URL, csrf, map[string]any{
		"account_id": acc.ID, "category_id": expenseCat, "type": "expense", "status": "pending",
		"amount": "999.00", "description": "Conta a pagar", "occurred_on": "2026-01-07",
	})
	if resp.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("esperava 201 na despesa pendente, obteve %d: %s", resp.StatusCode, b)
	}
	resp.Body.Close()

	updated, status := getAccount(t, client, ts.URL, acc.ID)
	if status != http.StatusOK {
		t.Fatalf("esperava 200 ao buscar conta, obteve %d", status)
	}
	// 1000 + 500 - 200 = 1300.00 (a despesa pendente não entra no cálculo).
	if updated.CurrentBalance != "1300.00" {
		t.Fatalf("saldo incorreto: esperava 1300.00, obteve %s", updated.CurrentBalance)
	}
}

func TestFinance_CategoryKindMismatchIsRejected(t *testing.T) {
	ts, client := newTestServer(t)
	registerUser(t, client, ts.URL, "Helena", "helena@example.com", "senhaForte123")
	loginUser(t, client, ts.URL, "helena@example.com", "senhaForte123")
	csrf := mustCSRF(t, client, ts.URL)

	acc := createAccount(t, client, ts.URL, csrf, "Carteira", "0.00", "2026-01-01")
	cats := listCategories(t, client, ts.URL)
	incomeCat := firstCategoryOfKind(cats, "income")

	resp := createTransaction(t, client, ts.URL, csrf, map[string]any{
		"account_id": acc.ID, "category_id": incomeCat, "type": "expense", "status": "paid",
		"amount": "50.00", "description": "Erro proposital", "occurred_on": "2026-01-10",
	})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("esperava 422 para categoria incompatível, obteve %d", resp.StatusCode)
	}
}

func TestFinance_TransferIsAtomicAndUpdatesBothAccounts(t *testing.T) {
	ts, client := newTestServer(t)
	registerUser(t, client, ts.URL, "Igor", "igor@example.com", "senhaForte123")
	loginUser(t, client, ts.URL, "igor@example.com", "senhaForte123")
	csrf := mustCSRF(t, client, ts.URL)

	accA := createAccount(t, client, ts.URL, csrf, "Conta A", "1000.00", "2026-01-01")
	accB := createAccount(t, client, ts.URL, csrf, "Conta B", "0.00", "2026-01-01")

	resp := doJSON(t, client, http.MethodPost, ts.URL+"/api/v1/transfers", map[string]any{
		"from_account_id": accA.ID, "to_account_id": accB.ID,
		"amount": "300.00", "description": "Reserva", "occurred_on": "2026-01-15",
	}, csrf)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("esperava 201 na transferência, obteve %d: %s", resp.StatusCode, b)
	}

	updatedA, _ := getAccount(t, client, ts.URL, accA.ID)
	updatedB, _ := getAccount(t, client, ts.URL, accB.ID)
	if updatedA.CurrentBalance != "700.00" {
		t.Fatalf("saldo da conta de origem incorreto: %s", updatedA.CurrentBalance)
	}
	if updatedB.CurrentBalance != "300.00" {
		t.Fatalf("saldo da conta de destino incorreto: %s", updatedB.CurrentBalance)
	}
}

func TestFinance_TransferRejectsAccountFromAnotherUser(t *testing.T) {
	ts, clientA := newTestServer(t)
	jarB, _ := cookiejar.New(nil)
	clientB := &http.Client{Jar: jarB, Timeout: 5 * time.Second}

	registerUser(t, clientA, ts.URL, "Julia", "julia@example.com", "senhaForte123")
	registerUser(t, clientB, ts.URL, "Kleber", "kleber@example.com", "senhaForte123")
	loginUser(t, clientA, ts.URL, "julia@example.com", "senhaForte123")
	loginUser(t, clientB, ts.URL, "kleber@example.com", "senhaForte123")
	csrfA := mustCSRF(t, clientA, ts.URL)
	csrfB := mustCSRF(t, clientB, ts.URL)

	accA := createAccount(t, clientA, ts.URL, csrfA, "Conta de Julia", "500.00", "2026-01-01")
	accB := createAccount(t, clientB, ts.URL, csrfB, "Conta de Kleber", "500.00", "2026-01-01")

	// Kleber tenta transferir usando a conta de Julia como destino.
	resp := doJSON(t, clientB, http.MethodPost, ts.URL+"/api/v1/transfers", map[string]any{
		"from_account_id": accB.ID, "to_account_id": accA.ID,
		"amount": "100.00", "description": "Tentativa indevida", "occurred_on": "2026-01-16",
	}, csrfB)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("esperava 422 ao usar conta de outro usuário na transferência, obteve %d", resp.StatusCode)
	}

	// O saldo de Julia não deve ter sido alterado pela tentativa.
	unchanged, status := getAccount(t, clientA, ts.URL, accA.ID)
	if status != http.StatusOK {
		t.Fatalf("esperava 200 ao consultar conta de Julia, obteve %d", status)
	}
	if unchanged.CurrentBalance != "500.00" {
		t.Fatalf("saldo de Julia deveria permanecer 500.00, obteve %s", unchanged.CurrentBalance)
	}
}

func TestFinance_UserCannotAccessOrModifyAnotherUsersAccount(t *testing.T) {
	ts, clientA := newTestServer(t)
	jarB, _ := cookiejar.New(nil)
	clientB := &http.Client{Jar: jarB, Timeout: 5 * time.Second}

	registerUser(t, clientA, ts.URL, "Larissa", "larissa@example.com", "senhaForte123")
	registerUser(t, clientB, ts.URL, "Marcos", "marcos@example.com", "senhaForte123")
	loginUser(t, clientA, ts.URL, "larissa@example.com", "senhaForte123")
	loginUser(t, clientB, ts.URL, "marcos@example.com", "senhaForte123")
	csrfA := mustCSRF(t, clientA, ts.URL)
	csrfB := mustCSRF(t, clientB, ts.URL)

	accA := createAccount(t, clientA, ts.URL, csrfA, "Conta de Larissa", "1000.00", "2026-01-01")

	// Marcos não pode ler a conta de Larissa pelo ID.
	_, status := getAccount(t, clientB, ts.URL, accA.ID)
	if status != http.StatusNotFound {
		t.Fatalf("esperava 404 ao ler conta de outro usuário, obteve %d", status)
	}

	// Marcos não pode editar a conta de Larissa.
	patchResp := doJSON(t, clientB, http.MethodPatch, ts.URL+"/api/v1/accounts/"+accA.ID, map[string]string{
		"name": "Hackeada", "type": "conta_corrente", "status": "active",
	}, csrfB)
	defer patchResp.Body.Close()
	if patchResp.StatusCode != http.StatusNotFound {
		t.Fatalf("esperava 404 ao editar conta de outro usuário, obteve %d", patchResp.StatusCode)
	}

	// Marcos não pode excluir a conta de Larissa.
	delResp := doJSON(t, clientB, http.MethodDelete, ts.URL+"/api/v1/accounts/"+accA.ID, nil, csrfB)
	defer delResp.Body.Close()
	if delResp.StatusCode != http.StatusNotFound {
		t.Fatalf("esperava 404 ao excluir conta de outro usuário, obteve %d", delResp.StatusCode)
	}

	// A conta de Larissa continua intacta.
	stillThere, status := getAccount(t, clientA, ts.URL, accA.ID)
	if status != http.StatusOK || stillThere.CurrentBalance != "1000.00" {
		t.Fatalf("conta de Larissa deveria permanecer intacta, obteve status=%d saldo=%s", status, stillThere.CurrentBalance)
	}
}

func TestFinance_TransactionListIsIsolatedPerUser(t *testing.T) {
	ts, clientA := newTestServer(t)
	jarB, _ := cookiejar.New(nil)
	clientB := &http.Client{Jar: jarB, Timeout: 5 * time.Second}

	registerUser(t, clientA, ts.URL, "Nadia", "nadia@example.com", "senhaForte123")
	registerUser(t, clientB, ts.URL, "Otavio", "otavio@example.com", "senhaForte123")
	loginUser(t, clientA, ts.URL, "nadia@example.com", "senhaForte123")
	loginUser(t, clientB, ts.URL, "otavio@example.com", "senhaForte123")
	csrfA := mustCSRF(t, clientA, ts.URL)
	csrfB := mustCSRF(t, clientB, ts.URL)

	accA := createAccount(t, clientA, ts.URL, csrfA, "Conta de Nadia", "0.00", "2026-01-01")
	accB := createAccount(t, clientB, ts.URL, csrfB, "Conta de Otavio", "0.00", "2026-01-01")
	catsA := listCategories(t, clientA, ts.URL)
	catsB := listCategories(t, clientB, ts.URL)

	createTransaction(t, clientA, ts.URL, csrfA, map[string]any{
		"account_id": accA.ID, "category_id": firstCategoryOfKind(catsA, "income"), "type": "income", "status": "paid",
		"amount": "100.00", "description": "Receita de Nadia", "occurred_on": "2026-01-10",
	}).Body.Close()
	createTransaction(t, clientB, ts.URL, csrfB, map[string]any{
		"account_id": accB.ID, "category_id": firstCategoryOfKind(catsB, "income"), "type": "income", "status": "paid",
		"amount": "200.00", "description": "Receita de Otavio", "occurred_on": "2026-01-10",
	}).Body.Close()

	resp := doJSON(t, clientA, http.MethodGet, ts.URL+"/api/v1/transactions", nil, "")
	defer resp.Body.Close()
	var out struct {
		Data []struct {
			Description string `json:"description"`
		} `json:"data"`
	}
	json.NewDecoder(resp.Body).Decode(&out)

	if len(out.Data) != 1 {
		t.Fatalf("esperava exatamente 1 lançamento para Nadia, obteve %d", len(out.Data))
	}
	if out.Data[0].Description != "Receita de Nadia" {
		t.Fatalf("lançamento de Nadia contém dados de outro usuário: %+v", out.Data)
	}
}
