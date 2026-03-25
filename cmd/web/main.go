package main

import (
	"log"
	"net/http"

	"bountystash/internal/app"
)

func main() {
	cfg := app.LoadConfig()

	router, err := app.NewRouter(cfg)
	if err != nil {
		log.Fatalf("failed to initialize router: %v", err)
	}

	srv := &http.Server{
		Addr:    cfg.Addr,
		Handler: router,
	}

	log.Printf("web server listening on %s", cfg.Addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server failed: %v", err)
	}
}
