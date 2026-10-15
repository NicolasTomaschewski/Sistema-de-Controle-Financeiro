package integration

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

func TestRecurring_GenerateOccurrenceIsIdempotent(t *testing.T) {
	ts, client := newTestServer(t)
	registerUser(t, client, ts.URL, "Paulo", "paulo@example.com", "senhaForte123")
	loginUser(t, client, ts.URL, "paulo@example.com", "senhaForte123")
	csrf := mustCSRF(t, client, ts.URL)

	acc := createAccount(t, client, ts.URL, csrf, "Conta Paulo", "0.00", "2026-01-01")
	cats := listCategories(t, client, ts.URL)
	expenseCat := firstCategoryOfKind(cats, "expense")

	createResp := doJSON(t, client, http.MethodPost, ts.URL+"/api/v1/recurring-rules", map[string]any{
		"account_id": acc.ID, "category_id": expenseCat, "description": "Aluguel",
		"planned_amount": "1500.00", "periodicity": "monthly", "start_date": "2026-01-01", "due_day": 5,
	}, csrf)
	defer createResp.Body.Close()
	if createResp.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(createResp.Body)
		t.Fatalf("esperava 201 ao criar regra, obteve %d: %s", createResp.StatusCode, b)
	}
	var created struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	json.NewDecoder(createResp.Body).Decode(&created)
	ruleID := created.Data.ID

	genURL := ts.URL + "/api/v1/recurring-rules/" + ruleID + "/generate?year=2026&month=3"

	first := doJSON(t, client, http.MethodPost, genURL, nil, csrf)
	defer first.Body.Close()
	if first.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(first.Body)
		t.Fatalf("esperava 201 na primeira geração, obteve %d: %s", first.StatusCode, b)
	}
	var firstOut struct {
		Data struct {
			TransactionID string `json:"transaction_id"`
			Created       bool   `json:"created"`
		} `json:"data"`
	}
	json.NewDecoder(first.Body).Decode(&firstOut)
	if !firstOut.Data.Created {
		t.Fatal("esperava created=true na primeira geração")
	}

	second := doJSON(t, client, http.MethodPost, genURL, nil, csrf)
	defer second.Body.Close()
	if second.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(second.Body)
		t.Fatalf("esperava 200 (idempotente) na segunda geração, obteve %d: %s", second.StatusCode, b)
	}
	var secondOut struct {
		Data struct {
			TransactionID string `json:"transaction_id"`
			Created       bool   `json:"created"`
		} `json:"data"`
	}
	json.NewDecoder(second.Body).Decode(&secondOut)
	if secondOut.Data.Created {
		t.Fatal("segunda geração não deveria criar uma nova ocorrência")
	}
	if secondOut.Data.TransactionID != firstOut.Data.TransactionID {
		t.Fatalf("a geração repetida deveria retornar a mesma ocorrência: %s != %s", secondOut.Data.TransactionID, firstOut.Data.TransactionID)
	}

	// Confere que apenas um lançamento foi efetivamente criado para o período.
	listResp := doJSON(t, client, http.MethodGet, ts.URL+"/api/v1/transactions?account_id="+acc.ID, nil, "")
	defer listResp.Body.Close()
	var list struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	json.NewDecoder(listResp.Body).Decode(&list)
	if len(list.Data) != 1 {
		t.Fatalf("esperava exatamente 1 lançamento gerado, obteve %d", len(list.Data))
	}
}

