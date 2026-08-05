package handler

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewTemplates_LoadsAllSets(t *testing.T) {
	tmpl := newTemplates(testTemplatesDir)
	if tmpl.base == nil || tmpl.dashboard == nil || tmpl.clients == nil || tmpl.logs == nil {
		t.Fatal("tous les sets de templates devraient être chargés")
	}
}

func TestRenderPage_Success(t *testing.T) {
	tmpl := newTemplates(testTemplatesDir)
	rec := httptest.NewRecorder()

	tmpl.renderPage(rec, tmpl.dashboard, "dashboard", "Dashboard", "dashboard", map[string]any{
		"Stats":  map[string]any{},
		"Recent": nil,
	})

	if rec.Code != 200 {
		t.Fatalf("attendu 200 par défaut, obtenu %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "<html") {
		t.Error("la page complète devrait inclure la mise en page de base")
	}
}

func TestRenderPage_ContentTemplateError(t *testing.T) {
	tmpl := newTemplates(testTemplatesDir)
	rec := httptest.NewRecorder()

	// Un nom de contenu inexistant doit provoquer une erreur 500, pas un panic.
	tmpl.renderPage(rec, tmpl.dashboard, "nom_de_template_inexistant", "Dashboard", "dashboard", nil)

	if rec.Code != 500 {
		t.Fatalf("attendu 500, obtenu %d", rec.Code)
	}
}

func TestRenderFragment(t *testing.T) {
	tmpl := newTemplates(testTemplatesDir)
	rec := httptest.NewRecorder()

	renderFragment(rec, tmpl.clients, "clients_table", []any{})

	if !strings.Contains(rec.Body.String(), "Aucun client") {
		t.Error("le fragment devrait afficher l'état vide")
	}
}

func TestRenderFragment_UnknownName_DoesNotPanic(t *testing.T) {
	tmpl := newTemplates(testTemplatesDir)
	rec := httptest.NewRecorder()

	// Ne doit pas paniquer même si le nom de fragment n'existe pas.
	renderFragment(rec, tmpl.clients, "fragment_inexistant", nil)
}
