# Capture Client

The CryoSpy capture client is a Go application that continuously captures video from a webcam, processes it according to server settings, and uploads clips to the CryoSpy capture server.

## Features

- **Continuous video capture** from webcam devices
- **Motion detection** using OpenCV (optional, configurable via server settings)
- **Real-time video processing** with compression, downscaling, and grayscaling
- **Chunked recording** with configurable duration
- **Concurrent upload** while recording continues
- **Buffer management** to prevent memory issues
- **Automatic authentication** with the capture server

## Prerequisites

### System Requirements

- Linux or Windows operating system (tested on Fedora Linux and Windows 10/11)
- Go 1.24.3 or later
- OpenCV 4.x
- FFmpeg (for video processing)
- Webcam device

### OpenCV Installation

#### Ubuntu/Debian:
```bash
sudo apt update
sudo apt install -y libopencv-dev pkg-config
```

#### CentOS/RHEL/Fedora:
```bash
# Fedora
sudo dnf install opencv-devel pkgconf-pkg-config

# CentOS/RHEL (requires EPEL)
sudo yum install epel-release
sudo yum install opencv-devel pkgconfig
```

#### Arch Linux:
```bash
sudo pacman -S opencv pkgconf
```

#### Windows:
1. **Install OpenCV:**
   - Download OpenCV from https://opencv.org/releases/
   - Extract to `C:\opencv` 
   - Add `C:\opencv\build\x64\vc16\bin` to your PATH environment variable

2. **Set environment variables:**
   ```cmd
   set CGO_CPPFLAGS=-IC:\opencv\build\include
   set CGO_LDFLAGS=-LC:\opencv\build\x64\vc16\lib -lopencv_world4100
   ```

   Or for PowerShell:
   ```powershell
   $env:CGO_CPPFLAGS="-IC:\opencv\build\include"
   $env:CGO_LDFLAGS="-LC:\opencv\build\x64\vc16\lib -lopencv_world4100"
   ```

3. **Alternative - Use vcpkg (recommended):**
   ```cmd
   vcpkg install opencv4[contrib,nonfree]:x64-windows
   vcpkg integrate install
   ```

### FFmpeg Installation

#### Ubuntu/Debian:
```bash
sudo apt install ffmpeg
```

#### CentOS/RHEL/Fedora:
```bash
sudo dnf install ffmpeg
# or
sudo yum install ffmpeg
```

#### Arch Linux:
```bash
sudo pacman -S ffmpeg
```

#### Windows:
1. **Download FFmpeg:**
   - Download from https://ffmpeg.org/download.html#build-windows
   - Extract to `C:\ffmpeg`
   - Add `C:\ffmpeg\bin` to your PATH environment variable

2. **Alternative - Use winget:**
   ```cmd
   winget install Gyan.FFmpeg
   ```

3. **Alternative - Use Chocolatey:**
   ```cmd
   choco install ffmpeg
   ```

## Installation

### Linux

1. **Clone and build:**
   ```bash
   cd /path/to/cryospy/client/capture-client
   go mod tidy
   go build -o capture-client .
   ```

2. **Create configuration:**
   ```bash
   cp config.example.json config.json
   # Edit config.json with your settings
   # The example includes all available video processing options
   ```

3. **Configure your camera device:**
   ```bash
   # Find available video devices
   ls /dev/video*
   
   # Test your camera
   ffmpeg -f v4l2 -i /dev/video0 -t 5 test.mp4
   ```

### Windows

1. **Clone and build:**
   ```cmd
   cd C:\path\to\cryospy\client\capture-client
   go mod tidy
   go build -o capture-client.exe .
   ```

2. **Create configuration:**
   ```cmd
   copy config.example.json config.json
   rem Edit config.json with your settings
   rem The example includes all available video processing options
   ```

3. **Configure your camera device:**
   ```cmd
   rem List available video devices
   ffmpeg -list_devices true -f dshow -i dummy
   
   rem Test your camera (replace "Your Camera Name" with actual camera name)
   ffmpeg -f dshow -i video="Your Camera Name" -t 5 test.mp4
   ```

## Configuration

