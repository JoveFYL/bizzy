package worker

import (
	"fmt"
	"log/slog"
	"math"
	"time"

	"github.com/JoveFYL/bizzy/internal/model"
	"github.com/JoveFYL/bizzy/internal/queue"
)

type HandlerFunc func(*model.Job) (any, error)

// pull job from queue and work on executing the job
type Pool struct {
	workers  int
	jobs     <-chan *model.Job
	queue    *queue.MemoryQueue
	handlers map[model.JobType]HandlerFunc
	requeue  func(*model.Job) error
	// channel to signal all workers to stop
	quit chan struct{}
}

func NewPool(workers int, queue *queue.MemoryQueue, jobs <-chan *model.Job, requeue func(*model.Job) error) *Pool {
	return &Pool{
		workers:  workers,
		jobs:     jobs,
		queue:    queue,
		handlers: make(map[model.JobType]HandlerFunc),
		requeue:  requeue,
		quit:     make(chan struct{}),
	}
}

// RegisterHandler allows us to add logic for different job types
func (p *Pool) RegisterHandler(jobType model.JobType, handler HandlerFunc) {
	p.handlers[jobType] = handler
}

// Start launches the goroutines
func (p *Pool) Start() {
	for i := 0; i < p.workers; i++ {
		go p.RunWorker(i)
	}
	slog.Info("worker pool started", "count", p.workers)
}

func (p *Pool) RunWorker(id int) {
	slog.Info("worker started", "worker_id", id)

	for job := range p.jobs {
		p.ProcessJob(id, job)
	}

	slog.Info("worker stopped", "worker_id", id)
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

	// mark job in-progress
	// get store copy
	current, ok := p.queue.UpdateJob(job.ID, func(job *model.Job) {
		job.Status = model.StatusProcessing
		job.UpdatedAt = time.Now()
	})
	if !ok {
		logger.Error("job not found in store")
		return
	}

	// run handler on store copy
	result, err := handler(current)

	// if error, retry
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
				// Get a fresh copy from the store to enqueue.
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
