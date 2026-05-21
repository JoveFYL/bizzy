package main_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/JoveFYL/bizzy/internal/api"
	"github.com/JoveFYL/bizzy/internal/model"
	"github.com/JoveFYL/bizzy/internal/queue"
	"github.com/JoveFYL/bizzy/internal/worker"
	"github.com/gin-gonic/gin"
)

func TestEndToEnd_JobSubmittedAndProcessed(t *testing.T) {
	gin.SetMode(gin.TestMode)

	q := queue.NewMemoryQueue(100)
	r := api.NewRouter(q)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	processed := make(chan struct{})
	pool := worker.NewPool(1, q, q.Dequeue(), q.Enqueue)
	pool.RegisterHandler(model.TypeImageProcessing, func(job *model.Job) (any, error) {
		close(processed)
		return "done", nil
	})
	pool.Start(ctx)

	w := httptest.NewRecorder()
	body := `{"type":"image_processing","payload":{"data":"test"}}`
	req, _ := http.NewRequest("POST", "/submit", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", w.Code)
	}

	select {
	case <-processed:
	case <-time.After(2 * time.Second):
		t.Fatal("job was not processed within timeout")
	}
}
