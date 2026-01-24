# Briefly

A Go-based background service that watches a directory for URL input files, processes content (YouTube videos via transcription or web articles via text extraction), summarizes using Claude or Gemini, and notifies via ntfy.sh.

## Features

- **Directory watching**: Monitors a folder for new URL files with debouncing
- **YouTube support**: Downloads audio with yt-dlp, transcribes with Whisper
- **Web article support**: Extracts readable content using go-readability
- **LLM summarization**: Supports Claude (Anthropic) and Gemini (Google)
- **Push notifications**: Sends completion alerts via ntfy.sh
- **Queue persistence**: Survives restarts with JSON-based job queue
- **Retry logic**: Exponential backoff for failed jobs
- **Custom prompts**: Override default summarization instructions per-file

## Requirements

### For local development

- Go 1.21+
- yt-dlp (for YouTube processing)
- ffmpeg (for audio processing)
- openai-whisper (Python package for transcription)

### For container deployment

- Podman or Docker

## Installation

### From source

```bash
git clone https://github.com/clobrano/briefly.git
cd briefly
go build -o briefly ./cmd/briefly
```

### Container build

```bash
podman build -t briefly:latest -f Containerfile .
```

## Docker Compose Configuration

The project includes a `docker-compose.yml` file that sets up Briefly with Watchtower for automatic image updates.

### Features

- **Automatic updates**: Watchtower checks for new images every 24 hours (configurable)
- **Zero-downtime updates**: Containers are restarted automatically when updates are available
- **Cleanup**: Old images are removed after successful updates
- **Easy configuration**: All settings in `.env` file

### Quick Start

1. Copy the example environment file:
   ```bash
   cp .env.example .env
   ```

2. Edit `.env` and set your API key:
   ```bash
   ANTHROPIC_API_KEY=your-actual-api-key-here
   ```

3. Start the services:
   ```bash
   docker compose up -d
   ```

4. Check the logs:
   ```bash
   docker compose logs -f briefly
   ```

### Watchtower Configuration

Watchtower settings in `.env`:

| Variable | Default | Description |
|----------|---------|-------------|
| `WATCHTOWER_POLL_INTERVAL` | `86400` | Update check interval in seconds (86400 = 24 hours) |
| `WATCHTOWER_DEBUG` | `false` | Enable verbose logging |

**Notification options:**
- Email: Set `WATCHTOWER_NOTIFICATION_URL` to SMTP URL
- ntfy.sh: Set `WATCHTOWER_NOTIFICATION_URL` to `ntfy://ntfy.sh/your-topic`

See `.env.example` for detailed notification configuration examples.

### Using with Podman

To use with Podman instead of Docker:

```bash
# Update the docker-compose.yml Watchtower volume to use Podman socket
# Replace:
#   - /var/run/docker.sock:/var/run/docker.sock
# With:
#   - /run/user/1000/podman/podman.sock:/var/run/docker.sock

podman-compose up -d
```

## Configuration

Briefly uses environment variables for configuration:

| Variable | Default | Description |
|----------|---------|-------------|
| `BRIEFLY_WATCH_DIR` | `/data/inbox` | Directory to watch for input files |
| `BRIEFLY_OUTPUT_DIR` | `/data/output` | Where summaries are saved |
| `BRIEFLY_LLM_PROVIDER` | `claude` | LLM provider: `claude` or `gemini` |
| `BRIEFLY_LLM_MODEL` | (auto) | LLM model name (see below for defaults) |
| `ANTHROPIC_API_KEY` | - | API key for Claude (required if using claude) |
| `GOOGLE_API_KEY` | - | API key for Gemini (required if using gemini) |
| `BRIEFLY_NTFY_TOPIC` | - | ntfy.sh topic for notifications (optional) |
| `BRIEFLY_WHISPER_MODEL` | `base` | Whisper model: `tiny`, `base`, `small`, `medium`, `large` |

### LLM Model Defaults

If `BRIEFLY_LLM_MODEL` is not set, it defaults based on provider:

| Provider | Default Model |
|----------|---------------|
| `claude` | `claude-3-7-sonnet-latest` |
| `gemini` | `gemini-2.5-flash` |

**Example models:**
- Claude: `claude-3-7-sonnet-latest`, `claude-sonnet-4-5`, `claude-opus-4-5-20251101`
- Gemini: `gemini-2.5-flash`, `gemini-2.5-pro`, `gemini-2.0-flash`

## Usage

### Running locally

```bash
# Set required environment variables
export ANTHROPIC_API_KEY=sk-ant-...
export BRIEFLY_WATCH_DIR=$HOME/briefly/inbox
export BRIEFLY_OUTPUT_DIR=$HOME/briefly/output
export BRIEFLY_NTFY_TOPIC=my-briefly-notifications  # optional

# Create directories
mkdir -p $BRIEFLY_WATCH_DIR $BRIEFLY_OUTPUT_DIR

# Run the service
./briefly
```

### Running with container

**Recommended (Docker Compose with automatic updates):**

```bash
# Copy and configure environment file
cp .env.example .env
# Edit .env with your API keys and settings
nano .env

# Start services (Briefly + Watchtower for auto-updates)
docker compose up -d

# View logs
docker compose logs -f briefly

# Stop services
docker compose down
```

