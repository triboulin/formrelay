package handler

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/google/uuid"

	"formrelay-admin/internal/model"
	"formrelay-admin/internal/repository"
)

func newAdminTestEnv(t *testing.T) (*repository.ClientRepository, *repository.SubmissionRepository, *AdminHandler) {
	t.Helper()
	db := newTestDB(t)
	clientRepo := repository.NewClientRepository(db)
	subRepo := repository.NewSubmissionRepository(db)
	h := NewAdminHandler(clientRepo, subRepo, testTemplatesDir)
	return clientRepo, subRepo, h
}

func TestAdminHandler_Dashboard(t *testing.T) {
	clientRepo, subRepo, h := newAdminTestEnv(t)
	c := newTestClient(t, clientRepo, true)
	if err := subRepo.Create(model.Submission{ID: uuid.NewString(), ClientID: c.ID, SenderIP: "1.2.3.4", Payload: "{}", Status: model.StatusSuccess}); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	rec := httptest.NewRecorder()
	h.Dashboard(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("attendu 200, obtenu %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "Dashboard") {
		t.Error("la page dashboard devrait contenir le titre 'Dashboard'")
	}
	if !strings.Contains(rec.Body.String(), c.Name) {
		t.Error("la page dashboard devrait lister le client récent")
	}
}

func TestAdminHandler_ClientsPage(t *testing.T) {
	clientRepo, _, h := newAdminTestEnv(t)
	c := newTestClient(t, clientRepo, true)

	req := httptest.NewRequest(http.MethodGet, "/admin/clients", nil)
	rec := httptest.NewRecorder()
	h.ClientsPage(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("attendu 200, obtenu %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), c.Name) {
		t.Error("la page clients devrait lister le client créé")
	}
}

func TestAdminHandler_ClientsPage_Empty(t *testing.T) {
	_, _, h := newAdminTestEnv(t)

	req := httptest.NewRequest(http.MethodGet, "/admin/clients", nil)
	rec := httptest.NewRecorder()
	h.ClientsPage(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("attendu 200, obtenu %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Aucun client") {
		t.Error("le message d'état vide devrait apparaître")
	}
}

func TestAdminHandler_CreateClient_Success(t *testing.T) {
	clientRepo, _, h := newAdminTestEnv(t)

	form := url.Values{"name": {"Nouveau Client"}, "destination_email": {"dest@example.com"}}
	req := httptest.NewRequest(http.MethodPost, "/admin/clients", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	h.CreateClient(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("attendu 200, obtenu %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "Nouveau Client") {
		t.Error("le fragment retourné devrait contenir le nouveau client")
	}

	clients, err := clientRepo.ListWithStats()
	if err != nil {
		t.Fatal(err)
	}
	if len(clients) != 1 {
		t.Fatalf("attendu 1 client en base, obtenu %d", len(clients))
	}
}

func TestAdminHandler_CreateClient_MissingFields(t *testing.T) {
	_, _, h := newAdminTestEnv(t)

	form := url.Values{"name": {""}, "destination_email": {""}}
	req := httptest.NewRequest(http.MethodPost, "/admin/clients", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	h.CreateClient(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("attendu 400, obtenu %d", rec.Code)
	}
}

func TestAdminHandler_EditClientForm(t *testing.T) {
	clientRepo, _, h := newAdminTestEnv(t)
	c := newTestClient(t, clientRepo, true)

	req := httptest.NewRequest(http.MethodGet, "/admin/clients/"+c.ID+"/edit", nil)
	req.SetPathValue("id", c.ID)
	rec := httptest.NewRecorder()
	h.EditClientForm(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("attendu 200, obtenu %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), c.Name) {
		t.Error("le formulaire d'édition devrait pré-remplir le nom actuel")
	}
	if !strings.Contains(rec.Body.String(), c.DestinationEmail) {
		t.Error("le formulaire d'édition devrait pré-remplir l'email actuel")
	}
	if !strings.Contains(rec.Body.String(), "hx-put=\"/admin/clients/"+c.ID+"\"") {
		t.Error("le formulaire devrait soumettre en PUT vers /admin/clients/{id}")
	}
}

func TestAdminHandler_EditClientForm_NotFound(t *testing.T) {
	_, _, h := newAdminTestEnv(t)

	req := httptest.NewRequest(http.MethodGet, "/admin/clients/inexistant/edit", nil)
	req.SetPathValue("id", "inexistant")
	rec := httptest.NewRecorder()
	h.EditClientForm(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("attendu 404, obtenu %d", rec.Code)
	}
}

func TestAdminHandler_UpdateClient_Success(t *testing.T) {
	clientRepo, _, h := newAdminTestEnv(t)
	c := newTestClient(t, clientRepo, true)

	form := url.Values{"name": {"Acme Renamed"}, "destination_email": {"new@example.com"}}
	req := httptest.NewRequest(http.MethodPut, "/admin/clients/"+c.ID, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetPathValue("id", c.ID)
	rec := httptest.NewRecorder()
	h.UpdateClient(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("attendu 200, obtenu %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "Acme Renamed") {
		t.Error("le fragment retourné devrait contenir le nom mis à jour")
	}

	got, err := clientRepo.GetByID(c.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != c.ID {
		t.Errorf("ID = %q, ne devrait jamais changer (attendu %q)", got.ID, c.ID)
	}
	if got.Name != "Acme Renamed" || got.DestinationEmail != "new@example.com" {
		t.Errorf("client non mis à jour: %+v", got)
	}
}

func TestAdminHandler_UpdateClient_NotFound(t *testing.T) {
	_, _, h := newAdminTestEnv(t)

	form := url.Values{"name": {"X"}, "destination_email": {"x@example.com"}}
	req := httptest.NewRequest(http.MethodPut, "/admin/clients/inexistant", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetPathValue("id", "inexistant")
	rec := httptest.NewRecorder()
	h.UpdateClient(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("attendu 404, obtenu %d", rec.Code)
	}
}

func TestAdminHandler_UpdateClient_MissingFields(t *testing.T) {
	clientRepo, _, h := newAdminTestEnv(t)
	c := newTestClient(t, clientRepo, true)

	form := url.Values{"name": {""}, "destination_email": {""}}
	req := httptest.NewRequest(http.MethodPut, "/admin/clients/"+c.ID, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetPathValue("id", c.ID)
	rec := httptest.NewRecorder()
	h.UpdateClient(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("attendu 400, obtenu %d", rec.Code)
	}
}

func TestAdminHandler_ToggleClient(t *testing.T) {
	clientRepo, _, h := newAdminTestEnv(t)
	c := newTestClient(t, clientRepo, true)

	req := httptest.NewRequest(http.MethodPost, "/admin/clients/"+c.ID+"/toggle", nil)
	req.SetPathValue("id", c.ID)
	rec := httptest.NewRecorder()
	h.ToggleClient(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("attendu 200, obtenu %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Inactif") {
		t.Errorf("le fragment devrait indiquer le client comme inactif après toggle: %s", rec.Body.String())
	}

	got, err := clientRepo.GetByID(c.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Active {
		t.Error("le client devrait être inactif en base après toggle")
	}
}

func TestAdminHandler_ToggleClient_NotFound(t *testing.T) {
	_, _, h := newAdminTestEnv(t)

	req := httptest.NewRequest(http.MethodPost, "/admin/clients/inexistant/toggle", nil)
	req.SetPathValue("id", "inexistant")
	rec := httptest.NewRecorder()
	h.ToggleClient(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("attendu 404, obtenu %d", rec.Code)
	}
}

func TestAdminHandler_DeleteClient(t *testing.T) {
	clientRepo, _, h := newAdminTestEnv(t)
	c := newTestClient(t, clientRepo, true)

	req := httptest.NewRequest(http.MethodDelete, "/admin/clients/"+c.ID, nil)
	req.SetPathValue("id", c.ID)
	rec := httptest.NewRecorder()
	h.DeleteClient(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("attendu 200, obtenu %d", rec.Code)
	}

	got, err := clientRepo.GetByID(c.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Error("le client aurait dû être supprimé")
	}
}

func TestAdminHandler_LogsPage_FullPage(t *testing.T) {
	clientRepo, subRepo, h := newAdminTestEnv(t)
	c := newTestClient(t, clientRepo, true)
	if err := subRepo.Create(model.Submission{ID: uuid.NewString(), ClientID: c.ID, SenderIP: "1.2.3.4", Payload: "{}", Status: model.StatusSuccess}); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/admin/logs", nil)
	rec := httptest.NewRecorder()
	h.LogsPage(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("attendu 200, obtenu %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "Logs des soumissions") {
		t.Error("la page complète devrait contenir le titre")
	}
}

func TestAdminHandler_LogsPage_HTMXFragment(t *testing.T) {
	clientRepo, subRepo, h := newAdminTestEnv(t)
	c := newTestClient(t, clientRepo, true)
	if err := subRepo.Create(model.Submission{ID: uuid.NewString(), ClientID: c.ID, SenderIP: "1.2.3.4", Payload: "{}", Status: model.StatusBlocked}); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/admin/logs?status=BLOCKED", nil)
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()
	h.LogsPage(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("attendu 200, obtenu %d", rec.Code)
	}
	// Le fragment ne doit pas contenir la mise en page complète (pas de balise <html>).
	if strings.Contains(rec.Body.String(), "<html") {
		t.Error("une réponse HTMX ne devrait pas contenir la mise en page complète")
	}
	if !strings.Contains(rec.Body.String(), "BLOCKED") {
		t.Error("le fragment devrait contenir la soumission bloquée")
	}
}

func TestAdminHandler_LogsPage_Pagination(t *testing.T) {
	clientRepo, subRepo, h := newAdminTestEnv(t)
	c := newTestClient(t, clientRepo, true)
	for i := 0; i < 3; i++ {
		if err := subRepo.Create(model.Submission{ID: uuid.NewString(), ClientID: c.ID, SenderIP: "1.2.3.4", Payload: "{}", Status: model.StatusSuccess}); err != nil {
			t.Fatal(err)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/admin/logs?page=2", nil)
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()
	h.LogsPage(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("attendu 200, obtenu %d", rec.Code)
	}
}

func TestAdminHandler_LogDetail(t *testing.T) {
	clientRepo, subRepo, h := newAdminTestEnv(t)
	c := newTestClient(t, clientRepo, true)
	sub := model.Submission{ID: uuid.NewString(), ClientID: c.ID, SenderIP: "1.2.3.4", Payload: `{"name":"John"}`, Status: model.StatusSuccess, Subject: "Bonjour"}
	if err := subRepo.Create(sub); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/admin/logs/"+sub.ID, nil)
	req.SetPathValue("id", sub.ID)
	rec := httptest.NewRecorder()
	h.LogDetail(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("attendu 200, obtenu %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "John") {
		t.Error("le détail devrait contenir le payload formaté")
	}
	if !strings.Contains(rec.Body.String(), "Bonjour") {
		t.Error("le détail devrait contenir le sujet")
	}
}

func TestAdminHandler_LogDetail_NotFound(t *testing.T) {
	_, _, h := newAdminTestEnv(t)

	req := httptest.NewRequest(http.MethodGet, "/admin/logs/inexistant", nil)
	req.SetPathValue("id", "inexistant")
	rec := httptest.NewRecorder()
	h.LogDetail(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("attendu 404, obtenu %d", rec.Code)
	}
}

func TestAdminHandler_LogDetail_InvalidJSONPayload(t *testing.T) {
	clientRepo, subRepo, h := newAdminTestEnv(t)
	c := newTestClient(t, clientRepo, true)
	sub := model.Submission{ID: uuid.NewString(), ClientID: c.ID, SenderIP: "1.2.3.4", Payload: `not-json`, Status: model.StatusFailed, ErrorMessage: "boom"}
	if err := subRepo.Create(sub); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/admin/logs/"+sub.ID, nil)
	req.SetPathValue("id", sub.ID)
	rec := httptest.NewRecorder()
	h.LogDetail(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("attendu 200, obtenu %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "not-json") {
		t.Error("le payload brut devrait être affiché tel quel s'il n'est pas du JSON valide")
	}
	if !strings.Contains(rec.Body.String(), "boom") {
		t.Error("le message d'erreur devrait être affiché")
	}
}
