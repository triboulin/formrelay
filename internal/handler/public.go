package handler

import (
	"html/template"
	"log"
	"net/http"
	"path"

	"formrelay-admin/internal/middleware"
	"formrelay-admin/internal/repository"
	"formrelay-admin/internal/service"
)

// PublicHandler gère la réception des formulaires côté public.
type PublicHandler struct {
	clientRepo  *repository.ClientRepository
	formService *service.FormService
	confirmTmpl *template.Template
}

func NewPublicHandler(clientRepo *repository.ClientRepository, formService *service.FormService, templatesDir string) *PublicHandler {
	tmpl := template.Must(template.ParseFiles(path.Join(templatesDir, "public", "confirmation.html")))
	return &PublicHandler{
		clientRepo:  clientRepo,
		formService: formService,
		confirmTmpl: tmpl,
	}
}

// Submit gère POST /f/{client_hash}
func (h *PublicHandler) Submit(w http.ResponseWriter, r *http.Request) {
	clientHash := r.PathValue("client_hash")
	if clientHash == "" {
		http.NotFound(w, r)
		return
	}

	client, err := h.clientRepo.GetByID(clientHash)
	if err != nil {
		log.Printf("erreur lecture client %s: %v", clientHash, err)
		http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
		return
	}
	if client == nil || !client.Active {
		http.Error(w, "404 Not Found: client inconnu ou inactif", http.StatusNotFound)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "400 Bad Request: formulaire invalide", http.StatusBadRequest)
		return
	}

	ip := middleware.ClientIP(r)

	result, err := h.formService.Process(*client, ip, r.Form)
	if err != nil {
		log.Printf("erreur traitement soumission pour client %s: %v", client.ID, err)
		http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
		return
	}

	if result.NextURL != "" {
		http.Redirect(w, r, result.NextURL, http.StatusSeeOther)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.confirmTmpl.Execute(w, nil); err != nil {
		log.Printf("erreur rendu template confirmation: %v", err)
	}
}
