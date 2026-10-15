package integration

import (
	"context"
	"testing"

	"github.com/nicolastomaschewski/controle-financeiro/internal/database"
)

func TestMigrate_ReproducibleOnEmptyDatabase(t *testing.T) {
	db := openTestSchema(t)
	ctx := context.Background()
	migrationsDir := findMigrationsDir(t)

	applied, err := database.Migrate(ctx, db, migrationsDir)
	if err != nil {
		t.Fatalf("primeira execução de migrações falhou: %v", err)
	}
	if len(applied) == 0 {
		t.Fatal("esperava ao menos uma migração aplicada (ex.: usuários e sessões)")
	}

	appliedAgain, err := database.Migrate(ctx, db, migrationsDir)
	if err != nil {
		t.Fatalf("segunda execução de migrações falhou: %v", err)
	}
	if len(appliedAgain) != 0 {
		t.Fatalf("execução repetida não deveria reaplicar migrações, reaplicou: %v", appliedAgain)
	}
}
