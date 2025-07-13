package server

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/yureien/anihash/database"
	"gorm.io/gorm"
)

func (s server) hashQueryHandler(w http.ResponseWriter, r *http.Request) {
	hash := r.URL.Query().Get("hash")
	if len(hash) != 32 && len(hash) != 40 {
		s.errorResponse(w, http.StatusBadRequest, "invalid hash")
		return
	}

	file, err := database.QueryFileByHash(s.db, hash)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			s.errorResponseWithJson(w, http.StatusNotFound, map[string]any{
				"file": nil,
				"state": database.FileState{
					FileID: nil,
					State:  uint8(database.FILE_NOT_FOUND),
					Error:  "File not in database, please use the ed2k query instead.",
				},
			})
			return
		}

		slog.Error("failed to query file", "hash", hash, "error", err)
		s.errorResponse(w, http.StatusInternalServerError, "failed to query file")
		return
	}

	fileState := database.FileState{
		FileID: &file.FileID,
		State:  uint8(database.FILE_AVAILABLE),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"file":  file,
		"state": fileState,
	})
}
