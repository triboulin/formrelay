package repository

import (
	"database/sql"
	"testing"

	"formrelay-admin/internal/database"
)

// newTestDB ouvre une base SQLite en mémoire avec le schéma appliqué, pour les tests.
func newTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := database.New(":memory:")
	if err != nil {
		t.Fatalf("erreur ouverture base de test: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}
