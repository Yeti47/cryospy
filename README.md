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

## Distribution Methods

CryoSpy is available in multiple distribution formats to suit different needs:

### Pre-built Releases (Recommended)
- **Linux Server**: Static binaries with installation scripts (handles FFmpeg installation)
- **Linux Client**: AppImage with all dependencies bundled (includes FFmpeg)
- **Windows**: Complete packages with installation scripts (handles FFmpeg installation)

### Building from Source
- Full source code available for customization
- Requires manual dependency installation
- Recommended only for developers or advanced users

**For most users, we recommend using the pre-built releases** which include installation scripts that handle all dependency management automatically.

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

**Runtime Dependencies (All Deployments):**
- **FFmpeg**: Required for all CryoSpy components (video processing)
- **SQLite**: Bundled with Go SQLite driver
- **Webcam devices**: For capture clients

**Build Dependencies (Source Builds Only):**
- **Go 1.24.3** or later
- **OpenCV 4.x** development libraries (capture client only)
- **C/C++ compiler** (for OpenCV CGO bindings)

## Installation

### Option 1: Pre-built Releases (Recommended)

Pre-built releases include installation scripts that automatically handle all dependency management.

**Linux Server:**
```bash
# Download and extract
wget https://github.com/Yeti47/cryospy/releases/latest/download/cryospy-server-linux-amd64.tar.gz
tar -xzf cryospy-server-linux-amd64.tar.gz
cd cryospy-server-linux-amd64

# Install with automatic dependency management
chmod +x install-server-linux.sh
./install-server-linux.sh --with-systemd

# Optional: Set up nginx reverse proxy
chmod +x setup-nginx-proxy.sh
./setup-nginx-proxy.sh --domain yourdomain.com --email admin@yourdomain.com
```

**Linux Client (AppImage):**
```bash
# Download AppImage (all dependencies bundled)
wget https://github.com/Yeti47/cryospy/releases/latest/download/cryospy-capture-client-linux-x86_64.AppImage
chmod +x cryospy-capture-client-linux-x86_64.AppImage

# Configure and run
./cryospy-capture-client-linux-x86_64.AppImage --configure
./cryospy-capture-client-linux-x86_64.AppImage
```

**Windows (Server and Client):**
```powershell
# Download and extract appropriate package
# Run installation scripts (handle dependency management automatically)
.\install-server-windows.ps1  # For server
.\install-client-windows.ps1  # For client

# Optional: Set up nginx reverse proxy
.\setup-nginx-proxy.ps1 -Domain "yourdomain.com" -CertPath "C:\ssl\certs\yourdomain.com"
```

### Option 2: Building from Source

**Only recommended for developers or users requiring customization.**

#### Step 1: Install Development Dependencies

**Linux (Ubuntu/Debian):**
```bash
# Runtime dependencies
sudo apt update
sudo apt install -y ffmpeg

# Build dependencies
sudo apt install -y build-essential pkg-config

# For capture client: OpenCV with contrib modules
sudo apt install -y libopencv-dev libopencv-contrib-dev
```

**Note:** Ubuntu/Debian's OpenCV packages may lack some contrib modules. If you encounter ArUco-related errors, consider using the pre-built AppImage or building OpenCV from source with contrib modules.

**Linux (Fedora/RHEL):**
```bash
# Runtime dependencies
sudo dnf install -y ffmpeg

# Build dependencies
sudo dnf install -y gcc gcc-c++ pkgconf-pkg-config

# For capture client: OpenCV with contrib modules (included by default)
sudo dnf install -y opencv-devel
```

**Windows:**
```powershell
# Install FFmpeg via Chocolatey
choco install ffmpeg

# Install OpenCV via vcpkg
git clone https://github.com/Microsoft/vcpkg.git C:\vcpkg
C:\vcpkg\bootstrap-vcpkg.bat
C:\vcpkg\vcpkg.exe install opencv[contrib]:x64-windows

# Set environment variables
$env:PKG_CONFIG_PATH = "C:\vcpkg\installed\x64-windows\lib\pkgconfig"
$env:CGO_ENABLED = "1"
$env:PATH += ";C:\vcpkg\installed\x64-windows\bin"
```

#### Step 2: Build Components

