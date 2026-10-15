package investment

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/shopspring/decimal"

	"github.com/nicolastomaschewski/controle-financeiro/internal/account"
	"github.com/nicolastomaschewski/controle-financeiro/internal/audit"
)

type Service struct {
	repo     *Repository
	accounts *account.Repository
	audit    *audit.Recorder
}

func NewService(repo *Repository, accounts *account.Repository, audit *audit.Recorder) *Service {
	return &Service{repo: repo, accounts: accounts, audit: audit}
}

type CreateInput struct {
	Name              string
	AssetType         string
	Institution       *string
	Ticker            *string
	Quantity          *decimal.Decimal
	AveragePrice      *decimal.Decimal
	InvestedAmount    decimal.Decimal
	MarketValue       decimal.Decimal
	PositionDate      time.Time
	AcquisitionDate   *time.Time
	MaturityDate      *time.Time
	IncludeInNetWorth bool
	Notes             *string
}

func (s *Service) Create(ctx context.Context, userID string, in CreateInput) (*Investment, error) {
	if err := ValidateName(in.Name); err != nil {
		return nil, err
	}
	if err := ValidateAssetType(in.AssetType); err != nil {
		return nil, err
	}
	if in.InvestedAmount.IsNegative() || in.MarketValue.IsNegative() {
		return nil, ErrInvalidAmount
	}

	inv := &Investment{
		UserID: userID, Name: in.Name, AssetType: in.AssetType, Institution: in.Institution, Ticker: in.Ticker,
		Quantity: in.Quantity, AveragePrice: in.AveragePrice, InvestedAmount: in.InvestedAmount, MarketValue: in.MarketValue,
		PositionDate: in.PositionDate, AcquisitionDate: in.AcquisitionDate, MaturityDate: in.MaturityDate,
		IncludeInNetWorth: in.IncludeInNetWorth, Notes: in.Notes, Status: StatusActive,
	}
	if err := s.repo.Create(ctx, inv); err != nil {
		return nil, fmt.Errorf("criar investimento: %w", err)
	}
	s.audit.Record(ctx, &userID, "investment.created", "investment", inv.ID, nil)
	return inv, nil
}

func (s *Service) Get(ctx context.Context, userID, id string) (*Investment, error) {
	return s.repo.GetByID(ctx, id, userID)
}

func (s *Service) List(ctx context.Context, userID string, includeClosed bool) ([]*Investment, error) {
	return s.repo.List(ctx, userID, includeClosed)
}

type UpdateInput struct {
	Name              string
	AssetType         string
	Institution       *string
	Ticker            *string
	AcquisitionDate   *time.Time
	MaturityDate      *time.Time
	IncludeInNetWorth bool
	Notes             *string
	Status            string
}

func (s *Service) Update(ctx context.Context, userID, id string, in UpdateInput) (*Investment, error) {
	if err := ValidateName(in.Name); err != nil {
		return nil, err
	}
	if err := ValidateAssetType(in.AssetType); err != nil {
		return nil, err
	}
	if in.Status != StatusActive && in.Status != StatusClosed {
		return nil, fmt.Errorf("status inválido: %s", in.Status)
	}

	inv := &Investment{
		Name: in.Name, AssetType: in.AssetType, Institution: in.Institution, Ticker: in.Ticker,
		AcquisitionDate: in.AcquisitionDate, MaturityDate: in.MaturityDate, IncludeInNetWorth: in.IncludeInNetWorth,
		Notes: in.Notes, Status: in.Status,
	}
	if err := s.repo.UpdateDetails(ctx, id, userID, inv); err != nil {
		return nil, err
	}
	s.audit.Record(ctx, &userID, "investment.updated", "investment", id, nil)
	return s.repo.GetByID(ctx, id, userID)
}

// Delete exclui o investimento somente se não houver movimentações
// registradas; caso contrário, encerra a posição (status "closed") para
// preservar o histórico de aportes e resgates.
func (s *Service) Delete(ctx context.Context, userID, id string) (closed bool, err error) {
	hasMovements, err := s.repo.HasMovements(ctx, id, userID)
	if err != nil {
		return false, err
	}
	if hasMovements {
		if err := s.repo.Close(ctx, id, userID); err != nil {
			return false, err
		}
		s.audit.Record(ctx, &userID, "investment.closed", "investment", id, nil)
		return true, nil
	}
	if err := s.repo.Delete(ctx, id, userID); err != nil {
		return false, err
	}
	s.audit.Record(ctx, &userID, "investment.deleted", "investment", id, nil)
	return false, nil
}