Edit `config.json`:

```json
{
  "client_id": "your-client-id",
  "client_secret": "your-client-secret", 
  "server_url": "http://your-server:8080",
  "camera_device": "/dev/video0",
  "buffer_size": 5,
  "settings_sync_seconds": 300,
  "video_codec": "mpeg4",
  "video_output_format": "mp4",
  "video_bitrate": "500k",
  "capture_codec": "MJPG",
  "capture_framerate": 15.0,
  "motion_sensitivity": 10.0
}
```

### Configuration Options

#### Basic Settings
- **client_id**: Client identifier for authentication
- **client_secret**: Client secret for authentication  
- **server_url**: URL of the capture server
- **camera_device**: Video device path
  - Linux: `/dev/video0` (device path) or `0` (OpenCV device ID)
  - Windows: Camera name (e.g., `"USB2.0 Camera"`) or `0` (OpenCV device ID)
- **buffer_size**: Number of clips to buffer in memory during upload
- **settings_sync_seconds**: How often to sync settings from server (in seconds, default: 300)

#### Video Processing Settings
- **video_codec**: Video codec for final processing (default: `"mpeg4"`)
  - Common options: `"mpeg4"`, `"libopenh264"`, `"libx264"` (if available)
- **video_output_format**: Output container format (default: `"mp4"`)
  - Common options: `"mp4"`, `"avi"`, `"mkv"`
- **video_bitrate**: Video bitrate for compression (default: `"500k"`)
  - Examples: `"500k"`, `"1M"`, `"2M"` (higher = better quality, larger files)
- **capture_codec**: Codec for initial capture (default: `"MJPG"`)
  - Common options: `"MJPG"`, `"MP4V"`, `"YUYV"`
- **capture_framerate**: Frame rate for video capture (default: `15.0`)
  - Examples: `15.0`, `30.0` (higher = smoother video, larger files)
- **motion_sensitivity**: Motion detection sensitivity as percentage (default: `10.0`)
  - Examples: `10.0` (10% of pixels), `5.0` (5% - more sensitive), `20.0` (20% - less sensitive)

## Usage

### Linux

1. **Start the capture client:**
   ```bash
   ./capture-client
   ```

2. **Stop gracefully:**
   ```bash
   # Press Ctrl+C for graceful shutdown
   ```

### Windows

1. **Start the capture client:**
   ```cmd
   capture-client.exe
   ```

2. **Stop gracefully:**
   ```cmd
   rem Press Ctrl+C for graceful shutdown
   ```

### Command Line Options

You can override configuration values using command line parameters. CLI parameters take precedence over the JSON configuration file.

#### Basic Options

- `--test`: Run in test mode with mock server client (no real server required)

#### Configuration Overrides

**Basic Settings:**
- `--client-id <id>`: Override client ID from config
- `--client-secret <secret>`: Override client secret from config  
- `--server-url <url>`: Override server URL from config
- `--camera-device <device>`: Override camera device from config
- `--buffer-size <size>`: Override buffer size from config
- `--settings-sync-seconds <seconds>`: Override settings sync interval from config

**Video Processing Settings:**
- `--video-codec <codec>`: Override video codec (e.g., `mpeg4`, `libopenh264`)
- `--video-output-format <format>`: Override video output format (e.g., `mp4`, `avi`)
- `--video-bitrate <bitrate>`: Override video bitrate (e.g., `500k`, `1M`)
- `--capture-codec <codec>`: Override capture codec (e.g., `MJPG`, `MP4V`)
- `--capture-framerate <rate>`: Override capture frame rate (e.g., `15.0`, `30.0`)
- `--motion-sensitivity <percentage>`: Override motion detection sensitivity (e.g., `10.0`, `5.0`)

#### Examples

**Linux:**
```bash
# Run in test mode
./capture-client --test

# Override camera device for testing
./capture-client --camera-device /dev/video1

# Override video processing settings
./capture-client --video-codec mpeg4 --video-bitrate 1M --capture-framerate 30.0

# Override motion detection sensitivity (lower = more sensitive)
./capture-client --motion-sensitivity 5.0

# Override multiple settings
./capture-client --server-url http://192.168.1.100:8080 --camera-device 0 --buffer-size 10

# Test mode with different settings
./capture-client --test --settings-sync-seconds 60 --video-bitrate 2M --motion-sensitivity 15.0
```

