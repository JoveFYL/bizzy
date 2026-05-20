package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/JoveFYL/bizzy/internal/queue"
	"github.com/gin-gonic/gin"
)

func CreateRouterHelper(t *testing.T) (*gin.Engine, *queue.MemoryQueue) {
	// skips printing this function in test logs
	t.Helper()
	q := queue.NewMemoryQueue(100)

	return NewRouter(q), q
}

func TestSubmitJob(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router, q := CreateRouterHelper(t)

	// create fake browser tab/connection
	w := httptest.NewRecorder()
	payload := `{"type":"image_processing","payload":{"data":"hello"}}`
	req, _ := http.NewRequest("POST", "/submit", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Errorf("expected status code %d, got %d", http.StatusAccepted, w.Code)
	}

	var res map[string]any
	err := json.Unmarshal(w.Body.Bytes(), &res)
	returnedID := res["job_id"].(string)

	if err != nil {
		t.Errorf("failed to parse json response: %v", err)
	}

	savedJob, found := q.GetJob(returnedID)

	if !found {
		t.Errorf("The API returned ID '%s', but that job does not exist inside the queue storage!", returnedID)
	}

	if savedJob.ID != returnedID {
		t.Errorf("Storage ID '%s' does not match API returned ID '%s'", savedJob.ID, returnedID)
	}

	if res["message"] != "job submitted successfully" {
		t.Errorf("expected message 'job submitted successfully', got %s", res["message"])
	}

}

func TestGetAllJobs(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router, _ := CreateRouterHelper(t)

	// submit 3 jobs
	for range 3 {
		recorder := httptest.NewRecorder()
		payload := `{"type":"image_processing","payload":{"data":"hello"}}`
		req, _ := http.NewRequest("POST", "/submit", strings.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(recorder, req)
	}

	req, _ := http.NewRequest("GET", "/jobs", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status code %d, got %d", http.StatusOK, w.Code)
	}

	var res map[string]any

	err := json.Unmarshal(w.Body.Bytes(), &res)
	if err != nil {
		t.Errorf("failed to parse json response: %v", err)
	}

	if res["message"] != "success" {
		t.Errorf("expected message 'job submitted successfully', got %s", res["message"])
	}

	jobs, ok := res["jobs"].([]any)

	if !ok {
		t.Errorf("expected 'jobs' to be an array, got %T", res["jobs"])
	}

	if len(jobs) != 3 {
		t.Errorf("expected 3 jobs, got %d", len(jobs))
	}

}
