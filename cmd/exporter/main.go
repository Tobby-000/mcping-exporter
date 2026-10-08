package main

import (
	"context"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"mcping-exporter/internal/collector"
	"mcping-exporter/internal/config"
	"mcping-exporter/internal/probe"
	"mcping-exporter/internal/watch"
)

func main() {
	// Flags
	listenAddr := flag.String("listen", ":9090", "address to listen on")
	configPath := flag.String("config", "config.yaml", "path to the YAML config file")
	flag.Parse()
	// Logger Init
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)
	// load config
	cfg, modTime, err := config.Load(*configPath)
	if err != nil {
		logger.Error("failed to load config", "path", *configPath, "err", err)
		os.Exit(1)
	}

	for _, w := range cfg.Warns() {
		logger.Warn(w)
	}

	// Cache Init
	cache := probe.NewCache()
	// targets
	targets := make([]probe.Target, 0, len(cfg.Targets))
	for _, t := range cfg.Targets {
		targets = append(targets, probe.Target{
			Name: t.Name,
			Addr: t.Addr,
		})
	}
	// prober init
	prober := probe.NewProber(
		targets,
		cache,
		time.Duration(cfg.Probe.Interval)*time.Second,
		time.Duration(cfg.Probe.Timeout)*time.Second,
		cfg.Probe.Limit,
		logger)
	// context init
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	go prober.Run(ctx)

	// watcher init
	reload := func() error {
		newCfg, _, err := config.Load(*configPath)
		if err != nil {
			return err
		}

		newTargets := make([]probe.Target, 0, len(newCfg.Targets))
		for _, t := range newCfg.Targets {
			newTargets = append(newTargets, probe.Target{Name: t.Name, Addr: t.Addr})
		}
		return prober.Reload(ctx, newTargets)
	}

	watcher := watch.New(*configPath, 5*time.Second, modTime, reload, logger)
	go watcher.Run(ctx)
	// collector regist and init

	mc := collector.NewMCCollector(cache)

	reg := prometheus.NewRegistry()
	reg.MustRegister(mc)

	// http service init and routes regist
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	server := &http.Server{
		Addr:    *listenAddr,
		Handler: mux,
	}
	// http service run
	go func() {
		logger.Info("exporter listening", "addr", *listenAddr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("http server run failed", "err", err)
			cancel()
		}
	}()
	// wait for shutdown signal
	<-ctx.Done()
	logger.Info("find shutdown signal,exiting")
	// wait 5s for http requests before program shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("http server shutdown error", "err", err)
	}

	logger.Info("exporter stopped")
}
