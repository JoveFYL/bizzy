package queue

import (
	"fmt"
	"sync"

	"github.com/JoveFYL/bizzy/internal/model"
)

type MemoryQueue struct {
	jobs chan *model.Job

	// reader writers mutex
	// reading much more frequent than writing
	// RWMutex allows many users to read map simultaneously
	mu sync.RWMutex

	// keep track of status and persistence
	store map[string]*model.Job
}

func NewMemoryQueue(size int) *MemoryQueue {
	return &MemoryQueue{
		jobs:  make(chan *model.Job, size),
		store: make(map[string]*model.Job),
	}
}

// Adds job to queue and stores job in map for future lookup
func (q *MemoryQueue) Enqueue(job *model.Job) error {
	// save job to map using mutex
	q.mu.Lock()
	q.store[job.ID] = job
	q.mu.Unlock()

	// send job to channel
	select {
	case q.jobs <- job:
		return nil

	// if buffer is full, block
	default:
		return fmt.Errorf("Queue is full (capacity: %d)", cap(q.jobs))
	}
}

// Return receive-only channel, don't return a job
// worker will read from this receive-only channel
func (q *MemoryQueue) Dequeue() <-chan *model.Job {
	return q.jobs
}

func (q *MemoryQueue) GetJob(id string) (*model.Job, bool) {
	q.mu.RLock()
	defer q.mu.RUnlock()

	job, ok := q.store[id]
	return job, ok
}
