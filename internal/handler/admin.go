package handler

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/google/uuid"

	"formrelay-admin/internal/model"
	"formrelay-admin/internal/repository"
)

// AdminHandler regroupe les handlers du panel d'administration HTMX.
type AdminHandler struct {
	clientRepo *repository.ClientRepository
	subRepo    *repository.SubmissionRepository
	tmpl       *templates
}

func NewAdminHandler(clientRepo *repository.ClientRepository, subRepo *repository.SubmissionRepository, templatesDir string) *AdminHandler {
	return &AdminHandler{
		clientRepo: clientRepo,
		subRepo:    subRepo,
		tmpl:       newTemplates(templatesDir),
	}
}

// Dashboard gère GET /admin
func (h *AdminHandler) Dashboard(w http.ResponseWriter, r *http.Request) {
	stats, err := h.subRepo.Stats()
	if err != nil {
		log.Printf("erreur stats: %v", err)
		http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
		return
	}
	totalClients, activeClients, err := h.clientRepo.Count()
	if err != nil {
		log.Printf("erreur count clients: %v", err)
		http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
		return
	}
	stats.TotalClients = totalClients
	stats.ActiveClients = activeClients

	recent, err := h.subRepo.Recent(10)
	if err != nil {
		log.Printf("erreur recent: %v", err)
		http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
		return
	}

	h.tmpl.renderPage(w, h.tmpl.dashboard, "dashboard", "Dashboard", "dashboard", map[string]any{
		"Stats":  stats,
		"Recent": recent,
	})
}

// ClientsPage gère GET /admin/clients
func (h *AdminHandler) ClientsPage(w http.ResponseWriter, r *http.Request) {
	clients, err := h.clientRepo.ListWithStats()
	if err != nil {
		log.Printf("erreur liste clients: %v", err)
		http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
		return
	}
	h.tmpl.renderPage(w, h.tmpl.clients, "clients", "Clients", "clients", clients)
}

// CreateClient gère POST /admin/clients
func (h *AdminHandler) CreateClient(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "400 Bad Request", http.StatusBadRequest)
		return
	}

	name := r.FormValue("name")
	destEmail := r.FormValue("destination_email")

	if name == "" || destEmail == "" {
		http.Error(w, "400 Bad Request: nom et email de destination requis", http.StatusBadRequest)
		return
	}

	c := model.Client{
		ID:               uuid.NewString(),
		Name:             name,
		DestinationEmail: destEmail,
		Active:           true,
	}

	if err := h.clientRepo.Create(c); err != nil {
		log.Printf("erreur création client: %v", err)
		http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
		return
	}

	clients, err := h.clientRepo.ListWithStats()
	if err != nil {
		log.Printf("erreur liste clients: %v", err)
		http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
		return
	}
	renderFragment(w, h.tmpl.clients, "clients_table", clients)
}

// EditClientForm gère GET /admin/clients/{id}/edit - fragment affiché dans le modal d'édition.
func (h *AdminHandler) EditClientForm(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	c, err := h.clientRepo.GetByID(id)
	if err != nil {
		log.Printf("erreur lecture client %s: %v", id, err)
		http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
		return
	}
	if c == nil {
		http.Error(w, "404 Not Found", http.StatusNotFound)
		return
	}
	renderFragment(w, h.tmpl.clients, "client_edit_form", *c)
}

