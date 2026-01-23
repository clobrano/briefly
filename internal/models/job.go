package models

import (
	"time"
)

type ContentType string

const (
	ContentTypeYouTube ContentType = "youtube"
	ContentTypeText    ContentType = "text"
	ContentTypeUnknown ContentType = "unknown"
)

type JobStatus string

const (
	JobStatusPending    JobStatus = "pending"
	JobStatusProcessing JobStatus = "processing"
	JobStatusCompleted  JobStatus = "completed"
	JobStatusFailed     JobStatus = "failed"
)

type Job struct {
	ID           string      `json:"id"`
	FilePath     string      `json:"file_path"`
	URL          string      `json:"url"`
	CustomPrompt string      `json:"custom_prompt,omitempty"`
	ContentType  ContentType `json:"content_type"`
	Status       JobStatus   `json:"status"`
	Content      string      `json:"content,omitempty"`
	Summary      string      `json:"summary,omitempty"`
	Error        string      `json:"error,omitempty"`
	CreatedAt    time.Time   `json:"created_at"`
	UpdatedAt    time.Time   `json:"updated_at"`
	Retries      int         `json:"retries"`
}

func NewJob(filePath, url, customPrompt string) *Job {
	now := time.Now()
	return &Job{
		ID:           generateID(),
		FilePath:     filePath,
		URL:          url,
		CustomPrompt: customPrompt,
		ContentType:  ContentTypeUnknown,
		Status:       JobStatusPending,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

func generateID() string {
	return time.Now().Format("20060102-150405.000")
}
