package config

import (
	"os"
	"strings"
)

type Config struct {
	WatchDir       string
	OutputDir      string
	LLMProvider    string
	LLMModel       string
	AnthropicKey   string
	GoogleKey      string
	NtfyTopic      string
	WhisperModel   string
	WhisperThreads string
}

func Load() *Config {
	provider := strings.ToLower(getEnv("BRIEFLY_LLM_PROVIDER", "claude"))
	model := getEnv("BRIEFLY_LLM_MODEL", "")

	// Set default model based on provider if not specified
	if model == "" {
		switch provider {
		case "claude":
			model = "claude-3-7-sonnet-latest"
		case "gemini":
			model = "gemini-2.5-flash"
		}
	}

	return &Config{
		WatchDir:       getEnv("BRIEFLY_WATCH_DIR", "/data/inbox"),
		OutputDir:      getEnv("BRIEFLY_OUTPUT_DIR", "/data/output"),
		LLMProvider:    provider,
		LLMModel:       model,
		AnthropicKey:   getEnv("ANTHROPIC_API_KEY", ""),
		GoogleKey:      getEnv("GOOGLE_API_KEY", ""),
		NtfyTopic:      getEnv("BRIEFLY_NTFY_TOPIC", ""),
		WhisperModel:   getEnv("BRIEFLY_WHISPER_MODEL", "base"),
		WhisperThreads: getEnv("BRIEFLY_WHISPER_THREADS", ""),
	}
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