// UpdateClient gère PUT /admin/clients/{id}. Le nom et l'email de destination
// peuvent être modifiés, mais l'identifiant (et donc l'endpoint /f/{id}) ne change jamais.
func (h *AdminHandler) UpdateClient(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if err := r.ParseForm(); err != nil {
		http.Error(w, "400 Bad Request", http.StatusBadRequest)
		return
	}

	name := r.FormValue("name")
	destEmail := r.FormValue("destination_email")
	if name == "" || destEmail == "" {
		http.Error(w, "400 Bad Request: nom et email de destination requis", http.StatusBadRequest)
		return
	}

	existing, err := h.clientRepo.GetByID(id)
	if err != nil {
		log.Printf("erreur lecture client %s: %v", id, err)
		http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
		return
	}
	if existing == nil {
		http.Error(w, "404 Not Found", http.StatusNotFound)
		return
	}

	if err := h.clientRepo.Update(id, name, destEmail); err != nil {
		log.Printf("erreur mise à jour client %s: %v", id, err)
		http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
		return
	}

	clients, err := h.clientRepo.ListWithStats()
	if err != nil {
		log.Printf("erreur liste clients: %v", err)
		http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
		return
	}
	renderFragment(w, h.tmpl.clients, "clients_table", clients)
}

// ToggleClient gère POST /admin/clients/{id}/toggle
func (h *AdminHandler) ToggleClient(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.clientRepo.ToggleActive(id); err != nil {
		log.Printf("erreur toggle client %s: %v", id, err)
		http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
		return
	}
	c, err := h.clientRepo.GetByID(id)
	if err != nil || c == nil {
		http.Error(w, "404 Not Found", http.StatusNotFound)
		return
	}
	renderFragment(w, h.tmpl.clients, "client_row", *c)
}

// DeleteClient gère DELETE /admin/clients/{id}
func (h *AdminHandler) DeleteClient(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.clientRepo.Delete(id); err != nil {
		log.Printf("erreur suppression client %s: %v", id, err)
		http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	// Réponse vide : HTMX (hx-swap="outerHTML") retire la ligne du tableau.
}

// LogsPage gère GET /admin/logs (page complète ou fragment filtré via HTMX)
func (h *AdminHandler) LogsPage(w http.ResponseWriter, r *http.Request) {
	filter := repository.ListFilter{
		ClientID: r.URL.Query().Get("client_id"),
		Status:   r.URL.Query().Get("status"),
		PageSize: 25,
	}
	if p, err := strconv.Atoi(r.URL.Query().Get("page")); err == nil && p > 0 {
		filter.Page = p
	} else {
		filter.Page = 1
	}

	subs, total, err := h.subRepo.List(filter)
	if err != nil {
		log.Printf("erreur liste soumissions: %v", err)
		http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
		return
	}
	totalPages := (total + filter.PageSize - 1) / filter.PageSize
	if totalPages == 0 {
		totalPages = 1
	}

	data := map[string]any{
		"Submissions": subs,
		"Total":       total,
		"TotalPages":  totalPages,
		"Filter":      filter,
	}

	// Si la requête vient de HTMX (filtre ou pagination), on ne renvoie que le fragment.
	if r.Header.Get("HX-Request") == "true" {
		renderFragment(w, h.tmpl.logs, "logs_table", data)
		return
	}

	clients, err := h.clientRepo.ListWithStats()
	if err != nil {
		log.Printf("erreur liste clients: %v", err)
		http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
		return
	}
	data["Clients"] = clients
	h.tmpl.renderPage(w, h.tmpl.logs, "logs", "Logs", "logs", data)
}

// LogDetail gère GET /admin/logs/{id} - fragment affiché dans l'offcanvas.
func (h *AdminHandler) LogDetail(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	sub, err := h.subRepo.GetByID(id)
	if err != nil {
		log.Printf("erreur lecture soumission %s: %v", id, err)
		http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
		return
	}
	if sub == nil {
		http.Error(w, "404 Not Found", http.StatusNotFound)
		return
	}

	pretty := sub.Payload
	var raw map[string]any
	if err := json.Unmarshal([]byte(sub.Payload), &raw); err == nil {
		var buf bytes.Buffer
		if err := json.Indent(&buf, []byte(sub.Payload), "", "  "); err == nil {
			pretty = buf.String()
		}
	}

	renderFragment(w, h.tmpl.logs, "log_detail", struct {
		model.Submission
		PayloadPretty string
	}{*sub, pretty})
}
