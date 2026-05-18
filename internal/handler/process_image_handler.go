package handler

import (
	"time"

	"github.com/JoveFYL/bizzy/internal/model"
)

func ProcessImageHandler(job *model.Job) (any, error) {
	// Simulate image processing time
	time.Sleep(2 * time.Second)

	// For demonstration, we just return a success message
	return "Image processed successfully", nil
}
