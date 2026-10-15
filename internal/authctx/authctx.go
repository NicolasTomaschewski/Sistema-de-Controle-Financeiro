// Package authctx propaga a identidade do usuário autenticado através do
// contexto da requisição. É o único ponto de verdade sobre "quem está
// fazendo a requisição" para todos os domínios da aplicação — nenhum
// handler deve aceitar um identificador de usuário vindo do cliente para
// fins de autorização.
package authctx

import "context"

type contextKey string

const userIDKey contextKey = "authenticated_user_id"

// WithUserID retorna um novo contexto associando o ID do usuário autenticado.
// Deve ser chamado exclusivamente pelo middleware de autenticação, após
// validar a sessão contra o banco de dados.
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

// UserID extrai o ID do usuário autenticado do contexto da requisição.
func UserID(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(userIDKey).(string)
	return id, ok && id != ""
}
