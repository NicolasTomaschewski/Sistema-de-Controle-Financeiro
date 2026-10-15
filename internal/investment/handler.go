package investment

import (
	"errors"
	"net/http"
	"time"

	"github.com/shopspring/decimal"

	"github.com/nicolastomaschewski/controle-financeiro/internal/account"
	"github.com/nicolastomaschewski/controle-financeiro/internal/authctx"
	"github.com/nicolastomaschewski/controle-financeiro/internal/httpx"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/investments", h.list)
	mux.HandleFunc("POST /api/v1/investments", h.create)
	mux.HandleFunc("GET /api/v1/investments/summary", h.summary)
	mux.HandleFunc("GET /api/v1/investments/{id}", h.get)
	mux.HandleFunc("PATCH /api/v1/investments/{id}", h.update)
	mux.HandleFunc("DELETE /api/v1/investments/{id}", h.delete)
	mux.HandleFunc("GET /api/v1/investments/{id}/transactions", h.listMovements)
	mux.HandleFunc("POST /api/v1/investments/{id}/transactions", h.recordMovement)
	mux.HandleFunc("GET /api/v1/investments/{id}/valuations", h.listValuations)
	mux.HandleFunc("POST /api/v1/investments/{id}/valuations", h.addValuation)
}

type investmentResponse struct {
	ID                string  `json:"id"`
	Name              string  `json:"name"`
	AssetType         string  `json:"asset_type"`
	Institution       *string `json:"institution"`
	Ticker            *string `json:"ticker"`
	Quantity          *string `json:"quantity"`
	AveragePrice      *string `json:"average_price"`
	InvestedAmount    string  `json:"invested_amount"`
	MarketValue       string  `json:"market_value"`
	UnrealizedGain    string  `json:"unrealized_gain"`
	SimpleReturnPct   *string `json:"simple_return_percent"`
	PositionDate      string  `json:"position_date"`
	AcquisitionDate   *string `json:"acquisition_date"`
	MaturityDate      *string `json:"maturity_date"`
	IncludeInNetWorth bool    `json:"include_in_net_worth"`
	Notes             *string `json:"notes"`
	Status            string  `json:"status"`
	IsEstimate        bool    `json:"is_estimate"`
}

func toResponse(inv *Investment) investmentResponse {
	resp := investmentResponse{
		ID: inv.ID, Name: inv.Name, AssetType: inv.AssetType, Institution: inv.Institution, Ticker: inv.Ticker,
		InvestedAmount: inv.InvestedAmount.StringFixed(2), MarketValue: inv.MarketValue.StringFixed(2),
		UnrealizedGain: inv.UnrealizedGain().StringFixed(2), PositionDate: inv.PositionDate.Format("2006-01-02"),
		IncludeInNetWorth: inv.IncludeInNetWorth, Notes: inv.Notes, Status: inv.Status,
		// Sem integração externa de mercado configurada, todo valor é uma
		// estimativa informada manualmente pelo usuário.
		IsEstimate: true,
	}
	if inv.Quantity != nil {
		q := inv.Quantity.String()
		resp.Quantity = &q
	}
	if inv.AveragePrice != nil {
		p := inv.AveragePrice.StringFixed(6)
		resp.AveragePrice = &p
	}
	if pct := inv.SimpleReturnPercent(); pct != nil {
		p := pct.StringFixed(2)
		resp.SimpleReturnPct = &p
	}
	if inv.AcquisitionDate != nil {
		d := inv.AcquisitionDate.Format("2006-01-02")
		resp.AcquisitionDate = &d
	}
	if inv.MaturityDate != nil {
		d := inv.MaturityDate.Format("2006-01-02")
		resp.MaturityDate = &d
	}
	return resp
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	userID, _ := authctx.UserID(r.Context())
	includeClosed := r.URL.Query().Get("include_closed") == "true"

	items, err := h.service.List(r.Context(), userID, includeClosed)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "Não foi possível listar os investimentos.")
		return
	}
	resp := make([]investmentResponse, 0, len(items))
	for _, inv := range items {
		resp = append(resp, toResponse(inv))
	}
	httpx.JSON(w, http.StatusOK, resp)
}

