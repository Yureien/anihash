package server

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/yureien/anihash/database"
	"gorm.io/gorm"
)

type queryByEd2KSizeRequest struct {
	Size int64
	Ed2K string
}

func (s server) queryHandler(w http.ResponseWriter, r *http.Request) {
	sizeStr := r.URL.Query().Get("size")
	ed2k := r.URL.Query().Get("ed2k")

	size, err := strconv.ParseInt(sizeStr, 10, 64)
	if err != nil {
		s.errorResponse(w, http.StatusBadRequest, "invalid size")
		return
	}

	request := queryByEd2KSizeRequest{
		Size: size,
		Ed2K: ed2k,
	}

	file, err := database.QueryFileByED2KSize(s.db, request.Ed2K, int(request.Size))
	// No need to query file state if file is already available. Override if errored.
	fileState := database.FileState{
		FileID: &file.FileID,
		State:  uint8(database.FILE_AVAILABLE),
	}

	if err != nil {
		// Query file state first, check if anidb has not errored.
		fileState, err = database.QueryFileStateByEd2KSize(s.db, request.Ed2K, request.Size)
		if err != nil && err != gorm.ErrRecordNotFound {
			slog.Error("failed to query file state", "error", err)
			s.errorResponse(w, http.StatusInternalServerError, "failed to query file state")
			return
		}

		if err == gorm.ErrRecordNotFound {
			s.anidbQueryChan <- request

			fileState, err = database.CreatePendingFileState(s.db, request.Ed2K, request.Size)
			if err != nil {
				slog.Error("failed to create pending file state", "error", err)
				s.errorResponse(w, http.StatusInternalServerError, "failed to create pending file state")
				return
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"file":  nil,
				"state": fileState,
			})
			return
		}

		if fileState.State != uint8(database.FILE_PENDING) && fileState.State != uint8(database.FILE_AVAILABLE) {
			statusCode := http.StatusBadRequest
			if fileState.State == uint8(database.FILE_NOT_FOUND) {
				statusCode = http.StatusNotFound
			}
			s.errorResponseWithJson(w, statusCode, map[string]any{
				"file":  nil,
				"state": fileState,
			})
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"file":  file,
		"state": fileState,
	})
}
