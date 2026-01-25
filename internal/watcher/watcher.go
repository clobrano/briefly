package watcher

import (
	"bufio"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"gopkg.in/yaml.v3"

	"github.com/clobrano/briefly/internal/models"
	"github.com/clobrano/briefly/internal/queue"
)

type Watcher struct {
	fsWatcher    *fsnotify.Watcher
	watchDir     string
	queue        *queue.Queue
	debounceTime time.Duration
	pending      map[string]time.Time
	mu           sync.Mutex
	done         chan struct{}
}

func New(watchDir string, q *queue.Queue) (*Watcher, error) {
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	return &Watcher{
		fsWatcher:    fsw,
		watchDir:     watchDir,
		queue:        q,
		debounceTime: 500 * time.Millisecond,
		pending:      make(map[string]time.Time),
		done:         make(chan struct{}),
	}, nil
}

func (w *Watcher) Start() error {
	if err := w.fsWatcher.Add(w.watchDir); err != nil {
		return err
	}

	// Process existing files on startup
	if err := w.processExisting(); err != nil {
		log.Printf("Warning: error processing existing files: %v", err)
	}

	go w.run()
	go w.debounceLoop()

	return nil
}

func (w *Watcher) Stop() error {
	close(w.done)
	return w.fsWatcher.Close()
}

func (w *Watcher) processExisting() error {
	entries, err := os.ReadDir(w.watchDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if w.isValidFile(entry.Name()) {
			path := filepath.Join(w.watchDir, entry.Name())
			w.processFile(path)
		}
	}

	return nil
}

func (w *Watcher) run() {
	for {
		select {
		case <-w.done:
			return
		case event, ok := <-w.fsWatcher.Events:
			if !ok {
				return
			}
			if event.Has(fsnotify.Create) || event.Has(fsnotify.Write) {
				if w.isValidFile(filepath.Base(event.Name)) {
					w.scheduleProcess(event.Name)
				}
			}
		case err, ok := <-w.fsWatcher.Errors:
			if !ok {
				return
			}
			log.Printf("Watcher error: %v", err)
		}
	}
}

func (w *Watcher) scheduleProcess(path string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.pending[path] = time.Now().Add(w.debounceTime)
}

func (w *Watcher) debounceLoop() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-w.done:
			return
		case <-ticker.C:
			w.mu.Lock()
			now := time.Now()
			var toProcess []string
			for path, deadline := range w.pending {
				if now.After(deadline) {
					toProcess = append(toProcess, path)
					delete(w.pending, path)
				}
			}
			w.mu.Unlock()

			for _, path := range toProcess {
				w.processFile(path)
			}
		}
	}
}

func (w *Watcher) processFile(path string) {
	result, err := parseInputFile(path)
	if err != nil {
		log.Printf("Error parsing file %s: %v", path, err)
		return
	}

	var job *models.Job
	if result.IsDirectText {
		if result.Text == "" {
			log.Printf("Error: empty text content in file %s", path)
			return
		}
		job = models.NewJobWithContent(path, result.Text, result.CustomPrompt)
		log.Printf("Queued job %s for direct text summarization", job.Filename)
	} else {
		if result.URL == "" {
			log.Printf("Error: empty URL in file %s", path)
			return
		}
		job = models.NewJob(path, result.URL, result.CustomPrompt)
		log.Printf("Queued job %s for URL: %s", job.Filename, result.URL)
	}

	if err := w.queue.Enqueue(job); err != nil {
		log.Printf("Error enqueuing job for %s: %v", path, err)
		return
	}
}

func (w *Watcher) isValidFile(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	return ext == ".briefly" || ext == ".url" || ext == ".txt"
}

type inputFile struct {
	URL    string `yaml:"url"`
	Text   string `yaml:"text"`
	Prompt string `yaml:"prompt"`
}

// parseResult holds the result of parsing an input file
type parseResult struct {
	URL          string
	Text         string
	CustomPrompt string
	IsDirectText bool
}

func parseInputFile(path string) (*parseResult, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	content := strings.Join(lines, "\n")
	content = strings.TrimSpace(content)

	// Check for YAML front matter
	if strings.HasPrefix(content, "---") {
		parts := strings.SplitN(content, "---", 3)
		if len(parts) >= 3 {
			var input inputFile
			if err := yaml.Unmarshal([]byte(parts[1]), &input); err == nil {
				// Check if text field is provided (direct text summarization)
				if input.Text != "" {
					return &parseResult{
						Text:         strings.TrimSpace(input.Text),
						CustomPrompt: strings.TrimSpace(input.Prompt),
						IsDirectText: true,
					}, nil
				}
				// URL-based summarization
				if input.URL != "" {
					return &parseResult{
						URL:          strings.TrimSpace(input.URL),
						CustomPrompt: strings.TrimSpace(input.Prompt),
						IsDirectText: false,
					}, nil
				}
			}
		}
	}

	// Simple format: check if content looks like a URL
	if len(lines) > 0 {
		firstLine := strings.TrimSpace(lines[0])
		if isURL(firstLine) {
			return &parseResult{
				URL:          firstLine,
				IsDirectText: false,
			}, nil
		}
	}

	// Treat as direct text if no URL found
	if content != "" {
		return &parseResult{
			Text:         content,
			IsDirectText: true,
		}, nil
	}

	return &parseResult{}, nil
}

// isURL checks if the string looks like a URL
func isURL(s string) bool {
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}
