// Command demoseed preenche a conta de demonstração com um ano de dados
// fictícios (receitas, custos fixos, despesas, orçamentos, bens e
// investimentos) para apresentar o sistema. Uso exclusivo em desenvolvimento,
// chamado pelo dev.sh.
//
// Tudo é gravado pela API HTTP, como faria a interface: as mesmas validações,
// regras financeiras e isolamento por usuário se aplicam. Nada é escrito
// direto no banco. É idempotente: se a conta já tiver lançamentos, não faz
// nada.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"time"
)

func main() {
	apiURL := flag.String("api", "http://localhost:8080", "URL base da API")
	email := flag.String("email", "demo@finctrl.local", "e-mail da conta de demonstração")
	password := flag.String("password", "demo12345", "senha da conta de demonstração")
	flag.Parse()

	c, err := newClient(*apiURL)
	if err != nil {
		log.Fatal(err)
	}
	if err := c.login(*email, *password); err != nil {
		log.Fatalf("demoseed: login: %v", err)
	}

	empty, err := c.isEmpty()
	if err != nil {
		log.Fatalf("demoseed: %v", err)
	}
	if !empty {
		fmt.Println("dados de demonstração já existem; nada a fazer")
		return
	}

	s := &seeder{c: c, today: today(), rnd: rand.New(rand.NewSource(42))}
	if err := s.run(); err != nil {
		log.Fatalf("demoseed: %v", err)
	}
	fmt.Println("dados de demonstração criados")
}

// ---------------------------------------------------------------- cliente HTTP

type client struct {
	base string
	http *http.Client
}

func newClient(base string) (*client, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}
	return &client{base: base, http: &http.Client{Jar: jar, Timeout: 10 * time.Second}}, nil
}

func (c *client) csrf() string {
	u, _ := url.Parse(c.base)
	for _, ck := range c.http.Jar.Cookies(u) {
		if ck.Name == "csrf_token" {
			return ck.Value
		}
	}
	return ""
}

// do envia a requisição e decodifica o campo "data" da resposta em out.
func (c *client) do(method, path string, body, out any) error {
	var r io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		r = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, c.base+path, r)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if tok := c.csrf(); tok != "" {
		req.Header.Set("X-CSRF-Token", tok)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return fmt.Errorf("%s %s: HTTP %d: %s", method, path, resp.StatusCode, raw)
	}
	if out == nil {
		return nil
	}
	var env struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		return fmt.Errorf("%s %s: resposta inválida: %w", method, path, err)
	}
	return json.Unmarshal(env.Data, out)
}

func (c *client) login(email, password string) error {
	return c.do(http.MethodPost, "/api/v1/auth/login", map[string]string{"email": email, "password": password}, nil)
}

// isEmpty considera a conta vazia enquanto ela não tiver lançamentos: é o
// núcleo do que o gerador cria. Bens ou investimentos cadastrados à mão não
// impedem a geração e são preservados.
func (c *client) isEmpty() (bool, error) {
	var tx []json.RawMessage
	if err := c.do(http.MethodGet, "/api/v1/transactions?limit=1", nil, &tx); err != nil {
		return false, err
	}
	return len(tx) == 0, nil
}

type idOnly struct {
	ID string `json:"id"`
}

func (c *client) create(path string, body any) (string, error) {
	var out idOnly
	if err := c.do(http.MethodPost, path, body, &out); err != nil {
		return "", err
	}
	return out.ID, nil
}

// --------------------------------------------------------------- dados fictícios

// Os valores monetários são montados em centavos inteiros e formatados como
// string decimal: nada de float para dinheiro, como no restante do sistema.
func brl(cents int64) string { return fmt.Sprintf("%d.%02d", cents/100, cents%100) }

func today() time.Time {
	now := time.Now()
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
}

func iso(t time.Time) string { return t.Format("2006-01-02") }

// day devolve o dia d do mês, limitado ao último dia do mês.
func day(month time.Time, d int) time.Time {
	last := time.Date(month.Year(), month.Month()+1, 0, 0, 0, 0, 0, time.UTC).Day()
	if d > last {
		d = last
	}
	return time.Date(month.Year(), month.Month(), d, 0, 0, 0, 0, time.UTC)
}

type seeder struct {
	c          *client
	today      time.Time
	rnd        *rand.Rand
	categories map[string]string
	accountID  string
}

// months devolve o primeiro dia de cada um dos últimos 12 meses, do mais
// antigo ao atual.
func (s *seeder) months() []time.Time {
	cur := time.Date(s.today.Year(), s.today.Month(), 1, 0, 0, 0, 0, time.UTC)
	out := make([]time.Time, 0, 12)
	for i := 11; i >= 0; i-- {
		out = append(out, cur.AddDate(0, -i, 0))
	}
	return out
}

