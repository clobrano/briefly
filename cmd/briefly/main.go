package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/clobrano/briefly/internal/config"
	"github.com/clobrano/briefly/internal/notifier"
	"github.com/clobrano/briefly/internal/processor"
	"github.com/clobrano/briefly/internal/queue"
	"github.com/clobrano/briefly/internal/summarizer"
	"github.com/clobrano/briefly/internal/watcher"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("Starting Briefly...")

	// Load configuration
	cfg := config.Load()

	// Validate configuration
	if err := validateConfig(cfg); err != nil {
		log.Fatalf("Configuration error: %v", err)
	}

	// Ensure directories exist and are writable
	if err := os.MkdirAll(cfg.WatchDir, 0755); err != nil {
		log.Fatalf("Failed to create watch directory: %v", err)
	}
	if err := os.MkdirAll(cfg.OutputDir, 0755); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	// Verify write permissions
	if err := checkWritePermission(cfg.WatchDir); err != nil {
		log.Fatalf("Watch directory not readable: %v", err)
	}
	if err := checkWritePermission(cfg.OutputDir); err != nil {
		log.Fatalf("Output directory not writable: %v", err)
	}

	// Initialize queue with persistence
	queuePath := filepath.Join(cfg.OutputDir, ".queue.json")
	q, err := queue.New(queuePath)
	if err != nil {
		log.Fatalf("Failed to initialize queue: %v", err)
	}
	log.Printf("Queue initialized (persistence: %s)", queuePath)

	// Initialize summarizer
	sum, err := initSummarizer(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize summarizer: %v", err)
	}
	log.Printf("Summarizer initialized (provider: %s, model: %s)", cfg.LLMProvider, cfg.LLMModel)

	// Initialize notifier
	ntfy := notifier.New(cfg.NtfyTopic)
	if ntfy != nil {
		log.Printf("Notifier initialized (topic: %s)", cfg.NtfyTopic)
	}

	// Initialize processor
	proc := processor.New(cfg, q, sum, ntfy)
	proc.Start()
	log.Println("Processor started")

	// Initialize watcher
	watch, err := watcher.New(cfg.WatchDir, q)
	if err != nil {
		log.Fatalf("Failed to initialize watcher: %v", err)
	}
	if err := watch.Start(); err != nil {
		log.Fatalf("Failed to start watcher: %v", err)
	}
	log.Printf("Watching directory: %s", cfg.WatchDir)

	log.Println("Briefly is running. Press Ctrl+C to stop.")

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down...")

	// Graceful shutdown
	watch.Stop()
	proc.Stop()

	log.Println("Briefly stopped.")
}

func validateConfig(cfg *config.Config) error {
	if cfg.LLMProvider == "claude" && cfg.AnthropicKey == "" {
		log.Println("Warning: ANTHROPIC_API_KEY not set, Claude summarization will fail")
	}
	if cfg.LLMProvider == "gemini" && cfg.GoogleKey == "" {
		log.Println("Warning: GOOGLE_API_KEY not set, Gemini summarization will fail")
	}
	return nil
}

func initSummarizer(cfg *config.Config) (summarizer.Summarizer, error) {
	switch cfg.LLMProvider {
	case "claude":
		return summarizer.NewClaudeSummarizer(cfg.AnthropicKey, cfg.LLMModel)
	case "gemini":
		ctx := context.Background()
		return summarizer.NewGeminiSummarizer(ctx, cfg.GoogleKey, cfg.LLMModel)
	default:
		return summarizer.NewClaudeSummarizer(cfg.AnthropicKey, cfg.LLMModel)
	}
}

func checkWritePermission(dir string) error {
	testFile := filepath.Join(dir, ".write_test")
	f, err := os.Create(testFile)
	if err != nil {
		return err
	}
	f.Close()
	os.Remove(testFile)
	return nil
}
