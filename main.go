package main

import (
	"github.com/JoveFYL/bizzy/internal/api"
	"github.com/JoveFYL/bizzy/internal/handler"
	"github.com/JoveFYL/bizzy/internal/model"
	"github.com/JoveFYL/bizzy/internal/queue"
	"github.com/JoveFYL/bizzy/internal/worker"
)

func main() {
	q := queue.NewMemoryQueue(100)
	r := api.NewRouter(q)

	pool := worker.NewPool(3, q, q.Dequeue(), q.Enqueue)
	pool.RegisterHandler(model.TypeImageProcessing, handler.ProcessImageHandler)
	pool.Start()

	r.Run(":8080")
}
