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
- **Proxy Authentication**: Optional custom header authentication for reverse proxy deployments
- **Proxy Security**: Configurable trusted proxy settings for production deployments
- **HTTPS Ready**: Designed to work with reverse proxies for HTTPS termination

### Defense-in-Depth Authentication

CryoSpy supports a layered authentication approach for enhanced security:

1. **Application Layer**: Standard Basic Authentication using client credentials
2. **Proxy Layer**: Optional custom header authentication for reverse proxies

This dual-layer approach is particularly valuable for internet-exposed deployments where additional security barriers are essential. The proxy authentication layer can be configured independently from the application authentication, allowing for flexible security policies.

**Configuration Example:**
```json
{
  "proxy_auth_header": "X-Proxy-Auth",
  "proxy_auth_value": "secure-proxy-token-12345"
}
```

The proxy authentication headers are automatically included in all API requests when configured, providing seamless integration with nginx, Apache, or other reverse proxy solutions that support custom authentication headers.

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
sudo apt install -y libopencv-dev libopencv-contrib-dev pkg-config ffmpeg
```

**Note:** On Ubuntu/Debian, installing OpenCV via `libopencv-dev` and `libopencv-contrib-dev` may not provide all required modules (such as ArUco) for CryoSpy. If you encounter build errors related to missing OpenCV types (e.g., ArUco), you will need to build OpenCV from source with contrib modules enabled.


**Building OpenCV from source (with contrib modules):**
```bash
# Install build dependencies
sudo apt update
sudo apt install -y \
  build-essential cmake git pkg-config ffmpeg \
  libgtk-3-dev libavcodec-dev libavformat-dev libswscale-dev \
  libtbb12 libtbb-dev \
  libjpeg-dev libpng-dev libtiff-dev libdc1394-22-dev

# Download OpenCV and contrib sources
OPENCV_VERSION=4.9.0
git clone --branch ${OPENCV_VERSION} https://github.com/opencv/opencv.git
git clone --branch ${OPENCV_VERSION} https://github.com/opencv/opencv_contrib.git
mkdir -p opencv/build
cd opencv/build

# Configure and build
cmake -D CMAKE_BUILD_TYPE=Release \
      -D CMAKE_INSTALL_PREFIX=/usr/local \
      -D OPENCV_EXTRA_MODULES_PATH=../../opencv_contrib/modules \
      -D BUILD_EXAMPLES=OFF \
      -D BUILD_TESTS=OFF \
      -D BUILD_PERF_TESTS=OFF \
      -D BUILD_opencv_python3=OFF \
      -D WITH_TBB=ON \
      -D WITH_FFMPEG=ON \
      ..
