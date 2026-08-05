package service

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/google/uuid"

	"formrelay-admin/internal/model"
	"formrelay-admin/internal/repository"
)

// controlFields liste les champs spéciaux du formulaire qui ne font pas partie
// du contenu métier à transmettre par email.
var controlFields = map[string]bool{
	"_next":    true,
	"_replyto": true,
	"_subject": true,
}

// FormService contient la logique métier de traitement d'une soumission de formulaire.
type FormService struct {
	subRepo *repository.SubmissionRepository
	mailer  *MailWorker
}

func NewFormService(subRepo *repository.SubmissionRepository, mailer *MailWorker) *FormService {
	return &FormService{subRepo: subRepo, mailer: mailer}
}

// ProcessResult résume le traitement effectué, utilisé par le handler HTTP
// pour construire la réponse (redirection, etc.).
type ProcessResult struct {
	Submission model.Submission
	NextURL    string
}

// Process traite une soumission: construction du payload, enregistrement en
// base et mise en queue de l'envoi SMTP.
func (s *FormService) Process(client model.Client, ip string, form url.Values) (ProcessResult, error) {
	payloadJSON, err := buildPayloadJSON(form)
	if err != nil {
		return ProcessResult{}, fmt.Errorf("construction payload: %w", err)
	}

	senderEmail := firstNonEmpty(form.Get("email"), form.Get("_replyto"))
	subject := firstNonEmpty(form.Get("subject"), form.Get("_subject"), fmt.Sprintf("Nouveau message via %s", client.Name))
	nextURL := form.Get("_next")

	sub := model.Submission{
		ID:          uuid.NewString(),
		ClientID:    client.ID,
		SenderIP:    ip,
		SenderEmail: senderEmail,
		Subject:     subject,
		Payload:     payloadJSON,
	}

	// Statut initial en attendant le résultat de l'envoi SMTP asynchrone.
	sub.Status = model.StatusFailed
	sub.ErrorMessage = "en attente d'envoi"
	if err := s.subRepo.Create(sub); err != nil {
		return ProcessResult{}, err
	}

	body := buildEmailBody(form)
	s.mailer.Enqueue(model.MailJob{
		SubmissionID: sub.ID,
		To:           client.DestinationEmail,
		ReplyTo:      senderEmail,
		Subject:      subject,
		Body:         body,
	})

	return ProcessResult{Submission: sub, NextURL: nextURL}, nil
}

func buildPayloadJSON(form url.Values) (string, error) {
	fields := make(map[string]string)
	for k, v := range form {
		if controlFields[k] || len(v) == 0 {
			continue
		}
		fields[k] = v[0]
	}
	b, err := json.Marshal(fields)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// buildEmailBody construit le corps de l'email relayé au format :
// "Nouvelle soumission de formulaire de la part de : NOM Prénom (email)"
// suivi du message.
func buildEmailBody(form url.Values) string {
	name := form.Get("name")
	email := firstNonEmpty(form.Get("email"), form.Get("_replyto"))
	message := form.Get("message")

	var b strings.Builder
	b.WriteString(fmt.Sprintf("Nouvelle soumission de formulaire de la part de : %s (%s)\n\n", name, email))
	b.WriteString(message)
	return b.String()
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
