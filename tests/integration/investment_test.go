package integration

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

type testInvestment struct {
	ID             string  `json:"id"`
	Quantity       *string `json:"quantity"`
	AveragePrice   *string `json:"average_price"`
	InvestedAmount string  `json:"invested_amount"`
	MarketValue    string  `json:"market_value"`
	UnrealizedGain string  `json:"unrealized_gain"`
}

func createInvestment(t *testing.T, client *http.Client, baseURL, csrf string, body map[string]any) testInvestment {
	t.Helper()
	resp := doJSON(t, client, http.MethodPost, baseURL+"/api/v1/investments", body, csrf)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("esperava 201 ao criar investimento, obteve %d: %s", resp.StatusCode, b)
	}
	var out struct {
		Data testInvestment `json:"data"`
	}
	json.NewDecoder(resp.Body).Decode(&out)
	return out.Data
}

func getInvestment(t *testing.T, client *http.Client, baseURL, id string) (testInvestment, int) {
	t.Helper()
	resp := doJSON(t, client, http.MethodGet, baseURL+"/api/v1/investments/"+id, nil, "")
	defer resp.Body.Close()
	var out struct {
		Data testInvestment `json:"data"`
	}
	json.NewDecoder(resp.Body).Decode(&out)
	return out.Data, resp.StatusCode
}

func TestInvestment_ContributionIncreasesPositionAndDebitsLinkedAccount(t *testing.T) {
	ts, client := newTestServer(t)
	registerUser(t, client, ts.URL, "Xavier", "xavier@example.com", "senhaForte123")
	loginUser(t, client, ts.URL, "xavier@example.com", "senhaForte123")
	csrf := mustCSRF(t, client, ts.URL)

	acc := createAccount(t, client, ts.URL, csrf, "Conta Xavier", "10000.00", "2026-01-01")

	inv := createInvestment(t, client, ts.URL, csrf, map[string]any{
		"name": "Tesouro Selic", "asset_type": "tesouro_direto",
		"invested_amount": "0.00", "market_value": "0.00", "position_date": "2026-01-01",
	})

	contribResp := doJSON(t, client, http.MethodPost, ts.URL+"/api/v1/investments/"+inv.ID+"/transactions", map[string]any{
		"account_id": acc.ID, "type": "contribution", "amount": "1000.00", "occurred_on": "2026-01-10",
	}, csrf)
	defer contribResp.Body.Close()
	if contribResp.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(contribResp.Body)
		t.Fatalf("esperava 201 no aporte, obteve %d: %s", contribResp.StatusCode, b)
	}

	updatedInv, status := getInvestment(t, client, ts.URL, inv.ID)
	if status != http.StatusOK {
		t.Fatalf("esperava 200, obteve %d", status)
	}
	if updatedInv.InvestedAmount != "1000.00" {
		t.Fatalf("valor investido incorreto: %s", updatedInv.InvestedAmount)
	}

	updatedAcc, _ := getAccount(t, client, ts.URL, acc.ID)
	// 10000 - 1000 (aporte debitado da conta) = 9000.00
	if updatedAcc.CurrentBalance != "9000.00" {
		t.Fatalf("saldo da conta deveria ter sido debitado pelo aporte: %s", updatedAcc.CurrentBalance)
	}

	// O aporte não deve aparecer como despesa no resumo mensal.
	summaryResp := doJSON(t, client, http.MethodGet, ts.URL+"/api/v1/transactions/monthly-summary?year=2026&month=1", nil, "")
	defer summaryResp.Body.Close()
	var summary struct {
		Data struct {
			TotalExpense string `json:"total_expense"`
		} `json:"data"`
	}
	json.NewDecoder(summaryResp.Body).Decode(&summary)
	if summary.Data.TotalExpense != "0.00" {
		t.Fatalf("aporte em investimento não deveria contar como despesa de consumo: %s", summary.Data.TotalExpense)
	}
}

