package server

import (
	"context"
	"log/slog"
	"time"

	"github.com/yureien/anihash/database"
)

func (s server) startProcessor() {
	go func() {
		for request := range s.anidbQueryChan {
			s.processAnidbQuery(request)
		}
	}()

	go func() {
		for {
			s.processPendingFiles()
			time.Sleep(1 * time.Hour)
		}
	}()
}

func (s server) processAnidbQuery(request queryByEd2KSizeRequest) {
	fileState, err := database.QueryFileStateByEd2KSize(s.db, request.Ed2K, request.Size)
	if err != nil {
		slog.Error("failed to query file state", "error", err)
		return
	}

	// Only process pending files.
	if fileState.State != uint8(database.FILE_PENDING) {
		return
	}

	slog.Info("fetching file from anidb", "ed2k", request.Ed2K, "size", request.Size)
	anidbFile, err := s.anidbClient.FileByHash(context.Background(), request.Size, request.Ed2K)
	if err != nil {
		database.UpdateErroredFileState(s.db, request.Ed2K, request.Size, database.FILE_ERROR, err.Error())
		return
	}

	file := database.AniDBFile{
		FileID:          anidbFile.FileID,
		AnimeID:         anidbFile.AnimeID,
		EpisodeID:       anidbFile.EpisodeID,
		GroupID:         anidbFile.GroupID,
		State:           anidbFile.State,
		Size:            anidbFile.Size,
		Ed2K:            anidbFile.Ed2K,
		MD5:             anidbFile.MD5,
		SHA1:            anidbFile.SHA1,
		CRC:             anidbFile.CRC,
		Quality:         anidbFile.Quality,
		Source:          anidbFile.Source,
		AudioCodec:      anidbFile.AudioCodec,
		AudioBitrate:    anidbFile.AudioBitrate,
		VideoCodec:      anidbFile.VideoCodec,
		VideoBitrate:    anidbFile.VideoBitrate,
		VideoResolution: anidbFile.VideoResolution,
		Extension:       anidbFile.Extension,
		Year:            anidbFile.Year,
		Type:            anidbFile.Type,
		RomajiName:      anidbFile.RomajiName,
		EnglishName:     anidbFile.EnglishName,
		EpNum:           anidbFile.EpNum,
		EpName:          anidbFile.EpName,
		EpRomajiName:    anidbFile.EpRomajiName,
		GroupName:       anidbFile.GroupName,
	}

	fileID, err := database.CreateFile(s.db, file)
	if err != nil {
		slog.Error("failed to create file", "error", err)
		database.UpdateErroredFileState(s.db, request.Ed2K, request.Size, database.FILE_ERROR, "failed to create file")
		return
	}

	err = database.UpdateAvailableFileState(s.db, fileID, request.Ed2K, request.Size)
	if err != nil {
		slog.Error("failed to update file state", "error", err)
		database.UpdateErroredFileState(s.db, request.Ed2K, request.Size, database.FILE_ERROR, "failed to update file state")
		return
	}
}

func (s server) processPendingFiles() {
	files, err := database.QueryPendingFiles(s.db)
	if err != nil {
		slog.Error("failed to query pending files", "error", err)
		return
	}

	for _, file := range files {
		s.anidbQueryChan <- queryByEd2KSizeRequest{
			Ed2K: file.Ed2K,
			Size: file.Size,
		}
	}
}
