package repository

import (
	"testing"

	"github.com/google/uuid"

	"formrelay-admin/internal/model"
)

func setupClientAndSubRepo(t *testing.T) (*ClientRepository, *SubmissionRepository, model.Client) {
	t.Helper()
	db := newTestDB(t)
	clientRepo := NewClientRepository(db)
	subRepo := NewSubmissionRepository(db)

	c := newTestClient("Acme")
	if err := clientRepo.Create(c); err != nil {
		t.Fatal(err)
	}
	return clientRepo, subRepo, c
}

func TestSubmissionRepository_CreateAndGetByID(t *testing.T) {
	_, subRepo, c := setupClientAndSubRepo(t)

	s := model.Submission{
		ID:          uuid.NewString(),
		ClientID:    c.ID,
		SenderIP:    "203.0.113.5",
		SenderEmail: "user@example.com",
		Subject:     "Bonjour",
		Payload:     `{"name":"John"}`,
		Status:      model.StatusSuccess,
	}
	if err := subRepo.Create(s); err != nil {
		t.Fatalf("Create() erreur: %v", err)
	}

	got, err := subRepo.GetByID(s.ID)
	if err != nil {
		t.Fatalf("GetByID() erreur: %v", err)
	}
	if got == nil {
		t.Fatal("GetByID() = nil")
	}
	if got.ClientName != c.Name {
		t.Errorf("ClientName = %q, attendu %q", got.ClientName, c.Name)
	}
	if got.SenderEmail != s.SenderEmail || got.Subject != s.Subject || got.Payload != s.Payload {
		t.Errorf("soumission récupérée incorrecte: %+v", got)
	}
	if got.Status != model.StatusSuccess {
		t.Errorf("Status = %q, attendu SUCCESS", got.Status)
	}
}

func TestSubmissionRepository_GetByID_NotFound(t *testing.T) {
	_, subRepo, _ := setupClientAndSubRepo(t)

	got, err := subRepo.GetByID("inexistant")
	if err != nil {
		t.Fatalf("erreur inattendue: %v", err)
	}
	if got != nil {
		t.Errorf("attendu nil, obtenu %+v", got)
	}
}

