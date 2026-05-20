package api

import (
	"time"

	"github.com/JoveFYL/bizzy/internal/model"
	"github.com/JoveFYL/bizzy/internal/queue"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type JobRequest struct {
	model.JobType `json:"type" binding:"required"`
	Payload       any `json:"payload" binding:"required"`
}

func NewRouter(q *queue.MemoryQueue) *gin.Engine {
	r := gin.Default()
	r.POST("/submit", func(g *gin.Context) {
		submitJob(g, q)
	})
	r.GET("/jobs", func(g *gin.Context) {
		getAllJobs(g, q)
	})
	r.GET("/job/:id", func(g *gin.Context) {
		getJob(g, q)
	})
	return r
}

// submit job
func submitJob(c *gin.Context, q *queue.MemoryQueue) {
	var req JobRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"message": "bad request", "error": err.Error()})
		return
	}

	// create job
	job := &model.Job{
		ID:        uuid.New().String(),
		Type:      req.JobType,
		Status:    model.StatusPending,
		Payload:   req.Payload,
		Retries:   0,
		MaxRetry:  3,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := q.Enqueue(job); err != nil {
		c.JSON(500, gin.H{"message": "failed to enqueue job", "error": err.Error()})
		return
	}

	c.JSON(202, gin.H{"message": "job submitted successfully", "job_id": job.ID})
}

func getAllJobs(c *gin.Context, q *queue.MemoryQueue) {
	jobs := q.GetAllJobs()
	c.JSON(200, gin.H{
		"message": "success",
		"jobs":    jobs,
	})
}

// get job by ID
func getJob(c *gin.Context, q *queue.MemoryQueue) {
	id := c.Param("id")
	job, ok := q.GetJob(id)

	if !ok {
		c.JSON(404, gin.H{"message": "job not found"})
		return
	}

	c.JSON(200, gin.H{
		"message": "success",
		"job":     job,
	})
}
