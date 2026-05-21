package worker

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"sync"
	"time"

	"github.com/JoveFYL/bizzy/internal/model"
	"github.com/JoveFYL/bizzy/internal/queue"
)

type HandlerFunc func(*model.Job) (any, error)

type Pool struct {
	workers  int
	jobs     <-chan *model.Job
	queue    queue.Queue
	handlers map[model.JobType]HandlerFunc
	requeue  func(*model.Job) error
	wg       sync.WaitGroup
}

func NewPool(workers int, q queue.Queue, jobs <-chan *model.Job, requeue func(*model.Job) error) *Pool {
	return &Pool{
		workers:  workers,
		jobs:     jobs,
		queue:    q,
		handlers: make(map[model.JobType]HandlerFunc),
		requeue:  requeue,
	}
}

func (p *Pool) RegisterHandler(jobType model.JobType, handler HandlerFunc) {
	p.handlers[jobType] = handler
}

// launch worker goroutines
// Call Wait() to block until all workers have drained and finished.
func (p *Pool) Start(ctx context.Context) {
	for i := 0; i < p.workers; i++ {
		p.wg.Add(1)
		go func(id int) {
			defer p.wg.Done()
			p.runWorker(ctx, id)
		}(i)
	}
	slog.Info("worker pool started", "count", p.workers)
}

// Wait blocks until all workers have stopped processing.
func (p *Pool) Wait() {
	p.wg.Wait()
}

func (p *Pool) runWorker(ctx context.Context, id int) {
	slog.Info("worker started", "worker_id", id)
	for {
		select {
		case <-ctx.Done():
			slog.Info("worker stopping (context cancelled)", "worker_id", id)
			return
		case job, ok := <-p.jobs:
			if !ok {
				slog.Info("worker stopped (channel closed)", "worker_id", id)
				return
			}
			p.ProcessJob(id, job)
		}
	}
}

func (p *Pool) ProcessJob(workerID int, job *model.Job) {
	logger := slog.With("worker_id", workerID, "job_id", job.ID, "job_type", job.Type)
	logger.Info("processing job")

	handler, ok := p.handlers[job.Type]
	if !ok {
		p.queue.UpdateJob(job.ID, func(job *model.Job) {
			job.Status = model.StatusFailed
			job.Error = fmt.Sprintf("no handler registered for job type: %s", job.Type)
			job.UpdatedAt = time.Now()
		})
		logger.Error("no handler for job type")
		return
	}

	current, ok := p.queue.UpdateJob(job.ID, func(job *model.Job) {
		job.Status = model.StatusProcessing
		job.UpdatedAt = time.Now()
	})
	if !ok {
		logger.Error("job not found in store")
		return
	}

	result, err := handler(current)

	if err != nil {
		logger.Info("job failed", "error", err, "retries", job.Retries, "max_retry", job.MaxRetry)

		var retryCount int
		var shouldRetry bool

		p.queue.UpdateJob(job.ID, func(j *model.Job) {
			j.Error = err.Error()
			j.UpdatedAt = time.Now()
			if j.Retries < j.MaxRetry {
				j.Retries++
				j.Status = model.StatusPending
				shouldRetry = true
				retryCount = j.Retries
			} else {
				j.Status = model.StatusFailed
			}
		})

		if shouldRetry {
			backoff := time.Duration(math.Exp2(float64(retryCount))) * time.Second
			logger.Info("scheduling retry", "attempt", retryCount, "backoff", backoff)
			time.AfterFunc(backoff, func() {
				copy, ok := p.queue.GetJob(job.ID)
				if !ok {
					return
				}
				if err := p.requeue(copy); err != nil {
					logger.Error("failed to requeue", "error", err)
					p.queue.UpdateJob(job.ID, func(j *model.Job) {
						j.Status = model.StatusFailed
						j.Error = fmt.Sprintf("requeue failed: %v", err)
						j.UpdatedAt = time.Now()
					})
				}
			})
		} else {
			logger.Info("job permanently failed, no more retries")
		}
	} else {
		p.queue.UpdateJob(job.ID, func(j *model.Job) {
			j.Status = model.StatusCompleted
			j.Result = result
			j.UpdatedAt = time.Now()
		})
		logger.Info("job completed")
	}
}
