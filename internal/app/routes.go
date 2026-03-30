package app

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/shaoyanji/bountystash/internal/http/handlers"
	"github.com/shaoyanji/bountystash/internal/service"
)

// NewRouter wires the thin 0.1 HTTP surface.
func NewRouter(cfg Config) (http.Handler, error) {
	db, err := openPostgres(cfg.DatabaseURL)
	if err != nil {
		return nil, err
	}

	svc := service.NewService(db)

	draftHandler, err := handlers.NewDraftHandler(svc)
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("initialize draft handler: %w", err)
	}

	reviewHandler, err := handlers.NewReviewHandler(svc)
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("initialize review handler: %w", err)
	}
	apiHandler := handlers.NewAPIHandler(svc)

	r := chi.NewRouter()

	r.Get("/", draftHandler.HandleHome)
	r.Get("/.well-known/bountystash-manifest", handlers.HandleManifest)
	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	r.Post("/draft", draftHandler.HandleDraftPost)
	r.Get("/work/{id}", draftHandler.HandleWorkShow)
	r.Get("/work/{id}/history", draftHandler.HandleWorkHistory)
	r.Get("/examples/{slug}", handlers.HandleExampleShow)
	r.Get("/review", reviewHandler.HandleQueue)

	r.Route("/api", func(api chi.Router) {
		api.Get("/healthz", apiHandler.HandleHealthz)
		api.Get("/examples", apiHandler.HandleExamplesList)
		api.Get("/examples/{slug}", apiHandler.HandleExampleShow)
		api.Get("/review", apiHandler.HandleReview)
		api.Get("/work", apiHandler.HandleWorkList)
		api.Get("/work/{id}", apiHandler.HandleWorkShow)
		api.Get("/work/{id}/history", apiHandler.HandleWorkHistory)
		api.Post("/draft", apiHandler.HandleDraftCreate)
	})

	return r, nil
}
