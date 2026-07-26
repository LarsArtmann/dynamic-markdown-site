package main

import (
	"context"
	"log/slog"
	"time"

	"github.com/larsartmann/dynamic-markdown-site/internal/content"
	"github.com/larsartmann/dynamic-markdown-site/internal/server"
	filewatcher "github.com/larsartmann/go-filewatcher/v2"
)

const debounceDelay = 500 * time.Millisecond

// watchForChanges monitors the root directory for markdown file changes in dev
// mode using go-filewatcher for recursive watching, extension filtering, and
// debouncing. The watcher exits cleanly when ctx is cancelled.
func watchForChanges(
	ctx context.Context,
	rootDir string,
	repo content.Repository,
	liveReload *server.LiveReload,
	logger *slog.Logger,
) {
	watcher, err := filewatcher.New(
		[]string{rootDir},
		filewatcher.WithExtensions(".md", ".markdown"),
		filewatcher.WithDebounce(debounceDelay),
		filewatcher.WithIgnoreDirs(content.SkipDirs...),
		filewatcher.WithOnError(func(err error) {
			logger.Error("file watcher error", slog.Any("error", err))
		}),
	)
	if err != nil {
		logger.Error("failed to create file watcher", slog.Any("error", err))

		return
	}

	defer func() {
		if closeErr := watcher.Close(); closeErr != nil {
			logger.Error("failed to close file watcher", slog.Any("error", closeErr))
		}
	}()

	logger.Info("file watcher initialized in dev mode", slog.String("root_dir", rootDir))

	events, err := watcher.Watch(ctx)
	if err != nil {
		logger.Error("failed to start watching", slog.Any("error", err))

		return
	}

	for event := range events {
		logger.Debug(
			"filesystem event",
			slog.String("path", event.Path),
			slog.String("op", event.Op.String()),
		)

		doRefresh(repo, liveReload, logger)
	}
}

func doRefresh(repo content.Repository, liveReload *server.LiveReload, logger *slog.Logger) {
	logger.Info("refreshing content repository due to filesystem changes")

	result := repo.Refresh()
	if !result.Success {
		logger.Error("failed to refresh repository", slog.String("error", result.Error))
	} else {
		logger.Info(
			"content repository refreshed",
			slog.Time("last_modified", result.LastModified),
			slog.Int("total_files", result.TotalFiles),
			slog.Int("total_dirs", result.TotalDirs),
		)

		if liveReload != nil {
			liveReload.Notify("")
		}
	}
}
