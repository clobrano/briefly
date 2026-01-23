package notifier

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/clobrano/briefly/internal/models"
)

type Notifier struct {
	topic  string
	client *http.Client
}

func New(topic string) *Notifier {
	if topic == "" {
		return nil
	}
	return &Notifier{
		topic: topic,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (n *Notifier) SendSuccess(ctx context.Context, job *models.Job) error {
	if n == nil || n.topic == "" {
		return nil
	}

	title := fmt.Sprintf("Briefly: %s summary ready", job.ContentType)
	message := fmt.Sprintf("Summary for %s is ready.\n\nJob ID: %s", job.URL, job.ID)

	return n.send(ctx, title, message, "default", "white_check_mark")
}

func (n *Notifier) SendFailure(ctx context.Context, job *models.Job) error {
	if n == nil || n.topic == "" {
		return nil
	}

	title := fmt.Sprintf("Briefly: %s processing failed", job.ContentType)
	message := fmt.Sprintf("Failed to process %s\n\nError: %s\n\nJob ID: %s", job.URL, job.Error, job.ID)

	return n.send(ctx, title, message, "high", "x")
}

func (n *Notifier) send(ctx context.Context, title, message, priority, tags string) error {
	url := fmt.Sprintf("https://ntfy.sh/%s", n.topic)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBufferString(message))
	if err != nil {
		return err
	}

	req.Header.Set("Title", title)
	req.Header.Set("Priority", priority)
	req.Header.Set("Tags", tags)

	resp, err := n.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("ntfy returned status %d", resp.StatusCode)
	}

	return nil
}
