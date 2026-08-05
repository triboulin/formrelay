package database

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNew_CreatesSchemaInMemory(t *testing.T) {
	db, err := New(":memory:")
	if err != nil {
		t.Fatalf("New() erreur inattendue: %v", err)
	}
	defer db.Close()

	tables := []string{"clients", "submissions"}
	for _, table := range tables {
		var name string
		err := db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&name)
		if err != nil {
			t.Errorf("table %q introuvable: %v", table, err)
		}
	}
}

func TestNew_CreatesNestedDataDirectory(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "nested", "sub", "formrelay.db")

	db, err := New(dbPath)
	if err != nil {
		t.Fatalf("New() erreur inattendue: %v", err)
	}
	defer db.Close()

	if _, err := os.Stat(filepath.Dir(dbPath)); err != nil {
		t.Errorf("le dossier parent aurait dû être créé: %v", err)
	}
	if _, err := os.Stat(dbPath); err != nil {
		t.Errorf("le fichier de base de données aurait dû être créé: %v", err)
	}
}

func TestNew_SchemaIsIdempotent(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "formrelay.db")

	db1, err := New(dbPath)
	if err != nil {
		t.Fatalf("premier New() erreur: %v", err)
	}
	db1.Close()

	// Rouvrir la même base ne doit pas échouer (CREATE TABLE IF NOT EXISTS).
	db2, err := New(dbPath)
	if err != nil {
		t.Fatalf("second New() erreur: %v", err)
	}
	defer db2.Close()
}

func TestNew_InvalidDirectory(t *testing.T) {
	dir := t.TempDir()
	blockingFile := filepath.Join(dir, "not-a-dir")
	if err := os.WriteFile(blockingFile, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Le chemin de la base pointe dans un sous-dossier d'un fichier existant :
	// MkdirAll doit échouer.
	dbPath := filepath.Join(blockingFile, "sub", "formrelay.db")

	_, err := New(dbPath)
	if err == nil {
		t.Fatal("attendu une erreur car le parent n'est pas un dossier valide")
	}
}