make -j$(nproc)
sudo make install
sudo ldconfig
```

After building, ensure `/usr/local/lib/pkgconfig` is in your `PKG_CONFIG_PATH`:
```bash
export PKG_CONFIG_PATH=/usr/local/lib/pkgconfig
```

This will make the custom-built OpenCV available for Go builds.

#### Fedora/CentOS/RHEL:
```bash
sudo dnf install opencv-devel pkgconf-pkg-config ffmpeg
```

#### Windows:
1. Install OpenCV and set environment variables
2. Download FFmpeg from https://ffmpeg.org and add to PATH
3. Install Go for Windows

### Production Deployment

For production deployments, especially when exposing the capture-server to the internet, consider:

1. **Use release builds** with `-tags release` for optimized performance
2. **Configure reverse proxy** (nginx/Apache) for HTTPS termination
3. **Enable proxy authentication** for defense-in-depth security
4. **Use automation scripts** available in the `scripts/` directory:
   - `setup-nginx-proxy.ps1` - Windows nginx reverse proxy setup with configurable proxy authentication
   - `install-server-linux.sh` - Linux server installation
   - `install-client-linux.sh` - Linux client installation with configuration setup
   - `install-server-windows.ps1` - Windows server installation with dependency management
   - `install-client-windows.ps1` - Windows client installation with configuration setup

**Cross-Platform Installation Features:**
- **Linux & Windows**: Interactive configuration setup with proxy authentication support
- **Linux & Windows**: Command-line parameters for automated deployment
- **Linux**: Systemd service installation options
- **Windows**: Windows service installation options
- **Windows**: Automated dependency installation via Chocolatey
- **Windows**: Firewall configuration scripts

**Example automated Linux client installation:**
```bash
# Install with all configuration in one command
./install-client-linux.sh --server-url "https://yourdomain.com" --client-id "camera-01" --client-secret "secure-secret" --proxy-auth-header "X-Proxy-Auth" --proxy-auth-value "proxy-token" --with-systemd --force
```

**Example automated Windows client installation:**
```powershell
# Install with all configuration in one command
.\install-client-windows.ps1 -ServerUrl "https://yourdomain.com" -ClientId "camera-01" -ClientSecret "secure-secret" -ProxyAuthHeader "X-Proxy-Auth" -ProxyAuthValue "proxy-token" -InstallAsService -SetupFirewall -Force
```

**Example production client configuration with proxy auth:**
```json
{
  "client_id": "camera-01",
  "client_secret": "secure-client-secret",
  "server_url": "https://yourdomain.com",
  "proxy_auth_header": "X-Proxy-Auth",
  "proxy_auth_value": "your-secure-proxy-token"
}
```

The proxy authentication provides an additional security layer when your capture-server is exposed to the internet through a reverse proxy. This way, unauthenticated requests can be rejected before even entering the app domain.

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

> **Related Feature**: For additional security when using reverse proxies, consider configuring **proxy authentication** in your capture clients. This adds a second layer of authentication at the proxy level, complementing the trusted proxy IP configuration. See the [Client Configuration](#client-configuration) section for details on proxy authentication headers.

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
  "server_timeout_seconds": 30,
  "proxy_auth_header": "X-Proxy-Auth",
  "proxy_auth_value": "your-proxy-secret"
}
```

#### Optional Proxy Authentication
The `proxy_auth_header` and `proxy_auth_value` fields enable additional authentication when your capture-server is deployed behind a reverse proxy (such as nginx) that requires custom authentication headers. This provides a defense-in-depth security model:

- **Application-level authentication**: Standard Basic Auth using `client_id` and `client_secret`
- **Proxy-level authentication**: Custom header authentication for the reverse proxy layer

**Common use cases:**
- nginx reverse proxy with custom authentication headers
- Load balancers requiring specific authentication tokens
- Additional security layer for internet-exposed deployments

**Example nginx configuration with proxy auth:**
```nginx
# In your nginx server block
if ($http_x_proxy_auth != "your-proxy-secret") {
    return 401 "Unauthorized - Invalid proxy authentication";
}
```

Leave these fields empty (`""`) if you're not using proxy authentication.

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

Example configuration:
```json
{
  "auth_event_settings": {
    "time_window_minutes": 60,
    "auto_disable_threshold": 10,
    "notification_recipient": "admin@example.com",
    "notification_threshold": 5,
    "min_interval_minutes": 30
  }
}
```

Set `auto_disable_threshold` to `0` to disable this feature.

#### Authentication Event Configuration
All authentication-related settings are unified under `auth_event_settings`:

- `time_window_minutes`: Time window for counting authentication failures
- `auto_disable_threshold`: Number of failures that trigger automatic client disabling (0 to disable)
- `notification_recipient`: Email address for notifications (empty string to disable)
- `notification_threshold`: Number of failures that trigger notifications (0 to disable)
- `min_interval_minutes`: Minimum time between notifications for rate limiting

**Configuration Examples:**

**Notifications only** (no auto-disable):
```json
{
  "auth_event_settings": {
    "time_window_minutes": 60,
    "auto_disable_threshold": 0,
    "notification_recipient": "admin@example.com",
    "notification_threshold": 5,
    "min_interval_minutes": 30
  }
}
```

**Auto-disable only** (no notifications):
```json
{
  "auth_event_settings": {
    "time_window_minutes": 60,
    "auto_disable_threshold": 10,
    "notification_recipient": "",
    "notification_threshold": 0,
    "min_interval_minutes": 0
  }
}
```

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
