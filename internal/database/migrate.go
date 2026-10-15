package database

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Migrate aplica, em ordem, todos os arquivos .sql do diretório informado que
// ainda não constam na tabela schema_migrations. Cada migração é executada
// dentro de uma transação: se falhar, nenhuma alteração parcial é mantida.
// É seguro executar repetidamente em um banco já migrado (idempotente) e
// reproduzível em um banco vazio.
func Migrate(ctx context.Context, db *sql.DB, dir string) ([]string, error) {
	if err := ensureMigrationsTable(ctx, db); err != nil {
		return nil, fmt.Errorf("preparar tabela de controle de migrações: %w", err)
	}

	files, err := listMigrationFiles(dir)
	if err != nil {
		return nil, fmt.Errorf("listar arquivos de migração: %w", err)
	}

	applied, err := appliedVersions(ctx, db)
	if err != nil {
		return nil, fmt.Errorf("consultar migrações aplicadas: %w", err)
	}

	var ran []string
	for _, f := range files {
		version := versionFromFilename(f)
		if applied[version] {
			continue
		}

		content, err := os.ReadFile(filepath.Join(dir, f))
		if err != nil {
			return ran, fmt.Errorf("ler migração %s: %w", f, err)
		}

		if err := applyMigration(ctx, db, version, string(content)); err != nil {
			return ran, fmt.Errorf("aplicar migração %s: %w", f, err)
		}
		ran = append(ran, f)
	}

	return ran, nil
}

func ensureMigrationsTable(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version     TEXT PRIMARY KEY,
			applied_at  TIMESTAMPTZ NOT NULL DEFAULT now()
		)
	`)
	return err
}

func appliedVersions(ctx context.Context, db *sql.DB) (map[string]bool, error) {
	rows, err := db.QueryContext(ctx, `SELECT version FROM schema_migrations`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	applied := make(map[string]bool)
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		applied[v] = true
	}
	return applied, rows.Err()
}

func applyMigration(ctx context.Context, db *sql.DB, version, sqlContent string) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, sqlContent); err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, `INSERT INTO schema_migrations (version) VALUES ($1)`, version); err != nil {
		return err
	}

	return tx.Commit()
}

func listMigrationFiles(dir string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if strings.HasSuffix(d.Name(), ".sql") {
			files = append(files, d.Name())
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	return files, nil
}

func versionFromFilename(filename string) string {
	return strings.TrimSuffix(filename, ".sql")
}
