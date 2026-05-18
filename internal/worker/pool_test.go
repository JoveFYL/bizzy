package worker

import (
	"errors"
	"sync/atomic"
	"testing"

	"github.com/JoveFYL/bizzy/internal/model"
	"github.com/JoveFYL/bizzy/internal/queue"
)

func CreatePoolAndQueueHelper(t *testing.T, workers int) (*Pool, *queue.MemoryQueue) {
	// skips printing this function in test logs
	t.Helper()

	q := queue.NewMemoryQueue(100)
	p := NewPool(workers, q, q.Dequeue(), q.Enqueue)

	// close queue when test finishes to stop workers
	t.Cleanup(func() {
		q.Close()
	})

	return p, q
}

func TestWorkerPool_Success(t *testing.T) {
	pool, q := CreatePoolAndQueueHelper(t, 2)

	doneSignal := make(chan struct{})

	pool.RegisterHandler("test_job", func(job *model.Job) (any, error) {
		close(doneSignal) // signal that job is done
		return "done", nil
	})

	// launch goroutines in background
	pool.Start()

	job := &model.Job{
		ID:   "123",
		Type: "test_job",
	}
	q.Enqueue(job)

	// wait here for worker to finish job
	<-doneSignal

	updatedJob, _ := q.GetJob("123")
	if updatedJob.Status != model.StatusCompleted {
		t.Errorf("expected completed status, got %s", updatedJob.Status)
	}
}

func TestWorkerPool_RetryLogic(t *testing.T) {
	q := queue.NewMemoryQueue(10)
	pool := NewPool(1, q, q.Dequeue(), q.Enqueue)

	doneSignal := make(chan struct{})
	var attempts atomic.Int32

	pool.RegisterHandler("fail_job", func(job *model.Job) (any, error) {
		if attempts.Add(1) == 2 {
			close(doneSignal) // signal that job is done after retry
		}
		return nil, errors.New("temporary error")
	})

	pool.Start()

	job := &model.Job{
		ID:       "retry_test",
		Type:     "fail_job",
		MaxRetry: 1,
	}
	q.Enqueue(job)

	<-doneSignal

	updatedJob, _ := q.GetJob("retry_test")

	if updatedJob.Status != model.StatusFailed {
		t.Errorf("expected failed status, got %s", updatedJob.Status)
	}
	if updatedJob.Retries != 1 {
		t.Errorf("expected 1 retry, got %d", updatedJob.Retries)
	}
}
