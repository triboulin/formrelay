package repository

import (
	"database/sql"
	"fmt"

	"formrelay-admin/internal/model"
)

// SubmissionRepository gère l'accès aux données des soumissions.
type SubmissionRepository struct {
	db *sql.DB
}

func NewSubmissionRepository(db *sql.DB) *SubmissionRepository {
	return &SubmissionRepository{db: db}
}

func (r *SubmissionRepository) Create(s model.Submission) error {
	_, err := r.db.Exec(
		`INSERT INTO submissions (id, client_id, sender_ip, sender_email, subject, payload, status, error_message)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		s.ID, s.ClientID, s.SenderIP, s.SenderEmail, s.Subject, s.Payload, s.Status, s.ErrorMessage,
	)
	if err != nil {
		return fmt.Errorf("création soumission: %w", err)
	}
	return nil
}

func (r *SubmissionRepository) UpdateStatus(id string, status model.SubmissionStatus, errMsg string) error {
	_, err := r.db.Exec(`UPDATE submissions SET status = ?, error_message = ? WHERE id = ?`, status, errMsg, id)
	if err != nil {
		return fmt.Errorf("mise à jour statut soumission: %w", err)
	}
	return nil
}

func (r *SubmissionRepository) GetByID(id string) (*model.Submission, error) {
	row := r.db.QueryRow(`
		SELECT s.id, s.client_id, c.name, s.sender_ip, s.sender_email, s.subject, s.payload, s.status, s.error_message, s.created_at
		FROM submissions s
		LEFT JOIN clients c ON c.id = s.client_id
		WHERE s.id = ?`, id)

	var s model.Submission
	var clientName sql.NullString
	var senderEmail, subject, errMsg sql.NullString
	if err := row.Scan(&s.ID, &s.ClientID, &clientName, &s.SenderIP, &senderEmail, &subject, &s.Payload, &s.Status, &errMsg, &s.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("lecture soumission: %w", err)
	}
	s.ClientName = clientName.String
	s.SenderEmail = senderEmail.String
	s.Subject = subject.String
	s.ErrorMessage = errMsg.String
	return &s, nil
}

// ListFilter définit les filtres possibles pour la liste paginée des soumissions.
type ListFilter struct {
	ClientID string
	Status   string
	Page     int
	PageSize int
}

func (r *SubmissionRepository) List(f ListFilter) ([]model.Submission, int, error) {
	where := "WHERE 1=1"
	args := []interface{}{}

	if f.ClientID != "" {
		where += " AND s.client_id = ?"
		args = append(args, f.ClientID)
	}
	if f.Status != "" {
		where += " AND s.status = ?"
		args = append(args, f.Status)
	}

	var total int
	countQuery := "SELECT COUNT(*) FROM submissions s " + where
	if err := r.db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("comptage soumissions: %w", err)
	}

	if f.PageSize <= 0 {
		f.PageSize = 25
	}
	if f.Page <= 0 {
		f.Page = 1
	}
	offset := (f.Page - 1) * f.PageSize

	query := `
		SELECT s.id, s.client_id, c.name, s.sender_ip, s.sender_email, s.subject, s.payload, s.status, s.error_message, s.created_at
		FROM submissions s
		LEFT JOIN clients c ON c.id = s.client_id
		` + where + `
		ORDER BY s.created_at DESC
		LIMIT ? OFFSET ?`
	args = append(args, f.PageSize, offset)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("liste soumissions: %w", err)
	}
	defer rows.Close()

	var subs []model.Submission
	for rows.Next() {
		var s model.Submission
		var clientName, senderEmail, subject, errMsg sql.NullString
		if err := rows.Scan(&s.ID, &s.ClientID, &clientName, &s.SenderIP, &senderEmail, &subject, &s.Payload, &s.Status, &errMsg, &s.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan soumission: %w", err)
		}
		s.ClientName = clientName.String
		s.SenderEmail = senderEmail.String
		s.Subject = subject.String
		s.ErrorMessage = errMsg.String
		subs = append(subs, s)
	}
	return subs, total, rows.Err()
}

// Recent retourne les N dernières soumissions (pour le dashboard).
func (r *SubmissionRepository) Recent(limit int) ([]model.Submission, error) {
	rows, err := r.db.Query(`
		SELECT s.id, s.client_id, c.name, s.sender_ip, s.sender_email, s.subject, s.payload, s.status, s.error_message, s.created_at
		FROM submissions s
		LEFT JOIN clients c ON c.id = s.client_id
		ORDER BY s.created_at DESC
		LIMIT ?`, limit)
	if err != nil {
		return nil, fmt.Errorf("soumissions récentes: %w", err)
	}
	defer rows.Close()

	var subs []model.Submission
	for rows.Next() {
		var s model.Submission
		var clientName, senderEmail, subject, errMsg sql.NullString
		if err := rows.Scan(&s.ID, &s.ClientID, &clientName, &s.SenderIP, &senderEmail, &subject, &s.Payload, &s.Status, &errMsg, &s.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan soumission: %w", err)
		}
		s.ClientName = clientName.String
		s.SenderEmail = senderEmail.String
		s.Subject = subject.String
		s.ErrorMessage = errMsg.String
		subs = append(subs, s)
	}
	return subs, rows.Err()
}

// Stats calcule les statistiques agrégées pour le dashboard.
func (r *SubmissionRepository) Stats() (model.Stats, error) {
	var s model.Stats

	if err := r.db.QueryRow(`SELECT COUNT(*) FROM submissions`).Scan(&s.TotalSubmissions); err != nil {
		return s, err
	}
	if err := r.db.QueryRow(`SELECT COUNT(*) FROM submissions WHERE status = 'SUCCESS'`).Scan(&s.SuccessCount); err != nil {
		return s, err
	}
	if err := r.db.QueryRow(`SELECT COUNT(*) FROM submissions WHERE status = 'FAILED'`).Scan(&s.FailedCount); err != nil {
		return s, err
	}
	if err := r.db.QueryRow(`SELECT COUNT(*) FROM submissions WHERE status = 'BLOCKED'`).Scan(&s.BlockedCount); err != nil {
		return s, err
	}
	if err := r.db.QueryRow(`SELECT COUNT(*) FROM submissions WHERE created_at >= DATETIME('now', '-1 day')`).Scan(&s.SubmissionsToday); err != nil {
		return s, err
	}
	if err := r.db.QueryRow(`SELECT COUNT(*) FROM submissions WHERE created_at >= DATETIME('now', '-7 day')`).Scan(&s.SubmissionsWeek); err != nil {
		return s, err
	}

	return s, nil
}

// PurgeOlderThanOneYear supprime les soumissions de plus d'un an (politique de rétention).
func (r *SubmissionRepository) PurgeOlderThanOneYear() (int64, error) {
	res, err := r.db.Exec(`DELETE FROM submissions WHERE created_at < DATETIME('now', '-1 year')`)
	if err != nil {
		return 0, fmt.Errorf("purge soumissions: %w", err)
	}
	return res.RowsAffected()
}
