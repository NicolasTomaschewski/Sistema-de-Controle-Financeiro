package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"time"
)

const (
	SessionCookieName = "session_token"
	SessionTTL        = 7 * 24 * time.Hour
	tokenByteLen      = 32
)

var ErrSessionInvalid = errors.New("sessão inválida ou expirada")

type Session struct {
	ID        string
	UserID    string
	ExpiresAt time.Time
	RevokedAt *sql.NullTime
}

type SessionRepository struct {
	db *sql.DB
}

func NewSessionRepository(db *sql.DB) *SessionRepository {
	return &SessionRepository{db: db}
}

// generateToken cria um token opaco e aleatório. Apenas seu hash SHA-256 é
// persistido — o valor em texto puro existe somente no cookie do cliente,
// nunca no banco de dados ou em logs.
func generateToken() (plain string, hash string, err error) {
	b := make([]byte, tokenByteLen)
	if _, err := rand.Read(b); err != nil {
		return "", "", fmt.Errorf("gerar token de sessão: %w", err)
	}
	plain = base64.RawURLEncoding.EncodeToString(b)
	sum := sha256.Sum256([]byte(plain))
	hash = hex.EncodeToString(sum[:])
	return plain, hash, nil
}

func hashToken(plain string) string {
	sum := sha256.Sum256([]byte(plain))
	return hex.EncodeToString(sum[:])
}

// Create gera uma nova sessão para o usuário e retorna o token em texto
// puro, que deve ser entregue ao cliente apenas via cookie HttpOnly.
func (r *SessionRepository) Create(ctx context.Context, userID, userAgent string) (token string, expiresAt time.Time, err error) {
	plain, hash, err := generateToken()
	if err != nil {
		return "", time.Time{}, err
	}
	expiresAt = time.Now().Add(SessionTTL)

	const q = `
		INSERT INTO sessions (user_id, token_hash, user_agent, expires_at)
		VALUES ($1, $2, $3, $4)
	`
	if _, err := r.db.ExecContext(ctx, q, userID, hash, userAgent, expiresAt); err != nil {
		return "", time.Time{}, fmt.Errorf("criar sessão: %w", err)
	}
	return plain, expiresAt, nil
}

// ValidateToken confere o token recebido contra o hash armazenado, rejeita
// sessões expiradas ou revogadas, e atualiza o último uso.
func (r *SessionRepository) ValidateToken(ctx context.Context, plainToken string) (userID string, err error) {
	hash := hashToken(plainToken)

	const q = `
		SELECT user_id FROM sessions
		WHERE token_hash = $1 AND revoked_at IS NULL AND expires_at > now()
	`
	err = r.db.QueryRowContext(ctx, q, hash).Scan(&userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrSessionInvalid
		}
		return "", fmt.Errorf("validar sessão: %w", err)
	}

	// Atualização best-effort do último uso; uma falha aqui não deve
	// invalidar uma sessão já comprovadamente válida.
	_, _ = r.db.ExecContext(ctx, `UPDATE sessions SET last_used_at = now() WHERE token_hash = $1`, hash)

	return userID, nil
}

// Revoke invalida a sessão correspondente ao token (logout).
func (r *SessionRepository) Revoke(ctx context.Context, plainToken string) error {
	hash := hashToken(plainToken)
	_, err := r.db.ExecContext(ctx, `UPDATE sessions SET revoked_at = now() WHERE token_hash = $1 AND revoked_at IS NULL`, hash)
	if err != nil {
		return fmt.Errorf("revogar sessão: %w", err)
	}
	return nil
}

func SetSessionCookie(w http.ResponseWriter, token string, expiresAt time.Time, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    token,
		Path:     "/",
		Expires:  expiresAt,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

func ClearSessionCookie(w http.ResponseWriter, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

func ReadSessionToken(r *http.Request) (string, bool) {
	c, err := r.Cookie(SessionCookieName)
	if err != nil || c.Value == "" {
		return "", false
	}
	return c.Value, true
}
