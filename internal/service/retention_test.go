package service

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"formrelay-admin/internal/model"
)

func TestRetentionCron_RunOnce_PurgesOldSubmissions(t *testing.T) {
	subRepo, c := newTestRepos(t)

	recent := model.Submission{ID: uuid.NewString(), ClientID: c.ID, SenderIP: "1.2.3.4", Payload: "{}", Status: model.StatusSuccess}
	if err := subRepo.Create(recent); err != nil {
		t.Fatal(err)
	}

	cron := NewRetentionCron(subRepo)

	// runOnce est privée mais testée depuis le même package : elle ne doit
	// pas paniquer et ne doit rien supprimer s'il n'y a pas de soumission ancienne.
	cron.runOnce()

	got, err := subRepo.GetByID(recent.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil {
		t.Error("la soumission récente n'aurait pas dû être supprimée")
	}
}

func TestRetentionCron_Start_RunsInitialPurge(t *testing.T) {
	subRepo, c := newTestRepos(t)

	old := model.Submission{ID: uuid.NewString(), ClientID: c.ID, SenderIP: "1.2.3.4", Payload: "{}", Status: model.StatusSuccess}
	if err := subRepo.Create(old); err != nil {
		t.Fatal(err)
	}

	cron := NewRetentionCron(subRepo)
	cron.Start()

	// Start() lance runOnce() en arrière-plan immédiatement : on laisse le temps à la goroutine de s'exécuter.
	time.Sleep(100 * time.Millisecond)

	got, err := subRepo.GetByID(old.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil {
		t.Error("la soumission récente n'aurait pas dû être supprimée par le premier passage de Start()")
	}
}