// vary devolve base ± pct% em centavos, de forma determinística.
func (s *seeder) vary(base int64, pct int) int64 {
	delta := base * int64(s.rnd.Intn(2*pct+1)-pct) / 100
	return base + delta
}

func (s *seeder) run() error {
	steps := []struct {
		name string
		fn   func() error
	}{
		{"categorias", s.loadCategories},
		{"conta", s.createAccount},
		{"receitas", s.seedIncome},
		{"custos fixos", s.seedFixedCosts},
		{"despesas do dia a dia", s.seedVariableExpenses},
		{"orçamentos", s.seedBudgets},
		{"bens", s.seedAssets},
		{"investimentos", s.seedInvestments},
	}
	for _, st := range steps {
		if err := st.fn(); err != nil {
			return fmt.Errorf("%s: %w", st.name, err)
		}
	}
	return nil
}

func (s *seeder) loadCategories() error {
	var cats []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
		Kind string `json:"kind"`
	}
	if err := s.c.do(http.MethodGet, "/api/v1/categories", nil, &cats); err != nil {
		return err
	}
	s.categories = make(map[string]string, len(cats))
	for _, c := range cats {
		s.categories[c.Kind+":"+c.Name] = c.ID
	}
	return nil
}

func (s *seeder) cat(kind, name string) (string, error) {
	id, ok := s.categories[kind+":"+name]
	if !ok {
		return "", fmt.Errorf("categoria do sistema %q não encontrada", name)
	}
	return id, nil
}

// createAccount cria a conta principal com saldo inicial no começo do
// histórico. Por ser a mais antiga, ela é a conta padrão dos lançamentos.
func (s *seeder) createAccount() error {
	id, err := s.c.create("/api/v1/accounts", map[string]any{
		"name": "Conta corrente", "type": "conta_corrente", "currency": "BRL",
		"initial_balance": brl(4_200_00), "initial_balance_date": iso(s.months()[0]),
	})
	s.accountID = id
	return err
}

func (s *seeder) transaction(kind, category, description string, amount int64, on time.Time, status string) (string, error) {
	catID, err := s.cat(kind, category)
	if err != nil {
		return "", err
	}
	return s.c.create("/api/v1/transactions", map[string]any{
		"account_id": s.accountID, "category_id": catID, "type": kind, "status": status,
		"amount": brl(amount), "description": description, "occurred_on": iso(on),
	})
}

// statusFor: o que já passou foi pago/recebido; o que ainda vai acontecer é
// previsto. Assim nenhuma despesa futura aparece como paga.
func (s *seeder) statusFor(on time.Time) string {
	if on.After(s.today) {
		return "planned"
	}
	return "paid"
}

func (s *seeder) seedIncome() error {
	months := s.months()
	for i, m := range months {
		salary := day(m, 5)
		if _, err := s.transaction("income", "Salário", "Salário", 7_850_00, salary, s.statusFor(salary)); err != nil {
			return err
		}
		if m.Month() == time.December {
			if _, err := s.transaction("income", "Salário", "13º salário", 7_850_00, day(m, 20), s.statusFor(day(m, 20))); err != nil {
				return err
			}
		}
		// Aluguel da sala comercial, recebido a partir da metade do período.
		if i >= 6 {
			on := day(m, 10)
			if _, err := s.transaction("income", "Aluguel recebido", "Aluguel da sala comercial", 1_150_00, on, s.statusFor(on)); err != nil {
				return err
			}
		}
		// Trabalhos avulsos em alguns meses.
		if i%3 == 1 {
			on := day(m, 18)
			if !on.After(s.today) {
				if _, err := s.transaction("income", "Trabalho autônomo", "Projeto freelance", s.vary(1_800_00, 30), on, "paid"); err != nil {
					return err
				}
			}
		}
		// Dividendos trimestrais.
		if m.Month()%3 == 0 {
			on := day(m, 28)
			if !on.After(s.today) {
				if _, err := s.transaction("income", "Rendimentos", "Dividendos", s.vary(310_00, 20), on, "paid"); err != nil {
					return err
				}
			}
		}
	}
	cur := months[len(months)-1]
	_, err := s.transaction("income", "Outras receitas", "Reembolso do plano de saúde", 280_00, day(cur, 26), "pending")
	return err
}

type fixedCost struct {
	description string
	category    string
	cents       int64
	dueDay      int
	dueMonth    time.Month // zero = mensal
	variesPct   int        // variação do valor efetivamente pago (contas de consumo)
}

