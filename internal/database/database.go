// Package database centraliza a conexão com o PostgreSQL e a execução de migrações.
package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// Options controla o comportamento do pool de conexões.
type Options struct {
	MaxOpenConns    int32
	MaxIdleConns    int32
	ConnMaxLifetime time.Duration
	ConnTimeout     time.Duration
}

// Connect abre o pool de conexões com o PostgreSQL usando database/sql com o
// driver pgx, valida a conectividade com um ping e aplica limites do pool.
func Connect(databaseURL string, opts Options) (*sql.DB, error) {
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("abrir conexão com o banco: %w", err)
	}

	if opts.MaxOpenConns > 0 {
		db.SetMaxOpenConns(int(opts.MaxOpenConns))
	}
	if opts.MaxIdleConns > 0 {
		db.SetMaxIdleConns(int(opts.MaxIdleConns))
	} else {
		db.SetMaxIdleConns(int(opts.MaxOpenConns))
	}
	if opts.ConnMaxLifetime > 0 {
		db.SetConnMaxLifetime(opts.ConnMaxLifetime)
	}

	timeout := opts.ConnTimeout
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("verificar conectividade com o banco: %w", err)
	}

	return db, nil
}
