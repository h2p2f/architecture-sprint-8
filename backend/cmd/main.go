package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/h2p2f/architecture-sprint-8/backend/internal/config"

	"github.com/h2p2f/architecture-sprint-8/backend/internal/handlers"
	customMiddleware "github.com/h2p2f/architecture-sprint-8/backend/internal/middleware"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal("Cannot load config:", err)
	}

	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	// CORS
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Маршруты API
	r.Route("/", func(r chi.Router) {
		r.Use(customMiddleware.AuthMiddleware(
			cfg.KeycloakURL,
			cfg.KeycloakRealm,
			cfg.RequiredRole,
		))

		r.Get("/reports", handlers.GenerateReports)
	})

	// Здоровье сервиса
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("healthy"))
		if err != nil {
			return
		}
	})

	serverAddr := fmt.Sprintf(":%s", cfg.Port)
	log.Printf("Starting server on %s", serverAddr)
	if err := http.ListenAndServe(serverAddr, r); err != nil {
		log.Fatal(err)
	}
}
