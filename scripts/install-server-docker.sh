#!/bin/bash

# install-server-docker.sh
# Automated installer for CryoSpy Server (Docker-based) with Nginx and SSL
# This script sets up the complete server environment including Docker, Nginx, and Certbot.

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

# Default values
INSTALL_DIR="$HOME/cryospy"
DOCKER_IMAGE="ghcr.io/yeti47/cryospy-server:latest"
DOMAIN=""
EMAIL=""
PROXY_AUTH_HEADER=""
PROXY_AUTH_VALUE=""

# Helper functions
log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

check_root() {
    if [[ $EUID -eq 0 ]]; then
        log_error "This script should not be run as root. Please run as a regular user with sudo access."
        exit 1
    fi
    if ! sudo -n true 2>/dev/null; then
        log_warn "Sudo access is required. You may be prompted for your password."
        sudo true
    fi
}

detect_os() {
    if [ -f /etc/os-release ]; then
        . /etc/os-release
        OS=$NAME
        DISTRO=$ID
    else
        log_error "Cannot detect OS. This script supports Linux only."
        exit 1
    fi
}

install_packages() {
    local packages="$1"
    log_info "Installing packages: $packages"
    
    if command -v apt-get &> /dev/null; then
        sudo apt-get update
        sudo apt-get install -y $packages
    elif command -v dnf &> /dev/null; then
        sudo dnf install -y $packages
    elif command -v yum &> /dev/null; then
        sudo yum install -y $packages
    elif command -v pacman &> /dev/null; then
        sudo pacman -S --noconfirm $packages
    elif command -v zypper &> /dev/null; then
        sudo zypper install -y $packages
    else
        log_error "No supported package manager found."
        exit 1
    fi
}

check_docker() {
    if ! command -v docker &> /dev/null; then
        log_warn "Docker not found. Attempting to install..."
        curl -fsSL https://get.docker.com -o get-docker.sh
        sudo sh get-docker.sh
        rm get-docker.sh
        
        # Add user to docker group
        sudo usermod -aG docker $USER
        log_warn "Docker installed. You may need to log out and back in for group changes to take effect."
        log_warn "Please re-run this script after logging back in."
        exit 1
    else
        log_success "Docker is installed."
    fi

    # Check for docker compose
    if ! docker compose version &> /dev/null; then
        log_error "Docker Compose plugin not found. Please install 'docker-compose-plugin'."
        exit 1
    fi
}

setup_directory() {
    log_info "Setting up installation directory at $INSTALL_DIR"
    mkdir -p "$INSTALL_DIR/data"
}

create_docker_compose() {
    log_info "Creating docker-compose.yml"
    cat > "$INSTALL_DIR/docker-compose.yml" <<EOF
services:
  cryospy-server:
    image: $DOCKER_IMAGE
    container_name: cryospy-server
    restart: unless-stopped
    ports:
      - "8080:8080"           # Dashboard (Direct access, e.g., http://local-ip:8080)
      - "127.0.0.1:8081:8081" # Capture API (Proxied via Nginx only)
    volumes:
      - ./data:/home/cryospy
EOF
}

