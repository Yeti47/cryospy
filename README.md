# CryoSpy

<div align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="resources/logo-dark.svg">
    <source media="(prefers-color-scheme: light)" srcset="resources/logo.svg">
    <img src="resources/logo.svg" alt="CryoSpy Logo" width="200" height="200">
  </picture>
  
  **Self-Hosted Surveillance System with Privacy-First Design**
  
  A comprehensive surveillance and monitoring solution designed for secure self-hosting, keeping your sensitive data within your own managed boundaries.
</div>

## Overview

CryoSpy is a privacy-focused surveillance system developed for secure self-hosting environments. Born from the need to avoid dependence on external services and third-party hardware, CryoSpy ensures that all sensitive and private data remains within your self-managed infrastructure. The system is designed for home security applications while prioritizing user privacy and data sovereignty.

**⚠️ Important:** This software is intended for lawful home security purposes only. The developers do not condone or support the use of this software for non-consensual monitoring, invasion of privacy, or any other illegal activities.

## Key Features

- **🔒 End-to-End Encryption**: All video data is encrypted at rest using AES-GCM encryption
- **🏠 Complete Self-Hosting**: No dependency on external services or cloud providers
- **📹 Continuous Recording**: Real-time video capture with configurable chunk duration
- **🎯 Motion Detection**: Intelligent motion detection with configurable sensitivity
- **📱 Web Dashboard**: Modern web interface for monitoring and management
- **🎬 Live Streaming**: HLS-based live video streaming with low latency
- **📧 Smart Notifications**: Email alerts for motion detection, storage warnings, and security events
- **🔄 Automatic Settings Sync**: Centralized configuration management
- **📊 Storage Management**: Automatic cleanup with configurable storage limits
- **🛡️ Client Authentication**: Secure client-server communication with encryption keys
- **📋 Daily Logs**: Comprehensive logging with daily rotation

## Architecture

CryoSpy consists of three main components:

### 🎥 Capture Client
A lightweight Go application that runs on devices with camera access:
- Continuous video recording using OpenCV
- Real-time motion detection with background subtraction
- Video post-processing for storage optimization (compression, downscaling, format conversion)
- Secure upload to capture server
- Automatic settings synchronization from server

### 🌐 Capture Server
The core backend service that handles video ingestion:
- RESTful API for video clip uploads
- Client authentication and authorization
- Video metadata extraction and thumbnail generation
- Encrypted storage management with automatic cleanup
- Email notifications for motion detection and storage alerts
- Authentication failure monitoring with configurable thresholds

### 🖥️ Dashboard
A web-based administration interface designed for local access on the host system:
- Client management and configuration
- Video playback and clip browsing
- Live streaming with HLS support
- System monitoring and statistics
- Settings management for all clients
- Intended for local network access; public internet exposure strongly discouraged for security

## Getting Started

### Prerequisites

**Hardware:**
- A server or PC to host the backend (Linux recommended)
- One or more devices with cameras (webcams, USB cameras) to act as clients

**Software:**
- **Docker** (Recommended): The easiest way to run both server and client components.
- **FFmpeg**: Required on the host machine if running binaries directly (not needed for Docker).

### Quick Setup Guide (Docker)

This guide covers setting up a production-ready CryoSpy server using Docker, Nginx, and Let's Encrypt SSL, followed by a Docker-based client setup.

#### 1. Server Setup (Docker + Nginx + SSL)

**Start CryoSpy Server**

Create a `docker-compose.yml` file:

```yaml
services:
  cryospy-server:
    image: ghcr.io/yeti47/cryospy-server:latest
    container_name: cryospy-server
    restart: unless-stopped
    ports:
      - "8080:8080"           # Dashboard (Direct access, e.g., http://local-ip:8080)
      - "127.0.0.1:8081:8081" # Capture API (Proxied via Nginx only)
    volumes:
      - ./data:/home/cryospy
```

Start the server:
```bash
mkdir -p cryospy/data
cd cryospy
# Save the above yaml to docker-compose.yml
docker compose up -d
```

**Configure Nginx**

Install Nginx and Certbot:
```bash
sudo apt update
sudo apt install nginx certbot python3-certbot-nginx
```