var fixedCosts = []fixedCost{
	{"Aluguel", "Moradia", 2_150_00, 5, 0, 0},
	{"Condomínio", "Moradia", 590_00, 10, 0, 0},
	{"Energia elétrica", "Água e energia", 235_00, 15, 0, 18},
	{"Água e esgoto", "Água e energia", 98_00, 15, 0, 12},
	{"Internet e celular", "Internet e telefone", 179_90, 20, 0, 0},
	{"Streaming", "Assinaturas", 55_90, 8, 0, 0},
	{"Academia", "Saúde", 119_90, 12, 0, 0},
	{"Curso de inglês", "Educação", 380_00, 25, 0, 0},
	{"IPVA", "Impostos", 1_980_00, 20, time.February, 0},
	{"IPTU", "Impostos", 1_650_00, 10, time.March, 0},
	{"Seguro do carro", "Transporte", 2_890_00, 15, time.September, 0},
}

type generated struct {
	TransactionID string `json:"transaction_id"`
	OccurredOn    string `json:"occurred_on"`
}

func (s *seeder) seedFixedCosts() error {
	months := s.months()
	start := months[0]
	for _, fc := range fixedCosts {
		catID, err := s.cat("expense", fc.category)
		if err != nil {
			return err
		}
		rule := map[string]any{
			"account_id": s.accountID, "category_id": catID, "description": fc.description,
			"planned_amount": brl(fc.cents), "periodicity": "monthly", "start_date": iso(start), "due_day": fc.dueDay,
		}
		if fc.dueMonth != 0 {
			rule["periodicity"] = "yearly"
			rule["due_month"] = int(fc.dueMonth)
		}
		ruleID, err := s.c.create("/api/v1/recurring-rules", rule)
		if err != nil {
			return err
		}

		for _, m := range months {
			if fc.dueMonth != 0 && m.Month() != fc.dueMonth {
				continue
			}
			var g generated
			path := fmt.Sprintf("/api/v1/recurring-rules/%s/generate?year=%d&month=%d", ruleID, m.Year(), int(m.Month()))
			if err := s.c.do(http.MethodPost, path, nil, &g); err != nil {
				return err
			}
			due, err := time.Parse("2006-01-02", g.OccurredOn)
			if err != nil {
				return err
			}
			// Ocorrências vencidas viram pagas, com o valor real da conta;
			// a internet do mês atual fica pendente para ilustrar esse estado.
			if due.After(s.today) || (fc.description == "Internet e celular" && m.Equal(months[len(months)-1])) {
				continue
			}
			amount := fc.cents
			if fc.variesPct > 0 {
				amount = s.vary(fc.cents, fc.variesPct)
			}
			err = s.c.do(http.MethodPatch, "/api/v1/transactions/"+g.TransactionID, map[string]any{
				"account_id": s.accountID, "category_id": catID, "status": "paid",
				"amount": brl(amount), "description": fc.description, "occurred_on": g.OccurredOn,
			}, nil)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

type variableExpense struct {
	description string
	category    string
	cents       int64
	days        []int
	variesPct   int
}

var variableExpenses = []variableExpense{
	{"Supermercado", "Alimentação", 520_00, []int{3, 12, 21, 28}, 25},
	{"Padaria e feira", "Alimentação", 85_00, []int{7, 17, 26}, 30},
	{"Restaurante", "Lazer", 140_00, []int{9, 23}, 40},
	{"Combustível", "Transporte", 260_00, []int{6, 20}, 15},
	{"Aplicativo de transporte", "Transporte", 45_00, []int{14}, 40},
	{"Farmácia", "Saúde", 95_00, []int{16}, 50},
	{"Cinema e passeios", "Lazer", 120_00, []int{27}, 40},
}

func (s *seeder) seedVariableExpenses() error {
	months := s.months()
	for i, m := range months {
		for _, ve := range variableExpenses {
			for _, d := range ve.days {
				on := day(m, d)
				if on.After(s.today) {
					continue
				}
				if _, err := s.transaction("expense", ve.category, ve.description, s.vary(ve.cents, ve.variesPct), on, "paid"); err != nil {
					return err
				}
			}
		}
		// Compras pontuais em alguns meses.
		if i%4 == 2 {
			on := day(m, 11)
			if !on.After(s.today) {
				if _, err := s.transaction("expense", "Compras pessoais", "Roupas", s.vary(380_00, 30), on, "paid"); err != nil {
					return err
				}
			}
		}
		if i == 4 {
			if _, err := s.transaction("expense", "Saúde", "Consulta odontológica", 450_00, day(m, 19), "paid"); err != nil {
				return err
			}
		}
	}
	cur := months[len(months)-1]
	// Gasto no dia 1 do mês atual que estoura o orçamento de Lazer, para que a
	// tela mostre uma categoria acima do limite em qualquer dia do mês.
	if _, err := s.transaction("expense", "Lazer", "Ingressos para show", 260_00, cur, "paid"); err != nil {
		return err
	}
	if _, err := s.transaction("expense", "Compras pessoais", "Presente de aniversário", 250_00, day(cur, 29), "planned"); err != nil {
		return err
	}
	return nil
}

func (s *seeder) seedBudgets() error {
	budgets := []struct {
		category string
		cents    int64
	}{
		{"Alimentação", 2_300_00}, {"Lazer", 400_00}, {"Transporte", 650_00}, {"Saúde", 350_00},
	}
	months := s.months()
	for _, m := range months[len(months)-2:] {
		for _, b := range budgets {
			catID, err := s.cat("expense", b.category)
			if err != nil {
				return err
			}
			if _, err := s.c.create("/api/v1/budgets", map[string]any{
				"category_id": catID, "year": m.Year(), "month": int(m.Month()), "planned_amount": brl(b.cents),
			}); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *seeder) seedAssets() error {
	months := s.months()
	assets := []struct {
		name, kind string
		history    []int64 // avaliações: há 12 meses, há 6 meses e hoje
	}{
		{"Apartamento (residência)", "imovel", []int64{430_000_00, 450_000_00, 470_000_00}},
		{"Sala comercial", "imovel", []int64{195_000_00, 198_000_00, 205_000_00}},
		{"Carro", "veiculo", []int64{84_000_00, 79_500_00, 74_000_00}},
	}
	dates := []time.Time{months[0], months[6], s.today}
	for _, a := range assets {
		id, err := s.c.create("/api/v1/assets", map[string]any{
			"name": a.name, "type": a.kind, "current_value": brl(a.history[0]), "valuation_date": iso(dates[0]),
			"notes": "Valor estimado pelo usuário (dados de demonstração).",
		})
		if err != nil {
			return err
		}
		for i := 1; i < len(a.history); i++ {
			if err := s.c.do(http.MethodPost, "/api/v1/assets/"+id+"/valuations", map[string]any{
				"value": brl(a.history[i]), "valuation_date": iso(dates[i]),
			}, nil); err != nil {
				return err
			}
		}
	}
	return nil
}

type movement struct {
	kind  string // contribution | redemption | income | fee
	cents int64
	month int // índice em months()
	day   int
}

func (s *seeder) seedInvestments() error {
	months := s.months()
	investments := []struct {
		name, kind, institution string
		initial, finalMarket    int64
		movements               []movement
	}{
		{"Tesouro Selic 2029", "tesouro_direto", "Tesouro Nacional", 18_000_00, 0, monthly("contribution", 1_000_00, 1, 11, 7)},
		{"CDB 110% do CDI", "cdb", "Banco Digital", 15_000_00, 16_480_00, nil},
		{"Carteira de ações", "acao", "Corretora XP", 22_000_00, 0, []movement{{"contribution", 3_000_00, 4, 15}, {"redemption", 2_000_00, 9, 15}}},
		{"Fundos imobiliários", "fii", "Corretora XP", 12_000_00, 11_640_00, monthly("income", 96_00, 1, 11, 15)},
		{"Bitcoin", "cripto", "Exchange", 4_000_00, 5_870_00, []movement{{"fee", 15_00, 3, 2}}},
		{"Previdência privada", "previdencia", "Seguradora", 28_000_00, 0, monthly("contribution", 500_00, 1, 11, 10)},
	}
	for _, inv := range investments {
		id, err := s.c.create("/api/v1/investments", map[string]any{
			"name": inv.name, "asset_type": inv.kind, "institution": inv.institution,
			"invested_amount": brl(inv.initial), "market_value": brl(inv.initial),
			"position_date": iso(months[0]), "acquisition_date": iso(months[0]),
			"include_in_net_worth": true,
		})
		if err != nil {
			return err
		}
		market := inv.initial
		for _, mv := range inv.movements {
			on := day(months[mv.month], mv.day)
			if on.After(s.today) {
				continue
			}
			if err := s.c.do(http.MethodPost, "/api/v1/investments/"+id+"/transactions", map[string]any{
				"type": mv.kind, "amount": brl(mv.cents), "fees": "0.00", "occurred_on": iso(on),
			}, nil); err != nil {
				return err
			}
			switch mv.kind {
			case "contribution":
				market += mv.cents
			case "redemption":
				market -= mv.cents
			}
		}
		// Avaliação atual informada pelo usuário: é ela, e não os aportes, que
		// gera ganho ou perda não realizado.
		final := inv.finalMarket
		if final == 0 {
			final = market + market*6/100
		}
		if err := s.c.do(http.MethodPost, "/api/v1/investments/"+id+"/valuations", map[string]any{
			"market_value": brl(final), "position_date": iso(s.today),
			"notes": "Valor informado pelo usuário (dados de demonstração).",
		}, nil); err != nil {
			return err
		}
	}
	return nil
}

func monthly(kind string, cents int64, from, to, d int) []movement {
	out := make([]movement, 0, to-from+1)
	for i := from; i <= to; i++ {
		out = append(out, movement{kind, cents, i, d})
	}
	return out
}
