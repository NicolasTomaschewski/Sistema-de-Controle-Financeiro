package auth

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

const (
	loginAttemptWindow  = 15 * time.Minute
	maxFailuresPerEmail = 5
	maxFailuresPerIP    = 20
)

type LoginAttemptRepository struct {
	db *sql.DB
}

func NewLoginAttemptRepository(db *sql.DB) *LoginAttemptRepository {
	return &LoginAttemptRepository{db: db}
}

func (r *LoginAttemptRepository) Record(ctx context.Context, emailNormalized, ip string, success bool) error {
	const q = `INSERT INTO login_attempts (email_normalized, ip_address, success) VALUES ($1, $2, $3)`
	if _, err := r.db.ExecContext(ctx, q, emailNormalized, ip, success); err != nil {
		return fmt.Errorf("registrar tentativa de login: %w", err)
	}
	return nil
}

// IsRateLimited verifica, de forma persistente (sobrevive a reinícios do
// processo), se o e-mail ou o IP informados excederam o número de
// tentativas falhas permitidas na janela recente.
func (r *LoginAttemptRepository) IsRateLimited(ctx context.Context, emailNormalized, ip string) (bool, error) {
	since := time.Now().Add(-loginAttemptWindow)

	var emailFailures int
	const qEmail = `
		SELECT count(*) FROM login_attempts
		WHERE email_normalized = $1 AND success = false AND attempted_at > $2
	`
	if err := r.db.QueryRowContext(ctx, qEmail, emailNormalized, since).Scan(&emailFailures); err != nil {
		return false, fmt.Errorf("contar tentativas por e-mail: %w", err)
	}
	if emailFailures >= maxFailuresPerEmail {
		return true, nil
	}

	var ipFailures int
	const qIP = `
		SELECT count(*) FROM login_attempts
		WHERE ip_address = $1 AND success = false AND attempted_at > $2
	`
	if err := r.db.QueryRowContext(ctx, qIP, ip, since).Scan(&ipFailures); err != nil {
		return false, fmt.Errorf("contar tentativas por IP: %w", err)
	}
	return ipFailures >= maxFailuresPerIP, nil
}