type createRequest struct {
	Name              string  `json:"name"`
	AssetType         string  `json:"asset_type"`
	Institution       *string `json:"institution"`
	Ticker            *string `json:"ticker"`
	Quantity          *string `json:"quantity"`
	AveragePrice      *string `json:"average_price"`
	InvestedAmount    string  `json:"invested_amount"`
	MarketValue       string  `json:"market_value"`
	PositionDate      string  `json:"position_date"`
	AcquisitionDate   *string `json:"acquisition_date"`
	MaturityDate      *string `json:"maturity_date"`
	IncludeInNetWorth *bool   `json:"include_in_net_worth"`
	Notes             *string `json:"notes"`
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	userID, _ := authctx.UserID(r.Context())

	var req createRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid_body", "Corpo da requisição inválido.")
		return
	}

	invested, err := decimal.NewFromString(req.InvestedAmount)
	if err != nil {
		httpx.ValidationError(w, map[string]string{"invested_amount": "Valor inválido."})
		return
	}
	marketValue, err := decimal.NewFromString(req.MarketValue)
	if err != nil {
		httpx.ValidationError(w, map[string]string{"market_value": "Valor inválido."})
		return
	}
	positionDate, err := time.Parse("2006-01-02", req.PositionDate)
	if err != nil {
		httpx.ValidationError(w, map[string]string{"position_date": "Data inválida (use AAAA-MM-DD)."})
		return
	}

	quantity, err := parseOptionalDecimal(req.Quantity)
	if err != nil {
		httpx.ValidationError(w, map[string]string{"quantity": "Quantidade inválida."})
		return
	}
	averagePrice, err := parseOptionalDecimal(req.AveragePrice)
	if err != nil {
		httpx.ValidationError(w, map[string]string{"average_price": "Preço médio inválido."})
		return
	}
	acquisitionDate, err := parseOptionalDate(req.AcquisitionDate)
	if err != nil {
		httpx.ValidationError(w, map[string]string{"acquisition_date": "Data inválida."})
		return
	}
	maturityDate, err := parseOptionalDate(req.MaturityDate)
	if err != nil {
		httpx.ValidationError(w, map[string]string{"maturity_date": "Data inválida."})
		return
	}
	includeInNetWorth := true
	if req.IncludeInNetWorth != nil {
		includeInNetWorth = *req.IncludeInNetWorth
	}

	inv, err := h.service.Create(r.Context(), userID, CreateInput{
		Name: req.Name, AssetType: req.AssetType, Institution: req.Institution, Ticker: req.Ticker,
		Quantity: quantity, AveragePrice: averagePrice, InvestedAmount: invested, MarketValue: marketValue,
		PositionDate: positionDate, AcquisitionDate: acquisitionDate, MaturityDate: maturityDate,
		IncludeInNetWorth: includeInNetWorth, Notes: req.Notes,
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, toResponse(inv))
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	userID, _ := authctx.UserID(r.Context())
	inv, err := h.service.Get(r.Context(), userID, r.PathValue("id"))
	if err != nil {
		writeDomainError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, toResponse(inv))
}

type updateRequest struct {
	Name              string  `json:"name"`
	AssetType         string  `json:"asset_type"`
	Institution       *string `json:"institution"`
	Ticker            *string `json:"ticker"`
	AcquisitionDate   *string `json:"acquisition_date"`
	MaturityDate      *string `json:"maturity_date"`
	IncludeInNetWorth bool    `json:"include_in_net_worth"`
	Notes             *string `json:"notes"`
	Status            string  `json:"status"`
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	userID, _ := authctx.UserID(r.Context())

	var req updateRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid_body", "Corpo da requisição inválido.")
		return
	}
	acquisitionDate, err := parseOptionalDate(req.AcquisitionDate)
	if err != nil {
		httpx.ValidationError(w, map[string]string{"acquisition_date": "Data inválida."})
		return
	}
	maturityDate, err := parseOptionalDate(req.MaturityDate)
	if err != nil {
		httpx.ValidationError(w, map[string]string{"maturity_date": "Data inválida."})
		return
	}

	inv, err := h.service.Update(r.Context(), userID, r.PathValue("id"), UpdateInput{
		Name: req.Name, AssetType: req.AssetType, Institution: req.Institution, Ticker: req.Ticker,
		AcquisitionDate: acquisitionDate, MaturityDate: maturityDate, IncludeInNetWorth: req.IncludeInNetWorth,
		Notes: req.Notes, Status: req.Status,
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, toResponse(inv))
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	userID, _ := authctx.UserID(r.Context())
	closed, err := h.service.Delete(r.Context(), userID, r.PathValue("id"))
	if err != nil {
		writeDomainError(w, err)
		return
	}
	if closed {
		httpx.JSON(w, http.StatusOK, map[string]string{"message": "Investimento possui movimentações e foi encerrado em vez de excluído."})
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"message": "Investimento excluído."})
}

