package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"formrelay-admin/internal/repository"
)

func newAPITestEnv(t *testing.T) (*repository.ClientRepository, *APIHandler) {
	t.Helper()
	db := newTestDB(t)
	clientRepo := repository.NewClientRepository(db)
	h := NewAPIHandler(clientRepo)
	return clientRepo, h
}

func TestAPIHandler_CreateClient_Success(t *testing.T) {
	_, h := newAPITestEnv(t)

	body := `{"name":"Acme","destination_email":"dest@example.com"}`
	req := httptest.NewRequest(http.MethodPost, "/api/clients", strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.CreateClient(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("attendu 201, obtenu %d: %s", rec.Code, rec.Body.String())
	}

	var got clientResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("réponse JSON invalide: %v", err)
	}
	if got.ID == "" {
		t.Error("l'ID du client devrait être renseigné")
	}
	if got.Endpoint != "/f/"+got.ID {
		t.Errorf("endpoint = %q, attendu /f/%s", got.Endpoint, got.ID)
	}
	if got.Name != "Acme" || got.DestinationEmail != "dest@example.com" {
		t.Errorf("client inattendu: %+v", got)
	}
	if !got.Active {
		t.Error("le client devrait être actif par défaut")
	}
	if got.CreatedAt == "" {
		t.Error("created_at devrait être renseigné")
	}
}

func TestAPIHandler_CreateClient_MissingFields(t *testing.T) {
	_, h := newAPITestEnv(t)

	body := `{"name":"","destination_email":""}`
	req := httptest.NewRequest(http.MethodPost, "/api/clients", strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.CreateClient(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("attendu 400, obtenu %d", rec.Code)
	}
}

func TestAPIHandler_CreateClient_InvalidJSON(t *testing.T) {
	_, h := newAPITestEnv(t)

	req := httptest.NewRequest(http.MethodPost, "/api/clients", strings.NewReader("{invalid"))
	rec := httptest.NewRecorder()
	h.CreateClient(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("attendu 400, obtenu %d", rec.Code)
	}
}

func TestAPIHandler_ListClients(t *testing.T) {
	clientRepo, h := newAPITestEnv(t)
	newTestClient(t, clientRepo, true)

	req := httptest.NewRequest(http.MethodGet, "/api/clients", nil)
	rec := httptest.NewRecorder()
	h.ListClients(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("attendu 200, obtenu %d: %s", rec.Code, rec.Body.String())
	}

	var got []clientResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("réponse JSON invalide: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("attendu 1 client, obtenu %d", len(got))
	}
}

func TestAPIHandler_ListClients_Empty(t *testing.T) {
	_, h := newAPITestEnv(t)

	req := httptest.NewRequest(http.MethodGet, "/api/clients", nil)
	rec := httptest.NewRecorder()
	h.ListClients(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("attendu 200, obtenu %d", rec.Code)
	}
	if strings.TrimSpace(rec.Body.String()) != "[]" {
		t.Errorf("attendu tableau vide, obtenu %s", rec.Body.String())
	}
}
