package main

import (
	"fmt"
	"time"

	"github.com/JoveFYL/bizzy/internal/handler"
	"github.com/JoveFYL/bizzy/internal/model"
	"github.com/JoveFYL/bizzy/internal/queue"
	"github.com/JoveFYL/bizzy/internal/worker"
)

func main() {
	q := queue.NewMemoryQueue(100)
	pool := worker.NewPool(3, q, q.Dequeue(), q.Enqueue)
	pool.RegisterHandler(model.TypeImageProcessing, handler.ProcessImageHandler)
	pool.Start()

	q.Enqueue(&model.Job{
		ID:     "test-1",
		Type:   model.TypeImageProcessing,
		Status: model.StatusPending,
	})

	time.Sleep(5 * time.Second) // Wait for processing
	job, _ := q.GetJob("test-1")
	fmt.Println("Final status:", job.Status)
}
