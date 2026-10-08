package probe

import (
	"context"
	"log/slog"
	"sync/atomic"
	"time"

	"golang.org/x/sync/errgroup"

	"mcping-exporter/internal/mc"
)

type Target struct {
	Name string
	Addr string
}
type Prober struct {
	targets  atomic.Pointer[[]Target]
	cache    *Cache
	interval time.Duration
	timeout  time.Duration
	limit    int
	logger   *slog.Logger
	// running 防止上一轮探测未完成时，下一轮 Ticker 触发导致探测重叠。
	running  atomic.Bool
	reloadCh chan []Target
}

func NewProber(targets []Target, cache *Cache, interval time.Duration, timeout time.Duration, limit int, logger *slog.Logger) *Prober {
	p := &Prober{
		cache:    cache,
		interval: interval,
		timeout:  timeout,
		limit:    limit,
		logger:   logger,
		reloadCh: make(chan []Target),
	}
	p.targets.Store(&targets)
	return p
}

func (p *Prober) Run(ctx context.Context) {
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()
	p.probeAll(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case newTargets := <-p.reloadCh:
			p.applyReload(newTargets)
		case <-ticker.C:
			p.probeAll(ctx)
		}
	}
}

func (p *Prober) probeAll(ctx context.Context) {
	// 当上一次探测还在运行时，跳过本轮
	if p.running.Load() {
		p.logger.Warn("previous probe still running, skipping this round")
		return
	}
	p.running.Store(true)
	defer p.running.Store(false)

	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(p.limit)

	targets := *p.targets.Load() // 解引用
	for _, t := range targets {
		g.Go(func() error {
			res, err := mc.Ping(ctx, t.Addr, p.timeout)
			p.cache.Set(t.Name, ProbeResult{
				Online:        res.Online && err == nil,
				Players:       res.Players,
				Timestamp:     time.Now(),
				Err:           err,
				RTTDuration:   res.RTTTime,
				TotalDuration: res.TotalTime,
			})
			return nil
		})
	}
	_ = g.Wait()
}

func (p *Prober) Reload(ctx context.Context, targets []Target) error {
	select {
	case p.reloadCh <- targets:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (p *Prober) applyReload(newTargets []Target) {
	p.logger.Info("applyReload start", "count", len(newTargets))
	oldTargets := *p.targets.Load()
	// 移除已经被删除的targets缓存
	newNames := make(map[string]struct{}, len(newTargets))
	for _, t := range newTargets {
		newNames[t.Name] = struct{}{}
	}
	removed := 0
	for _, t := range oldTargets {
		if _, ok := newNames[t.Name]; !ok {
			p.cache.Delete(t.Name)
			removed++
		}
	}

	// 更换新目标
	p.targets.Store(&newTargets)

	p.logger.Info("targets updated",
		"old", len(oldTargets),
		"new", len(newTargets),
		"removed", removed,
	)
}