configure_nginx() {
    log_info "Configuring Nginx..."
    
    # Install Nginx and Certbot if missing
    if ! command -v nginx &> /dev/null; then
        install_packages "nginx"
    fi
    
    # Install Certbot
    if ! command -v certbot &> /dev/null; then
        if [[ "$DISTRO" =~ ^(ubuntu|debian|fedora|centos|rhel)$ ]]; then
            install_packages "certbot python3-certbot-nginx"
        else
             log_warn "Could not automatically install certbot packages for $DISTRO. Please install 'certbot' and the nginx plugin manually."
        fi
    fi

    # Prepare Proxy Auth config
    local proxy_auth_block=""
    if [[ -n "$PROXY_AUTH_HEADER" && -n "$PROXY_AUTH_VALUE" ]]; then
        local nginx_header_var="\$http_$(echo "$PROXY_AUTH_HEADER" | tr '[:upper:]' '[:lower:]' | tr '-' '_')"
        proxy_auth_block="
        # Proxy Authentication
        if ($nginx_header_var != \"$PROXY_AUTH_VALUE\") {
            return 401 \"Unauthorized\";
        }"
    fi

    # Create Nginx Config
    local config_file="/etc/nginx/conf.d/cryospy.conf"
    # Remove default site if it exists (common on Debian/Ubuntu)
    if [ -f /etc/nginx/sites-enabled/default ]; then
        sudo rm -f /etc/nginx/sites-enabled/default
    fi

    log_info "Writing Nginx configuration to $config_file"
    sudo tee "$config_file" > /dev/null <<EOF
# Rate limiting zones
limit_req_zone \$binary_remote_addr zone=api:10m rate=10r/s;
limit_req_zone \$binary_remote_addr zone=upload:10m rate=2r/s;

server {
    listen 80;
    server_name $DOMAIN;
    
    # Redirect all HTTP traffic to HTTPS
    return 301 https://\$host\$request_uri;
}

server {
    listen 443 ssl http2;
    server_name $DOMAIN;

    # Security headers
    add_header X-Frame-Options DENY;
    add_header X-Content-Type-Options nosniff;
    add_header X-XSS-Protection "1; mode=block";

    # SSL configuration will be managed by Certbot

    # Dashboard (Web UI) - Blocked in Nginx
    # Access directly via http://your-server-ip:8080 on LAN
    location / {
        return 403 "Access denied.";
        add_header Content-Type text/plain;
    }

    # Capture Server API
    location /capture-server/ {
        # Apply rate limiting for uploads
        limit_req zone=upload burst=10 nodelay;

        $proxy_auth_block

        # Increase timeout and body size for video uploads
        client_max_body_size 100M;
        proxy_read_timeout 300s;
        proxy_connect_timeout 10s;
        proxy_send_timeout 300s;

        proxy_pass http://127.0.0.1:8081/;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
        
        # Security headers for API
        add_header X-Robots-Tag "noindex, nofollow";
    }
}
EOF

    # Test and Reload Nginx
    log_info "Testing Nginx configuration..."
    sudo nginx -t
    sudo systemctl reload nginx
}

setup_ssl() {
    log_info "Setting up SSL with Certbot..."
    sudo certbot --nginx -d "$DOMAIN" --email "$EMAIL" --agree-tos --non-interactive --redirect
}

# Interactive Wizard
wizard() {
    echo -e "${CYAN}CryoSpy Server Installer${NC}"
    echo "------------------------"
    
    if [[ -z "$DOMAIN" ]]; then
        read -p "Enter your domain name (e.g., cryospy.example.com): " DOMAIN
    fi
    
    if [[ -z "$EMAIL" ]]; then
        read -p "Enter your email for SSL certificates: " EMAIL
    fi
    
    if [[ -z "$PROXY_AUTH_HEADER" ]]; then
        read -p "Do you want to enable Proxy Authentication? (y/N): " enable_auth
        if [[ "$enable_auth" =~ ^[Yy]$ ]]; then
            read -p "Enter Header Name (default: X-Proxy-Auth): " PROXY_AUTH_HEADER
            PROXY_AUTH_HEADER=${PROXY_AUTH_HEADER:-X-Proxy-Auth}
            read -p "Enter Secret Token: " PROXY_AUTH_VALUE
        fi
    fi
}

main() {
    check_root
    detect_os
    wizard
    
    if [[ -z "$DOMAIN" ]] || [[ -z "$EMAIL" ]]; then
        log_error "Domain and Email are required."
        exit 1
    fi

    check_docker
    setup_directory
    create_docker_compose
    
    log_info "Starting CryoSpy Server container..."
    cd "$INSTALL_DIR"
    docker compose up -d
    
    configure_nginx
    setup_ssl
    
    log_success "Installation Complete!"
    echo -e "
${GREEN}CryoSpy is now running!${NC}

Dashboard: http://<server-ip>:8080 (LAN Access Only)
Client Server URL: https://$DOMAIN/capture-server

Configuration is located at: $INSTALL_DIR/data/config.json
Docker Compose file: $INSTALL_DIR/docker-compose.yml

To view logs:
  cd $INSTALL_DIR
  docker compose logs -f
"
    if [[ -n "$PROXY_AUTH_HEADER" ]]; then
        echo -e "${YELLOW}IMPORTANT: Proxy Authentication is ENABLED.${NC}"
        echo "You must configure your clients with:"
        echo "  \"proxy_auth_header\": \"$PROXY_AUTH_HEADER\""
        echo "  \"proxy_auth_value\": \"$PROXY_AUTH_VALUE\""
    fi
}

main
