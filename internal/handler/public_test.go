package handler

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"formrelay-admin/internal/model"
	"formrelay-admin/internal/repository"
)

func newSubmitRequest(clientID string, form url.Values) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/f/"+clientID, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetPathValue("client_hash", clientID)
	return req
}

func TestPublicHandler_Submit_UnknownClient(t *testing.T) {
	clientRepo, _, formService := newTestEnv(t)
	h := NewPublicHandler(clientRepo, formService, testTemplatesDir)

	req := newSubmitRequest("inexistant", url.Values{"name": {"John"}})
	rec := httptest.NewRecorder()
	h.Submit(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("attendu 404, obtenu %d", rec.Code)
	}
}

func TestPublicHandler_Submit_InactiveClient(t *testing.T) {
	clientRepo, _, formService := newTestEnv(t)
	c := newTestClient(t, clientRepo, false)
	h := NewPublicHandler(clientRepo, formService, testTemplatesDir)

	req := newSubmitRequest(c.ID, url.Values{"name": {"John"}})
	rec := httptest.NewRecorder()
	h.Submit(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("attendu 404 pour un client inactif, obtenu %d", rec.Code)
	}
}

func TestPublicHandler_Submit_SuccessWithRedirect(t *testing.T) {
	clientRepo, subRepo, formService := newTestEnv(t)
	c := newTestClient(t, clientRepo, true)
	h := NewPublicHandler(clientRepo, formService, testTemplatesDir)

	form := url.Values{"name": {"John"}, "email": {"john@test.com"}, "_next": {"https://acme.test/merci"}}
	req := newSubmitRequest(c.ID, form)
	rec := httptest.NewRecorder()
	h.Submit(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("attendu 303, obtenu %d", rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != "https://acme.test/merci" {
		t.Errorf("Location = %q", loc)
	}

	time.Sleep(50 * time.Millisecond)
	subs, total, err := subRepo.List(repository.ListFilter{ClientID: c.ID})
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 {
		t.Fatalf("attendu 1 soumission enregistrée, obtenu %d", total)
	}
	if subs[0].SenderEmail != "john@test.com" {
		t.Errorf("SenderEmail = %q", subs[0].SenderEmail)
	}
}

func TestPublicHandler_Submit_SuccessWithDefaultConfirmationPage(t *testing.T) {
	clientRepo, _, formService := newTestEnv(t)
	c := newTestClient(t, clientRepo, true)
	h := NewPublicHandler(clientRepo, formService, testTemplatesDir)

	req := newSubmitRequest(c.ID, url.Values{"name": {"John"}})
	rec := httptest.NewRecorder()
	h.Submit(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("attendu 200, obtenu %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Merci, votre message a bien été envoyé") {
		t.Errorf("la page de confirmation devrait contenir le message de remerciement, body = %q", body)
	}
	if strings.Contains(body, "vous répondra dès que possible") {
		t.Error("la page de confirmation ne devrait plus contenir la phrase 'vous répondra dès que possible'")
	}
}

func TestPublicHandler_Submit_HoneypotFieldNoLongerBlocks(t *testing.T) {
	clientRepo, subRepo, formService := newTestEnv(t)
	c := newTestClient(t, clientRepo, true)
	h := NewPublicHandler(clientRepo, formService, testTemplatesDir)

	// Le champ _honey, s'il est encore présent dans un formulaire existant, n'a plus d'effet.
	form := url.Values{"name": {"Visiteur"}, "_honey": {"quelque chose"}}
	req := newSubmitRequest(c.ID, form)
	rec := httptest.NewRecorder()
	h.Submit(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("attendu 200, obtenu %d", rec.Code)
	}

	subs, total, err := subRepo.List(repository.ListFilter{ClientID: c.ID})
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 {
		t.Fatalf("attendu 1 soumission, obtenu %d", total)
	}
	if subs[0].Status == model.StatusBlocked {
		t.Errorf("Status = %q, ne devrait plus jamais être BLOCKED", subs[0].Status)
	}
}
