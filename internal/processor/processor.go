package processor

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/clobrano/briefly/internal/config"
	"github.com/clobrano/briefly/internal/models"
	"github.com/clobrano/briefly/internal/notifier"
	"github.com/clobrano/briefly/internal/queue"
	"github.com/clobrano/briefly/internal/summarizer"
)

// ErrOutputExists is returned when attempting to write a summary that already exists
var ErrOutputExists = errors.New("output file already exists")

const (
	maxRetries  = 3
	baseBackoff = 5 * time.Second
)

type Processor struct {
	cfg        *config.Config
	queue      *queue.Queue
	textProc   *TextExtractor
	ytProc     *YouTubeProcessor
	summarizer summarizer.Summarizer
	notifier   *notifier.Notifier
	done       chan struct{}
}

func New(cfg *config.Config, q *queue.Queue, sum summarizer.Summarizer, ntfy *notifier.Notifier) *Processor {
	return &Processor{
		cfg:        cfg,
		queue:      q,
		textProc:   NewTextExtractor(),
		ytProc:     NewYouTubeProcessor(cfg.WhisperModel),
		summarizer: sum,
		notifier:   ntfy,
		done:       make(chan struct{}),
	}
}

func (p *Processor) Start() {
	go p.run()
}

func (p *Processor) Stop() {
	close(p.done)
}

func (p *Processor) run() {
	for {
		select {
		case <-p.done:
			return
		case <-p.queue.Wait():
			p.processQueue()
		}
	}
}

func (p *Processor) processQueue() {
	for {
		job := p.queue.Dequeue()
		if job == nil {
			return
		}

		select {
		case <-p.done:
			return
		default:
			p.processJob(job)
		}
	}
}

func (p *Processor) processJob(job *models.Job) {
	log.Printf("Processing job %s: %s", job.Filename, job.URL)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	// Detect content type first
	job.ContentType = DetectContentType(job.URL)
	if job.ContentType == models.ContentTypeUnknown {
		p.failJob(job, fmt.Errorf("unknown content type for URL: %s", job.URL))
		return
	}

	// Check if output already exists (skip duplicate processing)
	exists, err := p.outputExists(job)
	if err != nil {
		log.Printf("Error checking output file for job %s: %v", job.Filename, err)
		p.failJob(job, err)
		return
	}
	if exists {
		log.Printf("Skipping job %s: output file already exists", job.Filename)
		if p.notifier != nil {
			if err := p.notifier.SendSkipped(ctx, job); err != nil {
				log.Printf("Warning: failed to send skipped notification for job %s: %v", job.Filename, err)
			}
		}
		p.completeJob(job)
		return
	}

	// Send start notification only on first attempt
	if p.notifier != nil && job.Retries == 0 {
		if err := p.notifier.SendStart(ctx, job); err != nil {
			log.Printf("Warning: failed to send start notification for job %s: %v", job.Filename, err)
		}
	}

	// Extract content
	var content string

	switch job.ContentType {
	case models.ContentTypeYouTube:
		content, err = p.ytProc.Process(ctx, job.URL)
	case models.ContentTypeText:
		content, err = p.textProc.Extract(ctx, job.URL)
	}

	if err != nil {
		if p.shouldRetry(job) {
			p.retryJob(job, err)
			return
		}
		p.failJob(job, err)
		return
	}

	job.Content = content

	// Summarize
	summary, err := p.summarizer.Summarize(ctx, content, job.CustomPrompt, job.ContentType)
	if err != nil {
		if p.shouldRetry(job) {
			p.retryJob(job, err)
			return
		}
		p.failJob(job, err)
		return
	}

	job.Summary = summary

	// Save summary
	if err := p.saveSummary(job); err != nil {
		// Race condition: another worker already created the output file
		if errors.Is(err, ErrOutputExists) {
			log.Printf("Skipping job %s: output file created by concurrent worker", job.Filename)
			if p.notifier != nil {
				if notifyErr := p.notifier.SendSkipped(ctx, job); notifyErr != nil {
					log.Printf("Warning: failed to send skipped notification for job %s: %v", job.Filename, notifyErr)
				}
			}
			p.completeJob(job)
			return
		}
		log.Printf("Error: failed to save summary for job %s: %v", job.Filename, err)
		job.Error = fmt.Sprintf("failed to save summary: %v", err)
		p.failJob(job, fmt.Errorf("failed to save summary: %w", err))
		return
	}

	// Notify success
	if p.notifier != nil {
		if err := p.notifier.SendSuccess(ctx, job); err != nil {
			log.Printf("Warning: failed to send notification for job %s: %v", job.Filename, err)
		}
	}

	// Complete job
	p.completeJob(job)
}

func (p *Processor) shouldRetry(job *models.Job) bool {
	return job.Retries < maxRetries
}

func (p *Processor) retryJob(job *models.Job, err error) {
	job.Retries++
	job.Status = models.JobStatusPending
	job.Error = err.Error()
	job.UpdatedAt = time.Now()

	backoff := time.Duration(job.Retries) * baseBackoff
	log.Printf("Job %s failed (attempt %d/%d): %v. Retrying in %v",
		job.Filename, job.Retries, maxRetries, err, backoff)

	p.queue.Update(job)

	// Schedule retry
	go func() {
		time.Sleep(backoff)
		p.queue.Notify()
	}()
}

func (p *Processor) failJob(job *models.Job, err error) {
	job.Status = models.JobStatusFailed
	job.Error = err.Error()
	job.UpdatedAt = time.Now()

	log.Printf("Job %s failed permanently: %v", job.Filename, err)

	// Notify failure
	if p.notifier != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if notifyErr := p.notifier.SendFailure(ctx, job); notifyErr != nil {
			log.Printf("Warning: failed to send failure notification for job %s: %v", job.Filename, notifyErr)
		}
	}

	p.queue.Update(job)
}

func (p *Processor) completeJob(job *models.Job) {
	job.Status = models.JobStatusCompleted
	job.UpdatedAt = time.Now()

	log.Printf("Job %s completed successfully", job.Filename)

	// Remove the input file
	if job.FilePath != "" {
		os.Remove(job.FilePath)
	}

	p.queue.Remove(job.ID)
}

func (p *Processor) getOutputPath(job *models.Job) string {
	// Use input filename as base for output, fallback to job ID
	var baseName string
	if job.FilePath != "" {
		baseName = filepath.Base(job.FilePath)
		ext := filepath.Ext(baseName)
		baseName = strings.TrimSuffix(baseName, ext)
	} else {
		baseName = job.ID
	}

	filename := fmt.Sprintf("%s.md", baseName)
	return filepath.Join(p.cfg.OutputDir, filename)
}

func (p *Processor) outputExists(job *models.Job) (bool, error) {
	path := p.getOutputPath(job)
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	// Other errors (permission denied, etc.)
	return false, fmt.Errorf("failed to check output file: %w", err)
}

func (p *Processor) saveSummary(job *models.Job) error {
	if err := os.MkdirAll(p.cfg.OutputDir, 0755); err != nil {
		return err
	}

	path := p.getOutputPath(job)

	content := fmt.Sprintf("# Summary\n\n**URL:** %s\n**Type:** %s\n**Generated:** %s\n\n---\n\n%s",
		job.URL,
		job.ContentType,
		time.Now().Format(time.RFC3339),
		job.Summary,
	)

	// Use O_EXCL for atomic creation - fails if file already exists (race condition)
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		if os.IsExist(err) {
			return ErrOutputExists
		}
		return err
	}
	defer f.Close()

	_, err = f.WriteString(content)
	return err
}
