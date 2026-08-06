package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"formrelay-admin/internal/config"
	"formrelay-admin/internal/database"
	"formrelay-admin/internal/handler"
	"formrelay-admin/internal/middleware"
	"formrelay-admin/internal/repository"
	"formrelay-admin/internal/service"
)

const templatesDir = "templates"

func main() {
	cfg := config.Load()

	db, err := database.New(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("erreur initialisation base de données: %v", err)
	}
	defer db.Close()

	clientRepo := repository.NewClientRepository(db)
	subRepo := repository.NewSubmissionRepository(db)

	mailer := service.NewMailWorker(cfg, subRepo, 100)
	mailer.Start(4)

	retention := service.NewRetentionCron(subRepo)
	retention.Start()

	formService := service.NewFormService(subRepo, mailer)

	publicHandler := handler.NewPublicHandler(clientRepo, formService, templatesDir)
	adminHandler := handler.NewAdminHandler(clientRepo, subRepo, templatesDir)
	apiHandler := handler.NewAPIHandler(clientRepo)

	rateLimiter := middleware.NewIPRateLimiter(5 * time.Second)
	basicAuth := middleware.BasicAuth(cfg.AdminUser, cfg.AdminPass)

	mux := http.NewServeMux()

	// Endpoint public de réception des formulaires, protégé par le rate limiter.
	mux.Handle("POST /f/{client_hash}", rateLimiter.RateLimit(http.HandlerFunc(publicHandler.Submit)))

	// Panel d'administration, protégé par Basic Auth.
	adminMux := http.NewServeMux()
	adminMux.HandleFunc("GET /admin", adminHandler.Dashboard)
	adminMux.HandleFunc("GET /admin/clients", adminHandler.ClientsPage)
	adminMux.HandleFunc("POST /admin/clients", adminHandler.CreateClient)
	adminMux.HandleFunc("GET /admin/clients/{id}/edit", adminHandler.EditClientForm)
	adminMux.HandleFunc("PUT /admin/clients/{id}", adminHandler.UpdateClient)
	adminMux.HandleFunc("POST /admin/clients/{id}/toggle", adminHandler.ToggleClient)
	adminMux.HandleFunc("DELETE /admin/clients/{id}", adminHandler.DeleteClient)
	adminMux.HandleFunc("GET /admin/logs", adminHandler.LogsPage)
	adminMux.HandleFunc("GET /admin/logs/{id}", adminHandler.LogDetail)
	mux.Handle("/admin", basicAuth(adminMux))
	mux.Handle("/admin/", basicAuth(adminMux))

	// API JSON pour les scripts de provisioning (mêmes identifiants que /admin).
	apiMux := http.NewServeMux()
	apiMux.HandleFunc("POST /api/clients", apiHandler.CreateClient)
	apiMux.HandleFunc("GET /api/clients", apiHandler.ListClients)
	mux.Handle("/api/", basicAuth(apiMux))

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("FormRelay Admin démarré sur le port %s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("erreur serveur: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	log.Println("arrêt en cours...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("erreur arrêt serveur: %v", err)
	}
}
