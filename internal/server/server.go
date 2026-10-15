// Package server monta o roteador HTTP completo da aplicação, conectando
// configuração, banco de dados e os handlers de cada domínio. Fica separado
// de cmd/api para poder ser reutilizado diretamente pelos testes de
// integração, exercitando a pilha HTTP real (rotas, middlewares e
// autorização) contra um PostgreSQL de teste.
package server

import (
	"database/sql"
	"log/slog"
	"net/http"

	"github.com/nicolastomaschewski/controle-financeiro/internal/account"
	"github.com/nicolastomaschewski/controle-financeiro/internal/asset"
	"github.com/nicolastomaschewski/controle-financeiro/internal/audit"
	"github.com/nicolastomaschewski/controle-financeiro/internal/auth"
	"github.com/nicolastomaschewski/controle-financeiro/internal/budget"
	"github.com/nicolastomaschewski/controle-financeiro/internal/category"
	"github.com/nicolastomaschewski/controle-financeiro/internal/config"
	"github.com/nicolastomaschewski/controle-financeiro/internal/dashboard"
	"github.com/nicolastomaschewski/controle-financeiro/internal/investment"
	"github.com/nicolastomaschewski/controle-financeiro/internal/liability"
	"github.com/nicolastomaschewski/controle-financeiro/internal/middleware"
	"github.com/nicolastomaschewski/controle-financeiro/internal/networth"
	"github.com/nicolastomaschewski/controle-financeiro/internal/recurring"
	"github.com/nicolastomaschewski/controle-financeiro/internal/transaction"
	"github.com/nicolastomaschewski/controle-financeiro/internal/user"
)

// New monta o handler HTTP completo, já com os middlewares globais
// aplicados (recuperação de panics, logging, cabeçalhos de segurança, CORS
// e limite de tamanho do corpo).
func New(db *sql.DB, cfg *config.Config, logger *slog.Logger) http.Handler {
	router := buildRouter(db, cfg)

	return middleware.Chain(router,
		middleware.Recover(logger),
		middleware.RequestID,
		middleware.Logger(logger),
		middleware.SecurityHeaders,
		middleware.CORS(cfg.CORSOrigins),
		middleware.MaxBodyBytes(1<<20), // 1 MiB
	)
}

func buildRouter(db *sql.DB, cfg *config.Config) http.Handler {
	auditRecorder := audit.NewRecorder(db)

	userRepo := user.NewRepository(db)
	userService := user.NewService(userRepo)
	userHandler := user.NewHandler(userService)

	sessionRepo := auth.NewSessionRepository(db)
	loginAttemptRepo := auth.NewLoginAttemptRepository(db)
	authService := auth.NewService(userRepo, sessionRepo, loginAttemptRepo, auditRecorder)
	authHandler := auth.NewHandler(authService, cfg.CookieSecure)

	root := http.NewServeMux()

	root.HandleFunc("GET /healthz", healthHandler(db))
	root.HandleFunc("GET /readyz", healthHandler(db))
	authHandler.Register(root)

	protected := http.NewServeMux()
	userHandler.Register(protected)
	registerDomainRoutes(protected, db)

	protectedHandler := middleware.Chain(http.Handler(protected),
		auth.RequireAuth(sessionRepo),
		middleware.CSRFProtect,
	)
	root.Handle("/api/v1/", protectedHandler)

	return root
}

// registerDomainRoutes registra as rotas protegidas dos domínios de negócio
// (contas, categorias, transações, orçamentos, patrimônio, investimentos,
// dashboard e relatórios) conforme cada etapa é implementada.
func registerDomainRoutes(mux *http.ServeMux, db *sql.DB) {
	auditRecorder := audit.NewRecorder(db)

	accountRepo := account.NewRepository(db)
	accountService := account.NewService(accountRepo, auditRecorder)
	account.NewHandler(accountService).Register(mux)

	categoryRepo := category.NewRepository(db)
	categoryService := category.NewService(categoryRepo, auditRecorder)
	category.NewHandler(categoryService).Register(mux)

	transactionRepo := transaction.NewRepository(db)
	transactionService := transaction.NewService(transactionRepo, accountRepo, categoryRepo, auditRecorder)
	transaction.NewHandler(transactionService).Register(mux)

	recurringRepo := recurring.NewRepository(db)
	recurringService := recurring.NewService(recurringRepo, transactionRepo, accountRepo, categoryRepo, auditRecorder)
	recurring.NewHandler(recurringService).Register(mux)

	budgetRepo := budget.NewRepository(db)
	budgetService := budget.NewService(budgetRepo, categoryRepo, auditRecorder)
	budget.NewHandler(budgetService).Register(mux)

	assetRepo := asset.NewRepository(db)
	assetService := asset.NewService(assetRepo, auditRecorder)
	asset.NewHandler(assetService).Register(mux)

	liabilityRepo := liability.NewRepository(db)
	liabilityService := liability.NewService(liabilityRepo, auditRecorder)
	liability.NewHandler(liabilityService).Register(mux)

	investmentRepo := investment.NewRepository(db)
	investmentService := investment.NewService(investmentRepo, accountRepo, auditRecorder)
	investment.NewHandler(investmentService).Register(mux)

	netWorthService := networth.NewService(accountRepo, assetRepo, liabilityRepo, investmentRepo)
	networth.NewHandler(netWorthService).Register(mux)

	dashboardService := dashboard.NewService(accountService, transactionService, netWorthService, investmentService)
	dashboard.NewHandler(dashboardService).Register(mux)
}
