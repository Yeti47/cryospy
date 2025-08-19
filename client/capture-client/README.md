# Capture Client

The CryoSpy capture client is a Go application that continuously captures video from a webcam, processes it according to server-provided settings, and uploads clips to the CryoSpy capture server.

## Features

- **Continuous video capture** from webcam devices using OpenCV
- **Motion detection** with configurable sensitivity (optional, server-controlled)
- **Video post-processing** with FFmpeg (compression, downscaling, grayscale conversion)
- **Chunked recording** with server-configurable duration
- **Concurrent upload** while recording continues
- **Buffer management** to prevent memory issues
- **Automatic settings synchronization** from server
- **Daily rotating log files**

## Prerequisites

- **Go 1.24.3** or later
- **OpenCV 4.x** development libraries
- **FFmpeg** for video processing
- **Webcam device**

### OpenCV Installation

#### Ubuntu/Debian:
```bash
sudo apt update
sudo apt install -y libopencv-dev pkg-config
```

#### Fedora/CentOS/RHEL:
```bash
sudo dnf install opencv-devel pkgconf-pkg-config
```

#### Windows:
Install OpenCV and set environment variables:
```cmd
set CGO_CPPFLAGS=-IC:\opencv\build\include
set CGO_LDFLAGS=-LC:\opencv\build\x64\vc16\lib -lopencv_world4100
```

### FFmpeg Installation

#### Linux:
```bash
# Ubuntu/Debian
sudo apt install ffmpeg

# Fedora/CentOS/RHEL  
sudo dnf install ffmpeg
```

#### Windows:
Download from https://ffmpeg.org and add to PATH.

## Installation

1. **Build the application:**
   ```bash
   cd /path/to/cryospy/client/capture-client
   go mod tidy
   go build -o capture-client .
   ```

2. **Create configuration:**
   ```bash
   cp config.example.json config.json
   # Edit config.json with your settings
   ```

3. **Test camera access:**
   ```bash
   # Linux - list video devices
   ls /dev/video*
   
   # Test camera with FFmpeg
   ffmpeg -f v4l2 -i /dev/video0 -t 5 test.mp4
   ```

## Configuration

The application uses a `config.json` file for basic settings. All video processing and recording settings are managed by the server.

```json
{
  "client_id": "your-client-id",
  "client_secret": "your-client-secret",
  "server_url": "http://localhost:8080",
  "camera_device": "/dev/video0",
  "buffer_size": 5,
  "settings_sync_seconds": 300,
  "server_timeout_seconds": 30
}
```

### Configuration Options

- **client_id**: Client identifier for authentication
- **client_secret**: Client secret for authentication  
- **server_url**: URL of the CryoSpy capture server
- **camera_device**: Video device path (`/dev/video0` on Linux, camera name or `0` on Windows)
- **buffer_size**: Number of clips to buffer in memory during upload (default: 5)
- **settings_sync_seconds**: How often to sync settings from server in seconds (default: 300)
- **server_timeout_seconds**: HTTP request timeout in seconds (default: 30)

## Usage

### Basic Usage

```bash
# Start the capture client
./capture-client

# Stop with Ctrl+C for graceful shutdown
```

### Command Line Options

You can override configuration values using command line flags:

```bash
./capture-client --client-id myClient --server-url http://192.168.1.100:8080
./capture-client --camera-device /dev/video1 --buffer-size 10
./capture-client --settings-sync-seconds 600 --server-timeout-seconds 60
```

Available flags:
- `--client-id`: Override client ID
- `--client-secret`: Override client secret  
- `--server-url`: Override server URL
- `--camera-device`: Override camera device
- `--buffer-size`: Override buffer size
- `--settings-sync-seconds`: Override settings sync interval
- `--server-timeout-seconds`: Override server timeout

## Architecture

The capture client uses a modular architecture with the following key components:

### Core Components