func TestBudget_SummaryReflectsSpendingAndLimit(t *testing.T) {
	ts, client := newTestServer(t)
	registerUser(t, client, ts.URL, "Quenia", "quenia@example.com", "senhaForte123")
	loginUser(t, client, ts.URL, "quenia@example.com", "senhaForte123")
	csrf := mustCSRF(t, client, ts.URL)

	acc := createAccount(t, client, ts.URL, csrf, "Conta Quenia", "5000.00", "2026-01-01")
	cats := listCategories(t, client, ts.URL)
	expenseCat := firstCategoryOfKind(cats, "expense")

	budgetResp := doJSON(t, client, http.MethodPost, ts.URL+"/api/v1/budgets", map[string]any{
		"category_id": expenseCat, "year": 2026, "month": 4, "planned_amount": "300.00",
	}, csrf)
	defer budgetResp.Body.Close()
	if budgetResp.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(budgetResp.Body)
		t.Fatalf("esperava 201 ao criar orçamento, obteve %d: %s", budgetResp.StatusCode, b)
	}

	createTransaction(t, client, ts.URL, csrf, map[string]any{
		"account_id": acc.ID, "category_id": expenseCat, "type": "expense", "status": "paid",
		"amount": "350.00", "description": "Gasto acima do limite", "occurred_on": "2026-04-10",
	}).Body.Close()

	summaryResp := doJSON(t, client, http.MethodGet, ts.URL+"/api/v1/budgets?year=2026&month=4", nil, "")
	defer summaryResp.Body.Close()
	var out struct {
		Data []struct {
			PlannedAmount   string `json:"planned_amount"`
			SpentAmount     string `json:"spent_amount"`
			AvailableAmount string `json:"available_amount"`
			OverLimit       bool   `json:"over_limit"`
		} `json:"data"`
	}
	json.NewDecoder(summaryResp.Body).Decode(&out)

	if len(out.Data) != 1 {
		t.Fatalf("esperava 1 item no resumo do orçamento, obteve %d", len(out.Data))
	}
	item := out.Data[0]
	if item.SpentAmount != "350.00" {
		t.Fatalf("gasto incorreto: %s", item.SpentAmount)
	}
	if item.AvailableAmount != "-50.00" {
		t.Fatalf("saldo disponível incorreto: %s", item.AvailableAmount)
	}
	if !item.OverLimit {
		t.Fatal("esperava over_limit=true quando o gasto excede o planejado")
	}
}

func TestTransaction_MonthlySummaryExcludesTransfers(t *testing.T) {
	ts, client := newTestServer(t)
	registerUser(t, client, ts.URL, "Renata", "renata@example.com", "senhaForte123")
	loginUser(t, client, ts.URL, "renata@example.com", "senhaForte123")
	csrf := mustCSRF(t, client, ts.URL)

	accA := createAccount(t, client, ts.URL, csrf, "Conta A Renata", "1000.00", "2026-01-01")
	accB := createAccount(t, client, ts.URL, csrf, "Conta B Renata", "0.00", "2026-01-01")
	cats := listCategories(t, client, ts.URL)
	incomeCat := firstCategoryOfKind(cats, "income")
	expenseCat := firstCategoryOfKind(cats, "expense")

	createTransaction(t, client, ts.URL, csrf, map[string]any{
		"account_id": accA.ID, "category_id": incomeCat, "type": "income", "status": "paid",
		"amount": "1000.00", "description": "Salário de maio", "occurred_on": "2026-05-05",
	}).Body.Close()
	createTransaction(t, client, ts.URL, csrf, map[string]any{
		"account_id": accA.ID, "category_id": expenseCat, "type": "expense", "status": "paid",
		"amount": "400.00", "description": "Despesas de maio", "occurred_on": "2026-05-06",
	}).Body.Close()

	// Transferência grande que NÃO deve aparecer no consolidado de receitas/despesas.
	doJSON(t, client, http.MethodPost, ts.URL+"/api/v1/transfers", map[string]any{
		"from_account_id": accA.ID, "to_account_id": accB.ID,
		"amount": "5000.00", "description": "Reserva de maio", "occurred_on": "2026-05-10",
	}, csrf).Body.Close()

	resp := doJSON(t, client, http.MethodGet, ts.URL+"/api/v1/transactions/monthly-summary?year=2026&month=5", nil, "")
	defer resp.Body.Close()
	var out struct {
		Data struct {
			TotalIncome  string `json:"total_income"`
			TotalExpense string `json:"total_expense"`
			Result       string `json:"result"`
		} `json:"data"`
	}
	json.NewDecoder(resp.Body).Decode(&out)

	if out.Data.TotalIncome != "1000.00" {
		t.Fatalf("receita do mês incorreta (transferência não deveria contar): %s", out.Data.TotalIncome)
	}
	if out.Data.TotalExpense != "400.00" {
		t.Fatalf("despesa do mês incorreta (transferência não deveria contar): %s", out.Data.TotalExpense)
	}
	if out.Data.Result != "600.00" {
		t.Fatalf("resultado do mês incorreto: %s", out.Data.Result)
	}
}
