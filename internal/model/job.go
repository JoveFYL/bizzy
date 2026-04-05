package model

import "time"

type JobStatus string
type JobType string

// emulate enum using custome types
const (
	StatusPending    JobStatus = "pending"
	StatusProcessing JobStatus = "processing"
	StatusCompleted  JobStatus = "completed"
	StatusFailed     JobStatus = "failed"

	TypeEmailSending    JobType = "email_sending"
	TypeReportCreating  JobType = "report_creating"
	TypeImageProcessing JobType = "image_processing"
)

// Job represents a unit of work to be processed.
type Job struct {
	// field type and struct tag for JSON serialisation
	ID        string    `json:"id"`
	Type      JobType   `json:"type"`
	Status    JobStatus `json:"status"`
	Payload   any       `json:"payload"` // payload can be any data structure first, depending on job type
	Retries   int       `json:"retries"`
	MaxRetry  int       `json:"max_retry"`
	CreatedAt time.Time `json:"created_at"`
}
