package main

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/fsnotify/fsnotify"
	"github.com/larsartmann/dynamic-markdown-site/internal/content"
	"github.com/larsartmann/dynamic-markdown-site/internal/server"
)

// watchForChanges monitors the root directory for filesystem changes in dev mode.
func watchForChanges(
	rootDir string,
	repo content.Repository,
	liveReload *server.LiveReload,
	logger *slog.Logger,
) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		logger.Error("failed to create file watcher", slog.Any("error", err))

		return
	}

	defer func() {
		err := watcher.Close()
		if err != nil {
			logger.Error("failed to close file watcher", slog.Any("error", err))
		}
	}()

	logger.Info("file watcher initialized in dev mode", slog.String("root_dir", rootDir))

	if err := addDirectoriesRecursive(watcher, rootDir, logger); err != nil {
		logger.Error("failed to add directories to watcher", slog.Any("error", err))

		return
	}

	var (
		refreshPending bool
		debounceTimer  *time.Timer
		debounceDelay  = 500 * time.Millisecond
	)

	scheduleRefresh := func() {
		if !refreshPending {
			refreshPending = true

			if debounceTimer != nil {
				debounceTimer.Stop()
			}

			debounceTimer = time.AfterFunc(debounceDelay, func() {
				logger.Info("refreshing content repository due to filesystem changes")

				result := repo.Refresh()
				if !result.Success {
					logger.Error("failed to refresh repository", slog.String("error", result.Error))
				} else {
					logger.Info("content repository refreshed",
						slog.Time("last_modified", result.LastModified),
						slog.Int("total_files", result.TotalFiles),
						slog.Int("total_dirs", result.TotalDirs),
					)

					// Notify live reload clients
					if liveReload != nil {
						liveReload.Notify("")
					}
				}

				refreshPending = false
			})
		}
	}

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}

			if event.Op&fsnotify.Chmod != 0 {
				continue
			}

			logger.Debug("filesystem event",
				slog.String("path", event.Name),
				slog.String("op", event.Op.String()),
			)

			if shouldTriggerRefresh(event.Name) {
				logger.Debug(
					"scheduling refresh for markdown change",
					slog.String("path", event.Name),
				)
				scheduleRefresh()
			}

			if event.Op&fsnotify.Create != 0 && isDirectory(event.Name) {
				logger.Debug("adding new directory to watcher", slog.String("path", event.Name))

				err := addDirectoriesRecursive(watcher, event.Name, logger)
				if err != nil {
					logger.Error("failed to add new directory to watcher",
						slog.String("path", event.Name),
						slog.Any("error", err),
					)
				}
			}

		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}

			logger.Error("file watcher error", slog.Any("error", err))
		}
	}
}

// addDirectoriesRecursive recursively adds all directories to the watcher.
func addDirectoriesRecursive(watcher *fsnotify.Watcher, root string, logger *slog.Logger) error {
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return errors.Wrapf(err, "walking directory tree at %s (root: %s)", path, root)
		}

		if info.IsDir() {
			if strings.HasPrefix(filepath.Base(path), ".") {
				return filepath.SkipDir
			}

			skipDirs := []string{"node_modules", ".git", "vendor", "dist", "build", "tmp", "temp"}

			baseName := filepath.Base(path)
			for _, skip := range skipDirs {
				if strings.EqualFold(baseName, skip) {
					return filepath.SkipDir
				}
			}

			err := watcher.Add(path)
			if err != nil {
				logger.Warn("failed to add directory to watcher",
					slog.String("path", path),
					slog.Any("error", err),
				)
			} else {
				logger.Debug("added directory to watcher", slog.String("path", path))
			}
		}

		return nil
	})
	if err != nil {
		return errors.Wrapf(err, "walk directories in root: %s", root)
	}

	return nil
}

// shouldTriggerRefresh returns true if the file change should trigger a repository refresh.
func shouldTriggerRefresh(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))

	return ext == ".md" || ext == ".markdown"
}

// isDirectory returns true if the path exists and is a directory.
func isDirectory(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}

	return info.IsDir()
}