func TestSubmissionRepository_UpdateStatus(t *testing.T) {
	_, subRepo, c := setupClientAndSubRepo(t)

	s := model.Submission{ID: uuid.NewString(), ClientID: c.ID, SenderIP: "1.2.3.4", Payload: "{}", Status: model.StatusFailed, ErrorMessage: "en attente"}
	if err := subRepo.Create(s); err != nil {
		t.Fatal(err)
	}

	if err := subRepo.UpdateStatus(s.ID, model.StatusSuccess, ""); err != nil {
		t.Fatalf("UpdateStatus() erreur: %v", err)
	}

	got, err := subRepo.GetByID(s.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != model.StatusSuccess {
		t.Errorf("Status = %q, attendu SUCCESS", got.Status)
	}
	if got.ErrorMessage != "" {
		t.Errorf("ErrorMessage = %q, attendu vide", got.ErrorMessage)
	}
}

func TestSubmissionRepository_ListAndFilters(t *testing.T) {
	_, subRepo, c := setupClientAndSubRepo(t)

	statuses := []model.SubmissionStatus{model.StatusSuccess, model.StatusFailed, model.StatusBlocked, model.StatusSuccess}
	for _, st := range statuses {
		s := model.Submission{ID: uuid.NewString(), ClientID: c.ID, SenderIP: "1.2.3.4", Payload: "{}", Status: st}
		if err := subRepo.Create(s); err != nil {
			t.Fatal(err)
		}
	}

	t.Run("sans filtre", func(t *testing.T) {
		subs, total, err := subRepo.List(ListFilter{})
		if err != nil {
			t.Fatalf("List() erreur: %v", err)
		}
		if total != 4 {
			t.Errorf("total = %d, attendu 4", total)
		}
		if len(subs) != 4 {
			t.Errorf("len(subs) = %d, attendu 4", len(subs))
		}
	})

	t.Run("filtre par statut", func(t *testing.T) {
		subs, total, err := subRepo.List(ListFilter{Status: "SUCCESS"})
		if err != nil {
			t.Fatalf("List() erreur: %v", err)
		}
		if total != 2 {
			t.Errorf("total = %d, attendu 2", total)
		}
		for _, s := range subs {
			if s.Status != model.StatusSuccess {
				t.Errorf("statut inattendu dans les résultats filtrés: %s", s.Status)
			}
		}
	})

	t.Run("filtre par client", func(t *testing.T) {
		subs, total, err := subRepo.List(ListFilter{ClientID: c.ID})
		if err != nil {
			t.Fatalf("List() erreur: %v", err)
		}
		if total != 4 {
			t.Errorf("total = %d, attendu 4", total)
		}
		if len(subs) != 4 {
			t.Errorf("len(subs) = %d, attendu 4", len(subs))
		}
	})

	t.Run("filtre par client inexistant", func(t *testing.T) {
		_, total, err := subRepo.List(ListFilter{ClientID: "autre"})
		if err != nil {
			t.Fatalf("List() erreur: %v", err)
		}
		if total != 0 {
			t.Errorf("total = %d, attendu 0", total)
		}
	})

	t.Run("pagination", func(t *testing.T) {
		subs, total, err := subRepo.List(ListFilter{Page: 1, PageSize: 2})
		if err != nil {
			t.Fatalf("List() erreur: %v", err)
		}
		if total != 4 {
			t.Errorf("total = %d, attendu 4", total)
		}
		if len(subs) != 2 {
			t.Errorf("len(subs) = %d, attendu 2 (page 1, taille 2)", len(subs))
		}

		subs2, _, err := subRepo.List(ListFilter{Page: 2, PageSize: 2})
		if err != nil {
			t.Fatalf("List() erreur: %v", err)
		}
		if len(subs2) != 2 {
			t.Errorf("len(subs2) = %d, attendu 2 (page 2, taille 2)", len(subs2))
		}
	})

	t.Run("page et pageSize par défaut si non spécifiés", func(t *testing.T) {
		subs, _, err := subRepo.List(ListFilter{Page: 0, PageSize: 0})
		if err != nil {
			t.Fatalf("List() erreur: %v", err)
		}
		if len(subs) != 4 {
			t.Errorf("len(subs) = %d, attendu 4 (dans les limites de la pageSize par défaut)", len(subs))
		}
	})
}

func TestSubmissionRepository_Recent(t *testing.T) {
	_, subRepo, c := setupClientAndSubRepo(t)

	for i := 0; i < 3; i++ {
		s := model.Submission{ID: uuid.NewString(), ClientID: c.ID, SenderIP: "1.2.3.4", Payload: "{}", Status: model.StatusSuccess}
		if err := subRepo.Create(s); err != nil {
			t.Fatal(err)
		}
	}

	recent, err := subRepo.Recent(2)
	if err != nil {
		t.Fatalf("Recent() erreur: %v", err)
	}
	if len(recent) != 2 {
		t.Errorf("len(recent) = %d, attendu 2", len(recent))
	}
}

func TestSubmissionRepository_Stats(t *testing.T) {
	_, subRepo, c := setupClientAndSubRepo(t)

	statuses := []model.SubmissionStatus{model.StatusSuccess, model.StatusSuccess, model.StatusFailed, model.StatusBlocked}
	for _, st := range statuses {
		s := model.Submission{ID: uuid.NewString(), ClientID: c.ID, SenderIP: "1.2.3.4", Payload: "{}", Status: st}
		if err := subRepo.Create(s); err != nil {
			t.Fatal(err)
		}
	}

	stats, err := subRepo.Stats()
	if err != nil {
		t.Fatalf("Stats() erreur: %v", err)
	}
	if stats.TotalSubmissions != 4 {
		t.Errorf("TotalSubmissions = %d, attendu 4", stats.TotalSubmissions)
	}
	if stats.SuccessCount != 2 {
		t.Errorf("SuccessCount = %d, attendu 2", stats.SuccessCount)
	}
	if stats.FailedCount != 1 {
		t.Errorf("FailedCount = %d, attendu 1", stats.FailedCount)
	}
	if stats.BlockedCount != 1 {
		t.Errorf("BlockedCount = %d, attendu 1", stats.BlockedCount)
	}
	if stats.SubmissionsToday != 4 {
		t.Errorf("SubmissionsToday = %d, attendu 4", stats.SubmissionsToday)
	}
	if stats.SubmissionsWeek != 4 {
		t.Errorf("SubmissionsWeek = %d, attendu 4", stats.SubmissionsWeek)
	}
}

func TestSubmissionRepository_PurgeOlderThanOneYear(t *testing.T) {
	db := newTestDB(t)
	clientRepo := NewClientRepository(db)
	subRepo := NewSubmissionRepository(db)

	c := newTestClient("Acme")
	if err := clientRepo.Create(c); err != nil {
		t.Fatal(err)
	}

	// Soumission récente (non purgée).
	recent := model.Submission{ID: uuid.NewString(), ClientID: c.ID, SenderIP: "1.2.3.4", Payload: "{}", Status: model.StatusSuccess}
	if err := subRepo.Create(recent); err != nil {
		t.Fatal(err)
	}

	// Soumission ancienne (> 1 an), insérée directement avec une date passée.
	oldID := uuid.NewString()
	if _, err := db.Exec(
		`INSERT INTO submissions (id, client_id, sender_ip, payload, status, created_at) VALUES (?, ?, ?, ?, ?, DATETIME('now', '-400 day'))`,
		oldID, c.ID, "1.2.3.4", "{}", string(model.StatusSuccess),
	); err != nil {
		t.Fatal(err)
	}

	n, err := subRepo.PurgeOlderThanOneYear()
	if err != nil {
		t.Fatalf("PurgeOlderThanOneYear() erreur: %v", err)
	}
	if n != 1 {
		t.Errorf("nombre de lignes purgées = %d, attendu 1", n)
	}

	got, err := subRepo.GetByID(oldID)
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Error("la soumission ancienne aurait dû être supprimée")
	}

	stillThere, err := subRepo.GetByID(recent.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stillThere == nil {
		t.Error("la soumission récente n'aurait pas dû être supprimée")
	}
}
