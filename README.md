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

## Security Features

- **Master Encryption Key (MEK)**: Password-protected master key for data encryption
- **Per-Client MEK Encryption**: Each capture device has its own secret used to re-encrypt the global MEK
- **Secure Authentication**: Basic authentication with encrypted client secrets
- **Data Encryption**: All video data encrypted at rest using AES-GCM with the global MEK
- **Session Management**: Secure web sessions for dashboard access
- **Proxy Security**: Configurable trusted proxy settings for production deployments
- **HTTPS Ready**: Designed to work with reverse proxies for HTTPS termination

> **Note**: In CryoSpy, a "client" refers to a capture device (end device running the capture-client application), not a user or browser session.

## Quick Start

### Prerequisites

- **Go 1.24.3** or later
- **OpenCV 4.x** development libraries
- **FFmpeg** for video processing
- **SQLite** (bundled with Go SQLite driver)
- **Webcam devices** for capture clients

### Installation

1. **Clone the repository:**
   ```bash
   git clone https://github.com/Yeti47/cryospy.git
   cd cryospy
   ```

2. **Build all components:**
   ```bash
   # Build capture server (debug mode)
   cd server/capture-server
   go mod tidy
   go build -o capture-server .
   
   # Build dashboard (debug mode)
   cd ../dashboard
   go mod tidy
   go build -o dashboard .
   
   # Build capture client
   cd ../../client/capture-client
   go mod tidy
   go build -o capture-client .
   ```

   **For production deployments**, use release builds which optimize Gin for production:
   ```bash
   # Build with release tags for production
   go build -tags release -o capture-server .
   go build -tags release -o dashboard .
   ```

3. **Configure the server:**
   ```bash
   # Copy example config to user's home directory
   mkdir -p ~/cryospy
   cp server/config.example.json ~/cryospy/config.json
   # Edit ~/cryospy/config.json with your settings
   ```

4. **Start the services:**
   ```bash
   # Start capture server (from server/ directory)
   ./capture-server/capture-server
   
   # Start dashboard (from server/ directory)
   ./dashboard/dashboard
   ```

5. **Set up first client:**
   - Open the dashboard at `http://localhost:8080`
   - Complete the initial setup to create your Master Encryption Key
   - Create a new client and note the client ID and secret
   - Configure and start the capture client

### Dependencies Installation

#### Ubuntu/Debian:
```bash
sudo apt update
sudo apt install -y libopencv-dev pkg-config ffmpeg
```

#### Fedora/CentOS/RHEL:
```bash
sudo dnf install opencv-devel pkgconf-pkg-config ffmpeg
```

#### Windows:
1. Install OpenCV and set environment variables
2. Download FFmpeg from https://ffmpeg.org and add to PATH
3. Install Go for Windows

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
  "auth_notification_settings": {
    "recipient": "admin@example.com",
    "min_interval_minutes": 30,
    "failure_threshold": 5,
    "time_window_minutes": 60
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

```json
{
  "trusted_proxies": {
    "capture_server": ["192.168.1.10", "10.0.0.5"],
    "dashboard": []
  }
}
```

**Configuration Guidelines:**
- **Capture Server**: Usually needs proxy configuration when deployed behind nginx/Apache for internet access
- **Dashboard**: Often accessed locally, so empty array `[]` is most secure
- **Development Builds**: Trust all proxies by default (no restrictions)
- **Production Builds**: Only trust explicitly configured proxy IPs

**Common Deployment Scenarios:**

1. **Capture server behind nginx, dashboard local-only:**
   ```json
   "trusted_proxies": {
     "capture_server": ["192.168.1.10"],  // nginx server IP
     "dashboard": []                       // no proxies (most secure)
   }
   ```

2. **Both services behind load balancer:**
   ```json
   "trusted_proxies": {
     "capture_server": ["10.0.0.5"],     // load balancer IP
     "dashboard": ["10.0.0.5"]           // same load balancer
   }
   ```

3. **Direct internet access (no proxies):**
   ```json
   "trusted_proxies": {
     "capture_server": [],               // don't trust any proxy headers
     "dashboard": []                     // don't trust any proxy headers
   }
   ```

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
  "settings_sync_seconds": 300,
  "server_timeout_seconds": 30
}
```

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

### Streaming Configuration
Configure streaming settings in the server configuration:
- **Resolution**: Target video resolution (default: 480p)
- **Bitrate**: Video compression bitrate (default: 1000k)
- **Codec**: Video codec for transcoding (default: libx264)
- **Cache**: Enable/disable segment caching for performance
- **Look-ahead**: Number of clips to pre-process for smooth streaming

## Email Notifications

CryoSpy can send intelligent email notifications for:

- **Motion Detection**: Instant alerts when motion is detected by any camera
- **Storage Warnings**: Notifications when storage usage exceeds thresholds
- **Authentication Failures**: Security alerts when repeated authentication failures are detected

Configure SMTP settings in the server configuration to enable notifications.

## Development

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
```bash
# Build all components with available VS Code tasks
# Or build manually:

# Development builds (debug mode)
cd server/capture-server && go build -o capture-server .
cd server/dashboard && go build -o dashboard .
cd client/capture-client && go build -o capture-client .

# Production builds (release mode)
cd server/capture-server && go build -tags release -o capture-server .
cd server/dashboard && go build -tags release -o dashboard .
# Note: capture-client doesn't use build tags
```

#### Build Modes

**Debug Mode (Development):**
- Default build without tags
- Gin runs in debug mode with verbose logging
- Trusts all proxy headers (permissive for development)
- Includes all debugging middleware

**Release Mode (Production):**
- Built with `-tags release`
- Gin runs in release mode (optimized, minimal logging)
- Only trusts explicitly configured proxy IPs from config
- Minimal middleware for better performance

### Running Tests
```bash
# Run all tests
go test ./...

# Run server core tests only
go test ./server/core/...

# Run capture client tests only
go test ./client/capture-client/...
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

### OpenCV Installation Issues
```bash
# Ubuntu/Debian
sudo apt install libopencv-dev pkg-config

# Fedora
sudo dnf install opencv-devel pkgconf-pkg-config
```

### Common Problems
- **Camera in use**: Kill other processes using the camera with provided scripts
- **FFmpeg errors**: Ensure FFmpeg is installed and in PATH
- **Database locked**: Check for multiple server instances running
- **Upload failures**: Verify client credentials and server connectivity
- **Codec issues**: On some Linux distributions (e.g., Fedora), FFmpeg may not include libx264 by default. Use "libopenh264" instead in your streaming and recording configuration

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
