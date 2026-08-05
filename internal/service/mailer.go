package service

import (
	"fmt"
	"log"
	"net/smtp"

	"formrelay-admin/internal/config"
	"formrelay-admin/internal/model"
	"formrelay-admin/internal/repository"
)

// MailWorker découple la réception HTTP du formulaire de l'envoi SMTP effectif,
// via une queue en mémoire (channel Go), afin d'éviter les timeouts HTTP en cas
// de latence SMTP.
type MailWorker struct {
	cfg     config.Config
	queue   chan model.MailJob
	subRepo *repository.SubmissionRepository
}

// NewMailWorker crée un worker avec une queue bufferisée.
func NewMailWorker(cfg config.Config, subRepo *repository.SubmissionRepository, queueSize int) *MailWorker {
	return &MailWorker{
		cfg:     cfg,
		queue:   make(chan model.MailJob, queueSize),
		subRepo: subRepo,
	}
}

// Enqueue ajoute un job d'envoi à la queue. Non bloquant si la queue a de la place.
func (w *MailWorker) Enqueue(job model.MailJob) {
	w.queue <- job
}

// Start démarre N goroutines consommatrices de la queue.
func (w *MailWorker) Start(workers int) {
	for i := 0; i < workers; i++ {
		go w.loop()
	}
}

func (w *MailWorker) loop() {
	for job := range w.queue {
		err := w.send(job)
		if err != nil {
			log.Printf("échec envoi email pour soumission %s: %v", job.SubmissionID, err)
			if uErr := w.subRepo.UpdateStatus(job.SubmissionID, model.StatusFailed, err.Error()); uErr != nil {
				log.Printf("échec mise à jour statut soumission %s: %v", job.SubmissionID, uErr)
			}
			continue
		}
		if uErr := w.subRepo.UpdateStatus(job.SubmissionID, model.StatusSuccess, ""); uErr != nil {
			log.Printf("échec mise à jour statut soumission %s: %v", job.SubmissionID, uErr)
		}
	}
}

func (w *MailWorker) send(job model.MailJob) error {
	if w.cfg.SMTPHost == "" {
		return fmt.Errorf("SMTP non configuré (SMTP_HOST manquant)")
	}

	addr := fmt.Sprintf("%s:%d", w.cfg.SMTPHost, w.cfg.SMTPPort)
	auth := smtp.PlainAuth("", w.cfg.SMTPUser, w.cfg.SMTPPass, w.cfg.SMTPHost)

	from := w.cfg.FromEmail
	headers := make(map[string]string)
	headers["From"] = fmt.Sprintf("%s <%s>", w.cfg.FromName, from)
	headers["To"] = job.To
	if job.ReplyTo != "" {
		headers["Reply-To"] = job.ReplyTo
	}
	headers["Subject"] = job.Subject
	headers["MIME-Version"] = "1.0"
	headers["Content-Type"] = "text/plain; charset=\"utf-8\""

	var msg string
	for k, v := range headers {
		msg += fmt.Sprintf("%s: %s\r\n", k, v)
	}
	msg += "\r\n" + job.Body

	return smtp.SendMail(addr, auth, from, []string{job.To}, []byte(msg))
}
