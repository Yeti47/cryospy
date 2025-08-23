#!/bin/bash
set -e

# Default values
WITH_SYSTEMD=false
SKIP_DEPENDENCIES=false
FORCE=false

echo "🚀 CryoSpy Server Components - Linux Installation Script"
echo "========================================================="

# Function to detect the package manager and install dependencies
install_dependencies() {
    echo "Installing CryoSpy server dependencies..."
    
    # Detect OS and package manager
    if command -v apt-get &> /dev/null; then
        echo "Detected Debian/Ubuntu system with apt"
        sudo apt-get update
        sudo apt-get install -y ffmpeg
    elif command -v dnf &> /dev/null; then
        echo "Detected Fedora/RHEL system with dnf"
        sudo dnf install -y ffmpeg ffmpeg-devel
    elif command -v yum &> /dev/null; then
        echo "Detected RHEL/CentOS system with yum"
        # Enable EPEL repository for FFmpeg
        if ! rpm -qa | grep -q epel-release; then
            echo "Installing EPEL repository..."
            sudo yum install -y epel-release
        fi
        sudo yum install -y ffmpeg ffmpeg-devel
    elif command -v pacman &> /dev/null; then
        echo "Detected Arch Linux system with pacman"
        sudo pacman -Sy --noconfirm ffmpeg
    elif command -v zypper &> /dev/null; then
        echo "Detected openSUSE system with zypper"
        sudo zypper install -y ffmpeg-4
    else
        echo "ERROR: Unsupported package manager."
        echo "Please install the following packages manually:"
        echo "  - FFmpeg"
        echo ""
        echo "On your system, this might be:"
        echo "  - ffmpeg"
        exit 1
    fi
}

# Function to set permissions
set_permissions() {
    echo "Setting executable permissions..."
    chmod +x capture-server 2>/dev/null || echo "⚠ capture-server not found"
    chmod +x dashboard 2>/dev/null || echo "⚠ dashboard not found"
}

# Function to create initial configuration
create_initial_config() {
    echo "📝 Creating initial server configuration..."
    
    # Get user's home directory and create cryospy folder
    CRYOSPY_DIR="$HOME/cryospy"
    CONFIG_PATH="$CRYOSPY_DIR/config.json"
    
    # Create directory if it doesn't exist
    if [ ! -d "$CRYOSPY_DIR" ]; then
        mkdir -p "$CRYOSPY_DIR"
        echo "✅ Created configuration directory: $CRYOSPY_DIR"
    fi
    
    # Check if config already exists
    if [ -f "$CONFIG_PATH" ] && [ "$FORCE" != true ]; then
        read -p "Configuration file already exists at $CONFIG_PATH. Overwrite? (y/N): " overwrite
        if [[ ! "$overwrite" =~ ^[Yy]$ ]]; then
            echo "⏭️  Skipping configuration file creation"
            echo "$CONFIG_PATH"
            return
        fi
    fi
    
    # Create logs directory path
    LOGS_PATH="$CRYOSPY_DIR/logs"
    DB_PATH="$CRYOSPY_DIR/cryospy.db"
    
    # Create initial configuration with appropriate paths
    cat > "$CONFIG_PATH" << EOF
{
  "web_addr": "127.0.0.1",
  "web_port": 8080,
  "capture_port": 8081,
  "database_path": "$DB_PATH",
  "log_path": "$LOGS_PATH",
  "log_level": "info",
  "trusted_proxies": {
    "capture_server": [],
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
EOF
    
    echo "✅ Configuration file created: $CONFIG_PATH"
    echo "🔧 Please customize the configuration before starting the server:"
    echo "   - Update email settings for notifications"
    echo "   - Configure trusted proxies if using reverse proxy"
    echo "   - Adjust storage and streaming settings as needed"
    echo ""
    echo "$CONFIG_PATH"
}

# Function to create systemd service files (optional)
create_systemd_services() {
    if [ "$1" = "--with-systemd" ]; then
        echo "Creating systemd service files..."
        
        # Get current directory
        INSTALL_DIR=$(pwd)
        
        # Create capture-server service
        cat > cryospy-capture-server.service << EOF
[Unit]
Description=CryoSpy Capture Server
After=network.target

[Service]
Type=simple
User=$USER
WorkingDirectory=$INSTALL_DIR
ExecStart=$INSTALL_DIR/capture-server
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

        # Create dashboard service
        cat > cryospy-dashboard.service << EOF
[Unit]
Description=CryoSpy Dashboard
After=network.target

[Service]
Type=simple
User=$USER
WorkingDirectory=$INSTALL_DIR
ExecStart=$INSTALL_DIR/dashboard
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

        echo "✓ Created systemd service files:"
        echo "  - cryospy-capture-server.service"
        echo "  - cryospy-dashboard.service"
        echo ""
        echo "To install and start services:"
        echo "  sudo cp *.service /etc/systemd/system/"
        echo "  sudo systemctl daemon-reload"
        echo "  sudo systemctl enable cryospy-capture-server cryospy-dashboard"
        echo "  sudo systemctl start cryospy-capture-server cryospy-dashboard"
    fi
}

# Main installation process
main() {
    echo "Starting server installation..."
    echo ""
    
    # Check if running as root
    if [ "$EUID" -eq 0 ]; then
        echo "⚠ Please do not run this script as root."
        echo "  The script will prompt for sudo when needed."
        exit 1
    fi
    
    # Install dependencies
    install_dependencies
    echo ""
    
    # Set permissions
    set_permissions
    echo ""
    
    # Create initial configuration
    CONFIG_PATH=$(create_initial_config)
    echo ""
    
    # Create systemd services if requested
    if [ "$WITH_SYSTEMD" = true ]; then
        create_systemd_services "--with-systemd"
        echo ""
    fi
    
    echo "🎉 Server installation complete!"
    echo "================================="
    echo ""
    echo "Next steps:"
    echo "1. Configuration file created at: $CONFIG_PATH"
    echo "   Please customize it before starting the server (especially email settings)"
    echo "2. Start the server components:"
    echo "   ./capture-server &"
    echo "   ./dashboard &"
    echo "3. Access the web dashboard at: http://localhost:8080"
    if [ "$WITH_SYSTEMD" = true ]; then
        echo ""
        echo "Systemd service files created. To install and start:"
        echo "  sudo cp *.service /etc/systemd/system/"
        echo "  sudo systemctl daemon-reload"
        echo "  sudo systemctl enable cryospy-capture-server cryospy-dashboard"
        echo "  sudo systemctl start cryospy-capture-server cryospy-dashboard"
    fi
    echo ""
    echo "For help and documentation, visit:"
    echo "https://github.com/Yeti47/cryospy"
}

# Function to parse command line arguments
parse_arguments() {
    while [[ $# -gt 0 ]]; do
        case $1 in
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
                echo "Usage: $0 [options]"
                echo ""
                echo "Options:"
                echo "  --with-systemd        Create systemd service files"
                echo "  --skip-dependencies   Skip dependency installation"
                echo "  --force               Overwrite existing config without prompting"
                echo "  --help, -h            Show this help message"
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

# Parse command line arguments and run installation
parse_arguments "$@"
main
