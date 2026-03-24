package app

import (
	"html/template"
	"net/http"

	"bountystash/internal/http/handlers"
	"bountystash/internal/views"
	"github.com/go-chi/chi/v5"
)

// NewRouter wires the minimal HTTP surface for milestone bootstrap.
func NewRouter() (http.Handler, error) {
	homeTemplate, err := template.ParseFS(views.FS, "home.tmpl")
	if err != nil {
		return nil, err
	}

	r := chi.NewRouter()

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := homeTemplate.Execute(w, nil); err != nil {
			http.Error(w, "template render error", http.StatusInternalServerError)
		}
	})

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	r.Post("/draft", handlers.HandleDraftPreviewPost)

	return r, nil
}
