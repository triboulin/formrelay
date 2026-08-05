package model

import "time"

// Client représente un client / endpoint de formulaire.
type Client struct {
	ID               string    `json:"id"`
	Name             string    `json:"name"`
	DestinationEmail string    `json:"destination_email"`
	Active           bool      `json:"active"`
	CreatedAt        time.Time `json:"created_at"`
	SubmissionCount  int       `json:"submission_count,omitempty"`
}

// SubmissionStatus énumère les statuts possibles d'une soumission.
type SubmissionStatus string

const (
	StatusSuccess SubmissionStatus = "SUCCESS"
	StatusFailed  SubmissionStatus = "FAILED"
	StatusBlocked SubmissionStatus = "BLOCKED"
)

// Submission représente une soumission de formulaire archivée.
type Submission struct {
	ID           string           `json:"id"`
	ClientID     string           `json:"client_id"`
	ClientName   string           `json:"client_name,omitempty"`
	SenderIP     string           `json:"sender_ip"`
	SenderEmail  string           `json:"sender_email"`
	Subject      string           `json:"subject"`
	Payload      string           `json:"payload"`
	Status       SubmissionStatus `json:"status"`
	ErrorMessage string           `json:"error_message"`
	CreatedAt    time.Time        `json:"created_at"`
}

// MailJob représente un travail d'envoi d'email mis en queue.
type MailJob struct {
	SubmissionID string
	To           string
	ReplyTo      string
	Subject      string
	Body         string
}

// Stats représente les statistiques agrégées du dashboard.
type Stats struct {
	TotalClients     int
	ActiveClients    int
	TotalSubmissions int
	SuccessCount     int
	FailedCount      int
	BlockedCount     int
	SubmissionsToday int
	SubmissionsWeek  int
}
