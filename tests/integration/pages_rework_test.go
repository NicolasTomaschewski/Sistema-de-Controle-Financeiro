package integration

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

// Testes das funcionalidades das páginas Despesas (custos fixos mensais/anuais),
// Receitas (resumo anual por fonte) e Patrimônio (composição de bens e investimentos).

type testCostSummary struct {
	MonthlyRulesTotal string `json:"monthly_rules_total"`
	YearlyRulesTotal  string `json:"yearly_rules_total"`
	AnnualTotal       string `json:"annual_total"`
	MonthlyEquivalent string `json:"monthly_equivalent"`
	ByCategory        []struct {
		CategoryID  *string `json:"category_id"`
		AnnualTotal string  `json:"annual_total"`
	} `json:"by_category"`
}

func createRule(t *testing.T, client *http.Client, baseURL, csrf string, body map[string]any) string {
	t.Helper()
	resp := doJSON(t, client, http.MethodPost, baseURL+"/api/v1/recurring-rules", body, csrf)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("esperava 201 ao criar regra, obteve %d: %s", resp.StatusCode, b)
	}
	var out struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	json.NewDecoder(resp.Body).Decode(&out)
	return out.Data.ID
}

func getCostSummary(t *testing.T, client *http.Client, baseURL string) testCostSummary {
	t.Helper()
	resp := doJSON(t, client, http.MethodGet, baseURL+"/api/v1/recurring-rules/summary", nil, "")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("esperava 200 no resumo de custos, obteve %d: %s", resp.StatusCode, b)
	}
	var out struct {
		Data testCostSummary `json:"data"`
	}
	json.NewDecoder(resp.Body).Decode(&out)
	return out.Data
}

func TestRecurring_YearlyRuleValidationAndGenerationOnlyInDueMonth(t *testing.T) {
	ts, client := newTestServer(t)
	registerUser(t, client, ts.URL, "Lia", "lia@example.com", "senhaForte123")
	loginUser(t, client, ts.URL, "lia@example.com", "senhaForte123")
	csrf := mustCSRF(t, client, ts.URL)

	acc := createAccount(t, client, ts.URL, csrf, "Conta Lia", "0.00", "2026-01-01")
	expenseCat := firstCategoryOfKind(listCategories(t, client, ts.URL), "expense")

	// Regra anual sem mês de vencimento e regra mensal com mês são rejeitadas.
	for _, body := range []map[string]any{
		{"account_id": acc.ID, "description": "IPVA", "planned_amount": "1200.00", "periodicity": "yearly", "start_date": "2026-01-01", "due_day": 10},
		{"account_id": acc.ID, "description": "Aluguel", "planned_amount": "1500.00", "periodicity": "monthly", "start_date": "2026-01-01", "due_day": 5, "due_month": 3},
	} {
		resp := doJSON(t, client, http.MethodPost, ts.URL+"/api/v1/recurring-rules", body, csrf)
		resp.Body.Close()
		if resp.StatusCode != http.StatusUnprocessableEntity {
			t.Fatalf("esperava 422 para %v, obteve %d", body["description"], resp.StatusCode)
		}
	}

	ruleID := createRule(t, client, ts.URL, csrf, map[string]any{
		"account_id": acc.ID, "category_id": expenseCat, "description": "IPVA",
		"planned_amount": "1200.00", "periodicity": "yearly", "start_date": "2026-01-01", "due_day": 10, "due_month": 3,
	})

	// Fora do mês de vencimento: rejeita e não cria lançamento.
	wrong := doJSON(t, client, http.MethodPost, ts.URL+"/api/v1/recurring-rules/"+ruleID+"/generate?year=2026&month=4", nil, csrf)
	wrong.Body.Close()
	if wrong.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("esperava 422 ao gerar fora do mês de vencimento, obteve %d", wrong.StatusCode)
	}

	right := doJSON(t, client, http.MethodPost, ts.URL+"/api/v1/recurring-rules/"+ruleID+"/generate?year=2026&month=3", nil, csrf)
	defer right.Body.Close()
	if right.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(right.Body)
		t.Fatalf("esperava 201 no mês de vencimento, obteve %d: %s", right.StatusCode, b)
	}
	var gen struct {
		Data struct {
			OccurredOn string `json:"occurred_on"`
			Status     string `json:"status"`
		} `json:"data"`
	}
	json.NewDecoder(right.Body).Decode(&gen)
	if gen.Data.OccurredOn != "2026-03-10" {
		t.Fatalf("ocorrência anual deveria vencer em 2026-03-10, obteve %s", gen.Data.OccurredOn)
	}
	if gen.Data.Status == "paid" {
		t.Fatal("ocorrência gerada não pode nascer como paga")
	}

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

