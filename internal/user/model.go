// Package user representa a identidade e o perfil dos usuários do sistema.
// Não depende de nenhum outro domínio da aplicação.
package user

import (
	"errors"
	"regexp"
	"strings"
	"time"
)

type User struct {
	ID              string
	Name            string
	Email           string
	EmailNormalized string
	PasswordHash    string
	EmailVerifiedAt *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

var (
	ErrEmailTaken      = errors.New("e-mail já cadastrado")
	ErrInvalidEmail    = errors.New("e-mail inválido")
	ErrInvalidName     = errors.New("nome inválido")
	ErrInvalidPassword = errors.New("senha inválida")
	ErrUserNotFound    = errors.New("usuário não encontrado")
	ErrWrongPassword   = errors.New("senha atual incorreta")
)

var emailPattern = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)

// NormalizeEmail aplica a normalização usada para garantir unicidade
// independente de maiúsculas/minúsculas e espaços acidentais.
func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// ValidateEmail garante um formato mínimo plausível de e-mail.
func ValidateEmail(email string) error {
	e := strings.TrimSpace(email)
	if e == "" || len(e) > 254 || !emailPattern.MatchString(e) {
		return ErrInvalidEmail
	}
	return nil
}

// ValidateName garante que o nome não é vazio e tem um tamanho razoável.
func ValidateName(name string) error {
	n := strings.TrimSpace(name)
	if len(n) < 2 || len(n) > 150 {
		return ErrInvalidName
	}
	return nil
}

// ValidatePassword define a política mínima de senha exigida no cadastro e
// na troca de senha.
func ValidatePassword(password string) error {
	if len(password) < 8 || len(password) > 128 {
		return ErrInvalidPassword
	}
	return nil
}