func TestInvestment_BuyAndSellTracksWeightedAverageAndRealizedGain(t *testing.T) {
	ts, client := newTestServer(t)
	registerUser(t, client, ts.URL, "Yara", "yara@example.com", "senhaForte123")
	loginUser(t, client, ts.URL, "yara@example.com", "senhaForte123")
	csrf := mustCSRF(t, client, ts.URL)

	inv := createInvestment(t, client, ts.URL, csrf, map[string]any{
		"name": "ACME3", "asset_type": "acao", "ticker": "ACME3",
		"invested_amount": "0.00", "market_value": "0.00", "position_date": "2026-01-01",
	})

	// Compra 1: 10 cotas a R$10,00 (amount=100.00)
	buy1 := doJSON(t, client, http.MethodPost, ts.URL+"/api/v1/investments/"+inv.ID+"/transactions", map[string]any{
		"type": "contribution", "amount": "100.00", "quantity": "10", "occurred_on": "2026-02-01",
	}, csrf)
	buy1.Body.Close()

	// Compra 2: 10 cotas a R$20,00 (amount=200.00) -> preço médio deveria ir para R$15,00
	buy2 := doJSON(t, client, http.MethodPost, ts.URL+"/api/v1/investments/"+inv.ID+"/transactions", map[string]any{
		"type": "contribution", "amount": "200.00", "quantity": "10", "occurred_on": "2026-02-02",
	}, csrf)
	buy2.Body.Close()

	afterBuys, _ := getInvestment(t, client, ts.URL, inv.ID)
	if afterBuys.InvestedAmount != "300.00" {
		t.Fatalf("valor investido incorreto após compras: %s", afterBuys.InvestedAmount)
	}
	if afterBuys.AveragePrice == nil || *afterBuys.AveragePrice != "15.000000" {
		t.Fatalf("preço médio incorreto após compras: %v", afterBuys.AveragePrice)
	}

	// Venda de 5 cotas por R$100,00 (custo removido = 5*15 = 75; ganho realizado = 25)
	sellResp := doJSON(t, client, http.MethodPost, ts.URL+"/api/v1/investments/"+inv.ID+"/transactions", map[string]any{
		"type": "redemption", "amount": "100.00", "quantity": "5", "occurred_on": "2026-03-01",
	}, csrf)
	defer sellResp.Body.Close()
	if sellResp.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(sellResp.Body)
		t.Fatalf("esperava 201 na venda, obteve %d: %s", sellResp.StatusCode, b)
	}
	var sellOut struct {
		Data struct {
			RealizedGain *string `json:"realized_gain"`
		} `json:"data"`
	}
	json.NewDecoder(sellResp.Body).Decode(&sellOut)
	if sellOut.Data.RealizedGain == nil || *sellOut.Data.RealizedGain != "25.00" {
		t.Fatalf("ganho realizado incorreto na venda: %v", sellOut.Data.RealizedGain)
	}

	afterSell, _ := getInvestment(t, client, ts.URL, inv.ID)
	// invested_amount = 300 - 75 = 225.00; quantity = 15
	if afterSell.InvestedAmount != "225.00" {
		t.Fatalf("valor investido incorreto após venda: %s", afterSell.InvestedAmount)
	}
	if afterSell.Quantity == nil || *afterSell.Quantity != "15" {
		t.Fatalf("quantidade incorreta após venda: %v", afterSell.Quantity)
	}
}

func TestInvestment_RedemptionExceedingQuantityIsRejected(t *testing.T) {
	ts, client := newTestServer(t)
	registerUser(t, client, ts.URL, "Zeca", "zeca@example.com", "senhaForte123")
	loginUser(t, client, ts.URL, "zeca@example.com", "senhaForte123")
	csrf := mustCSRF(t, client, ts.URL)

	inv := createInvestment(t, client, ts.URL, csrf, map[string]any{
		"name": "FII XPTO", "asset_type": "fii", "invested_amount": "0.00", "market_value": "0.00", "position_date": "2026-01-01",
	})
	doJSON(t, client, http.MethodPost, ts.URL+"/api/v1/investments/"+inv.ID+"/transactions", map[string]any{
		"type": "contribution", "amount": "100.00", "quantity": "10", "occurred_on": "2026-01-05",
	}, csrf).Body.Close()

	resp := doJSON(t, client, http.MethodPost, ts.URL+"/api/v1/investments/"+inv.ID+"/transactions", map[string]any{
		"type": "redemption", "amount": "500.00", "quantity": "50", "occurred_on": "2026-01-10",
	}, csrf)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusConflict {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("esperava 409 ao resgatar mais do que a posição possui, obteve %d: %s", resp.StatusCode, b)
	}
}

func TestInvestment_IncomeDoesNotChangeInvestedAmountButIsRealizedGain(t *testing.T) {
	ts, client := newTestServer(t)
	registerUser(t, client, ts.URL, "Washington", "washington@example.com", "senhaForte123")
	loginUser(t, client, ts.URL, "washington@example.com", "senhaForte123")
	csrf := mustCSRF(t, client, ts.URL)

	inv := createInvestment(t, client, ts.URL, csrf, map[string]any{
		"name": "CDB Banco X", "asset_type": "cdb", "invested_amount": "1000.00", "market_value": "1000.00", "position_date": "2026-01-01",
	})

	doJSON(t, client, http.MethodPost, ts.URL+"/api/v1/investments/"+inv.ID+"/transactions", map[string]any{
		"type": "income", "amount": "50.00", "occurred_on": "2026-02-01",
	}, csrf).Body.Close()

	after, _ := getInvestment(t, client, ts.URL, inv.ID)
	if after.InvestedAmount != "1000.00" {
		t.Fatalf("rendimento não deveria alterar o valor investido: %s", after.InvestedAmount)
	}

	summaryResp := doJSON(t, client, http.MethodGet, ts.URL+"/api/v1/investments/summary", nil, "")
	defer summaryResp.Body.Close()
	var summary struct {
		Data struct {
			TotalRealizedResult string `json:"total_realized_result"`
		} `json:"data"`
	}
	json.NewDecoder(summaryResp.Body).Decode(&summary)
	if summary.Data.TotalRealizedResult != "50.00" {
		t.Fatalf("resultado realizado deveria contabilizar o rendimento recebido: %s", summary.Data.TotalRealizedResult)
	}
}