**Windows:**
```cmd
rem Run in test mode
capture-client.exe --test

rem Override camera device for testing
capture-client.exe --camera-device "USB2.0 HD Camera"

rem Override video processing settings
capture-client.exe --video-codec mpeg4 --video-bitrate 1M --capture-framerate 30.0

rem Override motion detection sensitivity
capture-client.exe --motion-sensitivity 5.0

rem Override multiple settings
capture-client.exe --server-url http://192.168.1.100:8080 --camera-device 0 --buffer-size 10
```

## How It Works

### Architecture

The application uses a hybrid approach combining:

- **gocv (OpenCV)** for:
  - Real-time webcam capture
  - Motion detection
  - Frame-by-frame processing
  - Continuous recording management

- **goffmpeg** for:
  - Video compression and optimization
  - Format conversion  
  - Downscaling and effects
  - Final processing before upload

### Process Flow

1. **Initialization:**
   - Load configuration from `config.json`
   - Authenticate with capture server
   - Fetch client settings (chunk duration, motion detection, etc.)

2. **Continuous Capture:**
   - Start webcam capture using OpenCV
   - Record video in chunks based on server settings
   - Process chunks concurrently while recording continues

3. **Processing Pipeline:**
   ```
   Raw Capture → Motion Detection → Video Processing → Upload Queue → Server
   ```

4. **Motion Detection (if enabled):**
   - Analyze video frames using background subtraction
   - Skip uploading clips with no motion
   - Configurable sensitivity

5. **Video Processing:**
   - Apply server-specified settings (grayscale, downscaling)
   - Use configurable codec and compression settings
   - Compress video for smaller file sizes
   - Convert to optimized format based on configuration

6. **Upload Management:**
   - Queue processed clips for upload
   - Upload in background while recording continues
   - Retry logic for failed uploads

## Troubleshooting

### Common Issues

1. **OpenCV not found:**
   ```
   Package opencv4 was not found in the pkg-config search path
   ```
   **Solution:** Install OpenCV development packages (see Prerequisites)

2. **Camera access denied:**
   ```
   failed to open webcam: VideoIO: can't open camera by index 0
   ```
   **Linux Solutions:**
   - Check if camera is not being used by another application
   - Verify device permissions: `sudo chmod 666 /dev/video0`
   - Add user to video group: `sudo usermod -a -G video $USER` (logout/login required)
   
   **Windows Solutions:**
   - Check if camera is not being used by another application
   - Run application as administrator if needed
   - Check Windows Camera privacy settings (Settings > Privacy > Camera)
   - Ensure camera drivers are properly installed

3. **No video devices:**
   ```
   ls /dev/video*
   # No output
   ```
   **Linux Solutions:**
   - Check if webcam is properly connected
   - Install USB video drivers if needed
   - Check dmesg for USB/camera errors: `dmesg | grep video`
   
   **Windows Solutions:**
   - Check Device Manager for camera devices
   - Update camera drivers through Device Manager
   - Use `ffmpeg -list_devices true -f dshow -i dummy` to list available cameras

4. **FFmpeg errors:**
   ```
   failed to process video: ffmpeg transcoding failed
   ```
   **Solutions:**
   - Verify FFmpeg installation: `ffmpeg -version`
   - Check input video format compatibility
   - Ensure sufficient disk space in temp directory
   - Try different video codec (e.g., `--video-codec mpeg4`)

5. **Video codec not supported:**
   ```
   Unknown encoder 'libx264'
   ```
   **Solutions:**
   - Use `mpeg4` codec which is more widely supported
   - Check available encoders: `ffmpeg -encoders | grep video`
   - Build FFmpeg with additional codec support if needed

6. **Poor video quality or large files:**
   **Solutions:**
   - Adjust video bitrate: `--video-bitrate 1M` (higher) or `--video-bitrate 250k` (lower)
   - Change capture frame rate: `--capture-framerate 30.0` (smoother) or `--capture-framerate 10.0` (smaller files)
   - Try different video codec for better compression

