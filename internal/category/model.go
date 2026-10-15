// Package category implementa as categorias de receitas e despesas,
// distinguindo categorias globais do sistema (não editáveis pelo usuário)
// de categorias privadas de cada usuário.
package category

import (
	"errors"
	"strings"
	"time"
)

type Category struct {
	ID         string
	UserID     *string // nil = categoria global do sistema
	Name       string
	Kind       string // "income" ou "expense"
	ArchivedAt *time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

const (
	KindIncome  = "income"
	KindExpense = "expense"
)

var (
	ErrNotFound       = errors.New("categoria não encontrada")
	ErrInvalidName    = errors.New("nome da categoria inválido")
	ErrInvalidKind    = errors.New("tipo de categoria inválido")
	ErrNameTaken      = errors.New("já existe uma categoria com este nome e tipo")
	ErrSystemCategory = errors.New("categorias do sistema não podem ser alteradas")
	ErrInUse          = errors.New("categoria possui lançamentos vinculados")
)

func ValidateName(name string) error {
	if len(strings.TrimSpace(name)) == 0 || len(name) > 100 {
		return ErrInvalidName
	}
	return nil
}

func ValidateKind(kind string) error {
	if kind != KindIncome && kind != KindExpense {
		return ErrInvalidKind
	}
	return nil
}

func (c *Category) IsSystem() bool {
	return c.UserID == nil
}
