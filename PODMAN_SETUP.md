# Podman Setup Guide for Briefly with Watchtower

This guide shows how to set up Briefly with automatic updates using Podman and podman-compose.

## Prerequisites

1. **Install Podman**:
   ```bash
   # Fedora/RHEL/CentOS
   sudo dnf install podman

   # Ubuntu 20.10+
   sudo apt-get install podman

   # Check installation
   podman --version
   ```

2. **Install podman-compose**:
   ```bash
   # Using pip (recommended)
   pip3 install podman-compose

   # Or via package manager
   sudo dnf install podman-compose  # Fedora/RHEL
   sudo apt-get install podman-compose  # Ubuntu

   # Verify
   podman-compose --version
   ```

## Step 1: Enable Podman Socket

Watchtower needs access to the Podman socket to monitor and update containers.

### For Rootful Podman (running as root)

```bash
# Enable and start the Podman socket
sudo systemctl enable --now podman.socket

# Verify it's running
sudo systemctl status podman.socket

# Check socket exists
ls -la /run/podman/podman.sock
```

### For Rootless Podman (running as regular user - RECOMMENDED)

```bash
# Enable and start the Podman socket for your user
systemctl --user enable --now podman.socket

# Verify it's running
systemctl --user status podman.socket

# Check socket exists (replace 1000 with your UID from: echo $UID)
ls -la /run/user/$(id -u)/podman/podman.sock
```

## Step 2: Update docker-compose.yml for Podman

The `docker-compose.yml` file is already configured for Podman, but verify the socket path:

1. **For rootful Podman**: The file uses `/run/podman/podman.sock` (already set)

2. **For rootless Podman**: Update line 92 in `docker-compose.yml`:
   ```yaml
   # Change this line:
   - /run/podman/podman.sock:/var/run/docker.sock

   # To (replace 1000 with your UID):
   - /run/user/1000/podman/podman.sock:/var/run/docker.sock
   ```

   Find your UID:
   ```bash
   echo $UID
   ```

## Step 3: Create Required Directories

```bash
# Create inbox and output directories
mkdir -p inbox output

# For rootless Podman, ensure proper ownership
chown -R $USER:$USER inbox output
```

## Step 4: Configure Secrets

Choose one of these methods (see README.md "Security Best Practices" for details):

### Option A: Environment Variables (Recommended for Podman)

```bash
export ANTHROPIC_API_KEY="your-api-key"
export BRIEFLY_NTFY_TOPIC="your-topic"
export BRIEFLY_HOST_INBOX="$PWD/inbox"
export BRIEFLY_HOST_OUTPUT="$PWD/output"
```

### Option B: .env File (Quick Testing)

```bash
cp .env.example .env
chmod 600 .env
nano .env  # Add your API keys
```

## Step 5: Build the Briefly Image

```bash
# Build the image using Podman
podman build -t briefly:latest -f Containerfile .
```

## Step 6: Start Services

```bash
# Start Briefly and Watchtower
podman-compose up -d

# Check logs
podman-compose logs -f briefly

# Check both containers are running
podman-compose ps
```

## Verification

### Check Briefly is Running

```bash
# View logs
podman-compose logs briefly

# Should see: "Starting Briefly..." and "Watching directory: /data/inbox"
```

### Check Watchtower is Running

```bash
# View Watchtower logs
podman-compose logs watchtower

# Should see: "Watchtower" and no permission errors
```

### Test File Processing

```bash
# Create a test file
echo "https://www.youtube.com/watch?v=dQw4w9WgXcQ" > inbox/test.url

# Watch for processing
podman-compose logs -f briefly

# Check output directory
ls -la output/
```

## Troubleshooting

### Error: "permission denied" on Docker socket

**Problem**: Watchtower can't access the Podman socket.

**Solution**: Enable the Podman socket (see Step 1):
```bash
# For rootful
sudo systemctl enable --now podman.socket

# For rootless
systemctl --user enable --now podman.socket
```

### Error: "permission denied" in /data/inbox

**Problem**: Container can't write to mounted directories.

**Solution**: Ensure directories exist and have proper permissions:
```bash
mkdir -p inbox output
chmod 755 inbox output

# For rootless Podman
chown -R $USER:$USER inbox output
```

### Watchtower shows "no containers to watch"

**Problem**: Watchtower is running but not monitoring Briefly.

**Solution**: Check the Watchtower label on the Briefly service:
```bash
podman inspect briefly | grep watchtower
# Should show: "com.centurylinklabs.watchtower.enable": "true"
```

### SELinux "Permission Denied" Errors

**Problem**: SELinux blocking container access to volumes.

**Solution**: The `:Z` suffix is already added to volumes in docker-compose.yml, but you can also:
```bash
# Temporarily set SELinux to permissive (not recommended for production)
sudo setenforce 0

# Or add the correct context
chcon -Rt container_file_t inbox output
```

## Running Without Watchtower

If you don't need automatic updates, you can comment out the Watchtower service in `docker-compose.yml`:

```yaml
# Comment out or remove the watchtower service
# watchtower:
#   image: containrrr/watchtower:latest
#   ...
```

Then start only Briefly:
```bash
podman-compose up -d briefly
```

## Updating Manually

Without Watchtower, update containers manually:

```bash
# Pull latest image
podman pull briefly:latest

# Restart containers
podman-compose down
podman-compose up -d
```

## Advanced: Rootless Podman with systemd User Service

For a persistent setup that survives reboots:

```bash
# Generate systemd service files
cd /path/to/briefly
podman-compose up -d  # Start once to create containers

# Generate service files
podman generate systemd --new --name briefly > ~/.config/systemd/user/briefly.service
podman generate systemd --new --name watchtower > ~/.config/systemd/user/watchtower.service

# Stop compose-managed containers
podman-compose down

# Enable services
systemctl --user daemon-reload
systemctl --user enable --now briefly watchtower

# Enable lingering so services run without login
loginctl enable-linger $USER
```

## Next Steps

- Configure custom prompts (see README.md "Input file format")
- Set up ntfy.sh notifications
- Adjust Whisper model for better/faster transcription
- Monitor resource usage with `podman stats`
