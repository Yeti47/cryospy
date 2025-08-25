#!/bin/bash
set -e

# Default values
SERVER_URL=""
CLIENT_ID=""
CLIENT_SECRET=""
PROXY_AUTH_HEADER=""
PROXY_AUTH_VALUE=""
FORCE=false

echo "📹 CryoSpy Capture Client - Configuration Script"
echo "================================================"

# Function to create configuration file
create_config_file() {
    echo "📝 Creating client configuration..."
    
    # Always write config.json to user's working directory
    CONFIG_PATH="$PWD/config.json"
    if [ -f "$CONFIG_PATH" ] && [ "$FORCE" != true ]; then
        read -p "config.json already exists in $PWD. Overwrite? (y/N): " overwrite
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
    
    # Create configuration file in user's working directory
    cat > "$CONFIG_PATH" << EOF
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
    
    echo "✅ Configuration file created: $CONFIG_PATH"
    
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
    echo "  --force                   Overwrite existing files without prompting"
    echo "  --help, -h                Show this help message"
    echo ""
    echo "Examples:"
    echo "  # Interactive configuration"
    echo "  $0"
    echo ""
    echo "  # Automated configuration with proxy auth"
    echo "  $0 --server-url https://yourdomain.com --client-id camera-01 \\"
    echo "     --client-secret secure-secret --proxy-auth-header X-Proxy-Auth \\"
    echo "     --proxy-auth-value proxy-token --force"
}

# Main configuration process
main() {
    echo "Starting client configuration..."
    echo ""
    
    # Create configuration file
    create_config_file
    echo ""
    
    echo "🎉 Client configuration complete!"
    echo "================================="
    echo ""
    echo "Configuration created: config.json"
    echo ""
    echo "To start the capture client:"
    echo "  ./cryospy-capture-client-linux-x86_64.AppImage"
    echo ""
    echo "Troubleshooting:"
    echo "- If camera access is blocked, check camera permissions:"
    echo "  sudo usermod -a -G video \$USER"
    echo "- Make sure camera device exists (default: /dev/video0)"
    echo "- Check camera isn't being used by another application"
    echo ""
    echo "For help and documentation, visit:"
    echo "https://github.com/Yeti47/cryospy"
}

# Parse command line arguments and run configuration
parse_arguments "$@"
main
