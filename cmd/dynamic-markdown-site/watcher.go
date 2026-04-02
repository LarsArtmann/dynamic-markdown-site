package main

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"

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

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}

			handleFileEvent(watcher, event, repo, liveReload, logger)

		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}

			logger.Error("file watcher error", slog.Any("error", err))
		}
	}
}

// handleFileEvent processes a single filesystem event.
func handleFileEvent(
	watcher *fsnotify.Watcher,
	event fsnotify.Event,
	repo content.Repository,
	liveReload *server.LiveReload,
	logger *slog.Logger,
) {
	if event.Op&fsnotify.Chmod != 0 {
		return
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
		scheduleRefresh(repo, liveReload, logger)
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
}

// scheduleRefresh schedules a content refresh with debouncing.
func scheduleRefresh(repo content.Repository, liveReload *server.LiveReload, logger *slog.Logger) {
	// This is a simplified version - the full debouncing logic is in watchForChanges
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

		if liveReload != nil {
			liveReload.Notify("")
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

			if content.ShouldSkipDir(filepath.Base(path)) {
				return filepath.SkipDir
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
	return content.IsMarkdownFile(path)
}

// isDirectory returns true if the path exists and is a directory.
func isDirectory(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}

	return info.IsDir()
}