7. **Motion detection too sensitive or not sensitive enough:**
   **Solutions:**
   - For too many false positives (detecting motion when there is none):
     - Increase sensitivity value: `--motion-sensitivity 15.0` (less sensitive)
   - For missing actual motion:
     - Decrease sensitivity value: `--motion-sensitivity 5.0` (more sensitive)
   - Test different values: `2.0` (very sensitive) to `25.0` (very insensitive)
   - Note: Values below 5.0 may trigger on compression artifacts or minor lighting changes

### Codec Compatibility

**Widely Supported Codecs:**
- `mpeg4`: Native FFmpeg codec, works on most systems
- `libx264`: High-quality H.264 encoding (if FFmpeg built with x264 support)

**Capture Codecs:**
- `MJPG`: Motion JPEG, most compatible
- `MP4V`: MPEG-4 Part 2, good compatibility
- `YUYV`: Raw format, largest files but most compatible

### Debug Mode

Enable verbose logging by setting environment variable:

**Linux:**
```bash
export GOCV_DEBUG=1
./capture-client
```

**Windows:**
```cmd
set GOCV_DEBUG=1
capture-client.exe
```

### Testing Camera

Test your camera setup:

**Linux:**
```bash
# Test with OpenCV (requires OpenCV Python bindings)
python3 -c "import cv2; cap = cv2.VideoCapture(0); ret, frame = cap.read(); print('Camera working:', ret); cap.release()"

# Test with FFmpeg
ffmpeg -f v4l2 -i /dev/video0 -t 5 -y test.mp4 && echo "Camera test successful"
```

**Windows:**
```cmd
rem Test with OpenCV (requires OpenCV Python bindings)
python -c "import cv2; cap = cv2.VideoCapture(0); ret, frame = cap.read(); print('Camera working:', ret); cap.release()"

rem Test with FFmpeg (replace camera name with your actual camera)
ffmpeg -f dshow -i video="USB2.0 Camera" -t 5 -y test.mp4 && echo Camera test successful
```

## Development

### Building from Source

**Linux:**
```bash
# Get dependencies
go mod tidy

# Build
go build -o capture-client .

# Cross-compile for different architectures
GOOS=linux GOARCH=amd64 go build -o capture-client-amd64 .
GOOS=linux GOARCH=arm64 go build -o capture-client-arm64 .
GOOS=windows GOARCH=amd64 go build -o capture-client-windows.exe .
```

**Windows:**
```cmd
rem Get dependencies
go mod tidy

rem Build
go build -o capture-client.exe .

rem Cross-compile for different architectures
set GOOS=windows&& set GOARCH=amd64&& go build -o capture-client-amd64.exe .
set GOOS=linux&& set GOARCH=amd64&& go build -o capture-client-linux .
```

### Testing

```bash
# Run tests
go test ./...

# Test specific package
go test ./video
go test ./client
```

## Performance Tuning

### Memory Usage
- Adjust `buffer_size` in config based on available RAM
- Lower chunk duration reduces memory usage but increases processing overhead

### CPU Usage  
- Motion detection is CPU-intensive; disable if not needed
- Lower resolution reduces processing load
- Adjust video compression settings:
  - Lower bitrate: `--video-bitrate 250k` 
  - Lower frame rate: `--capture-framerate 10.0`
  - Use hardware-accelerated codecs if available

### Storage
- Processed videos are temporarily stored before upload
- Ensure sufficient disk space in temp directory
- Failed uploads are retried but may accumulate disk usage

### Video Quality vs Performance

**For best quality (higher resource usage):**
```bash
./capture-client --video-bitrate 2M --capture-framerate 30.0 --video-codec libx264
```

**For best performance (lower resource usage):**
```bash
./capture-client --video-bitrate 250k --capture-framerate 10.0 --video-codec mpeg4
```

**Balanced settings:**
```bash
./capture-client --video-bitrate 500k --capture-framerate 15.0 --video-codec mpeg4
```

