package application

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

const (
	defaultWorkers    = 2
	defaultPollTicker = time.Second
	defaultJobLease   = 120 * time.Second
)

// Worker polls JobStore with a configurable goroutine pool and processes claimed jobs.
type Worker struct {
	jobs       JobStore
	processor  *Processor
	workers    int
	pollTicker time.Duration
	lease      time.Duration
}

// NewWorker constructs a Worker pool (defaulting to 2 goroutines when workers < 1).
func NewWorker(jobs JobStore, processor *Processor, workers int) *Worker {
	if workers < 1 {
		workers = defaultWorkers
	}
	return &Worker{
		jobs:       jobs,
		processor:  processor,
		workers:    workers,
		pollTicker: defaultPollTicker,
		lease:      defaultJobLease,
	}
}

// Start runs the worker goroutines until ctx is cancelled and waits for all to exit.
func (w *Worker) Start(ctx context.Context) {
	if w == nil || w.jobs == nil || w.processor == nil {
		return
	}
	var wg sync.WaitGroup
	for i := 0; i < w.workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w.runLoop(ctx)
		}()
	}
	wg.Wait()
}

func (w *Worker) runLoop(ctx context.Context) {
	ticker := time.NewTicker(w.pollTicker)
	defer ticker.Stop()
	for {
		w.drain(ctx)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (w *Worker) drain(ctx context.Context) {
	for ctx.Err() == nil {
		job, ok, err := w.jobs.ClaimNext(ctx, w.lease)
		if err != nil {
			if ctx.Err() == nil {
				slog.Error("assistant worker claim failed", "error", err)
			}
			return
		}
		if !ok {
			return
		}
		if err := w.processor.Process(ctx, job); err != nil && ctx.Err() == nil {
			slog.Error("assistant worker process failed", "job_id", job.ID, "error", err)
		}
	}
}