type movementResponse struct {
	ID           string  `json:"id"`
	AccountID    *string `json:"account_id"`
	Type         string  `json:"type"`
	Amount       string  `json:"amount"`
	Quantity     *string `json:"quantity"`
	Fees         string  `json:"fees"`
	RealizedGain *string `json:"realized_gain"`
	OccurredOn   string  `json:"occurred_on"`
	Notes        *string `json:"notes"`
}

func toMovementResponse(m *Movement) movementResponse {
	resp := movementResponse{
		ID: m.ID, AccountID: m.AccountID, Type: m.Type, Amount: m.Amount.StringFixed(2),
		Fees: m.Fees.StringFixed(2), OccurredOn: m.OccurredOn.Format("2006-01-02"), Notes: m.Notes,
	}
	if m.Quantity != nil {
		q := m.Quantity.String()
		resp.Quantity = &q
	}
	if m.RealizedGain != nil {
		g := m.RealizedGain.StringFixed(2)
		resp.RealizedGain = &g
	}
	return resp
}

func (h *Handler) listMovements(w http.ResponseWriter, r *http.Request) {
	userID, _ := authctx.UserID(r.Context())
	items, err := h.service.ListMovements(r.Context(), userID, r.PathValue("id"))
	if err != nil {
		writeDomainError(w, err)
		return
	}
	resp := make([]movementResponse, 0, len(items))
	for _, m := range items {
		resp = append(resp, toMovementResponse(m))
	}
	httpx.JSON(w, http.StatusOK, resp)
}

type movementRequest struct {
	AccountID  *string `json:"account_id"`
	Type       string  `json:"type"`
	Amount     string  `json:"amount"`
	Quantity   *string `json:"quantity"`
	Fees       string  `json:"fees"`
	OccurredOn string  `json:"occurred_on"`
	Notes      *string `json:"notes"`
}

func (h *Handler) recordMovement(w http.ResponseWriter, r *http.Request) {
	userID, _ := authctx.UserID(r.Context())

	var req movementRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid_body", "Corpo da requisição inválido.")
		return
	}
	amount, err := decimal.NewFromString(req.Amount)
	if err != nil {
		httpx.ValidationError(w, map[string]string{"amount": "Valor inválido."})
		return
	}
	fees := decimal.Zero
	if req.Fees != "" {
		fees, err = decimal.NewFromString(req.Fees)
		if err != nil {
			httpx.ValidationError(w, map[string]string{"fees": "Valor de taxa inválido."})
			return
		}
	}
	quantity, err := parseOptionalDecimal(req.Quantity)
	if err != nil {
		httpx.ValidationError(w, map[string]string{"quantity": "Quantidade inválida."})
		return
	}
	occurredOn, err := time.Parse("2006-01-02", req.OccurredOn)
	if err != nil {
		httpx.ValidationError(w, map[string]string{"occurred_on": "Data inválida (use AAAA-MM-DD)."})
		return
	}

	m, err := h.service.RecordMovement(r.Context(), userID, r.PathValue("id"), MovementInput{
		AccountID: req.AccountID, Type: req.Type, Amount: amount, Quantity: quantity,
		Fees: fees, OccurredOn: occurredOn, Notes: req.Notes,
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, toMovementResponse(m))
}

type valuationResponse struct {
	ID           string  `json:"id"`
	MarketValue  string  `json:"market_value"`
	PositionDate string  `json:"position_date"`
	Notes        *string `json:"notes"`
}

func toValuationResponse(v *Valuation) valuationResponse {
	return valuationResponse{ID: v.ID, MarketValue: v.MarketValue.StringFixed(2), PositionDate: v.PositionDate.Format("2006-01-02"), Notes: v.Notes}
}