Create a new configuration file `/etc/nginx/conf.d/cryospy.conf` (replace `yourdomain.com` with your actual domain):

```nginx
# Rate limiting zones
limit_req_zone $binary_remote_addr zone=api:10m rate=10r/s;
limit_req_zone $binary_remote_addr zone=upload:10m rate=2r/s;

# Upstream for capture-server
upstream capture_server {
    server 127.0.0.1:8081;
}

server {
    listen 80;
    server_name yourdomain.com;
    
    # Redirect all HTTP traffic to HTTPS
    return 301 https://$host$request_uri;
}

server {
    listen 443 ssl http2;
    server_name yourdomain.com;

    # Security headers
    add_header X-Frame-Options DENY;
    add_header X-Content-Type-Options nosniff;
    add_header X-XSS-Protection "1; mode=block";

    # SSL configuration will be managed by Certbot
    # ssl_certificate /etc/letsencrypt/live/yourdomain.com/fullchain.pem;
    # ssl_certificate_key /etc/letsencrypt/live/yourdomain.com/privkey.pem;

    # Dashboard (Web UI) - Blocked in Nginx
    # Access directly via http://your-server-ip:8080 on LAN
    location / {
        default_type text/plain;
        return 403 "Access denied.";
    }

    # Capture Server API (for Clients)
    location /capture-server/ {
        # Apply rate limiting for uploads
        limit_req zone=upload burst=10 nodelay;

        # Optional: Proxy Authentication (Defense-in-Depth)
        # Uncomment to require a secret header from clients (recommended)
        # Replace "your-secure-proxy-token" with a strong secret
        # if ($http_x_proxy_auth != "your-secure-proxy-token") {
        #     return 401 "Unauthorized";
        # }

        # Increase timeout and body size for video uploads
        client_max_body_size 100M;
        proxy_read_timeout 300s;
        proxy_connect_timeout 10s;
        proxy_send_timeout 300s;

        proxy_pass http://capture_server/;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        
        # Security headers for API
        add_header X-Robots-Tag "noindex, nofollow";
    }

    # Health check endpoint
    location = /health {
        limit_req zone=api burst=5 nodelay;
        
        # Optional: Proxy Authentication
        # if ($http_x_proxy_auth != "your-secure-proxy-token") {
        #     return 401 "Unauthorized";
        # }
        
        proxy_pass http://capture_server;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

> **Security Tip:** Uncomment the proxy authentication block in the Nginx config above to add an extra layer of security. If enabled, you must also configure `proxy_auth_header` ("X-Proxy-Auth") and `proxy_auth_value` ("your-secure-proxy-token") in your client's `config.json`.

> **Note:** If certbot fails with SSL configuration errors, temporarily comment out the entire HTTPS (port 443) server block, run certbot, then uncomment it and reload nginx.

**Enable SSL**

Run Certbot to obtain a certificate and automatically configure SSL:
```bash
sudo certbot --nginx -d yourdomain.com
```

Certbot automatically sets up a scheduled task for certificate renewal. You can test automatic renewal with:
```bash
sudo certbot renew --dry-run
```

The Capture API is now accessible at `https://yourdomain.com/capture-server/api/`. Note that the root URL `https://yourdomain.com` will return "Access denied" for security. The dashboard is accessible locally at `http://<server-ip>:8080`. Clients should be configured with `server_url: "https://yourdomain.com/capture-server"`.

#### 2. Client Setup (Docker)

Run the capture client on your camera-equipped device:

```bash
# 1. Create configuration directory
mkdir -p client-config

# 2. Create config.json inside client-config/
# (Copy the client configuration from the Dashboard "Clients" page)

# 3. Run the container
docker run -d \
  --name cryospy-client \
  --restart unless-stopped \
  --device /dev/video0:/dev/video0 \
  -v $(pwd)/client-config:/config \
  ghcr.io/yeti47/cryospy-client:latest
```

### Alternative: Linux Server Binaries

If you prefer not to use Docker for the server, we provide pre-built binaries for Linux.

