#!/bin/bash
set -e

# Default values
SERVER_URL=""
CLIENT_ID=""
CLIENT_SECRET=""
PROXY_AUTH_HEADER=""
PROXY_AUTH_VALUE=""
WITH_SYSTEMD=false
SKIP_DEPENDENCIES=false
FORCE=false

echo "📹 CryoSpy Capture Client - Linux Installation Script"
echo "====================================================="

# Function to create configuration file
create_config_file() {
    echo "📝 Creating client configuration..."
    
    # Check if config.json already exists
    if [ -f "config.json" ] && [ "$FORCE" != true ]; then
        read -p "config.json already exists. Overwrite? (y/N): " overwrite
        if [[ ! "$overwrite" =~ ^[Yy]$ ]]; then
            echo "⏭️  Skipping configuration file creation"
            return
        fi
    fi
    
    # Prompt for missing values
    if [ -z "$SERVER_URL" ]; then
        read -p "Enter server URL (e.g., http://your-server-ip:8081 or https://yourdomain.com): " SERVER_URL
        if [ -z "$SERVER_URL" ]; then
            SERVER_URL="http://localhost:8081"
        fi
    fi
    
    if [ -z "$CLIENT_ID" ]; then
        read -p "Enter client ID (obtain from server admin): " CLIENT_ID
        if [ -z "$CLIENT_ID" ]; then
            CLIENT_ID="your-client-id"
        fi
    fi
    
    if [ -z "$CLIENT_SECRET" ]; then
        read -p "Enter client secret (obtain from server admin): " CLIENT_SECRET
        if [ -z "$CLIENT_SECRET" ]; then
            CLIENT_SECRET="your-client-secret"
        fi
    fi
    
    # Ask about proxy authentication if not provided
    if [ -z "$PROXY_AUTH_HEADER" ] && [ -z "$PROXY_AUTH_VALUE" ]; then
        read -p "Configure proxy authentication for reverse proxy? (y/N): " use_proxy_auth
        if [[ "$use_proxy_auth" =~ ^[Yy]$ ]]; then
            read -p "Enter proxy auth header name (e.g., X-Proxy-Auth): " PROXY_AUTH_HEADER
            read -p "Enter proxy auth header value: " PROXY_AUTH_VALUE
        fi
    fi
    
    # Create configuration file
    cat > config.json << EOF
{
  "client_id": "$CLIENT_ID",
  "client_secret": "$CLIENT_SECRET",
  "server_url": "$SERVER_URL",
  "camera_device": "/dev/video0",
  "buffer_size": 5,
  "settings_sync_seconds": 300,
  "server_timeout_seconds": 30,
  "proxy_auth_header": "${PROXY_AUTH_HEADER:-}",
  "proxy_auth_value": "${PROXY_AUTH_VALUE:-}"
}
EOF
    
    echo "✅ Configuration file created: config.json"
    
    if [ -n "$PROXY_AUTH_HEADER" ] && [ -n "$PROXY_AUTH_VALUE" ]; then
        echo "🔐 Proxy authentication configured for defense-in-depth security"
    fi
}

# Function to parse command line arguments
parse_arguments() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            --server-url)
                SERVER_URL="$2"
                shift 2
                ;;
            --client-id)
                CLIENT_ID="$2"
                shift 2
                ;;
            --client-secret)
                CLIENT_SECRET="$2"
                shift 2
                ;;
            --proxy-auth-header)
                PROXY_AUTH_HEADER="$2"
                shift 2
                ;;
            --proxy-auth-value)
                PROXY_AUTH_VALUE="$2"
                shift 2
                ;;
            --with-systemd)
                WITH_SYSTEMD=true
                shift
                ;;
            --skip-dependencies)
                SKIP_DEPENDENCIES=true
                shift
                ;;
            --force)
                FORCE=true
                shift
                ;;
            --help|-h)
                show_help
                exit 0
                ;;
            *)
                echo "Unknown option: $1"
                echo "Use --help for usage information."
                exit 1
                ;;
        esac
    done
}

# Function to show help
show_help() {
    echo "Usage: $0 [options]"
    echo ""
    echo "Options:"
    echo "  --server-url URL          Server URL (e.g., https://yourdomain.com)"
    echo "  --client-id ID            Client ID from server admin"
    echo "  --client-secret SECRET    Client secret from server admin"
    echo "  --proxy-auth-header NAME  Proxy authentication header name"
    echo "  --proxy-auth-value VALUE  Proxy authentication header value"
    echo "  --with-systemd            Create systemd service file"
    echo "  --skip-dependencies       Skip dependency installation"
    echo "  --force                   Overwrite existing files without prompting"
    echo "  --help, -h                Show this help message"
    echo ""
    echo "Examples:"
    echo "  # Interactive installation"
    echo "  $0"
    echo ""
    echo "  # Automated installation with proxy auth"
    echo "  $0 --server-url https://yourdomain.com --client-id camera-01 \\"
    echo "     --client-secret secure-secret --proxy-auth-header X-Proxy-Auth \\"
    echo "     --proxy-auth-value proxy-token --with-systemd --force"
}

