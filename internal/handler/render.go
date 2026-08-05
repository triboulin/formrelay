package handler

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
)

// funcMap regroupe les fonctions utilitaires disponibles dans tous les templates admin.
var funcMap = template.FuncMap{
	"statusColor": func(status any) string {
		switch fmt.Sprintf("%v", status) {
		case "SUCCESS":
			return "bg-green-100 text-green-700"
		case "FAILED":
			return "bg-red-100 text-red-700"
		case "BLOCKED":
			return "bg-yellow-100 text-yellow-700"
		default:
			return "bg-gray-100 text-gray-700"
		}
	},
	"add": func(a, b int) int { return a + b },
	"sub": func(a, b int) int { return a - b },
}

// templates regroupe les templates admin. Chaque "feature" (dashboard, clients,
// logs) est compilée dans son propre *template.Template afin d'éviter toute
// collision de noms entre les {{define}} de fragments partagés (ex: pagination).
// base.html est appliqué séparément autour du contenu déjà rendu de chaque page.
type templates struct {
	base      *template.Template
	dashboard *template.Template
	clients   *template.Template
	logs      *template.Template
}

func newTemplates(templatesDir string) *templates {
	dir := filepath.Join(templatesDir, "admin")
	base := func(name string) string { return filepath.Join(dir, name) }

	return &templates{
		base:      template.Must(template.New("base.html").Funcs(funcMap).ParseFiles(base("base.html"))),
		dashboard: template.Must(template.New("dashboard.html").Funcs(funcMap).ParseFiles(base("dashboard.html"))),
		clients:   template.Must(template.New("clients.html").Funcs(funcMap).ParseFiles(base("clients.html"))),
		logs:      template.Must(template.New("logs.html").Funcs(funcMap).ParseFiles(base("logs.html"))),
	}
}

type pageData struct {
	Title     string
	ActiveNav string
	Body      template.HTML
}

// renderPage exécute le template "name" du set fourni pour produire le contenu,
// puis l'injecte dans base.html pour obtenir la page complète.
func (t *templates) renderPage(w http.ResponseWriter, set *template.Template, contentName, title, activeNav string, data any) {
	var buf bytes.Buffer
	if err := set.ExecuteTemplate(&buf, contentName, data); err != nil {
		log.Printf("erreur rendu contenu %s: %v", contentName, err)
		http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	err := t.base.ExecuteTemplate(w, "base", pageData{
		Title:     title,
		ActiveNav: activeNav,
		Body:      template.HTML(buf.String()),
	})
	if err != nil {
		log.Printf("erreur rendu base: %v", err)
	}
}

// renderFragment exécute un fragment précis du set fourni (réponse HTMX partielle).
func renderFragment(w http.ResponseWriter, set *template.Template, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := set.ExecuteTemplate(w, name, data); err != nil {
		log.Printf("erreur rendu fragment %s: %v", name, err)
	}
}