```bash
# Download and extract
wget https://github.com/Yeti47/cryospy/releases/latest/download/cryospy-server-linux-amd64.tar.gz
tar -xzf cryospy-server-linux-amd64.tar.gz
cd cryospy-server-linux-amd64

# Install with automatic dependency management
chmod +x install-server-linux.sh
./install-server-linux.sh --with-systemd
```

> **Note:** The client is **only** distributed as a Docker image. If you need a standalone binary for the client (e.g., for Windows), you must build it from source.

## Configuration

### Server Configuration
The server uses a JSON configuration file. By default, it's located at `~/cryospy/config.json` in the user's home directory:

```json
{
  "web_addr": "127.0.0.1",
  "web_port": 8080,
  "capture_port": 8081,
  "database_path": "/path/to/cryospy.db",
  "log_path": "/path/to/logs",
  "log_level": "info",
  "trusted_proxies": {
    "capture_server": ["192.168.1.10", "nginx-ip"],
    "dashboard": []
  },
  "storage_notification_settings": {
    "recipient": "admin@example.com",
    "min_interval_minutes": 60,
    "warning_threshold": 0.8
  },
  "motion_notification_settings": {
    "recipient": "admin@example.com",
    "min_interval_minutes": 15
  },
  "auth_event_settings": {
    "time_window_minutes": 60,
    "auto_disable_threshold": 10,
    "notification_recipient": "admin@example.com",
    "notification_threshold": 5,
    "min_interval_minutes": 30
  },
  "smtp_settings": {
    "host": "smtp.gmail.com",
    "port": 587,
    "username": "your-email@gmail.com",
    "password": "your-app-password",
    "from_addr": "your-email@gmail.com"
  },
  "streaming_settings": {
    "cache": {
      "enabled": true,
      "max_size_bytes": 1073741824
    },
    "look_ahead": 10,
    "width": 854,
    "height": 480,
    "video_bitrate": "1000k",
    "video_codec": "libx264",
    "frame_rate": 25
  }
}
```

#### Trusted Proxies Configuration

The `trusted_proxies` configuration is important for production deployments behind reverse proxies or load balancers. This setting controls which proxy IP addresses are trusted to provide real client IP information through headers like `X-Forwarded-For`.

**Configuration Guidelines:**
- **Capture Server**: Usually needs proxy configuration when deployed behind nginx/Apache for internet access
- **Dashboard**: Often accessed locally, so empty array `[]` is most secure
- **Development Builds**: Trust all proxies by default (no restrictions)
- **Production Builds**: Only trust explicitly configured proxy IPs

> **Security Note**: Incorrectly configured trusted proxies can allow IP spoofing. Only add proxy IPs that you control and trust.

### Client Configuration
Each capture client uses a local JSON configuration file:

```json
{
  "client_id": "your-client-id",
  "client_secret": "your-client-secret",
  "server_url": "http://localhost:8081",
  "camera_device": "/dev/video0",
  "buffer_size": 5,
  "retry_buffer_size": 100,
  "settings_sync_seconds": 300,
  "server_timeout_seconds": 30,
  "upload_retry_minutes": 5,
  "upload_max_retries": 3,
  "proxy_auth_header": "X-Proxy-Auth",
  "proxy_auth_value": "your-proxy-secret"
}
```

#### Upload Retry Configuration

CryoSpy includes intelligent retry logic for handling temporary server outages:

- **`upload_retry_minutes`** (default: 5): Minutes to wait before retrying failed uploads
- **`upload_max_retries`** (default: 3): Maximum number of retry attempts before giving up
  - Set to `0` to disable retries entirely (uploads are dropped immediately on failure)
  - Set to `null` or omit to use the default value of 3
- **`buffer_size`** (default: 5): Number of clips to buffer for immediate upload
- **`retry_buffer_size`** (default: 100): Number of failed clips to buffer for retry during server outages

#### Optional Proxy Authentication
The `proxy_auth_header` and `proxy_auth_value` fields enable additional authentication when your capture-server is deployed behind a reverse proxy (such as nginx) that requires custom authentication headers. This provides a defense-in-depth security model.

## Video Streaming

CryoSpy includes a powerful live streaming feature that allows real-time viewing of camera feeds through the web dashboard:

### Features
- **HLS (HTTP Live Streaming)** for low-latency playback
- **Live and Historical Viewing** - stream new clips as they arrive or browse past recordings
- **Adaptive Quality** - configurable resolution, bitrate, and codec settings
- **Caching** - normalized video segments are cached for improved performance
- **Security** - all video data remains encrypted and is only decrypted during streaming

### Usage
1. Navigate to the "Stream" section in the dashboard
2. Select a client from the dropdown
3. Optionally set a reference time for historical playback
4. Click "Start Streaming" to begin viewing

## Email Notifications

CryoSpy can send intelligent email notifications for:

- **Motion Detection**: Instant alerts when motion is detected by any camera
- **Storage Warnings**: Notifications when storage usage exceeds thresholds
- **Authentication Failures**: Security alerts when repeated authentication failures are detected

Configure SMTP settings in the server configuration to enable notifications.

## Client Management

### Client Security Features

CryoSpy includes several security features for managing camera clients:

#### Client Disabling
- Clients can be disabled without deleting them entirely
- Disabled clients cannot authenticate and are treated as non-existent
- Useful for temporarily suspending problematic clients
- Can be managed through the dashboard interface

#### Automatic Client Disabling
When `auth_event_settings.auto_disable_threshold` is configured, clients will be automatically disabled after exceeding the specified number of authentication failures within the time window. This helps protect against brute force attacks and misconfigured clients.

## Development & Building from Source

### Tech Stack
- **Language**: Go 1.24+
- **Web Framework**: Gin
- **Database**: SQLite (with WAL mode)
- **Video Processing**: FFmpeg, OpenCV (GoCV)
- **Containerization**: Docker

### Project Structure
```
cryospy/
├── client/
│   └── capture-client/          # Camera capture application
├── server/
│   ├── capture-server/          # Video ingestion API
│   ├── dashboard/               # Web administration interface
│   └── core/                    # Shared libraries and utilities
├── docs/                        # Documentation
├── resources/                   # Assets (logo, etc.)
└── scripts/                     # Utility scripts
```

### Building from Source

**Only recommended for developers or users requiring customization (e.g., Windows clients).**

#### Prerequisites
- **Go 1.24.3** or later
- **FFmpeg**: Must be installed and in PATH.
- **OpenCV 4.x**: Required for the capture client.
  - **Linux**: Install `libopencv-dev` (Debian/Ubuntu) or `opencv-devel` (Fedora).
  - **Windows**: Install via `vcpkg` (`vcpkg install opencv[contrib]:x64-windows`). Ensure DLLs are in PATH or alongside the binary.

#### Build Commands
```bash
# Build server components (no OpenCV required)
cd server/capture-server && go build -tags release -o capture-server .
cd server/dashboard && go build -tags release -o dashboard .

# Build capture client (requires OpenCV)
cd client/capture-client && go build -o capture-client .
```

### Running Tests
```bash
# Run all tests
go test ./...

# Run End-to-End (E2E) tests
# Note: Requires Docker and Docker Compose
go test -v ./tests/e2e/...
```

## Troubleshooting

### Camera Access Issues
```bash
# Linux - check camera permissions
sudo chmod 666 /dev/video0
sudo usermod -a -G video $USER  # logout/login required

# Test camera access
ffmpeg -f v4l2 -i /dev/video0 -t 5 test.mp4
```

### Common Problems
- **Camera in use**: Kill other processes using the camera with provided scripts in `scripts/` directory
- **FFmpeg errors**: Ensure FFmpeg is installed and accessible in PATH
- **Database locked**: Check for multiple server instances running
- **Upload failures**: Verify client credentials and server connectivity
- **Missing DLLs on Windows**: Ensure you extracted the complete release package
- **ArUco module errors**: When building from source, ensure OpenCV is compiled with contrib modules.

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Disclaimer

This software is provided for lawful surveillance and security purposes only. Users are responsible for ensuring compliance with all applicable laws and regulations regarding video recording, privacy, and surveillance in their jurisdiction. The developers expressly disclaim any responsibility for misuse of this software for illegal or unethical purposes, including but not limited to non-consensual monitoring or invasion of privacy.

---

<div align="center">
  <sub>Built with 💙 for privacy-conscious home security</sub>
</div>
