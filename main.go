package main

import (
	"log/slog"
	"os"

	"github.com/yureien/anihash/anidb"
	"github.com/yureien/anihash/database"
	"github.com/yureien/anihash/server"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	cfg, err := LoadConfig("config.yaml")
	if err != nil {
		logger.Error("failed to load config", "error", err)
		return
	}

	anidbClient, closeAnidb, err := anidb.NewAuthenticatedClient(logger, &cfg.Anidb)
	if err != nil {
		logger.Error("failed to create anidb client", "error", err)
		return
	}
	defer closeAnidb()

	db, err := database.LoadDatabase(logger, &cfg.Database)
	if err != nil {
		logger.Error("failed to load database", "error", err)
		return
	}

	server, err := server.New(anidbClient, db)
	if err != nil {
		logger.Error("failed to create server", "error", err)
		return
	}

	if err := server.ListenAndServe(logger, &cfg.Server); err != nil {
		logger.Error("failed to start server", "error", err)
	}
}
