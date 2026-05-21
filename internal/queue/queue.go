package queue

import "github.com/JoveFYL/bizzy/internal/model"

// Queue is the contract the worker pool and router depend on.
// MemoryQueue implement this.
type Queue interface {
	Enqueue(job *model.Job) error
	Dequeue() <-chan *model.Job
	GetJob(id string) (*model.Job, bool)
	GetAllJobs() []*model.Job
	UpdateJob(id string, fn func(*model.Job)) (*model.Job, bool)
	Close()
}
