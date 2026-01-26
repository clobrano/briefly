package processor

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type YouTubeProcessor struct {
	whisperModel   string
	whisperThreads string
	tempDir        string
}

func NewYouTubeProcessor(whisperModel, whisperThreads string) *YouTubeProcessor {
	tempDir := os.TempDir()
	return &YouTubeProcessor{
		whisperModel:   whisperModel,
		whisperThreads: whisperThreads,
		tempDir:        tempDir,
	}
}

func (y *YouTubeProcessor) Process(ctx context.Context, url string) (string, error) {
	// Create temp directory for this job
	workDir, err := os.MkdirTemp(y.tempDir, "briefly-yt-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(workDir)

	audioPath := filepath.Join(workDir, "audio.mp3")

	// Download audio using yt-dlp
	if err := y.downloadAudio(ctx, url, audioPath); err != nil {
		return "", fmt.Errorf("failed to download audio: %w", err)
	}

	// Transcribe using Whisper
	transcript, err := y.transcribe(ctx, audioPath)
	if err != nil {
		return "", fmt.Errorf("failed to transcribe: %w", err)
	}

	return transcript, nil
}

func (y *YouTubeProcessor) downloadAudio(ctx context.Context, url, outputPath string) error {
	args := []string{
		"-x",                        // Extract audio
		"--audio-format", "mp3",     // Convert to mp3
		"--audio-quality", "0",      // Best quality
		"-o", outputPath,            // Output path
		"--no-playlist",             // Single video only
		"--no-warnings",             // Suppress warnings
		url,
	}

	cmd := exec.CommandContext(ctx, "yt-dlp", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("yt-dlp failed: %w, stderr: %s", err, stderr.String())
	}

	// yt-dlp might add extension, check for the file
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		// Try with .mp3 extension
		if _, err := os.Stat(outputPath + ".mp3"); err == nil {
			os.Rename(outputPath+".mp3", outputPath)
		}
	}

	return nil
}

func (y *YouTubeProcessor) transcribe(ctx context.Context, audioPath string) (string, error) {
	workDir := filepath.Dir(audioPath)
	outputBase := filepath.Join(workDir, "transcript")

	args := []string{
		audioPath,
		"--model", y.whisperModel,
		"--output_format", "txt",
		"--output_dir", workDir,
		"--language", "en", // Default to English, could be made configurable
		"--device", "cpu",  // Explicitly use CPU to avoid GPU memory issues
		"--fp16", "False",  // Disable FP16 on CPU to suppress warning
	}

	// Limit CPU threads if configured (helps reduce memory usage)
	if y.whisperThreads != "" {
		args = append(args, "--threads", y.whisperThreads)
	}

	// Use pre-downloaded models if available (container environment)
	if modelDir := os.Getenv("BRIEFLY_WHISPER_MODEL_DIR"); modelDir != "" {
		args = append(args, "--model_dir", modelDir)
	} else if _, err := os.Stat("/app/whisper-models"); err == nil {
		args = append(args, "--model_dir", "/app/whisper-models")
	}

	cmd := exec.CommandContext(ctx, "whisper", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("whisper failed: %w, stderr: %s", err, stderr.String())
	}

	// Read the transcript file
	transcriptPath := outputBase + ".txt"

	// Whisper names the output after the input file
	audioBase := strings.TrimSuffix(filepath.Base(audioPath), filepath.Ext(audioPath))
	transcriptPath = filepath.Join(workDir, audioBase+".txt")

	transcript, err := os.ReadFile(transcriptPath)
	if err != nil {
		return "", fmt.Errorf("failed to read transcript: %w", err)
	}

	return strings.TrimSpace(string(transcript)), nil
}
