package main

import (
	"log"
	"net/http"

	"github.com/InQaaaaGit/trunc_url.git/internal/config"
	"github.com/InQaaaaGit/trunc_url.git/internal/handler"
	"github.com/InQaaaaGit/trunc_url.git/internal/service"
	"github.com/go-chi/chi/v5"
)

func main() {
	cfg := config.NewConfig()
	urlService := service.NewURLService()
	handler := handler.NewHandler(urlService)

	r := chi.NewRouter()
	r.Post("/", handler.HandleCreateURL)
	r.Get("/{shortID}", handler.HandleRedirect)

	log.Printf("Сервер запускается на http://localhost%s\n", cfg.ServerAddress)
	if err := http.ListenAndServe(cfg.ServerAddress, r); err != nil {
		log.Fatal(err)
	}
}
