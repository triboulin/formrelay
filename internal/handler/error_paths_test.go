package handler

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"formrelay-admin/internal/config"
	"formrelay-admin/internal/database"
	"formrelay-admin/internal/repository"
	"formrelay-admin/internal/service"
)

// brokenReader simule une erreur de lecture du corps de la requête,
// pour déclencher l'échec de r.ParseForm().
type brokenReader struct{}

func (brokenReader) Read(p []byte) (int, error) { return 0, errors.New("erreur de lecture simulée") }

// closedDBEnv construit un environnement handler dont la base de données est
// immédiatement fermée, afin de forcer les branches d'erreur 500 des handlers.
func closedDBEnv(t *testing.T) (*repository.ClientRepository, *repository.SubmissionRepository, *AdminHandler, *PublicHandler) {
	t.Helper()
	db, err := database.New(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	clientRepo := repository.NewClientRepository(db)
	subRepo := repository.NewSubmissionRepository(db)
	adminH := NewAdminHandler(clientRepo, subRepo, testTemplatesDir)

	worker := service.NewMailWorker(config.Config{}, subRepo, 10)
	formService := service.NewFormService(subRepo, worker)
	publicH := NewPublicHandler(clientRepo, formService, testTemplatesDir)

	db.Close() // toute requête SQL suivante échouera désormais.

	return clientRepo, subRepo, adminH, publicH
}

func TestAdminHandler_Dashboard_DBError(t *testing.T) {
	_, _, adminH, _ := closedDBEnv(t)

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	rec := httptest.NewRecorder()
	adminH.Dashboard(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("attendu 500, obtenu %d", rec.Code)
	}
}

func TestAdminHandler_ClientsPage_DBError(t *testing.T) {
	_, _, adminH, _ := closedDBEnv(t)

	req := httptest.NewRequest(http.MethodGet, "/admin/clients", nil)
	rec := httptest.NewRecorder()
	adminH.ClientsPage(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("attendu 500, obtenu %d", rec.Code)
	}
}

func TestAdminHandler_CreateClient_DBError(t *testing.T) {
	_, _, adminH, _ := closedDBEnv(t)

	req := httptest.NewRequest(http.MethodPost, "/admin/clients", nil)
	req.Form = map[string][]string{"name": {"X"}, "destination_email": {"x@example.com"}}
	rec := httptest.NewRecorder()
	adminH.CreateClient(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("attendu 500, obtenu %d", rec.Code)
	}
}

func TestAdminHandler_ToggleClient_DBError(t *testing.T) {
	_, _, adminH, _ := closedDBEnv(t)

	req := httptest.NewRequest(http.MethodPost, "/admin/clients/x/toggle", nil)
	req.SetPathValue("id", "x")
	rec := httptest.NewRecorder()
	adminH.ToggleClient(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("attendu 500, obtenu %d", rec.Code)
	}
}

func TestAdminHandler_EditClientForm_DBError(t *testing.T) {
	_, _, adminH, _ := closedDBEnv(t)

	req := httptest.NewRequest(http.MethodGet, "/admin/clients/x/edit", nil)
	req.SetPathValue("id", "x")
	rec := httptest.NewRecorder()
	adminH.EditClientForm(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("attendu 500, obtenu %d", rec.Code)
	}
}

func TestAdminHandler_UpdateClient_DBError(t *testing.T) {
	_, _, adminH, _ := closedDBEnv(t)

	req := httptest.NewRequest(http.MethodPut, "/admin/clients/x", nil)
	req.Form = map[string][]string{"name": {"X"}, "destination_email": {"x@example.com"}}
	req.SetPathValue("id", "x")
	rec := httptest.NewRecorder()
	adminH.UpdateClient(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("attendu 500, obtenu %d", rec.Code)
	}
}

func TestAdminHandler_DeleteClient_DBError(t *testing.T) {
	_, _, adminH, _ := closedDBEnv(t)

	req := httptest.NewRequest(http.MethodDelete, "/admin/clients/x", nil)
	req.SetPathValue("id", "x")
	rec := httptest.NewRecorder()
	adminH.DeleteClient(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("attendu 500, obtenu %d", rec.Code)
	}
}

func TestAdminHandler_LogsPage_DBError(t *testing.T) {
	_, _, adminH, _ := closedDBEnv(t)

	req := httptest.NewRequest(http.MethodGet, "/admin/logs", nil)
	rec := httptest.NewRecorder()
	adminH.LogsPage(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("attendu 500, obtenu %d", rec.Code)
	}
}

func TestAdminHandler_LogDetail_DBError(t *testing.T) {
	_, _, adminH, _ := closedDBEnv(t)

	req := httptest.NewRequest(http.MethodGet, "/admin/logs/x", nil)
	req.SetPathValue("id", "x")
	rec := httptest.NewRecorder()
	adminH.LogDetail(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("attendu 500, obtenu %d", rec.Code)
	}
}

func TestPublicHandler_Submit_DBError(t *testing.T) {
	_, _, _, publicH := closedDBEnv(t)

	req := newSubmitRequest("x", nil)
	rec := httptest.NewRecorder()
	publicH.Submit(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("attendu 500, obtenu %d", rec.Code)
	}
}

func TestPublicHandler_Submit_ParseFormError(t *testing.T) {
	clientRepo, _, formService := newTestEnv(t)
	c := newTestClient(t, clientRepo, true)
	h := NewPublicHandler(clientRepo, formService, testTemplatesDir)

	req := httptest.NewRequest(http.MethodPost, "/f/"+c.ID, brokenReader{})
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetPathValue("client_hash", c.ID)
	rec := httptest.NewRecorder()
	h.Submit(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("attendu 400, obtenu %d", rec.Code)
	}
}
