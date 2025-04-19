package main

import (
	"log"
	"net/http"
	"time"

	"github.com/InQaaaaGit/trunc_url.git/internal/config"
	"github.com/InQaaaaGit/trunc_url.git/internal/handler"
	"github.com/InQaaaaGit/trunc_url.git/internal/service"
	"github.com/go-chi/chi/v5"
)

func main() {
	cfg := config.NewConfig()
	urlService := service.NewURLService()
	handler := handler.NewHandler(urlService, cfg)

	r := chi.NewRouter()
	r.Post("/", handler.HandleCreateURL)
	r.Get("/{shortID}", handler.HandleRedirect)

	server := &http.Server{
		Addr:         cfg.ServerAddress,
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	log.Printf("Сервер запускается на %s\n", cfg.ServerAddress)
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
