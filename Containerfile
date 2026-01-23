# Stage 1: Build Go binary
FROM docker.io/library/golang:1.25-alpine AS builder

WORKDIR /build

# Install build dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o briefly ./cmd/briefly

# Stage 2: Runtime with Python, ffmpeg, yt-dlp, and whisper
FROM docker.io/library/python:3.11-slim

WORKDIR /app

# Install system dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    ffmpeg \
    curl \
    && rm -rf /var/lib/apt/lists/*

# Install yt-dlp
RUN pip install --no-cache-dir yt-dlp

# Install OpenAI Whisper
RUN pip install --no-cache-dir openai-whisper

# Copy the Go binary from builder
COPY --from=builder /build/briefly /app/briefly

# Create data directories with open permissions (will be mounted over)
RUN mkdir -p /data/inbox /data/output

# Pre-download Whisper base model to shared cache
RUN mkdir -p /app/whisper-models \
    && python -c "import whisper; whisper.load_model('base', download_root='/app/whisper-models')" \
    && chmod -R 755 /app/whisper-models

# Set environment defaults
ENV BRIEFLY_WATCH_DIR=/data/inbox
ENV BRIEFLY_OUTPUT_DIR=/data/output
ENV BRIEFLY_LLM_PROVIDER=claude
ENV BRIEFLY_WHISPER_MODEL=base
ENV XDG_CACHE_HOME=/tmp/.cache

# Volume mounts
VOLUME ["/data/inbox", "/data/output"]

# Run the service
# Note: Use --userns=keep-id with podman for rootless operation
CMD ["/app/briefly"]
