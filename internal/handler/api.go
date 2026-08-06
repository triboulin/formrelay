package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"

	"formrelay-admin/internal/model"
	"formrelay-admin/internal/repository"
)

// APIHandler expose une API JSON minimale (en plus du panel HTMX) pour les
// scripts de provisioning : création/liste de clients sans avoir à parser
// les fragments HTML retournés par /admin/clients. Montée sous /api/ et
// protégée par la même Basic Auth (ADMIN_USER/ADMIN_PASS) que /admin.
type APIHandler struct {
	clientRepo *repository.ClientRepository
}

func NewAPIHandler(clientRepo *repository.ClientRepository) *APIHandler {
	return &APIHandler{clientRepo: clientRepo}
}

type createClientRequest struct {
	Name             string `json:"name"`
	DestinationEmail string `json:"destination_email"`
}

// clientResponse est la représentation JSON publique d'un client. Endpoint
// est dérivé de l'ID et rappelé explicitement pour éviter aux clients de
// l'API de devoir reconstruire eux-mêmes le chemin /f/{id}.
type clientResponse struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	DestinationEmail string `json:"destination_email"`
	Active           bool   `json:"active"`
	Endpoint         string `json:"endpoint"`
	CreatedAt        string `json:"created_at,omitempty"`
}

func toClientResponse(c model.Client) clientResponse {
	resp := clientResponse{
		ID:               c.ID,
		Name:             c.Name,
		DestinationEmail: c.DestinationEmail,
		Active:           c.Active,
		Endpoint:         "/f/" + c.ID,
	}
	if !c.CreatedAt.IsZero() {
		resp.CreatedAt = c.CreatedAt.Format(time.RFC3339)
	}
	return resp
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("erreur encodage JSON: %v", err)
	}
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

// CreateClient gère POST /api/clients. Corps JSON attendu :
// {"name": "...", "destination_email": "..."}. Retourne le client créé
// (201) avec son endpoint /f/{id} prêt à être intégré côté site.
func (h *APIHandler) CreateClient(w http.ResponseWriter, r *http.Request) {
	var req createClientRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "corps JSON invalide")
		return
	}

	if req.Name == "" || req.DestinationEmail == "" {
		writeJSONError(w, http.StatusBadRequest, "name et destination_email requis")
		return
	}

	c := model.Client{
		ID:               uuid.NewString(),
		Name:             req.Name,
		DestinationEmail: req.DestinationEmail,
		Active:           true,
	}

	if err := h.clientRepo.Create(c); err != nil {
		log.Printf("erreur création client (API): %v", err)
		writeJSONError(w, http.StatusInternalServerError, "erreur interne")
		return
	}

	created, err := h.clientRepo.GetByID(c.ID)
	if err != nil || created == nil {
		log.Printf("erreur relecture client créé (API) %s: %v", c.ID, err)
		writeJSONError(w, http.StatusInternalServerError, "erreur interne")
		return
	}

	writeJSON(w, http.StatusCreated, toClientResponse(*created))
}

// ListClients gère GET /api/clients. Permet à un script de vérifier si un
// client existe déjà (par nom) avant d'en créer un doublon.
func (h *APIHandler) ListClients(w http.ResponseWriter, r *http.Request) {
	clients, err := h.clientRepo.ListWithStats()
	if err != nil {
		log.Printf("erreur liste clients (API): %v", err)
		writeJSONError(w, http.StatusInternalServerError, "erreur interne")
		return
	}

	resp := make([]clientResponse, 0, len(clients))
	for _, c := range clients {
		resp = append(resp, toClientResponse(c))
	}
	writeJSON(w, http.StatusOK, resp)
}