func TestInvestment_IsIncludedInNetWorthUnlessExcluded(t *testing.T) {
	ts, client := newTestServer(t)
	registerUser(t, client, ts.URL, "Yolanda", "yolanda@example.com", "senhaForte123")
	loginUser(t, client, ts.URL, "yolanda@example.com", "senhaForte123")
	csrf := mustCSRF(t, client, ts.URL)

	createInvestment(t, client, ts.URL, csrf, map[string]any{
		"name": "Fundo Imobiliário", "asset_type": "fii",
		"invested_amount": "5000.00", "market_value": "5500.00", "position_date": "2026-01-01",
	})
	createInvestment(t, client, ts.URL, csrf, map[string]any{
		"name": "Posição já contabilizada em conta", "asset_type": "outro",
		"invested_amount": "2000.00", "market_value": "2000.00", "position_date": "2026-01-01",
		"include_in_net_worth": false,
	})

	nw := getNetWorth(t, client, ts.URL)
	// Apenas a primeira posição (5500.00) deve ser somada; a segunda está
	// marcada como já representada em outro lugar (não deve duplicar).
	if nw.InvestmentsValue != "5500.00" {
		t.Fatalf("valor de investimentos no patrimônio incorreto: %s", nw.InvestmentsValue)
	}
	if nw.GrossWorth != "5500.00" {
		t.Fatalf("patrimônio bruto incorreto: %s", nw.GrossWorth)
	}
}

func TestInvestment_IsIsolatedPerUser(t *testing.T) {
	ts, clientA := newTestServer(t)
	clientB := newJarClient(t)

	registerUser(t, clientA, ts.URL, "Yago", "yago@example.com", "senhaForte123")
	registerUser(t, clientB, ts.URL, "Zara", "zara@example.com", "senhaForte123")
	loginUser(t, clientA, ts.URL, "yago@example.com", "senhaForte123")
	loginUser(t, clientB, ts.URL, "zara@example.com", "senhaForte123")
	csrfA := mustCSRF(t, clientA, ts.URL)

	inv := createInvestment(t, clientA, ts.URL, csrfA, map[string]any{
		"name": "Investimento de Yago", "asset_type": "outro",
		"invested_amount": "100.00", "market_value": "100.00", "position_date": "2026-01-01",
	})

	_, status := getInvestment(t, clientB, ts.URL, inv.ID)
	if status != http.StatusNotFound {
		t.Fatalf("esperava 404 ao acessar investimento de outro usuário, obteve %d", status)
	}
}

func TestInvestment_ContributionAndRedemptionAreNotGainOrLoss(t *testing.T) {
	ts, client := newTestServer(t)
	registerUser(t, client, ts.URL, "Gil", "gil.aporte@example.com", "senhaForte123")
	loginUser(t, client, ts.URL, "gil.aporte@example.com", "senhaForte123")
	csrf := mustCSRF(t, client, ts.URL)

	inv := createInvestment(t, client, ts.URL, csrf, map[string]any{
		"name": "CDB", "asset_type": "cdb", "invested_amount": "30000.00", "market_value": "30000.00", "position_date": "2026-01-01",
	})

	for _, mv := range []map[string]any{
		{"type": "contribution", "amount": "1000.00", "occurred_on": "2026-02-01"},
		{"type": "redemption", "amount": "500.00", "occurred_on": "2026-03-01"},
	} {
		resp := doJSON(t, client, http.MethodPost, ts.URL+"/api/v1/investments/"+inv.ID+"/transactions", mv, csrf)
		resp.Body.Close()
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("esperava 201 em %v, obteve %d", mv["type"], resp.StatusCode)
		}
		after, _ := getInvestment(t, client, ts.URL, inv.ID)
		if after.UnrealizedGain != "0.00" {
			t.Fatalf("%v não pode gerar ganho/perda: unrealized_gain=%s (investido %s, mercado %s)",
				mv["type"], after.UnrealizedGain, after.InvestedAmount, after.MarketValue)
		}
	}

	final, _ := getInvestment(t, client, ts.URL, inv.ID)
	if final.InvestedAmount != "30500.00" || final.MarketValue != "30500.00" {
		t.Fatalf("posição final incorreta: investido %s, mercado %s", final.InvestedAmount, final.MarketValue)
	}
}
