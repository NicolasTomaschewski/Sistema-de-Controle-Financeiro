// Utilitários compartilhados pelos testes de integração. Todos os testes
// deste pacote executam contra um PostgreSQL real — nunca mocks — conforme
// exigido para validar migrações, restrições do banco e transações.
package integration

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"os"
	"path/filepath"
	"regexp"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/nicolastomaschewski/controle-financeiro/internal/database"
)

var nonAlnum = regexp.MustCompile(`[^a-zA-Z0-9]+`)

// newJarClient cria um http.Client com seu próprio cookie jar isolado,
// representando a sessão de um usuário distinto no mesmo servidor de teste.
func newJarClient(t *testing.T) *http.Client {
	t.Helper()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("criar cookie jar: %v", err)
	}
	return &http.Client{Jar: jar, Timeout: 5 * time.Second}
}

// openTestSchema conecta ao banco de teste e isola o teste em um schema
// PostgreSQL exclusivo e descartável, evitando interferência entre testes e
// com dados de desenvolvimento. O schema é recriado vazio no início e
// removido ao final do teste.
func openTestSchema(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL não definida; pulando teste de integração")
	}

	schema := fmt.Sprintf("test_%s_%d", nonAlnum.ReplaceAllString(t.Name(), "_"), os.Getpid())
	if len(schema) > 63 {
		schema = schema[:63]
	}

	admin, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("abrir conexão administrativa: %v", err)
	}
	if _, err := admin.Exec(fmt.Sprintf(`DROP SCHEMA IF EXISTS %s CASCADE`, schema)); err != nil {
		t.Fatalf("limpar schema anterior: %v", err)
	}
	if _, err := admin.Exec(fmt.Sprintf(`CREATE SCHEMA %s`, schema)); err != nil {
		t.Fatalf("criar schema de teste: %v", err)
	}
	admin.Close()

	// Repassa o schema de teste como parâmetro de conexão (search_path), para
	// que toda nova conexão do pool já nasça isolada no schema correto — um
	// SET por conexão individual não se propagaria para as demais conexões
	// do pool.
	testDSN := dsn + "&search_path=" + schema
	tdb, err := sql.Open("pgx", testDSN)
	if err != nil {
		t.Fatalf("abrir conexão de teste: %v", err)
	}

	t.Cleanup(func() {
		cleanup, err := sql.Open("pgx", dsn)
		if err == nil {
			cleanup.Exec(fmt.Sprintf(`DROP SCHEMA IF EXISTS %s CASCADE`, schema))
			cleanup.Close()
		}
		tdb.Close()
	})

	return tdb
}

// newMigratedTestDB cria um schema de teste isolado e aplica todas as
// migrações do projeto, retornando um banco pronto para os testes de
// domínio (autenticação, contas, transações etc.).
func newMigratedTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db := openTestSchema(t)
	if _, err := database.Migrate(context.Background(), db, findMigrationsDir(t)); err != nil {
		t.Fatalf("aplicar migrações no schema de teste: %v", err)
	}
	return db
}

func findMigrationsDir(t *testing.T) string {
	t.Helper()
	dir, err := filepath.Abs(filepath.Join("..", "..", "migrations"))
	if err != nil {
		t.Fatalf("resolver diretório de migrações: %v", err)
	}
	return dir
}
