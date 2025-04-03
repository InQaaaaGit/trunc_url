package main

import (
	"log"
	"net/http"

	"github.com/InQaaaaGit/trunc_url.git/internal/config"
	"github.com/InQaaaaGit/trunc_url.git/internal/handler"
	"github.com/InQaaaaGit/trunc_url.git/internal/service"
)

func main() {
	cfg := config.NewConfig()
	urlService := service.NewURLService()
	handler := handler.NewHandler(urlService)

	http.HandleFunc("/", handler.HandleRequest)

	log.Printf("Сервер запускается на http://localhost%s\n", cfg.ServerAddress)
	if err := http.ListenAndServe(cfg.ServerAddress, nil); err != nil {
		log.Fatal(err)
	}
}
