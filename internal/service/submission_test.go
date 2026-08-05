package service

import (
	"encoding/json"
	"net/url"
	"strings"
	"testing"
	"time"

	"formrelay-admin/internal/config"
	"formrelay-admin/internal/model"
)

func TestFormService_Process_HoneypotFieldNoLongerBlocks(t *testing.T) {
	subRepo, c := newTestRepos(t)
	worker := NewMailWorker(config.Config{}, subRepo, 10)
	worker.Start(1)
	svc := NewFormService(subRepo, worker)

	// Le champ _honey n'a plus d'effet spécial : la soumission doit être traitée normalement.
	form := url.Values{}
	form.Set("_honey", "je suis un bot")
	form.Set("name", "Visiteur")
	form.Set("_next", "https://example.com/thanks")

	result, err := svc.Process(c, "203.0.113.1", form)
	if err != nil {
		t.Fatalf("Process() erreur: %v", err)
	}
	if result.Submission.Status == model.StatusBlocked {
		t.Error("le honeypot est supprimé : la soumission ne doit plus être marquée BLOCKED")
	}
	if result.NextURL != "https://example.com/thanks" {
		t.Errorf("NextURL = %q", result.NextURL)
	}

	got, err := subRepo.GetByID(result.Submission.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil {
		t.Fatal("la soumission aurait dû être enregistrée en base")
	}
	if got.Status == model.StatusBlocked {
		t.Errorf("Status en base = %q, ne devrait plus jamais être BLOCKED", got.Status)
	}
}

func TestFormService_Process_LegitSubmission_EnqueuesMail(t *testing.T) {
	srv := startFakeSMTPServer(t)
	host, port := srv.hostPort(t)

	subRepo, c := newTestRepos(t)
	cfg := config.Config{SMTPHost: host, SMTPPort: port, FromEmail: "noreply@example.com", FromName: "FormRelay"}
	worker := NewMailWorker(cfg, subRepo, 10)
	worker.Start(1)
	svc := NewFormService(subRepo, worker)

	form := url.Values{}
	form.Set("name", "John Doe")
	form.Set("email", "john@example.com")
	form.Set("message", "Bonjour, ceci est un test.")
	form.Set("_subject", "Contact depuis le site")

	result, err := svc.Process(c, "198.51.100.7", form)
	if err != nil {
		t.Fatalf("Process() erreur: %v", err)
	}
	if result.Submission.Subject != "Contact depuis le site" {
		t.Errorf("Subject = %q", result.Submission.Subject)
	}
	if result.Submission.SenderEmail != "john@example.com" {
		t.Errorf("SenderEmail = %q", result.Submission.SenderEmail)
	}

	var payload map[string]string
	if err := json.Unmarshal([]byte(result.Submission.Payload), &payload); err != nil {
		t.Fatalf("payload JSON invalide: %v", err)
	}
	if payload["name"] != "John Doe" {
		t.Errorf("payload[name] = %q", payload["name"])
	}
	if _, ok := payload["_subject"]; ok {
		t.Error("les champs de contrôle (_subject) ne doivent pas figurer dans le payload")
	}

	pollUntil(t, 3*time.Second, func() bool {
		got, err := subRepo.GetByID(result.Submission.ID)
		return err == nil && got != nil && got.Status == model.StatusSuccess
	})
}

func TestFormService_Process_EmailBodyFormat(t *testing.T) {
	srv := startFakeSMTPServer(t)
	host, port := srv.hostPort(t)

	subRepo, c := newTestRepos(t)
	cfg := config.Config{SMTPHost: host, SMTPPort: port, FromEmail: "noreply@example.com", FromName: "FormRelay"}
	worker := NewMailWorker(cfg, subRepo, 10)
	worker.Start(1)
	svc := NewFormService(subRepo, worker)

	form := url.Values{}
	form.Set("name", "Jean Dupont")
	form.Set("email", "jean@example.com")
	form.Set("message", "Bonjour, je souhaite un devis.")

	result, err := svc.Process(c, "198.51.100.7", form)
	if err != nil {
		t.Fatalf("Process() erreur: %v", err)
	}

	pollUntil(t, 3*time.Second, func() bool { return srv.receivedCount() == 1 })
	msg := srv.lastMessage()

	if !strings.Contains(msg, "Nouvelle soumission de formulaire de la part de : Jean Dupont (jean@example.com)") {
		t.Errorf("le corps du mail ne contient pas la ligne d'en-tête attendue:\n%s", msg)
	}
	if !strings.Contains(msg, "Bonjour, je souhaite un devis.") {
		t.Errorf("le corps du mail ne contient pas le message:\n%s", msg)
	}

	pollUntil(t, 3*time.Second, func() bool {
		got, err := subRepo.GetByID(result.Submission.ID)
		return err == nil && got != nil && got.Status == model.StatusSuccess
	})
}

func TestFormService_Process_SubjectFieldTakesPriorityOverUnderscoreSubject(t *testing.T) {
	subRepo, c := newTestRepos(t)
	worker := NewMailWorker(config.Config{}, subRepo, 10)
	worker.Start(1)
	svc := NewFormService(subRepo, worker)

	form := url.Values{}
	form.Set("subject", "Sujet saisi par le visiteur")
	form.Set("_subject", "Sujet par défaut du site")

	result, err := svc.Process(c, "127.0.0.1", form)
	if err != nil {
		t.Fatalf("Process() erreur: %v", err)
	}
	if result.Submission.Subject != "Sujet saisi par le visiteur" {
		t.Errorf("Subject = %q, attendu la valeur du champ 'subject' en priorité", result.Submission.Subject)
	}
}

func TestFormService_Process_UnderscoreSubjectFallback(t *testing.T) {
	subRepo, c := newTestRepos(t)
	worker := NewMailWorker(config.Config{}, subRepo, 10)
	worker.Start(1)
	svc := NewFormService(subRepo, worker)

	form := url.Values{}
	form.Set("_subject", "Sujet par défaut du site")

	result, err := svc.Process(c, "127.0.0.1", form)
	if err != nil {
		t.Fatalf("Process() erreur: %v", err)
	}
	if result.Submission.Subject != "Sujet par défaut du site" {
		t.Errorf("Subject = %q, attendu le fallback _subject en l'absence de champ 'subject'", result.Submission.Subject)
	}
}

func TestFormService_Process_SubjectFallback(t *testing.T) {
	subRepo, c := newTestRepos(t)
	worker := NewMailWorker(config.Config{}, subRepo, 10)
	worker.Start(1)
	svc := NewFormService(subRepo, worker)

	form := url.Values{}
	form.Set("message", "Sans sujet explicite")

	result, err := svc.Process(c, "127.0.0.1", form)
	if err != nil {
		t.Fatalf("Process() erreur: %v", err)
	}
	if !strings.Contains(result.Submission.Subject, c.Name) {
		t.Errorf("Subject = %q, attendu contenant le nom du client %q", result.Submission.Subject, c.Name)
	}
}

func TestFormService_Process_ReplyToFallback(t *testing.T) {
	subRepo, c := newTestRepos(t)
	worker := NewMailWorker(config.Config{}, subRepo, 10)
	worker.Start(1)
	svc := NewFormService(subRepo, worker)

	form := url.Values{}
	form.Set("_replyto", "reply@example.com")

	result, err := svc.Process(c, "127.0.0.1", form)
	if err != nil {
		t.Fatalf("Process() erreur: %v", err)
	}
	if result.Submission.SenderEmail != "reply@example.com" {
		t.Errorf("SenderEmail = %q, attendu reply@example.com (fallback _replyto)", result.Submission.SenderEmail)
	}
}