type MovementInput struct {
	AccountID  *string
	Type       string
	Amount     decimal.Decimal
	Quantity   *decimal.Decimal
	Fees       decimal.Decimal
	OccurredOn time.Time
	Notes      *string
}

// RecordMovement registra aporte, resgate, rendimento ou taxa, recalculando
// a posição (quantidade, preço médio ponderado, valor investido) e, quando
// uma conta financeira é informada, debita/credita essa conta atomicamente
// na mesma transação SQL — nunca como uma despesa de consumo separada.
func (s *Service) RecordMovement(ctx context.Context, userID, investmentID string, in MovementInput) (*Movement, error) {
	if err := ValidateMovementType(in.Type); err != nil {
		return nil, err
	}
	if err := ValidateAmount(in.Amount); err != nil {
		return nil, err
	}
	if in.Quantity != nil && in.Quantity.IsNegative() {
		return nil, ErrInvalidQuantity
	}
	if in.Fees.IsNegative() {
		return nil, ErrInvalidAmount
	}

	inv, err := s.repo.GetByID(ctx, investmentID, userID)
	if err != nil {
		return nil, err
	}

	if in.AccountID != nil {
		if _, err := s.accounts.GetByID(ctx, *in.AccountID, userID); err != nil {
			if errors.Is(err, account.ErrNotFound) {
				return nil, ErrAccountNotOwned
			}
			return nil, fmt.Errorf("verificar conta: %w", err)
		}
	}

	newQuantity, newAveragePrice, newInvestedAmount, realizedGain, err := applyMovement(inv, in)
	if err != nil {
		return nil, err
	}

	m := &Movement{
		UserID: userID, InvestmentID: investmentID, AccountID: in.AccountID, Type: in.Type,
		Amount: in.Amount, Quantity: in.Quantity, Fees: in.Fees, RealizedGain: realizedGain,
		OccurredOn: in.OccurredOn, Notes: in.Notes,
	}
	if err := s.repo.RecordMovement(ctx, m, newQuantity, newAveragePrice, newInvestedAmount, MarketValueAfter(inv, in)); err != nil {
		return nil, fmt.Errorf("registrar movimentação: %w", err)
	}

	s.audit.Record(ctx, &userID, "investment.movement_recorded", "investment", investmentID, map[string]any{
		"type": in.Type, "amount": in.Amount.String(),
	})
	return m, nil
}

// applyMovement calcula o novo estado da posição (quantidade, preço médio,
// valor investido) e o ganho realizado da movimentação, quando aplicável.
// Nunca inventa um ganho realizado quando não há dados suficientes (ex.:
// resgate de ativo sem controle de quantidade/preço médio).
func applyMovement(inv *Investment, in MovementInput) (newQuantity, newAveragePrice *decimal.Decimal, newInvestedAmount decimal.Decimal, realizedGain *decimal.Decimal, err error) {
	currentQty := decimal.Zero
	if inv.Quantity != nil {
		currentQty = *inv.Quantity
	}
	currentAvg := decimal.Zero
	if inv.AveragePrice != nil {
		currentAvg = *inv.AveragePrice
	}

	switch in.Type {
	case TxContribution:
		newInvestedAmount = inv.InvestedAmount.Add(in.Amount)
		if in.Quantity != nil {
			q := currentQty.Add(*in.Quantity)
			totalCost := currentQty.Mul(currentAvg).Add(in.Amount)
			avg := decimal.Zero
			if !q.IsZero() {
				avg = totalCost.Div(q)
			}
			newQuantity, newAveragePrice = &q, &avg
		} else {
			newQuantity, newAveragePrice = inv.Quantity, inv.AveragePrice
		}
		return newQuantity, newAveragePrice, newInvestedAmount, nil, nil

	case TxRedemption:
		if in.Quantity != nil && inv.Quantity != nil {
			if in.Quantity.GreaterThan(currentQty) {
				return nil, nil, decimal.Zero, nil, ErrInsufficientQuantity
			}
			costRemoved := in.Quantity.Mul(currentAvg)
			q := currentQty.Sub(*in.Quantity)
			newInvestedAmount = maxZero(inv.InvestedAmount.Sub(costRemoved))
			gain := in.Amount.Sub(costRemoved)
			return &q, inv.AveragePrice, newInvestedAmount, &gain, nil
		}
		costRemoved := decimal.Min(in.Amount, inv.InvestedAmount)
		newInvestedAmount = maxZero(inv.InvestedAmount.Sub(costRemoved))
		return inv.Quantity, inv.AveragePrice, newInvestedAmount, nil, nil

	case TxIncome:
		gain := in.Amount
		return inv.Quantity, inv.AveragePrice, inv.InvestedAmount, &gain, nil

	case TxFee:
		loss := in.Amount.Neg()
		return inv.Quantity, inv.AveragePrice, inv.InvestedAmount, &loss, nil

	default:
		return nil, nil, decimal.Zero, nil, ErrInvalidMovementType
	}
}