- **Recorder** (OpenCV-based): Handles continuous webcam capture and raw video recording
- **Motion Detector**: Analyzes recorded clips for motion using background subtraction
- **Post Processor** (FFmpeg-based): Applies video compression, format conversion, and effects
- **Upload Queue**: Manages concurrent upload of processed clips to the server
- **File Tracker**: Handles temporary file management and cleanup
- **Settings Provider**: Synchronizes client settings from the server

### Process Flow

1. **Initialization**
   - Load local configuration and authenticate with server
   - Fetch client-specific settings (recording duration, motion detection, video processing)
   - Initialize camera and start continuous recording

2. **Recording Loop**
   - Capture video in chunks based on server-configured duration
   - Save raw clips to temporary directory
   - Queue clips for processing while recording continues

3. **Processing Pipeline**
   ```
   Raw Video → Motion Detection → Post Processing → Upload Queue → Server
   ```

4. **Motion Detection**
   - Analyze frames using OpenCV background subtraction
   - Always runs to determine if motion is present in clips
   - If server "Motion Only" setting is enabled, clips without motion are filtered out before upload
   - Configurable sensitivity and detection parameters

5. **Post Processing**
   - Apply server-specified transformations (downscaling, grayscale, compression)
   - Convert to specified output format and codec
   - Optimize for upload size while maintaining quality

6. **Upload Management**
   - Queue processed clips for background upload
   - Retry failed uploads with exponential backoff
   - Clean up temporary files after successful upload

## Server-Managed Settings

All video processing and recording settings are managed by the CryoSpy server through the dashboard:

### Recording Settings
- **Clip Duration**: Length of each recorded video chunk (30-1800 seconds)
- **Capture Codec**: Video codec for raw capture (MJPG, YUYV, H264)
- **Frame Rate**: Capture frame rate (FPS)

### Motion Detection
- **Motion Only**: Enable/disable motion detection filtering
- **Min Area**: Minimum area threshold for motion detection
- **Max Frames**: Maximum frames to analyze per clip
- **Warm-up Frames**: Initial frames to skip before detection

### Post Processing
- **Output Format**: Container format (mp4, avi, mkv, webm, mov)
- **Output Codec**: Video codec (libx264, libx265, libvpx-vp9, ffv1)
- **Video Bitrate**: Compression bitrate (500k to 15000k)
- **Downscale Resolution**: Target resolution (360p, 480p, 720p, 1080p, etc.)
- **Grayscale**: Convert to black and white

## Troubleshooting

### Common Issues

**Camera Access Denied:**
```bash
# Linux - check permissions
sudo chmod 666 /dev/video0
sudo usermod -a -G video $USER  # logout/login required

# Test camera
ffmpeg -f v4l2 -i /dev/video0 -t 5 test.mp4
```

**OpenCV Not Found:**
```bash
# Ubuntu/Debian
sudo apt install libopencv-dev pkg-config

# Fedora
sudo dnf install opencv-devel pkgconf-pkg-config
```

**No Video Devices:**
```bash
# Check for devices
ls /dev/video*

# Check system logs
dmesg | grep video
```

**FFmpeg Errors:**
- Ensure FFmpeg is installed and in PATH
- Check disk space in temp directory
- Verify camera is not used by another application

### Logging

The application creates daily rotating log files in the `logs/` directory and outputs to console. Log files are named with the format: `capture-client-YYYY-MM-DD.log`

## Development

### Building
```bash
go mod tidy
go build -o capture-client .
```

### Testing Camera
```bash
# List available cameras (Linux)
v4l2-ctl --list-devices

# Test with OpenCV
python3 -c "import cv2; cap = cv2.VideoCapture(0); ret, frame = cap.read(); print('Camera working:', ret); cap.release()"
```

### Performance Tuning
- Adjust `buffer_size` based on available memory
- Monitor CPU usage if motion detection is enabled
- Ensure sufficient disk space for temporary files
- Settings are optimized through the server dashboard

