package service

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"formrelay-admin/internal/config"
	"formrelay-admin/internal/database"
	"formrelay-admin/internal/model"
	"formrelay-admin/internal/repository"
)

// newTestRepos ouvre une base SQLite en mémoire et crée un client de test,
// retournant le repository de soumissions et le client créé.
func newTestRepos(t *testing.T) (*repository.SubmissionRepository, model.Client) {
	t.Helper()
	db, err := database.New(":memory:")
	if err != nil {
		t.Fatalf("erreur ouverture base de test: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	clientRepo := repository.NewClientRepository(db)
	c := model.Client{ID: uuid.NewString(), Name: "Acme", DestinationEmail: "dest@example.com", Active: true}
	if err := clientRepo.Create(c); err != nil {
		t.Fatalf("erreur création client de test: %v", err)
	}

	return repository.NewSubmissionRepository(db), c
}

func pollUntil(t *testing.T, timeout time.Duration, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("timeout en attente de la condition")
}

func TestMailWorker_Send_MissingSMTPHost(t *testing.T) {
	subRepo, _ := newTestRepos(t)
	cfg := config.Config{SMTPHost: "", FromEmail: "noreply@example.com", FromName: "FormRelay"}
	w := NewMailWorker(cfg, subRepo, 10)

	err := w.send(model.MailJob{SubmissionID: "x", To: "dest@example.com", Subject: "Test", Body: "Corps"})
	if err == nil {
		t.Fatal("attendu une erreur si SMTP_HOST n'est pas configuré")
	}
	if !strings.Contains(err.Error(), "SMTP non configuré") {
		t.Errorf("message d'erreur inattendu: %v", err)
	}
}

func TestMailWorker_SendSuccess_UpdatesSubmissionStatus(t *testing.T) {
	srv := startFakeSMTPServer(t)
	host, port := srv.hostPort(t)

	subRepo, c := newTestRepos(t)
	sub := model.Submission{ID: uuid.NewString(), ClientID: c.ID, SenderIP: "127.0.0.1", Payload: "{}", Status: model.StatusFailed, ErrorMessage: "en attente"}
	if err := subRepo.Create(sub); err != nil {
		t.Fatal(err)
	}

	cfg := config.Config{
		SMTPHost:  host,
		SMTPPort:  port,
		SMTPUser:  "user",
		SMTPPass:  "pass",
		FromEmail: "noreply@example.com",
		FromName:  "FormRelay",
	}
	w := NewMailWorker(cfg, subRepo, 10)
	w.Start(1)

	w.Enqueue(model.MailJob{
		SubmissionID: sub.ID,
		To:           c.DestinationEmail,
		ReplyTo:      "sender@example.com",
		Subject:      "Nouveau message",
		Body:         "Contenu du message",
	})

	pollUntil(t, 3*time.Second, func() bool {
		got, err := subRepo.GetByID(sub.ID)
		return err == nil && got != nil && got.Status == model.StatusSuccess
	})

	got, err := subRepo.GetByID(sub.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != model.StatusSuccess {
		t.Fatalf("Status = %q, attendu SUCCESS", got.Status)
	}
	if got.ErrorMessage != "" {
		t.Errorf("ErrorMessage = %q, attendu vide après succès", got.ErrorMessage)
	}

	pollUntil(t, 2*time.Second, func() bool { return srv.receivedCount() == 1 })
	msg := srv.lastMessage()
	if !strings.Contains(msg, "Nouveau message") {
		t.Errorf("le message reçu par le serveur SMTP ne contient pas le sujet attendu:\n%s", msg)
	}
	if !strings.Contains(msg, "Contenu du message") {
		t.Errorf("le message reçu par le serveur SMTP ne contient pas le corps attendu:\n%s", msg)
	}
}

func TestMailWorker_SendFailure_UpdatesSubmissionStatus(t *testing.T) {
	subRepo, c := newTestRepos(t)
	sub := model.Submission{ID: uuid.NewString(), ClientID: c.ID, SenderIP: "127.0.0.1", Payload: "{}", Status: model.StatusFailed, ErrorMessage: "en attente"}
	if err := subRepo.Create(sub); err != nil {
		t.Fatal(err)
	}

	// Port fermé : la connexion SMTP doit échouer et le statut rester FAILED avec un message d'erreur.
	cfg := config.Config{SMTPHost: "127.0.0.1", SMTPPort: 1, FromEmail: "noreply@example.com", FromName: "FormRelay"}
	w := NewMailWorker(cfg, subRepo, 10)
	w.Start(1)

	w.Enqueue(model.MailJob{SubmissionID: sub.ID, To: c.DestinationEmail, Subject: "Test", Body: "Corps"})

	pollUntil(t, 3*time.Second, func() bool {
		got, err := subRepo.GetByID(sub.ID)
		return err == nil && got != nil && got.ErrorMessage != "en attente"
	})

	got, err := subRepo.GetByID(sub.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != model.StatusFailed {
		t.Errorf("Status = %q, attendu FAILED", got.Status)
	}
	if got.ErrorMessage == "" {
		t.Error("ErrorMessage devrait être renseigné après un échec d'envoi")
	}
}
