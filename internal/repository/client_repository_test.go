package repository

import (
	"testing"

	"github.com/google/uuid"

	"formrelay-admin/internal/model"
)

func newTestClient(name string) model.Client {
	return model.Client{
		ID:               uuid.NewString(),
		Name:             name,
		DestinationEmail: "dest+" + name + "@example.com",
		Active:           true,
	}
}

func TestClientRepository_CreateAndGetByID(t *testing.T) {
	repo := NewClientRepository(newTestDB(t))
	c := newTestClient("Acme")

	if err := repo.Create(c); err != nil {
		t.Fatalf("Create() erreur: %v", err)
	}

	got, err := repo.GetByID(c.ID)
	if err != nil {
		t.Fatalf("GetByID() erreur: %v", err)
	}
	if got == nil {
		t.Fatal("GetByID() = nil, attendu un client")
	}
	if got.Name != c.Name || got.DestinationEmail != c.DestinationEmail {
		t.Errorf("client récupéré incorrect: %+v", got)
	}
	if !got.Active {
		t.Error("le client devrait être actif par défaut")
	}
	if got.CreatedAt.IsZero() {
		t.Error("CreatedAt ne devrait pas être zéro")
	}
}

func TestClientRepository_GetByID_NotFound(t *testing.T) {
	repo := NewClientRepository(newTestDB(t))

	got, err := repo.GetByID("inexistant")
	if err != nil {
		t.Fatalf("GetByID() erreur inattendue: %v", err)
	}
	if got != nil {
		t.Errorf("GetByID() = %+v, attendu nil", got)
	}
}

func TestClientRepository_ListWithStats(t *testing.T) {
	db := newTestDB(t)
	clientRepo := NewClientRepository(db)
	subRepo := NewSubmissionRepository(db)

	c1 := newTestClient("Acme")
	c2 := newTestClient("Beta")
	if err := clientRepo.Create(c1); err != nil {
		t.Fatal(err)
	}
	if err := clientRepo.Create(c2); err != nil {
		t.Fatal(err)
	}

	// 2 soumissions pour c1, 0 pour c2.
	for i := 0; i < 2; i++ {
		s := model.Submission{
			ID:       uuid.NewString(),
			ClientID: c1.ID,
			SenderIP: "127.0.0.1",
			Payload:  "{}",
			Status:   model.StatusSuccess,
		}
		if err := subRepo.Create(s); err != nil {
			t.Fatal(err)
		}
	}

	clients, err := clientRepo.ListWithStats()
	if err != nil {
		t.Fatalf("ListWithStats() erreur: %v", err)
	}
	if len(clients) != 2 {
		t.Fatalf("attendu 2 clients, obtenu %d", len(clients))
	}

	counts := map[string]int{}
	for _, c := range clients {
		counts[c.ID] = c.SubmissionCount
	}
	if counts[c1.ID] != 2 {
		t.Errorf("c1 devrait avoir 2 soumissions, a %d", counts[c1.ID])
	}
	if counts[c2.ID] != 0 {
		t.Errorf("c2 devrait avoir 0 soumission, a %d", counts[c2.ID])
	}
}

func TestClientRepository_ListWithStats_Empty(t *testing.T) {
	repo := NewClientRepository(newTestDB(t))
	clients, err := repo.ListWithStats()
	if err != nil {
		t.Fatalf("erreur: %v", err)
	}
	if len(clients) != 0 {
		t.Errorf("attendu 0 client, obtenu %d", len(clients))
	}
}

func TestClientRepository_Update(t *testing.T) {
	repo := NewClientRepository(newTestDB(t))
	c := newTestClient("Acme")
	if err := repo.Create(c); err != nil {
		t.Fatal(err)
	}

	if err := repo.Update(c.ID, "Acme Renamed", "new@example.com"); err != nil {
		t.Fatalf("Update() erreur: %v", err)
	}

	got, err := repo.GetByID(c.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil {
		t.Fatal("le client devrait toujours exister après mise à jour")
	}
	if got.ID != c.ID {
		t.Errorf("ID = %q, ne devrait jamais changer (attendu %q)", got.ID, c.ID)
	}
	if got.Name != "Acme Renamed" {
		t.Errorf("Name = %q, attendu 'Acme Renamed'", got.Name)
	}
	if got.DestinationEmail != "new@example.com" {
		t.Errorf("DestinationEmail = %q, attendu 'new@example.com'", got.DestinationEmail)
	}
	if !got.Active {
		t.Error("Active ne devrait pas être modifié par Update()")
	}
}

func TestClientRepository_ToggleActive(t *testing.T) {
	repo := NewClientRepository(newTestDB(t))
	c := newTestClient("Acme")
	if err := repo.Create(c); err != nil {
		t.Fatal(err)
	}

	if err := repo.ToggleActive(c.ID); err != nil {
		t.Fatalf("ToggleActive() erreur: %v", err)
	}
	got, err := repo.GetByID(c.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Active {
		t.Error("le client devrait être inactif après le premier toggle")
	}

	if err := repo.ToggleActive(c.ID); err != nil {
		t.Fatalf("ToggleActive() erreur: %v", err)
	}
	got, err = repo.GetByID(c.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Active {
		t.Error("le client devrait être actif après le second toggle")
	}
}

func TestClientRepository_Delete(t *testing.T) {
	db := newTestDB(t)
	clientRepo := NewClientRepository(db)
	subRepo := NewSubmissionRepository(db)

	c := newTestClient("Acme")
	if err := clientRepo.Create(c); err != nil {
		t.Fatal(err)
	}
	sub := model.Submission{ID: uuid.NewString(), ClientID: c.ID, SenderIP: "127.0.0.1", Payload: "{}", Status: model.StatusSuccess}
	if err := subRepo.Create(sub); err != nil {
		t.Fatal(err)
	}

	if err := clientRepo.Delete(c.ID); err != nil {
		t.Fatalf("Delete() erreur: %v", err)
	}

	got, err := clientRepo.GetByID(c.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Error("le client devrait avoir été supprimé")
	}

	// La suppression en cascade doit avoir retiré la soumission associée.
	gotSub, err := subRepo.GetByID(sub.ID)
	if err != nil {
		t.Fatal(err)
	}
	if gotSub != nil {
		t.Error("la soumission associée devrait avoir été supprimée en cascade")
	}
}

func TestClientRepository_Count(t *testing.T) {
	repo := NewClientRepository(newTestDB(t))

	total, active, err := repo.Count()
	if err != nil {
		t.Fatalf("Count() erreur: %v", err)
	}
	if total != 0 || active != 0 {
		t.Errorf("attendu 0/0, obtenu %d/%d", total, active)
	}

	c1 := newTestClient("Acme")
	c2 := newTestClient("Beta")
	c2.Active = false
	if err := repo.Create(c1); err != nil {
		t.Fatal(err)
	}
	if err := repo.Create(c2); err != nil {
		t.Fatal(err)
	}

	total, active, err = repo.Count()
	if err != nil {
		t.Fatalf("Count() erreur: %v", err)
	}
	if total != 2 {
		t.Errorf("total = %d, attendu 2", total)
	}
	if active != 1 {
		t.Errorf("active = %d, attendu 1", active)
	}
}