// MarketValueAfter ajusta o valor de mercado da posição pela movimentação:
// um aporte acrescenta o dinheiro aplicado e um resgate retira o dinheiro
// sacado, de modo que nenhum dos dois apareça como ganho ou perda. Rendimentos
// pagos e taxas não alteram a posição (o efeito no valor de mercado, se houver,
// é informado pelo usuário em uma nova avaliação).
func MarketValueAfter(inv *Investment, in MovementInput) decimal.Decimal {
	switch in.Type {
	case TxContribution:
		return inv.MarketValue.Add(in.Amount)
	case TxRedemption:
		return maxZero(inv.MarketValue.Sub(in.Amount))
	default:
		return inv.MarketValue
	}
}

func maxZero(d decimal.Decimal) decimal.Decimal {
	if d.IsNegative() {
		return decimal.Zero
	}
	return d
}

func (s *Service) ListMovements(ctx context.Context, userID, investmentID string) ([]*Movement, error) {
	if _, err := s.repo.GetByID(ctx, investmentID, userID); err != nil {
		return nil, err
	}
	return s.repo.ListMovements(ctx, investmentID, userID)
}

func (s *Service) AddValuation(ctx context.Context, userID, investmentID string, marketValue decimal.Decimal, date time.Time, notes *string) (*Valuation, error) {
	if marketValue.IsNegative() {
		return nil, ErrInvalidAmount
	}
	if _, err := s.repo.GetByID(ctx, investmentID, userID); err != nil {
		return nil, err
	}
	v, err := s.repo.AddValuation(ctx, investmentID, userID, marketValue, date, notes)
	if err != nil {
		return nil, err
	}
	s.audit.Record(ctx, &userID, "investment.valuation_added", "investment", investmentID, map[string]any{"market_value": marketValue.String()})
	return v, nil
}

func (s *Service) ListValuations(ctx context.Context, userID, investmentID string) ([]*Valuation, error) {
	if _, err := s.repo.GetByID(ctx, investmentID, userID); err != nil {
		return nil, err
	}
	return s.repo.ListValuations(ctx, investmentID, userID)
}

// PortfolioSummary consolida métricas da carteira completa do usuário.
type PortfolioSummary struct {
	TotalInvested       decimal.Decimal
	TotalMarketValue    decimal.Decimal
	UnrealizedGain      decimal.Decimal
	TotalRealizedResult decimal.Decimal
	ByAssetType         map[string]decimal.Decimal
}

func (s *Service) Summary(ctx context.Context, userID string) (PortfolioSummary, error) {
	investments, err := s.repo.List(ctx, userID, false)
	if err != nil {
		return PortfolioSummary{}, err
	}

	summary := PortfolioSummary{
		TotalInvested: decimal.Zero, TotalMarketValue: decimal.Zero, UnrealizedGain: decimal.Zero,
		TotalRealizedResult: decimal.Zero, ByAssetType: map[string]decimal.Decimal{},
	}
	for _, inv := range investments {
		summary.TotalInvested = summary.TotalInvested.Add(inv.InvestedAmount)
		summary.TotalMarketValue = summary.TotalMarketValue.Add(inv.MarketValue)
		summary.ByAssetType[inv.AssetType] = summary.ByAssetType[inv.AssetType].Add(inv.MarketValue)

		movements, err := s.repo.ListMovements(ctx, inv.ID, userID)
		if err != nil {
			return PortfolioSummary{}, err
		}
		for _, m := range movements {
			if m.RealizedGain != nil {
				summary.TotalRealizedResult = summary.TotalRealizedResult.Add(*m.RealizedGain)
			}
		}
	}
	summary.UnrealizedGain = summary.TotalMarketValue.Sub(summary.TotalInvested)
	return summary, nil
}
