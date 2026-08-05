package handler

import (
	"database/sql"
	"testing"

	"github.com/google/uuid"

	"formrelay-admin/internal/config"
	"formrelay-admin/internal/database"
	"formrelay-admin/internal/model"
	"formrelay-admin/internal/repository"
	"formrelay-admin/internal/service"
)

// testTemplatesDir pointe vers le dossier templates/ à la racine du projet,
// relatif au dossier du package internal/handler.
const testTemplatesDir = "../../templates"

func newTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := database.New(":memory:")
	if err != nil {
		t.Fatalf("erreur ouverture base de test: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

// newTestEnv construit un environnement complet (repos, worker mail, service métier)
// prêt à être injecté dans les handlers HTTP testés.
func newTestEnv(t *testing.T) (*repository.ClientRepository, *repository.SubmissionRepository, *service.FormService) {
	t.Helper()
	db := newTestDB(t)
	clientRepo := repository.NewClientRepository(db)
	subRepo := repository.NewSubmissionRepository(db)

	// SMTP non configuré : les envois échoueront silencieusement (statut FAILED),
	// ce qui est acceptable pour tester le flux HTTP indépendamment du SMTP.
	worker := service.NewMailWorker(config.Config{}, subRepo, 10)
	worker.Start(1)
	formService := service.NewFormService(subRepo, worker)

	return clientRepo, subRepo, formService
}

func newTestClient(t *testing.T, repo *repository.ClientRepository, active bool) model.Client {
	t.Helper()
	c := model.Client{
		ID:               uuid.NewString(),
		Name:             "Acme",
		DestinationEmail: "dest@example.com",
		Active:           active,
	}
	if err := repo.Create(c); err != nil {
		t.Fatalf("erreur création client de test: %v", err)
	}
	return c
}
