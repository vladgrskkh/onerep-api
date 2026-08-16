// @title OneRep Gym API
// @version 1.0
// @description Core gym domain service for OneRep — exercises, templates, workouts, progress
// @host localhost:8081
// @BasePath /v1
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/vladgrskkh/onerep-api/internal/application"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	ctx := context.Background()

	app, err := application.New(
		application.WithLogger(logger),
		application.WithDatabase(ctx),
		application.WithRedis(ctx),
		application.WithS3(ctx),
		application.WithJWKS(ctx),
		application.WithServices(),
		application.WithHandlers(),
	)
	if err != nil {
		logger.Error("failed to create app", "error", err)
		os.Exit(1)
	}

	if err := app.Run(ctx); err != nil {
		logger.Error("server error", "error", err)
		os.Exit(1)
	}

	logger.Info("stopped")
}
