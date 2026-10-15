package integration

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/cookiejar"
	"testing"
	"time"
)

type testNetWorth struct {
	AccountsBalance     string `json:"accounts_balance"`
	DeclaredAssetsValue string `json:"declared_assets_value"`
	InvestmentsValue    string `json:"investments_value"`
	GrossWorth          string `json:"gross_worth"`
	TotalLiabilities    string `json:"total_liabilities"`
	NetWorth            string `json:"net_worth"`
}

func getNetWorth(t *testing.T, client *http.Client, baseURL string) testNetWorth {
	t.Helper()
	resp := doJSON(t, client, http.MethodGet, baseURL+"/api/v1/net-worth", nil, "")
	defer resp.Body.Close()
	var out struct {
		Data testNetWorth `json:"data"`
	}
	json.NewDecoder(resp.Body).Decode(&out)
	return out.Data
}

func TestNetWorth_AggregatesAccountsAssetsAndLiabilities(t *testing.T) {
	ts, client := newTestServer(t)
	registerUser(t, client, ts.URL, "Sergio", "sergio@example.com", "senhaForte123")
	loginUser(t, client, ts.URL, "sergio@example.com", "senhaForte123")
	csrf := mustCSRF(t, client, ts.URL)

	createAccount(t, client, ts.URL, csrf, "Conta Sergio", "10000.00", "2026-01-01")

	assetResp := doJSON(t, client, http.MethodPost, ts.URL+"/api/v1/assets", map[string]any{
		"name": "Apartamento", "type": "imovel", "current_value": "300000.00", "valuation_date": "2026-01-01",
	}, csrf)
	defer assetResp.Body.Close()
	if assetResp.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(assetResp.Body)
		t.Fatalf("esperava 201 ao criar ativo, obteve %d: %s", assetResp.StatusCode, b)
	}

	liabilityResp := doJSON(t, client, http.MethodPost, ts.URL+"/api/v1/liabilities", map[string]any{
		"name": "Financiamento do apartamento", "type": "financiamento",
		"principal_amount": "250000.00", "current_balance": "200000.00", "reference_date": "2026-01-01",
	}, csrf)
	defer liabilityResp.Body.Close()
	if liabilityResp.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(liabilityResp.Body)
		t.Fatalf("esperava 201 ao criar passivo, obteve %d: %s", liabilityResp.StatusCode, b)
	}

	nw := getNetWorth(t, client, ts.URL)
	if nw.AccountsBalance != "10000.00" {
		t.Fatalf("saldo de contas incorreto: %s", nw.AccountsBalance)
	}
	if nw.DeclaredAssetsValue != "300000.00" {
		t.Fatalf("valor de ativos incorreto: %s", nw.DeclaredAssetsValue)
	}
	// Bruto = 10000 (contas) + 300000 (ativo) = 310000.00
	if nw.GrossWorth != "310000.00" {
		t.Fatalf("patrimônio bruto incorreto: %s", nw.GrossWorth)
	}
	if nw.TotalLiabilities != "200000.00" {
		t.Fatalf("total de passivos incorreto: %s", nw.TotalLiabilities)
	}
	// Líquido = 310000 - 200000 = 110000.00
	if nw.NetWorth != "110000.00" {
		t.Fatalf("patrimônio líquido incorreto: %s", nw.NetWorth)
	}
}

func TestAsset_ValuationHistoryIsPreservedAfterUpdate(t *testing.T) {
	ts, client := newTestServer(t)
	registerUser(t, client, ts.URL, "Tatiana", "tatiana@example.com", "senhaForte123")
	loginUser(t, client, ts.URL, "tatiana@example.com", "senhaForte123")
	csrf := mustCSRF(t, client, ts.URL)

	createResp := doJSON(t, client, http.MethodPost, ts.URL+"/api/v1/assets", map[string]any{
		"name": "Carro", "type": "veiculo", "current_value": "80000.00", "valuation_date": "2026-01-01",
	}, csrf)
	defer createResp.Body.Close()
	var created struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	json.NewDecoder(createResp.Body).Decode(&created)
	assetID := created.Data.ID

	// Nova avaliação (depreciação do veículo).
	valResp := doJSON(t, client, http.MethodPost, ts.URL+"/api/v1/assets/"+assetID+"/valuations", map[string]any{
		"value": "70000.00", "valuation_date": "2026-06-01",
	}, csrf)
	defer valResp.Body.Close()
	if valResp.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(valResp.Body)
		t.Fatalf("esperava 201 ao registrar avaliação, obteve %d: %s", valResp.StatusCode, b)
	}

	// O histórico deve conter as DUAS avaliações (a inicial e a nova).
	histResp := doJSON(t, client, http.MethodGet, ts.URL+"/api/v1/assets/"+assetID+"/valuations", nil, "")
	defer histResp.Body.Close()
	var hist struct {
		Data []struct {
			Value string `json:"value"`
		} `json:"data"`
	}
	json.NewDecoder(histResp.Body).Decode(&hist)
	if len(hist.Data) != 2 {
		t.Fatalf("esperava 2 avaliações no histórico, obteve %d", len(hist.Data))
	}

	// O valor atual do ativo deve refletir a avaliação mais recente.
	getResp := doJSON(t, client, http.MethodGet, ts.URL+"/api/v1/assets/"+assetID, nil, "")
	defer getResp.Body.Close()
	var out struct {
		Data struct {
			CurrentValue string `json:"current_value"`
		} `json:"data"`
	}
	json.NewDecoder(getResp.Body).Decode(&out)
	if out.Data.CurrentValue != "70000.00" {
		t.Fatalf("valor atual do ativo deveria refletir a avaliação mais recente: %s", out.Data.CurrentValue)
	}
}

