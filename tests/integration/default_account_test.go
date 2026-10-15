package integration

import (
	"encoding/json"
	"io"
	"net/http"
	"sync"
	"testing"
)

// A interface não expõe mais a gestão de contas: lançamentos e custos fixos
// sem conta usam a conta padrão "Carteira", criada uma única vez por usuário.

func listAccountIDs(t *testing.T, client *http.Client, baseURL string) []string {
	t.Helper()
	resp := doJSON(t, client, http.MethodGet, baseURL+"/api/v1/accounts", nil, "")
	defer resp.Body.Close()
	var out struct {
		Data []testAccount `json:"data"`
	}
	json.NewDecoder(resp.Body).Decode(&out)
	ids := make([]string, 0, len(out.Data))
	for _, a := range out.Data {
		ids = append(ids, a.ID)
	}
	return ids
}

func TestDefaultAccount_TransactionsWithoutAccountUseSingleCarteira(t *testing.T) {
	ts, client := newTestServer(t)
	registerUser(t, client, ts.URL, "Rui", "rui@example.com", "senhaForte123")
	loginUser(t, client, ts.URL, "rui@example.com", "senhaForte123")
	csrf := mustCSRF(t, client, ts.URL)

	if ids := listAccountIDs(t, client, ts.URL); len(ids) != 0 {
		t.Fatalf("usuário novo não deveria ter contas, tem %d", len(ids))
	}

	// Requisições simultâneas não podem criar mais de uma conta padrão.
	var wg sync.WaitGroup
	statuses := make([]int, 5)
	for i := range statuses {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			resp := createTransaction(t, client, ts.URL, csrf, map[string]any{
				"type": "income", "amount": "100.00", "description": "Freela", "occurred_on": "2026-03-10",
			})
			statuses[i] = resp.StatusCode
			resp.Body.Close()
		}(i)
	}
	wg.Wait()
	for _, s := range statuses {
		if s != http.StatusCreated {
			t.Fatalf("esperava 201 em todos os lançamentos sem conta, obteve %v", statuses)
		}
	}

	ids := listAccountIDs(t, client, ts.URL)
	if len(ids) != 1 {
		t.Fatalf("esperava exatamente uma conta padrão, obteve %d", len(ids))
	}
	acc, _ := getAccount(t, client, ts.URL, ids[0])
	if acc.CurrentBalance != "500.00" {
		t.Fatalf("saldo da Carteira deveria ser 500.00, obteve %s", acc.CurrentBalance)
	}

	// Custo fixo sem conta também gera a ocorrência na Carteira.
	ruleID := createRule(t, client, ts.URL, csrf, map[string]any{
		"description": "Internet", "planned_amount": "99.90", "periodicity": "monthly",
		"start_date": "2026-01-01", "due_day": 5,
	})
	gen := doJSON(t, client, http.MethodPost, ts.URL+"/api/v1/recurring-rules/"+ruleID+"/generate?year=2026&month=3", nil, csrf)
	defer gen.Body.Close()
	if gen.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(gen.Body)
		t.Fatalf("esperava 201 ao lançar custo fixo sem conta, obteve %d: %s", gen.StatusCode, b)
	}
	if ids := listAccountIDs(t, client, ts.URL); len(ids) != 1 {
		t.Fatalf("a geração não deveria criar outra conta, há %d", len(ids))
	}
}

func TestDefaultAccount_IsPerUser(t *testing.T) {
	ts, clientA := newTestServer(t)
	registerUser(t, clientA, ts.URL, "Ana", "ana.d@example.com", "senhaForte123")
	loginUser(t, clientA, ts.URL, "ana.d@example.com", "senhaForte123")
	csrfA := mustCSRF(t, clientA, ts.URL)

	clientB := newJarClient(t)
	registerUser(t, clientB, ts.URL, "Beto", "beto.d@example.com", "senhaForte123")
	loginUser(t, clientB, ts.URL, "beto.d@example.com", "senhaForte123")
	csrfB := mustCSRF(t, clientB, ts.URL)

	for _, c := range []struct {
		client *http.Client
		csrf   string
	}{{clientA, csrfA}, {clientB, csrfB}} {
		resp := createTransaction(t, c.client, ts.URL, c.csrf, map[string]any{
			"type": "expense", "amount": "10.00", "description": "Café", "occurred_on": "2026-03-10",
		})
		resp.Body.Close()
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("esperava 201, obteve %d", resp.StatusCode)
		}
	}

	idsA := listAccountIDs(t, clientA, ts.URL)
	idsB := listAccountIDs(t, clientB, ts.URL)
	if len(idsA) != 1 || len(idsB) != 1 || idsA[0] == idsB[0] {
		t.Fatalf("cada usuário deveria ter a própria Carteira: A=%v B=%v", idsA, idsB)
	}
	if _, status := getAccount(t, clientA, ts.URL, idsB[0]); status != http.StatusNotFound {
		t.Fatalf("A não deveria ver a Carteira de B, status %d", status)
	}
}
