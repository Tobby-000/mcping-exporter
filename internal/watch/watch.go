package watch

import (
	"context"
	"log/slog"
	"os"
	"time"
)

type Watcher struct {
	path     string
	interval time.Duration
	lastMod  time.Time
	onChange func() error
	logger   *slog.Logger
}

func New(path string, interval time.Duration, lastMod time.Time, onChange func() error, logger *slog.Logger) *Watcher {
	return &Watcher{
		path:     path,
		interval: interval,
		lastMod:  lastMod,
		onChange: onChange,
		logger:   logger,
	}
}

func (w *Watcher) Run(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.check()
		}
	}
}

func (w *Watcher) check() {
	info, err := os.Stat(w.path)
	if err != nil {
		w.logger.Debug("stat config failed", "path", w.path, "err", err)
		return
	}
	mod := info.ModTime()
	// please remove before commit
	w.logger.Info("watch tick", "path", w.path, "mod", mod, "lastMod", w.lastMod, "changed", !mod.Equal(w.lastMod))
	if mod.Equal(w.lastMod) {
		return
	}
	w.lastMod = mod
	if err := w.onChange(); err != nil {
		w.logger.Error("reload failed, keeping current config", "err", err.Error())
		return
	}
}