# Function to detect the package manager and install dependencies
install_dependencies() {
    echo "Installing CryoSpy client dependencies..."
    
    # Detect OS and package manager
    if command -v apt-get &> /dev/null; then
        echo "Detected Debian/Ubuntu system with apt"
        sudo apt-get update
        sudo apt-get install -y libopencv-dev pkg-config ffmpeg
    elif command -v dnf &> /dev/null; then
        echo "Detected Fedora/RHEL system with dnf"
        sudo dnf install -y opencv-devel pkgconf-pkg-config ffmpeg ffmpeg-devel
    elif command -v yum &> /dev/null; then
        echo "Detected RHEL/CentOS system with yum"
        # Enable EPEL repository for FFmpeg
        if ! rpm -qa | grep -q epel-release; then
            echo "Installing EPEL repository..."
            sudo yum install -y epel-release
        fi
        sudo yum install -y opencv-devel pkgconfig ffmpeg ffmpeg-devel
    elif command -v pacman &> /dev/null; then
        echo "Detected Arch Linux system with pacman"
        sudo pacman -Sy --noconfirm opencv pkgconf ffmpeg
    elif command -v zypper &> /dev/null; then
        echo "Detected openSUSE system with zypper"
        sudo zypper install -y opencv-devel pkg-config ffmpeg-4
    else
        echo "ERROR: Unsupported package manager."
        echo "Please install the following packages manually:"
        echo "  - OpenCV development libraries (for camera capture and motion detection)"
        echo "  - pkg-config (required for OpenCV compilation)"
        echo "  - FFmpeg (for video post-processing)"
        echo ""
        echo "On your system, this might be:"
        echo "  - opencv-dev or opencv-devel"
        echo "  - pkg-config or pkgconfig or pkgconf-pkg-config"
        echo "  - ffmpeg"
        exit 1
    fi
}

# Function to set permissions
set_permissions() {
    echo "Setting executable permissions..."
    chmod +x capture-client 2>/dev/null || echo "⚠ capture-client not found"
}

# Function to create systemd service file (optional)
create_systemd_service() {
    echo "Creating systemd service file..."
    
    # Get current directory
    INSTALL_DIR=$(pwd)
    
    # Create capture-client service
    cat > cryospy-capture-client.service << EOF
[Unit]
Description=CryoSpy Capture Client
After=network.target

[Service]
Type=simple
User=$USER
WorkingDirectory=$INSTALL_DIR
ExecStart=$INSTALL_DIR/capture-client
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
EOF

    echo "✅ Created systemd service file: cryospy-capture-client.service"
}

# Main installation process
main() {
    echo "Starting client installation..."
    echo ""
    
    # Check if running as root
    if [ "$EUID" -eq 0 ]; then
        echo "⚠ Please do not run this script as root."
        echo "  The script will prompt for sudo when needed."
        exit 1
    fi
    
    # Install dependencies unless skipped
    if [ "$SKIP_DEPENDENCIES" != true ]; then
        install_dependencies
        echo ""
    else
        echo "⏭️  Skipping dependency installation"
        echo ""
    fi
    
    # Set permissions
    set_permissions
    echo ""
    
    # Create configuration file
    create_config_file
    echo ""
    
    # Create systemd service if requested
    if [ "$WITH_SYSTEMD" = true ]; then
        create_systemd_service "--with-systemd"
        echo ""
    fi
    
    echo "🎉 Client installation complete!"
    echo "================================="
    echo ""
    echo "Configuration created: config.json"
    if [ "$WITH_SYSTEMD" = true ]; then
        echo ""
        echo "Systemd service file created. To install and start:"
        echo "  sudo cp cryospy-capture-client.service /etc/systemd/system/"
        echo "  sudo systemctl daemon-reload"
        echo "  sudo systemctl enable cryospy-capture-client"
        echo "  sudo systemctl start cryospy-capture-client"
    fi
    echo ""
    echo "To start the capture client manually:"
    echo "  ./capture-client"
    echo ""
    echo "Troubleshooting:"
    echo "- If camera access is blocked: ./kill-camera-processes.sh"
    echo "- Check camera permissions: sudo usermod -a -G video \$USER"
    echo ""
    echo "For help and documentation, visit:"
    echo "https://github.com/Yeti47/cryospy"
}

# Parse command line arguments and run installation
parse_arguments "$@"
main
