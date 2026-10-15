package category

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, userID, name, kind string) (*Category, error) {
	const q = `
		INSERT INTO categories (user_id, name, kind)
		VALUES ($1, $2, $3)
		RETURNING id, created_at, updated_at
	`
	c := &Category{UserID: &userID, Name: name, Kind: kind}
	err := r.db.QueryRowContext(ctx, q, userID, name, kind).Scan(&c.ID, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrNameTaken
		}
		return nil, fmt.Errorf("criar categoria: %w", err)
	}
	return c, nil
}

// ListVisible retorna as categorias globais do sistema somadas às
// categorias privadas do usuário autenticado.
func (r *Repository) ListVisible(ctx context.Context, userID string, includeArchived bool) ([]*Category, error) {
	q := `
		SELECT id, user_id, name, kind, archived_at, created_at, updated_at
		FROM categories
		WHERE (user_id IS NULL OR user_id = $1)
	`
	if !includeArchived {
		q += ` AND archived_at IS NULL`
	}
	q += ` ORDER BY (user_id IS NULL) DESC, kind, name`

	rows, err := r.db.QueryContext(ctx, q, userID)
	if err != nil {
		return nil, fmt.Errorf("listar categorias: %w", err)
	}
	defer rows.Close()

	var out []*Category
	for rows.Next() {
		c, err := scanRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// GetVisible busca uma categoria pelo ID, exigindo que ela seja global ou
// pertença ao usuário autenticado — nunca retorna categorias privadas de
// outro usuário.
func (r *Repository) GetVisible(ctx context.Context, id, userID string) (*Category, error) {
	const q = `
		SELECT id, user_id, name, kind, archived_at, created_at, updated_at
		FROM categories
		WHERE id = $1 AND (user_id IS NULL OR user_id = $2)
	`
	var c Category
	err := r.db.QueryRowContext(ctx, q, id, userID).Scan(&c.ID, &c.UserID, &c.Name, &c.Kind, &c.ArchivedAt, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("consultar categoria: %w", err)
	}
	return &c, nil
}

func (r *Repository) UpdateName(ctx context.Context, id, userID, name string) error {
	const q = `UPDATE categories SET name = $3, updated_at = now() WHERE id = $1 AND user_id = $2`
	res, err := r.db.ExecContext(ctx, q, id, userID, name)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrNameTaken
		}
		return fmt.Errorf("atualizar categoria: %w", err)
	}
	return checkAffected(res)
}

func (r *Repository) Archive(ctx context.Context, id, userID string) error {
	const q = `UPDATE categories SET archived_at = now(), updated_at = now() WHERE id = $1 AND user_id = $2 AND archived_at IS NULL`
	res, err := r.db.ExecContext(ctx, q, id, userID)
	if err != nil {
		return fmt.Errorf("arquivar categoria: %w", err)
	}
	return checkAffected(res)
}

func (r *Repository) Delete(ctx context.Context, id, userID string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM categories WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return fmt.Errorf("excluir categoria: %w", err)
	}
	return checkAffected(res)
}

func (r *Repository) InUse(ctx context.Context, id string) (bool, error) {
	var exists bool
	const q = `SELECT EXISTS (SELECT 1 FROM transactions WHERE category_id = $1)`
	if err := r.db.QueryRowContext(ctx, q, id).Scan(&exists); err != nil {
		return false, fmt.Errorf("verificar uso da categoria: %w", err)
	}
	return exists, nil
}

func scanRow(rows *sql.Rows) (*Category, error) {
	var c Category
	if err := rows.Scan(&c.ID, &c.UserID, &c.Name, &c.Kind, &c.ArchivedAt, &c.CreatedAt, &c.UpdatedAt); err != nil {
		return nil, fmt.Errorf("ler categoria: %w", err)
	}
	return &c, nil
}

func checkAffected(res sql.Result) error {
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("verificar linhas afetadas: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func isUniqueViolation(err error) bool {
	return errIsPgCode(err, "23505")
}
