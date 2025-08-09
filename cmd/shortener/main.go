package main

import (
	_ "net/http/pprof"

	"github.com/InQaaaaGit/trunc_url.git/internal/app"
	"github.com/InQaaaaGit/trunc_url.git/internal/buildinfo"
	"github.com/InQaaaaGit/trunc_url.git/internal/server"
	"go.uber.org/zap"
)

// Глобальные переменные для информации о сборке
var (
	buildVersion = "N/A"
	buildDate    = "N/A"
	buildCommit  = "N/A"
)

func main() {
	// Создаем и выводим информацию о сборке
	buildInfo := buildinfo.NewInfo(buildVersion, buildDate, buildCommit)
	buildInfo.Print()

	// Инициализация логгера
	logger, cleanup := server.InitLogger()
	defer cleanup()

	// Инициализация конфигурации
	cfg := server.InitConfig(logger)

	// Создание и настройка приложения
	application, err := app.NewApp(cfg)
	if err != nil {
		logger.Fatal("Ошибка создания приложения", zap.Error(err))
	}
	if err := application.Configure(); err != nil {
		logger.Fatal("Ошибка конфигурации приложения", zap.Error(err))
	}

	// Создание и запуск сервера
	httpServer := server.NewHTTPServer(application.GetServer(), cfg, logger)
	if err := httpServer.Start(); err != nil {
		logger.Fatal("Server failed to start", zap.Error(err))
	}
}