func TestRecurring_CostSummaryCountsOnlyActiveRulesOfTheUser(t *testing.T) {
	ts, clientA := newTestServer(t)
	clientB := newJarClient(t)

	registerUser(t, clientA, ts.URL, "Ana", "ana.custos@example.com", "senhaForte123")
	loginUser(t, clientA, ts.URL, "ana.custos@example.com", "senhaForte123")
	csrfA := mustCSRF(t, clientA, ts.URL)
	registerUser(t, clientB, ts.URL, "Bruno", "bruno.custos@example.com", "senhaForte123")
	loginUser(t, clientB, ts.URL, "bruno.custos@example.com", "senhaForte123")
	csrfB := mustCSRF(t, clientB, ts.URL)

	createRule(t, clientA, ts.URL, csrfA, map[string]any{
		"description": "Aluguel", "planned_amount": "1500.00", "periodicity": "monthly", "start_date": "2026-01-01", "due_day": 5,
	})
	createRule(t, clientA, ts.URL, csrfA, map[string]any{
		"description": "IPTU", "planned_amount": "1200.00", "periodicity": "yearly", "start_date": "2026-01-01", "due_day": 10, "due_month": 2,
	})
	inactiveID := createRule(t, clientA, ts.URL, csrfA, map[string]any{
		"description": "Academia", "planned_amount": "100.00", "periodicity": "monthly", "start_date": "2026-01-01",
	})
	deactivate := doJSON(t, clientA, http.MethodPatch, ts.URL+"/api/v1/recurring-rules/"+inactiveID, map[string]any{
		"description": "Academia", "planned_amount": "100.00", "periodicity": "monthly", "start_date": "2026-01-01", "active": false,
	}, csrfA)
	deactivate.Body.Close()
	if deactivate.StatusCode != http.StatusOK {
		t.Fatalf("esperava 200 ao desativar regra, obteve %d", deactivate.StatusCode)
	}

	createRule(t, clientB, ts.URL, csrfB, map[string]any{
		"description": "Aluguel do Bruno", "planned_amount": "999.00", "periodicity": "monthly", "start_date": "2026-01-01",
	})

	a := getCostSummary(t, clientA, ts.URL)
	if a.MonthlyRulesTotal != "1500.00" || a.YearlyRulesTotal != "1200.00" {
		t.Fatalf("totais por periodicidade incorretos: %+v", a)
	}
	if a.AnnualTotal != "19200.00" || a.MonthlyEquivalent != "1600.00" {
		t.Fatalf("custo anual/mensal equivalente incorreto (regra inativa ou de outro usuário contada?): %+v", a)
	}

	b := getCostSummary(t, clientB, ts.URL)
	if b.AnnualTotal != "11988.00" || b.MonthlyEquivalent != "999.00" {
		t.Fatalf("resumo do usuário B deveria conter apenas a própria regra: %+v", b)
	}
}

