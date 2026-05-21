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

func CreateRouterHelper(t *testing.T) (*gin.Engine, queue.Queue) {
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
	if err != nil {
		t.Fatalf("failed to parse json response: %v", err)
	}

	returnedID := res["job_id"].(string)

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
		t.Fatalf("failed to parse json response: %v", err)
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

func TestGetJob(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router, _ := CreateRouterHelper(t)

	// create fake browser tab/connection
	submitRecorder := httptest.NewRecorder()
	payload := `{"type":"image_processing","payload":{"data":"hello"}}`
	submitReq, _ := http.NewRequest("POST", "/submit", strings.NewReader(payload))
	submitReq.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(submitRecorder, submitReq)

	if submitRecorder.Code != http.StatusAccepted {
		t.Errorf("expected status code %d, got %d", http.StatusAccepted, submitRecorder.Code)
	}

	var submitRes map[string]any
	submitErr := json.Unmarshal(submitRecorder.Body.Bytes(), &submitRes)

	if submitErr != nil {
		t.Fatalf("failed to parse json response: %v", submitErr)
	}

	submittedID := submitRes["job_id"].(string)

	getRecorder := httptest.NewRecorder()
	getReq, _ := http.NewRequest("GET", "/job/"+submittedID, nil)
	router.ServeHTTP(getRecorder, getReq)

	var getRes map[string]any
	getErr := json.Unmarshal(getRecorder.Body.Bytes(), &getRes)

	if getErr != nil {
		t.Fatalf("failed to parse json response: %v", getErr)
	}

	returnedID := getRes["job_id"].(string)

	if getRecorder.Code != http.StatusOK {
		t.Errorf("The API returned ID '%s', but that job does not exist inside the queue storage!", submittedID)
	}

	if returnedID != submittedID {
		t.Errorf("Storage ID '%s' does not match API returned ID '%s'", returnedID, submittedID)
	}

	if getRes["message"] != "success" {
		t.Errorf("expected message 'success', got %s", getRes["message"])
	}

}