```bash
# Clone repository
git clone https://github.com/Yeti47/cryospy.git
cd cryospy

# Build server components (no OpenCV required)
cd server/capture-server
go mod tidy
go build -tags release -ldflags "-s -w" -o capture-server .

cd ../dashboard
go mod tidy
go build -tags release -ldflags "-s -w" -o dashboard .

# Build capture client (requires OpenCV)
cd ../../client/capture-client
go mod tidy
go build -ldflags "-s -w" -o capture-client .
```

#### Step 3: Configure and Run

```bash
# Create configuration
mkdir -p ~/cryospy
cp server/config.example.json ~/cryospy/config.json
# Edit ~/cryospy/config.json with your settings

# Start services
./server/capture-server/capture-server &
./server/dashboard/dashboard &

# Access dashboard at http://localhost:8080
# Create your first client through the web interface
```

## Dependency Details

### FFmpeg Requirement

FFmpeg is required on **all machines** running CryoSpy components because:
- CryoSpy spawns `ffmpeg` as external processes for video processing
- Cannot be bundled as a DLL/library - must be installed as executable
- Must be available in system PATH or same directory as CryoSpy binaries

**Installation:**
- **Linux**: `sudo apt install ffmpeg` or `sudo dnf install ffmpeg`
- **Windows**: `choco install ffmpeg` or download from ffmpeg.org
- **Linux AppImage**: Already bundled, no separate installation needed

### OpenCV Requirement

OpenCV is a runtime dependency for the capture client. The GoCV bindings require the OpenCV shared libraries (Linux: `.so` files, Windows: `.dll` files) to be present at runtime.

**Pre-built releases** bundle all required OpenCV libraries, so users do not need to install OpenCV themselves:
  - **Linux AppImage**: All OpenCV `.so` libraries are bundled inside the AppImage
  - **Windows releases**: All required OpenCV `.dll` files are included in the release package

If you build the capture client from source, you need to install the OpenCV development libraries and ensure the runtime libraries are available on your system.

### Production Deployment

For production deployments, especially when exposing the capture-server to the internet:

1. **Use release builds** with `-tags release` for optimized performance
2. **Configure reverse proxy** (nginx/Apache) for HTTPS termination
3. **Enable proxy authentication** for defense-in-depth security
4. **Use provided automation scripts** in the `scripts/` directory

**Available Installation Scripts:**
- `install-server-linux.sh` - Linux server installation with dependency management
- `install-server-windows.ps1` - Windows server installation with dependency management
- `install-client-windows.ps1` - Windows client installation and configuration
- `setup-nginx-proxy.sh` / `setup-nginx-proxy.ps1` - Nginx reverse proxy setup

**Example automated Windows client installation:**
```powershell
.\install-client-windows.ps1 -ServerUrl "https://yourdomain.com" -ClientId "camera-01" -ClientSecret "secure-secret" -ProxyAuthHeader "X-Proxy-Auth" -ProxyAuthValue "proxy-token" -InstallAsService -SetupFirewall -Force
```

**Example client configuration with proxy authentication:**
```json
{
  "client_id": "camera-01",
  "client_secret": "secure-client-secret",
  "server_url": "https://yourdomain.com",
  "proxy_auth_header": "X-Proxy-Auth",
  "proxy_auth_value": "your-secure-proxy-token"
}
```

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

### Dependency Issues

**Pre-built releases:** No dependency installation required! Installation scripts handle everything automatically.

**Building from source:**
```bash
# Linux: Install OpenCV development packages
sudo apt install libopencv-dev pkg-config  # Ubuntu/Debian
sudo dnf install opencv-devel pkgconf-pkg-config  # Fedora

# All platforms: Ensure FFmpeg is installed and in PATH
ffmpeg -version  # Should show version information
```

### Common Problems
- **Camera in use**: Kill other processes using the camera with provided scripts in `scripts/` directory
- **FFmpeg errors**: Ensure FFmpeg is installed and accessible in PATH
- **Database locked**: Check for multiple server instances running
- **Upload failures**: Verify client credentials and server connectivity
- **Missing DLLs on Windows**: Ensure you extracted the complete release package
- **ArUco module errors**: When building from source, ensure OpenCV is compiled with contrib modules. Install `libopencv-contrib-dev` (Ubuntu/Debian) or use `opencv-devel` on Fedora/RHEL (includes contrib modules by default), or build OpenCV from source with `-DOPENCV_EXTRA_MODULES_PATH=/path/to/opencv_contrib/modules`. For easier setup, consider using pre-built releases.
- **Codec issues**: On some Linux distributions, use "libopenh264" instead of "libx264" in configuration

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