func TestTransaction_AnnualSummaryGroupsPaidIncomeBySource(t *testing.T) {
	ts, clientA := newTestServer(t)
	clientB := newJarClient(t)

	registerUser(t, clientA, ts.URL, "Carla", "carla.renda@example.com", "senhaForte123")
	loginUser(t, clientA, ts.URL, "carla.renda@example.com", "senhaForte123")
	csrfA := mustCSRF(t, clientA, ts.URL)
	registerUser(t, clientB, ts.URL, "Davi", "davi.renda@example.com", "senhaForte123")
	loginUser(t, clientB, ts.URL, "davi.renda@example.com", "senhaForte123")
	csrfB := mustCSRF(t, clientB, ts.URL)

	accA := createAccount(t, clientA, ts.URL, csrfA, "Conta Carla", "0.00", "2025-01-01")
	incomeCat := firstCategoryOfKind(listCategories(t, clientA, ts.URL), "income")
	accB := createAccount(t, clientB, ts.URL, csrfB, "Conta Davi", "0.00", "2025-01-01")

	for _, tx := range []map[string]any{
		{"status": "paid", "amount": "500.00", "occurred_on": "2026-01-05"},
		{"status": "paid", "amount": "300.25", "occurred_on": "2026-06-05"},
		{"status": "pending", "amount": "999.00", "occurred_on": "2026-07-05"}, // não recebida: fora do total
		{"status": "paid", "amount": "777.00", "occurred_on": "2025-12-31"},    // outro ano
	} {
		tx["account_id"], tx["category_id"], tx["type"], tx["description"] = accA.ID, incomeCat, "income", "Salário"
		resp := createTransaction(t, clientA, ts.URL, csrfA, tx)
		resp.Body.Close()
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("esperava 201 ao criar receita, obteve %d", resp.StatusCode)
		}
	}
	resp := createTransaction(t, clientB, ts.URL, csrfB, map[string]any{
		"account_id": accB.ID, "category_id": incomeCat, "type": "income", "status": "paid",
		"amount": "5000.00", "description": "Salário Davi", "occurred_on": "2026-03-05",
	})
	resp.Body.Close()

	type annual struct {
		TotalIncome      string `json:"total_income"`
		IncomeByCategory []struct {
			CategoryID *string `json:"category_id"`
			Total      string  `json:"total"`
		} `json:"income_by_category"`
	}
	get := func(client *http.Client) annual {
		r := doJSON(t, client, http.MethodGet, ts.URL+"/api/v1/transactions/annual-summary?year=2026", nil, "")
		defer r.Body.Close()
		if r.StatusCode != http.StatusOK {
			t.Fatalf("esperava 200 no resumo anual, obteve %d", r.StatusCode)
		}
		var out struct {
			Data annual `json:"data"`
		}
		json.NewDecoder(r.Body).Decode(&out)
		return out.Data
	}

	a := get(clientA)
	if a.TotalIncome != "800.25" {
		t.Fatalf("receita anual de A deveria ser 800.25, obteve %s", a.TotalIncome)
	}
	if len(a.IncomeByCategory) != 1 || a.IncomeByCategory[0].Total != "800.25" {
		t.Fatalf("receitas por fonte incorretas: %+v", a.IncomeByCategory)
	}
	if b := get(clientB); b.TotalIncome != "5000.00" {
		t.Fatalf("receita anual de B deveria conter apenas os próprios lançamentos, obteve %s", b.TotalIncome)
	}

	bad := doJSON(t, clientA, http.MethodGet, ts.URL+"/api/v1/transactions/annual-summary?year=1800", nil, "")
	bad.Body.Close()
	if bad.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("esperava 422 para ano inválido, obteve %d", bad.StatusCode)
	}
}

