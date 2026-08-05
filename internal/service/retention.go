package service

import (
	"log"
	"time"

	"formrelay-admin/internal/repository"
)

// RetentionCron exécute périodiquement la purge des soumissions de plus d'un an.
type RetentionCron struct {
	subRepo *repository.SubmissionRepository
}

func NewRetentionCron(subRepo *repository.SubmissionRepository) *RetentionCron {
	return &RetentionCron{subRepo: subRepo}
}

// Start lance la purge immédiatement puis programme une exécution quotidienne à minuit.
func (c *RetentionCron) Start() {
	go func() {
		c.runOnce()
		for {
			now := time.Now()
			nextMidnight := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
			timer := time.NewTimer(nextMidnight.Sub(now))
			<-timer.C
			c.runOnce()
		}
	}()
}

func (c *RetentionCron) runOnce() {
	n, err := c.subRepo.PurgeOlderThanOneYear()
	if err != nil {
		log.Printf("erreur purge rétention: %v", err)
		return
	}
	if n > 0 {
		log.Printf("rétention: %d soumission(s) de plus d'un an supprimée(s)", n)
	}
}
