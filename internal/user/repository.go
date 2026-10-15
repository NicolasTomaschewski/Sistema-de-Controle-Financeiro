package user

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// Create insere um novo usuário. Retorna ErrEmailTaken se já existir um
// usuário com o mesmo e-mail normalizado (unicidade garantida pelo banco).
func (r *Repository) Create(ctx context.Context, u *User) error {
	const q = `
		INSERT INTO users (name, email, email_normalized, password_hash)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at
	`
	err := r.db.QueryRowContext(ctx, q, u.Name, u.Email, u.EmailNormalized, u.PasswordHash).
		Scan(&u.ID, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrEmailTaken
		}
		return fmt.Errorf("inserir usuário: %w", err)
	}
	return nil
}

func (r *Repository) GetByID(ctx context.Context, id string) (*User, error) {
	const q = `
		SELECT id, name, email, email_normalized, password_hash, email_verified_at, created_at, updated_at
		FROM users WHERE id = $1
	`
	return r.scanOne(r.db.QueryRowContext(ctx, q, id))
}

func (r *Repository) GetByEmailNormalized(ctx context.Context, emailNormalized string) (*User, error) {
	const q = `
		SELECT id, name, email, email_normalized, password_hash, email_verified_at, created_at, updated_at
		FROM users WHERE email_normalized = $1
	`
	return r.scanOne(r.db.QueryRowContext(ctx, q, emailNormalized))
}

func (r *Repository) scanOne(row *sql.Row) (*User, error) {
	var u User
	err := row.Scan(&u.ID, &u.Name, &u.Email, &u.EmailNormalized, &u.PasswordHash, &u.EmailVerifiedAt, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("consultar usuário: %w", err)
	}
	return &u, nil
}

// UpdateProfile atualiza nome e/ou e-mail de um usuário identificado pelo
// próprio ID (nunca por um identificador informado por outro usuário).
func (r *Repository) UpdateProfile(ctx context.Context, id, name, email, emailNormalized string) error {
	const q = `
		UPDATE users
		SET name = $2, email = $3, email_normalized = $4, updated_at = now()
		WHERE id = $1
	`
	res, err := r.db.ExecContext(ctx, q, id, name, email, emailNormalized)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrEmailTaken
		}
		return fmt.Errorf("atualizar perfil: %w", err)
	}
	return checkRowsAffected(res)
}

func (r *Repository) UpdatePasswordHash(ctx context.Context, id, passwordHash string) error {
	const q = `UPDATE users SET password_hash = $2, updated_at = now() WHERE id = $1`
	res, err := r.db.ExecContext(ctx, q, id, passwordHash)
	if err != nil {
		return fmt.Errorf("atualizar senha: %w", err)
	}
	return checkRowsAffected(res)
}

func checkRowsAffected(res sql.Result) error {
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("verificar linhas afetadas: %w", err)
	}
	if n == 0 {
		return ErrUserNotFound
	}
	return nil
}
