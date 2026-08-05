package repository

import (
	"database/sql"
	"fmt"

	"formrelay-admin/internal/model"
)

// ClientRepository gère l'accès aux données des clients.
type ClientRepository struct {
	db *sql.DB
}

func NewClientRepository(db *sql.DB) *ClientRepository {
	return &ClientRepository{db: db}
}

func (r *ClientRepository) Create(c model.Client) error {
	_, err := r.db.Exec(
		`INSERT INTO clients (id, name, destination_email, active) VALUES (?, ?, ?, ?)`,
		c.ID, c.Name, c.DestinationEmail, c.Active,
	)
	if err != nil {
		return fmt.Errorf("création client: %w", err)
	}
	return nil
}

func (r *ClientRepository) GetByID(id string) (*model.Client, error) {
	row := r.db.QueryRow(
		`SELECT id, name, destination_email, active, created_at FROM clients WHERE id = ?`, id,
	)
	var c model.Client
	if err := row.Scan(&c.ID, &c.Name, &c.DestinationEmail, &c.Active, &c.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("lecture client: %w", err)
	}
	return &c, nil
}

// ListWithStats retourne tous les clients avec leur nombre de soumissions.
func (r *ClientRepository) ListWithStats() ([]model.Client, error) {
	rows, err := r.db.Query(`
		SELECT c.id, c.name, c.destination_email, c.active, c.created_at,
		       COALESCE(COUNT(s.id), 0) as submission_count
		FROM clients c
		LEFT JOIN submissions s ON s.client_id = c.id
		GROUP BY c.id
		ORDER BY c.created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("liste clients: %w", err)
	}
	defer rows.Close()

	var clients []model.Client
	for rows.Next() {
		var c model.Client
		if err := rows.Scan(&c.ID, &c.Name, &c.DestinationEmail, &c.Active, &c.CreatedAt, &c.SubmissionCount); err != nil {
			return nil, fmt.Errorf("scan client: %w", err)
		}
		clients = append(clients, c)
	}
	return clients, rows.Err()
}

// Update modifie le nom et l'email de destination d'un client existant,
// sans jamais toucher à son identifiant (l'endpoint /f/{id} reste stable).
func (r *ClientRepository) Update(id, name, destinationEmail string) error {
	_, err := r.db.Exec(
		`UPDATE clients SET name = ?, destination_email = ? WHERE id = ?`,
		name, destinationEmail, id,
	)
	if err != nil {
		return fmt.Errorf("mise à jour client: %w", err)
	}
	return nil
}

func (r *ClientRepository) ToggleActive(id string) error {
	_, err := r.db.Exec(`UPDATE clients SET active = NOT active WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("toggle client: %w", err)
	}
	return nil
}

func (r *ClientRepository) Delete(id string) error {
	_, err := r.db.Exec(`DELETE FROM clients WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("suppression client: %w", err)
	}
	return nil
}

func (r *ClientRepository) Count() (total, active int, err error) {
	err = r.db.QueryRow(`SELECT COUNT(*), COALESCE(SUM(CASE WHEN active THEN 1 ELSE 0 END), 0) FROM clients`).Scan(&total, &active)
	return
}
