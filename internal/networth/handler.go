package networth

import (
	"net/http"

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
	mux.HandleFunc("GET /api/v1/net-worth", h.get)
	mux.HandleFunc("GET /api/v1/net-worth/composition", h.composition)
}

type snapshotResponse struct {
	AccountsBalance     string `json:"accounts_balance"`
	DeclaredAssetsValue string `json:"declared_assets_value"`
	InvestmentsValue    string `json:"investments_value"`
	GrossWorth          string `json:"gross_worth"`
	TotalLiabilities    string `json:"total_liabilities"`
	NetWorth            string `json:"net_worth"`
	AsOf                string `json:"as_of"`
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	userID, _ := authctx.UserID(r.Context())

	snap, err := h.service.Calculate(r.Context(), userID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "Não foi possível calcular o patrimônio líquido.")
		return
	}

	httpx.JSON(w, http.StatusOK, snapshotResponse{
		AccountsBalance:     snap.AccountsBalance.StringFixed(2),
		DeclaredAssetsValue: snap.DeclaredAssetsValue.StringFixed(2),
		InvestmentsValue:    snap.InvestmentsValue.StringFixed(2),
		GrossWorth:          snap.GrossWorth.StringFixed(2),
		TotalLiabilities:    snap.TotalLiabilities.StringFixed(2),
		NetWorth:            snap.NetWorth.StringFixed(2),
		AsOf:                snap.AsOf.Format("2006-01-02T15:04:05Z07:00"),
	})
}

type sliceDTO struct {
	Group        string `json:"group"`
	Type         string `json:"type"`
	Value        string `json:"value"`
	SharePercent string `json:"share_percent"`
}

type compositionResponse struct {
	AssetsValue      string     `json:"assets_value"`
	InvestmentsValue string     `json:"investments_value"`
	Total            string     `json:"total"`
	Slices           []sliceDTO `json:"slices"`
	AsOf             string     `json:"as_of"`
}

func (h *Handler) composition(w http.ResponseWriter, r *http.Request) {
	userID, _ := authctx.UserID(r.Context())

	c, err := h.service.Composition(r.Context(), userID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "Não foi possível calcular a composição do patrimônio.")
		return
	}
	resp := compositionResponse{
		AssetsValue:      c.AssetsValue.StringFixed(2),
		InvestmentsValue: c.InvestmentsValue.StringFixed(2),
		Total:            c.Total.StringFixed(2),
		Slices:           make([]sliceDTO, 0, len(c.Slices)),
		AsOf:             c.AsOf.Format("2006-01-02T15:04:05Z07:00"),
	}
	for _, sl := range c.Slices {
		resp.Slices = append(resp.Slices, sliceDTO{
			Group: sl.Group, Type: sl.Type, Value: sl.Value.StringFixed(2), SharePercent: sl.SharePercent.StringFixed(1),
		})
	}
	httpx.JSON(w, http.StatusOK, resp)
}
