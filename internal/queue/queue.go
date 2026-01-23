package queue

import (
	"encoding/json"
	"os"
	"sync"

	"github.com/clobrano/briefly/internal/models"
)

type Queue struct {
	mu           sync.Mutex
	jobs         []*models.Job
	persistPath  string
	notification chan struct{}
}

func New(persistPath string) (*Queue, error) {
	q := &Queue{
		jobs:         make([]*models.Job, 0),
		persistPath:  persistPath,
		notification: make(chan struct{}, 1),
	}

	if err := q.load(); err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	return q, nil
}

func (q *Queue) Enqueue(job *models.Job) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.jobs = append(q.jobs, job)

	select {
	case q.notification <- struct{}{}:
	default:
	}

	return q.persist()
}

func (q *Queue) Dequeue() *models.Job {
	q.mu.Lock()
	defer q.mu.Unlock()

	for _, job := range q.jobs {
		if job.Status == models.JobStatusPending {
			job.Status = models.JobStatusProcessing
			q.persist()
			return job
		}
	}
	return nil
}

func (q *Queue) Update(job *models.Job) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	for i, j := range q.jobs {
		if j.ID == job.ID {
			q.jobs[i] = job
			return q.persist()
		}
	}
	return nil
}

func (q *Queue) Remove(jobID string) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	for i, j := range q.jobs {
		if j.ID == jobID {
			q.jobs = append(q.jobs[:i], q.jobs[i+1:]...)
			return q.persist()
		}
	}
	return nil
}

func (q *Queue) Wait() <-chan struct{} {
	return q.notification
}

func (q *Queue) Notify() {
	select {
	case q.notification <- struct{}{}:
	default:
	}
}

func (q *Queue) Len() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.jobs)
}

func (q *Queue) PendingCount() int {
	q.mu.Lock()
	defer q.mu.Unlock()

	count := 0
	for _, job := range q.jobs {
		if job.Status == models.JobStatusPending {
			count++
		}
	}
	return count
}

func (q *Queue) persist() error {
	if q.persistPath == "" {
		return nil
	}

	data, err := json.MarshalIndent(q.jobs, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(q.persistPath, data, 0644)
}

func (q *Queue) load() error {
	if q.persistPath == "" {
		return nil
	}

	data, err := os.ReadFile(q.persistPath)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, &q.jobs)
}
