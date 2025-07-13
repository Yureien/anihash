package scanner

import (
	"context"
	"encoding/hex"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/yureien/anihash/anidb"
	"github.com/yureien/anihash/database"
	"github.com/zorchenhimer/go-ed2k"
	"gorm.io/gorm"
)

var videoExtensions = map[string]struct{}{
	".mp4":  {},
	".mkv":  {},
	".avi":  {},
	".mov":  {},
	".wmv":  {},
	".flv":  {},
	".webm": {},
	".mpg":  {},
	".mpeg": {},
	".m4v":  {},
	".m4a":  {},
}

type scanner struct {
	logger      *slog.Logger
	cfg         ScannerConfig
	anidbClient *anidb.Client
	db          *gorm.DB

	processChan chan string
	wg          sync.WaitGroup
}

func StartScanner(logger *slog.Logger, cfg ScannerConfig, anidbClient *anidb.Client, db *gorm.DB) {
	if cfg.ScanPath == "" {
		logger.Error("scan path is not set, disabling scanner")
		return
	}

	scanner := scanner{
		logger:      logger,
		cfg:         cfg,
		anidbClient: anidbClient,
		db:          db,
		processChan: make(chan string),
	}
	go scanner.start()
}

func (s *scanner) start() {
	numWorkers := s.cfg.NumWorkers
	if numWorkers <= 0 {
		numWorkers = runtime.GOMAXPROCS(0)
	}

	s.wg.Add(numWorkers)
	for i := 0; i < numWorkers; i++ {
		go s.startProcessor()
	}

	s.logger.Info("starting initial scan", "path", s.cfg.ScanPath)
	err := filepath.Walk(s.cfg.ScanPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		s.processChan <- path
		return nil
	})
	if err != nil {
		s.logger.Error("failed to walk scan path", "error", err)
	}
	s.logger.Info("initial scan finished")

	s.startWatcher()
}

func (s *scanner) startWatcher() {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		s.logger.Error("failed to create watcher", "error", err)
		return
	}
	defer watcher.Close()

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					s.logger.Info("watcher closed")
					return
				}

				if event.Has(fsnotify.Remove) {
					info, err := os.Stat(event.Name)
					if err != nil {
						continue
					}
					if info.IsDir() {
						s.logger.Info("removing directory from watcher", "path", event.Name)
						if err := watcher.Remove(event.Name); err != nil {
							s.logger.Error("failed to remove directory from watcher", "path", event.Name, "error", err)
						}
					}
				}

				if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) {
					info, err := os.Stat(event.Name)
					if err != nil {
						continue
					}

					if info.IsDir() {
						s.logger.Info("adding new directory to watcher", "path", event.Name)
						if err := watcher.Add(event.Name); err != nil {
							s.logger.Error("failed to add new directory to watcher", "path", event.Name, "error", err)
						}
					}
					if !info.IsDir() {
						s.processChan <- event.Name
					}
				}
			case err, ok := <-watcher.Errors:
				s.logger.Error("watcher error", "error", err)
				if !ok {
					return
				}
			}
		}
	}()

	s.logger.Info("adding paths to watcher recursively", "path", s.cfg.ScanPath)
	err = filepath.Walk(s.cfg.ScanPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return watcher.Add(path)
		}
		return nil
	})
	if err != nil {
		s.logger.Error("failed to add path to watcher", "path", s.cfg.ScanPath, "error", err)
		return
	}

	<-make(chan struct{})
}

func (s *scanner) startProcessor() {
	defer s.wg.Done()
	for path := range s.processChan {
		s.processFile(path)
	}
}

func (s *scanner) processFile(path string) {
	ext := strings.ToLower(filepath.Ext(path))
	if _, ok := videoExtensions[ext]; !ok {
		s.logger.Info("skipping non-video file", "path", path)
		return
	}

	info, err := os.Stat(path)
	if err != nil {
		s.logger.Error("failed to get file info", "path", path, "error", err)
		return
	}
	if info.IsDir() {
		return
	}

	s.logger.Info("scanning file", "path", path)

	file, err := os.Open(path)
	if err != nil {
		s.logger.Error("failed to open file", "path", path, "error", err)
		return
	}
	defer file.Close()

	hasher := ed2k.New()
	if _, err := io.Copy(hasher, file); err != nil {
		s.logger.Error("failed to hash file", "path", path, "error", err)
		return
	}
	ed2kHash := hex.EncodeToString(hasher.Sum(nil))
	size := info.Size()

	fileState, err := database.QueryFileStateByEd2KSize(s.db, ed2kHash, size)
	if err != nil && err != gorm.ErrRecordNotFound {
		s.logger.Error("failed to query file state", "path", path, "error", err)
		return
	}
	if err == gorm.ErrRecordNotFound {
		fileState, err = database.CreatePendingFileState(s.db, ed2kHash, size)
		if err != nil {
			s.logger.Error("failed to create pending file state", "path", path, "error", err)
			return
		}
	}

	if fileState.State == uint8(database.FILE_AVAILABLE) {
		s.logger.Info("file already in database", "path", path)
		return
	}

	if fileState.State != uint8(database.FILE_PENDING) {
		s.logger.Info("file state is not pending", "path", path, "state", fileState.State)
		return
	}

	s.logger.Info("fetching file from anidb", "path", path, "ed2k", ed2kHash, "size", size)
	anidbFile, err := s.anidbClient.FileByHash(context.Background(), size, ed2kHash)
	if err != nil {
		database.UpdateErroredFileState(s.db, ed2kHash, size, database.FILE_ERROR, err.Error())
		s.logger.Error("failed to fetch file from anidb", "path", path, "error", err)
		return
	}

	dbFile := database.AniDBFile{
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

	_, err = database.CreateFile(s.db, dbFile)
	if err != nil {
		database.UpdateErroredFileState(s.db, ed2kHash, size, database.FILE_ERROR, err.Error())
		s.logger.Error("failed to create file in database", "path", path, "error", err)
		return
	}

	err = database.UpdateAvailableFileState(s.db, uint(dbFile.FileID), ed2kHash, size)
	if err != nil {
		s.logger.Error("failed to update file state", "path", path, "error", err)
		return
	}

	s.logger.Info("successfully added file to database", "path", path)
}