func TestNetWorth_CompositionHasOnlyAssetsAndInvestmentsOfTheUser(t *testing.T) {
	ts, clientA := newTestServer(t)
	clientB := newJarClient(t)

	registerUser(t, clientA, ts.URL, "Elisa", "elisa.patrimonio@example.com", "senhaForte123")
	loginUser(t, clientA, ts.URL, "elisa.patrimonio@example.com", "senhaForte123")
	csrfA := mustCSRF(t, clientA, ts.URL)
	registerUser(t, clientB, ts.URL, "Fabio", "fabio.patrimonio@example.com", "senhaForte123")
	loginUser(t, clientB, ts.URL, "fabio.patrimonio@example.com", "senhaForte123")
	csrfB := mustCSRF(t, clientB, ts.URL)

	createAccount(t, clientA, ts.URL, csrfA, "Conta Elisa", "10000.00", "2026-01-01")
	for _, body := range []map[string]any{
		{"name": "Apartamento", "type": "imovel", "current_value": "300000.00", "valuation_date": "2026-01-01"},
		{"name": "Carro", "type": "veiculo", "current_value": "50000.00", "valuation_date": "2026-01-01"},
	} {
		r := doJSON(t, clientA, http.MethodPost, ts.URL+"/api/v1/assets", body, csrfA)
		r.Body.Close()
		if r.StatusCode != http.StatusCreated {
			t.Fatalf("esperava 201 ao criar ativo, obteve %d", r.StatusCode)
		}
	}
	r := doJSON(t, clientA, http.MethodPost, ts.URL+"/api/v1/liabilities", map[string]any{
		"name": "Financiamento", "type": "financiamento", "principal_amount": "250000.00",
		"current_balance": "200000.00", "reference_date": "2026-01-01",
	}, csrfA)
	r.Body.Close()
	createInvestment(t, clientA, ts.URL, csrfA, map[string]any{
		"name": "CDB", "asset_type": "cdb", "invested_amount": "1000.00", "market_value": "1250.50", "position_date": "2026-01-01",
	})
	createInvestment(t, clientA, ts.URL, csrfA, map[string]any{
		"name": "Ações na conta", "asset_type": "acao", "invested_amount": "500.00", "market_value": "500.00",
		"position_date": "2026-01-01", "include_in_net_worth": false,
	})

	r = doJSON(t, clientB, http.MethodPost, ts.URL+"/api/v1/assets", map[string]any{
		"name": "Casa do Fabio", "type": "imovel", "current_value": "900000.00", "valuation_date": "2026-01-01",
	}, csrfB)
	r.Body.Close()

	type composition struct {
		AssetsValue      string `json:"assets_value"`
		InvestmentsValue string `json:"investments_value"`
		Total            string `json:"total"`
		Slices           []struct {
			Group        string `json:"group"`
			Type         string `json:"type"`
			Value        string `json:"value"`
			SharePercent string `json:"share_percent"`
		} `json:"slices"`
	}
	get := func(client *http.Client) composition {
		resp := doJSON(t, client, http.MethodGet, ts.URL+"/api/v1/net-worth/composition", nil, "")
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("esperava 200 na composição, obteve %d", resp.StatusCode)
		}
		var out struct {
			Data composition `json:"data"`
		}
		json.NewDecoder(resp.Body).Decode(&out)
		return out.Data
	}

	a := get(clientA)
	// Sem o saldo da conta, sem o passivo e sem o investimento marcado como fora do patrimônio.
	if a.AssetsValue != "350000.00" || a.InvestmentsValue != "1250.50" || a.Total != "351250.50" {
		t.Fatalf("totais da composição incorretos: %+v", a)
	}
	if len(a.Slices) != 3 || a.Slices[0].Type != "imovel" || a.Slices[0].Group != "asset" {
		t.Fatalf("fatias incorretas: %+v", a.Slices)
	}
	if a.Slices[0].SharePercent != "85.4" {
		t.Fatalf("participação de imóveis deveria ser 85.4%%, obteve %s", a.Slices[0].SharePercent)
	}

	// A composição deve bater com os componentes do patrimônio líquido.
	nw := getNetWorth(t, clientA, ts.URL)
	if nw.DeclaredAssetsValue != a.AssetsValue || nw.InvestmentsValue != a.InvestmentsValue {
		t.Fatalf("composição diverge do patrimônio líquido: %+v vs %+v", a, nw)
	}

	b := get(clientB)
	if b.Total != "900000.00" || len(b.Slices) != 1 {
		t.Fatalf("composição de B deveria conter apenas os próprios bens: %+v", b)
	}
}

func TestPagesRework_NewEndpointsRequireAuthentication(t *testing.T) {
	ts, _ := newTestServer(t)
	anon := newJarClient(t)
	for _, path := range []string{
		"/api/v1/recurring-rules/summary",
		"/api/v1/transactions/annual-summary?year=2026",
		"/api/v1/net-worth/composition",
	} {
		resp := doJSON(t, anon, http.MethodGet, ts.URL+path, nil, "")
		resp.Body.Close()
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("%s sem autenticação deveria retornar 401, obteve %d", path, resp.StatusCode)
		}
	}
}