Docker Compose automatically:
- Pulls and runs the Briefly service
- Configures Watchtower to check for image updates every 24 hours
- Restarts containers when updates are available
- Cleans up old images

**Alternative (Manual Podman with user namespace mapping):**

```bash
podman run -d \
  --name briefly \
  --userns=keep-id \
  -e ANTHROPIC_API_KEY=sk-ant-... \
  -e BRIEFLY_NTFY_TOPIC=my-briefly-notifications \
  -v /path/to/inbox:/data/inbox:Z \
  -v /path/to/output:/data/output:Z \
  briefly:latest
```

The `--userns=keep-id` flag maps your host UID into the container, allowing the container to read/write files you own. The `:Z` suffix (capital Z) applies the correct SELinux context for private volumes.

**Alternative (if files are world-readable):**

```bash
podman run -d \
  --name briefly \
  -e ANTHROPIC_API_KEY=sk-ant-... \
  -v /path/to/inbox:/data/inbox:z \
  -v /path/to/output:/data/output:z \
  briefly:latest
```

### Input file format

Create files with `.briefly`, `.url`, or `.txt` extension in the watch directory.

**Simple format (URL only):**

```
https://www.youtube.com/watch?v=dQw4w9WgXcQ
```

**Extended format (YAML front matter):**

```yaml
---
url: https://www.youtube.com/watch?v=dQw4w9WgXcQ
prompt: |
  Summarize this video focusing on:
  - Main arguments presented
  - Key statistics mentioned
  - Action items for viewers
---
```

### Supported content types

| Type | Detection | Processing |
|------|-----------|------------|
| YouTube | URLs containing `youtube.com` or `youtu.be` | yt-dlp audio download + Whisper transcription |
| Web articles | Any other HTTP/HTTPS URL | go-readability text extraction |

### Output

Summaries are saved as Markdown files in the output directory:

```
output/
├── 20240115-143022.123.md
├── 20240115-144530.456.md
└── .queue.json  # Internal queue state
```

Each summary file contains:

```markdown
# Summary

**URL:** https://example.com/article
**Type:** text
**Generated:** 2024-01-15T14:30:22Z

---

[Summary content here]
```

### Notifications

If `BRIEFLY_NTFY_TOPIC` is set, you'll receive push notifications when summaries complete. Subscribe to your topic at `https://ntfy.sh/your-topic` or use the ntfy mobile app.

## Architecture

```
┌─────────────┐     ┌─────────┐     ┌───────────┐     ┌────────────┐
│   Watcher   │────▶│  Queue  │────▶│ Processor │────▶│ Summarizer │
│  (fsnotify) │     │  (JSON) │     │           │     │   (LLM)    │
└─────────────┘     └─────────┘     └───────────┘     └────────────┘
                                          │
                                          ▼
                                    ┌──────────┐
                                    │ Notifier │
                                    │  (ntfy)  │
                                    └──────────┘
```

### Components

- **Watcher**: Monitors the input directory using fsnotify with 500ms debouncing
- **Queue**: Thread-safe job queue with JSON persistence for crash recovery
- **Processor**: Orchestrates content extraction and summarization with retry logic
- **Summarizer**: Interface supporting Claude and Gemini backends
- **Notifier**: Sends completion notifications to ntfy.sh

## Whisper model selection

| Model | Size | Speed | Accuracy | Memory |
|-------|------|-------|----------|--------|
| `tiny` | 39M | Fastest | Lower | ~1GB |
| `base` | 74M | Fast | Good | ~1GB |
| `small` | 244M | Medium | Better | ~2GB |
| `medium` | 769M | Slow | High | ~5GB |
| `large` | 1550M | Slowest | Highest | ~10GB |

Default is `base` for a balance of speed and accuracy.

## Troubleshooting

### YouTube download fails

Ensure yt-dlp is installed and up to date:

```bash
pip install -U yt-dlp
```

### Whisper transcription fails

Ensure Whisper and ffmpeg are installed:

```bash
pip install openai-whisper
# On Fedora/RHEL
sudo dnf install ffmpeg
# On Ubuntu/Debian
sudo apt install ffmpeg
```

### API errors

- Verify your API key is set correctly
- Check you have sufficient API credits
- Ensure the API key has the required permissions

### Queue stuck

If the queue appears stuck, check `.queue.json` in the output directory. You can manually edit or delete it to reset the queue state.

### Permission denied errors in container

If you see errors like `permission denied` when reading input files:

1. **Use `--userns=keep-id`** (recommended for rootless Podman):
   ```bash
   podman run --userns=keep-id -v /path/to/inbox:/data/inbox:Z ...
   ```

2. **Check SELinux context** - use `:Z` (capital) for private volumes:
   ```bash
   -v /path/to/inbox:/data/inbox:Z
   ```

3. **Ensure directory permissions** allow the container user to read:
   ```bash
   # Make files readable by others
   chmod -R o+rX /path/to/inbox
   chmod -R o+rwX /path/to/output
   ```

4. **Check file ownership** matches the container user:
   ```bash
   # When using --userns=keep-id, files should be owned by your user
   ls -la /path/to/inbox/
   ```

5. **Verify the volume is mounted correctly**:
   ```bash
   podman exec briefly ls -la /data/inbox/
   ```

## License

MIT
