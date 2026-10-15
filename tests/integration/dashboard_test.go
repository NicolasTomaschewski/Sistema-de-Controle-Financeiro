package integration

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestDashboard_SummaryConsolidatesAllDomains(t *testing.T) {
	ts, client := newTestServer(t)
	registerUser(t, client, ts.URL, "Alberto", "alberto@example.com", "senhaForte123")
	loginUser(t, client, ts.URL, "alberto@example.com", "senhaForte123")
	csrf := mustCSRF(t, client, ts.URL)

	acc := createAccount(t, client, ts.URL, csrf, "Conta Alberto", "1000.00", "2026-01-01")
	cats := listCategories(t, client, ts.URL)
	incomeCat := firstCategoryOfKind(cats, "income")
	expenseCat := firstCategoryOfKind(cats, "expense")

	createTransaction(t, client, ts.URL, csrf, map[string]any{
		"account_id": acc.ID, "category_id": incomeCat, "type": "income", "status": "paid",
		"amount": "2000.00", "description": "Salário de julho", "occurred_on": "2026-07-05",
	}).Body.Close()
	createTransaction(t, client, ts.URL, csrf, map[string]any{
		"account_id": acc.ID, "category_id": expenseCat, "type": "expense", "status": "paid",
		"amount": "300.00", "description": "Despesas de julho", "occurred_on": "2026-07-06",
	}).Body.Close()

	resp := doJSON(t, client, http.MethodGet, ts.URL+"/api/v1/dashboard/summary?year=2026&month=7", nil, "")
	defer resp.Body.Close()
	var out struct {
		Data struct {
			AccountsBalance string `json:"accounts_balance"`
			MonthlyIncome   string `json:"monthly_income"`
			MonthlyExpense  string `json:"monthly_expense"`
			MonthlyResult   string `json:"monthly_result"`
			NetWorth        struct {
				NetWorth string `json:"net_worth"`
			} `json:"net_worth"`
			RecentTransactions []struct {
				Description string `json:"description"`
			} `json:"recent_transactions"`
		} `json:"data"`
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("esperava 200, obteve %d", resp.StatusCode)
	}
	json.NewDecoder(resp.Body).Decode(&out)

	// 1000 (saldo inicial) + 2000 (receita) - 300 (despesa) = 2700.00
	if out.Data.AccountsBalance != "2700.00" {
		t.Fatalf("saldo de contas incorreto no dashboard: %s", out.Data.AccountsBalance)
	}
	if out.Data.MonthlyIncome != "2000.00" {
		t.Fatalf("receita mensal incorreta no dashboard: %s", out.Data.MonthlyIncome)
	}
	if out.Data.MonthlyExpense != "300.00" {
		t.Fatalf("despesa mensal incorreta no dashboard: %s", out.Data.MonthlyExpense)
	}
	if out.Data.MonthlyResult != "1700.00" {
		t.Fatalf("resultado mensal incorreto no dashboard: %s", out.Data.MonthlyResult)
	}
	if out.Data.NetWorth.NetWorth != "2700.00" {
		t.Fatalf("patrimônio líquido incorreto no dashboard: %s", out.Data.NetWorth.NetWorth)
	}
	if len(out.Data.RecentTransactions) != 2 {
		t.Fatalf("esperava 2 movimentações recentes, obteve %d", len(out.Data.RecentTransactions))
	}
}

func TestDashboard_MonthlyFlowReturnsChronologicalSeries(t *testing.T) {
	ts, client := newTestServer(t)
	registerUser(t, client, ts.URL, "Beatriz", "beatriz@example.com", "senhaForte123")
	loginUser(t, client, ts.URL, "beatriz@example.com", "senhaForte123")
	csrf := mustCSRF(t, client, ts.URL)

	acc := createAccount(t, client, ts.URL, csrf, "Conta Beatriz", "0.00", "2026-01-01")
	cats := listCategories(t, client, ts.URL)
	incomeCat := firstCategoryOfKind(cats, "income")

	createTransaction(t, client, ts.URL, csrf, map[string]any{
		"account_id": acc.ID, "category_id": incomeCat, "type": "income", "status": "paid",
		"amount": "500.00", "description": "Receita de março", "occurred_on": "2026-03-10",
	}).Body.Close()
	createTransaction(t, client, ts.URL, csrf, map[string]any{
		"account_id": acc.ID, "category_id": incomeCat, "type": "income", "status": "paid",
		"amount": "700.00", "description": "Receita de maio", "occurred_on": "2026-05-10",
	}).Body.Close()

	resp := doJSON(t, client, http.MethodGet, ts.URL+"/api/v1/dashboard/monthly-flow?year=2026&month=5&months=3", nil, "")
	defer resp.Body.Close()
	var out struct {
		Data []struct {
			Year        int    `json:"year"`
			Month       int    `json:"month"`
			TotalIncome string `json:"total_income"`
		} `json:"data"`
	}
	json.NewDecoder(resp.Body).Decode(&out)

	if len(out.Data) != 3 {
		t.Fatalf("esperava 3 pontos (meses 3, 4, 5), obteve %d", len(out.Data))
	}
	if out.Data[0].Month != 3 || out.Data[0].TotalIncome != "500.00" {
		t.Fatalf("primeiro ponto incorreto: %+v", out.Data[0])
	}
	if out.Data[1].Month != 4 || out.Data[1].TotalIncome != "0.00" {
		t.Fatalf("segundo ponto incorreto: %+v", out.Data[1])
	}
	if out.Data[2].Month != 5 || out.Data[2].TotalIncome != "700.00" {
		t.Fatalf("terceiro ponto incorreto: %+v", out.Data[2])
	}
}

func TestDashboard_IsIsolatedPerUser(t *testing.T) {
	ts, clientA := newTestServer(t)
	clientB := newJarClient(t)

	registerUser(t, clientA, ts.URL, "Carlos", "carlos@example.com", "senhaForte123")
	registerUser(t, clientB, ts.URL, "Debora", "debora@example.com", "senhaForte123")
	loginUser(t, clientA, ts.URL, "carlos@example.com", "senhaForte123")
	loginUser(t, clientB, ts.URL, "debora@example.com", "senhaForte123")
	csrfA := mustCSRF(t, clientA, ts.URL)

	createAccount(t, clientA, ts.URL, csrfA, "Conta de Carlos", "5000.00", "2026-01-01")

	respB := doJSON(t, clientB, http.MethodGet, ts.URL+"/api/v1/dashboard/summary", nil, "")
	defer respB.Body.Close()
	var outB struct {
		Data struct {
			AccountsBalance string `json:"accounts_balance"`
		} `json:"data"`
	}
	json.NewDecoder(respB.Body).Decode(&outB)

	if outB.Data.AccountsBalance != "0.00" {
		t.Fatalf("dashboard de Debora não deveria incluir saldo da conta de Carlos: %s", outB.Data.AccountsBalance)
	}
}
