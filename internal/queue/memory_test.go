package queue

import (
	"testing"

	"github.com/JoveFYL/bizzy/internal/model"
)

func TestEnqueue(t *testing.T) {
	size := 2
	q := NewMemoryQueue(size)

	job1 := &model.Job{
		ID:   "job-1",
		Type: model.TypeEmailSending,
	}

	err := q.Enqueue(job1)
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}

	storedJob, ok := q.GetJob("job-1")
	if !ok {
		t.Errorf("job-1 was not found in the store")
	}
	if storedJob.ID != "job-1" {
		t.Errorf("expected job-1, got %s", storedJob.ID)
	}

	job2 := &model.Job{ID: "job-2"}
	job3 := &model.Job{ID: "job-3"}

	q.Enqueue(job2)

	err = q.Enqueue(job3) // Should fail (3rd job in size-2 queue)
	if err == nil {
		t.Errorf("expected error for full queue, but got nil")
	}
}