func (h *Handler) listValuations(w http.ResponseWriter, r *http.Request) {
	userID, _ := authctx.UserID(r.Context())
	items, err := h.service.ListValuations(r.Context(), userID, r.PathValue("id"))
	if err != nil {
		writeDomainError(w, err)
		return
	}
	resp := make([]valuationResponse, 0, len(items))
	for _, v := range items {
		resp = append(resp, toValuationResponse(v))
	}
	httpx.JSON(w, http.StatusOK, resp)
}

type addValuationRequest struct {
	MarketValue  string  `json:"market_value"`
	PositionDate string  `json:"position_date"`
	Notes        *string `json:"notes"`
}

func (h *Handler) addValuation(w http.ResponseWriter, r *http.Request) {
	userID, _ := authctx.UserID(r.Context())

	var req addValuationRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid_body", "Corpo da requisição inválido.")
		return
	}
	value, err := decimal.NewFromString(req.MarketValue)
	if err != nil {
		httpx.ValidationError(w, map[string]string{"market_value": "Valor inválido."})
		return
	}
	date, err := time.Parse("2006-01-02", req.PositionDate)
	if err != nil {
		httpx.ValidationError(w, map[string]string{"position_date": "Data inválida (use AAAA-MM-DD)."})
		return
	}

	v, err := h.service.AddValuation(r.Context(), userID, r.PathValue("id"), value, date, req.Notes)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, toValuationResponse(v))
}

type summaryResponse struct {
	TotalInvested       string            `json:"total_invested"`
	TotalMarketValue    string            `json:"total_market_value"`
	UnrealizedGain      string            `json:"unrealized_gain"`
	TotalRealizedResult string            `json:"total_realized_result"`
	ByAssetType         map[string]string `json:"by_asset_type"`
}

func (h *Handler) summary(w http.ResponseWriter, r *http.Request) {
	userID, _ := authctx.UserID(r.Context())
	s, err := h.service.Summary(r.Context(), userID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "Não foi possível calcular o resumo da carteira.")
		return
	}
	byType := make(map[string]string, len(s.ByAssetType))
	for k, v := range s.ByAssetType {
		byType[k] = v.StringFixed(2)
	}
	httpx.JSON(w, http.StatusOK, summaryResponse{
		TotalInvested: s.TotalInvested.StringFixed(2), TotalMarketValue: s.TotalMarketValue.StringFixed(2),
		UnrealizedGain: s.UnrealizedGain.StringFixed(2), TotalRealizedResult: s.TotalRealizedResult.StringFixed(2),
		ByAssetType: byType,
	})
}

func parseOptionalDecimal(s *string) (*decimal.Decimal, error) {
	if s == nil || *s == "" {
		return nil, nil
	}
	d, err := decimal.NewFromString(*s)
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func parseOptionalDate(s *string) (*time.Time, error) {
	if s == nil || *s == "" {
		return nil, nil
	}
	d, err := time.Parse("2006-01-02", *s)
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func writeDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrNotFound), errors.Is(err, ErrMovementNotFound):
		httpx.NotFoundOrForbidden(w)
	case errors.Is(err, ErrInvalidName):
		httpx.ValidationError(w, map[string]string{"name": "Nome inválido."})
	case errors.Is(err, ErrInvalidAssetType):
		httpx.ValidationError(w, map[string]string{"asset_type": "Tipo de ativo inválido."})
	case errors.Is(err, ErrInvalidAmount):
		httpx.ValidationError(w, map[string]string{"amount": "Valor inválido."})
	case errors.Is(err, ErrInvalidQuantity):
		httpx.ValidationError(w, map[string]string{"quantity": "Quantidade inválida."})
	case errors.Is(err, ErrInvalidMovementType):
		httpx.ValidationError(w, map[string]string{"type": "Tipo de movimentação inválido."})
	case errors.Is(err, account.ErrNotFound), errors.Is(err, ErrAccountNotOwned):
		httpx.ValidationError(w, map[string]string{"account_id": "Conta não encontrada ou não pertence ao usuário."})
	case errors.Is(err, ErrInsufficientQuantity):
		httpx.Error(w, http.StatusConflict, "insufficient_quantity", "Quantidade insuficiente na posição para este resgate.")
	case errors.Is(err, ErrValuationExists):
		httpx.Error(w, http.StatusConflict, "valuation_exists", "Já existe um valor de mercado registrado para esta data.")
	default:
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "Não foi possível concluir a operação.")
	}
}