func TestLiability_SettlingToZeroMarksAsSettledAndPreservesHistory(t *testing.T) {
	ts, client := newTestServer(t)
	registerUser(t, client, ts.URL, "Ulisses", "ulisses@example.com", "senhaForte123")
	loginUser(t, client, ts.URL, "ulisses@example.com", "senhaForte123")
	csrf := mustCSRF(t, client, ts.URL)

	createResp := doJSON(t, client, http.MethodPost, ts.URL+"/api/v1/liabilities", map[string]any{
		"name": "Empréstimo pessoal", "type": "emprestimo",
		"principal_amount": "5000.00", "current_balance": "5000.00", "reference_date": "2026-01-01",
	}, csrf)
	defer createResp.Body.Close()
	var created struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	json.NewDecoder(createResp.Body).Decode(&created)
	liabilityID := created.Data.ID

	// Baixa total do saldo devedor.
	settleResp := doJSON(t, client, http.MethodPost, ts.URL+"/api/v1/liabilities/"+liabilityID+"/valuations", map[string]any{
		"balance": "0.00", "reference_date": "2026-12-01",
	}, csrf)
	defer settleResp.Body.Close()
	if settleResp.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(settleResp.Body)
		t.Fatalf("esperava 201 na baixa, obteve %d: %s", settleResp.StatusCode, b)
	}

	getResp := doJSON(t, client, http.MethodGet, ts.URL+"/api/v1/liabilities/"+liabilityID, nil, "")
	defer getResp.Body.Close()
	var out struct {
		Data struct {
			Status         string `json:"status"`
			CurrentBalance string `json:"current_balance"`
		} `json:"data"`
	}
	json.NewDecoder(getResp.Body).Decode(&out)
	if out.Data.Status != "settled" {
		t.Fatalf("esperava status settled após baixa total, obteve %s", out.Data.Status)
	}
	if out.Data.CurrentBalance != "0.00" {
		t.Fatalf("saldo devedor deveria ser 0.00, obteve %s", out.Data.CurrentBalance)
	}

	// Passivo liquidado não deve entrar no total de passivos do patrimônio.
	nw := getNetWorth(t, client, ts.URL)
	if nw.TotalLiabilities != "0.00" {
		t.Fatalf("passivo liquidado não deveria contar no total de passivos: %s", nw.TotalLiabilities)
	}

	// O histórico de baixas permanece acessível.
	histResp := doJSON(t, client, http.MethodGet, ts.URL+"/api/v1/liabilities/"+liabilityID+"/valuations", nil, "")
	defer histResp.Body.Close()
	var hist struct {
		Data []struct {
			Balance string `json:"balance"`
		} `json:"data"`
	}
	json.NewDecoder(histResp.Body).Decode(&hist)
	if len(hist.Data) != 2 {
		t.Fatalf("esperava 2 registros no histórico de saldo, obteve %d", len(hist.Data))
	}
}

func TestAssetAndLiability_AreIsolatedPerUser(t *testing.T) {
	ts, clientA := newTestServer(t)
	jarB, _ := cookiejar.New(nil)
	clientB := &http.Client{Jar: jarB, Timeout: 5 * time.Second}

	registerUser(t, clientA, ts.URL, "Vitoria", "vitoria@example.com", "senhaForte123")
	registerUser(t, clientB, ts.URL, "Wagner", "wagner@example.com", "senhaForte123")
	loginUser(t, clientA, ts.URL, "vitoria@example.com", "senhaForte123")
	loginUser(t, clientB, ts.URL, "wagner@example.com", "senhaForte123")
	csrfA := mustCSRF(t, clientA, ts.URL)

	assetResp := doJSON(t, clientA, http.MethodPost, ts.URL+"/api/v1/assets", map[string]any{
		"name": "Ativo de Vitoria", "type": "outro", "current_value": "1000.00", "valuation_date": "2026-01-01",
	}, csrfA)
	defer assetResp.Body.Close()
	var created struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	json.NewDecoder(assetResp.Body).Decode(&created)

	getResp := doJSON(t, clientB, http.MethodGet, ts.URL+"/api/v1/assets/"+created.Data.ID, nil, "")
	defer getResp.Body.Close()
	if getResp.StatusCode != http.StatusNotFound {
		t.Fatalf("esperava 404 ao acessar ativo de outro usuário, obteve %d", getResp.StatusCode)
	}

	nwB := getNetWorth(t, clientB, ts.URL)
	if nwB.DeclaredAssetsValue != "0.00" {
		t.Fatalf("patrimônio de Wagner não deveria incluir ativos de Vitoria: %s", nwB.DeclaredAssetsValue)
	}
}
