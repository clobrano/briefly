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
	url, customPrompt, err := parseInputFile(path)
	if err != nil {
		log.Printf("Error parsing file %s: %v", path, err)
		return
	}

	job := models.NewJob(path, url, customPrompt)
	if err := w.queue.Enqueue(job); err != nil {
		log.Printf("Error enqueuing job for %s: %v", path, err)
		return
	}

	log.Printf("Queued job %s for URL: %s", job.ID, url)
}

func (w *Watcher) isValidFile(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	return ext == ".briefly" || ext == ".url" || ext == ".txt"
}

type inputFile struct {
	URL    string `yaml:"url"`
	Prompt string `yaml:"prompt"`
}

func parseInputFile(path string) (url, customPrompt string, err error) {
	file, err := os.Open(path)
	if err != nil {
		return "", "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return "", "", err
	}

	content := strings.Join(lines, "\n")
	content = strings.TrimSpace(content)

	// Check for YAML front matter
	if strings.HasPrefix(content, "---") {
		parts := strings.SplitN(content, "---", 3)
		if len(parts) >= 3 {
			var input inputFile
			if err := yaml.Unmarshal([]byte(parts[1]), &input); err == nil && input.URL != "" {
				return strings.TrimSpace(input.URL), strings.TrimSpace(input.Prompt), nil
			}
		}
	}

	// Simple URL-only format
	if len(lines) > 0 {
		return strings.TrimSpace(lines[0]), "", nil
	}

	return "", "", nil
}
