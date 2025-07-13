package server

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/yureien/anihash/anidb"
	"goji.io"
	"goji.io/pat"
	"gorm.io/gorm"
)

type server struct {
	db          *gorm.DB
	anidbClient *anidb.Client

	anidbQueryChan chan queryByEd2KSizeRequest
}

var _ http.Handler = server{}

func (s server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.queryHandler(w, r)
}

func (s server) errorResponse(w http.ResponseWriter, errCode int, errMsg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(errCode)
	json.NewEncoder(w).Encode(map[string]string{"error": errMsg})
}

func (s server) errorResponseWithJson(w http.ResponseWriter, errCode int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(errCode)
	json.NewEncoder(w).Encode(data)
}

func New(anidbClient *anidb.Client, db *gorm.DB) (*server, error) {
	anidbQueryChan := make(chan queryByEd2KSizeRequest)

	server := server{
		db:             db,
		anidbClient:    anidbClient,
		anidbQueryChan: anidbQueryChan,
	}
	server.startProcessor()

	return &server, nil
}

func (s server) ListenAndServe(logger *slog.Logger, cfg *ServerConfig) error {
	listenAddress := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)

	mux := goji.NewMux()
	mux.HandleFunc(pat.Get("/query/ed2k"), s.queryHandler)
	mux.HandleFunc(pat.Get("/query/hash"), s.hashQueryHandler)
	mux.HandleFunc(pat.Get("/"), s.homePageHandler)

	logger.Info("starting server", "address", listenAddress)
	return http.ListenAndServe(listenAddress, mux)
}
