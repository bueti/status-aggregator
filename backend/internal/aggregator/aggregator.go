package aggregator

import (
	"context"
	"log/slog"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/bbu/status-aggregator/backend/internal/providers"
	"github.com/bbu/status-aggregator/backend/internal/store"
)

type Entry struct {
	Config   providers.Config
	provider providers.Provider
	snap     atomic.Pointer[providers.Status]
}

func (e *Entry) Snapshot() providers.Status {
	if s := e.snap.Load(); s != nil {
		return *s
	}
	return providers.Status{Indicator: providers.IndicatorUnknown}
}

type Aggregator struct {
	store        *store.Store
	pollInterval time.Duration
	fetchTimeout time.Duration
	logger       *slog.Logger

	mu      sync.RWMutex
	entries []*Entry
	byID    map[string]*Entry

	reload chan struct{}
}

func New(s *store.Store, logger *slog.Logger) *Aggregator {
	return &Aggregator{
		store:        s,
		pollInterval: 60 * time.Second,
		fetchTimeout: 10 * time.Second,
		logger:       logger,
		byID:         map[string]*Entry{},
		reload:       make(chan struct{}, 1),
	}
}

func (a *Aggregator) Entries() []*Entry {
	a.mu.RLock()
	defer a.mu.RUnlock()
	out := make([]*Entry, len(a.entries))
	copy(out, a.entries)
	return out
}

func (a *Aggregator) Get(id string) (*Entry, bool) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	e, ok := a.byID[id]
	return e, ok
}

func (a *Aggregator) Reload() {
	select {
	case a.reload <- struct{}{}:
	default:
	}
}

func (a *Aggregator) Run(ctx context.Context) error {
	if err := a.loadEntries(ctx); err != nil {
		return err
	}
	a.fetchAll(ctx)

	ticker := time.NewTicker(a.pollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			a.fetchAll(ctx)
		case <-a.reload:
			if err := a.loadEntries(ctx); err != nil {
				a.logger.Error("reload failed", "err", err)
				continue
			}
			a.fetchAll(ctx)
		}
	}
}

func (a *Aggregator) loadEntries(ctx context.Context) error {
	cfgs, err := a.store.List(ctx)
	if err != nil {
		return err
	}
	sort.SliceStable(cfgs, func(i, j int) bool {
		if cfgs[i].SortOrder != cfgs[j].SortOrder {
			return cfgs[i].SortOrder < cfgs[j].SortOrder
		}
		return cfgs[i].ID < cfgs[j].ID
	})

	a.mu.Lock()
	defer a.mu.Unlock()

	prev := a.byID
	next := make(map[string]*Entry, len(cfgs))
	entries := make([]*Entry, 0, len(cfgs))

	for _, c := range cfgs {
		factory, err := providers.Lookup(c.Kind)
		if err != nil {
			a.logger.Warn("skip provider: unknown kind", "id", c.ID, "kind", c.Kind)
			continue
		}
		p, err := factory.Build(c)
		if err != nil {
			a.logger.Warn("skip provider: build failed", "id", c.ID, "err", err)
			continue
		}
		e := &Entry{Config: c, provider: p}
		if old, ok := prev[c.ID]; ok {
			if s := old.snap.Load(); s != nil {
				e.snap.Store(s)
			}
		}
		next[c.ID] = e
		entries = append(entries, e)
	}

	a.entries = entries
	a.byID = next
	return nil
}

func (a *Aggregator) fetchAll(ctx context.Context) {
	entries := a.Entries()
	var wg sync.WaitGroup
	for _, e := range entries {
		wg.Add(1)
		go func(e *Entry) {
			defer wg.Done()
			a.fetchOne(ctx, e)
		}(e)
	}
	wg.Wait()
}

func (a *Aggregator) fetchOne(ctx context.Context, e *Entry) {
	fctx, cancel := context.WithTimeout(ctx, a.fetchTimeout)
	defer cancel()
	s, err := e.provider.Fetch(fctx)
	if err != nil {
		a.logger.Warn("fetch failed", "id", e.Config.ID, "err", err)
		if prev := e.snap.Load(); prev != nil {
			stale := *prev
			stale.Err = err.Error()
			e.snap.Store(&stale)
			return
		}
		e.snap.Store(&providers.Status{
			Indicator: providers.IndicatorUnknown,
			Err:       err.Error(),
			FetchedAt: time.Time{},
		})
		return
	}
	e.snap.Store(&s)
}

func (a *Aggregator) StaleThreshold() time.Duration { return 3 * a.pollInterval }
