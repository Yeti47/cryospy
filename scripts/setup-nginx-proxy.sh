#!/bin/bash

# setup-nginx-proxy-linux.sh
# Bash script to set up nginx as a reverse proxy for CryoSpy capture-server on Linux
# This exposes ONLY the capture-server to the internet via HTTPS, keeping the dashboard local-only

set -e  # Exit on any error

# Default values
DOMAIN=""
CAPTURE_SERVER_PORT=8081
HTTPS_PORT=443
HTTP_PORT=80
PROXY_AUTH_HEADER=""
PROXY_AUTH_VALUE=""
EMAIL=""
SKIP_NGINX_INSTALL=false
SKIP_CERTBOT=false
FORCE=false

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Function to show help
show_help() {
    echo "Usage: $0 --domain DOMAIN [options]"
    echo ""
    echo "Required:"
    echo "  --domain DOMAIN           Domain name for SSL certificate (e.g., cryospy.example.com)"
    echo ""
    echo "Options:"
    echo "  --email EMAIL             Email for Let's Encrypt registration (required for SSL)"
    echo "  --capture-port PORT       Capture server port (default: 8081)"
    echo "  --https-port PORT         HTTPS port (default: 443)"
    echo "  --http-port PORT          HTTP port (default: 80)"
    echo "  --proxy-auth-header NAME  Custom proxy auth header name"
    echo "  --proxy-auth-value VALUE  Custom proxy auth header value"
    echo "  --skip-nginx-install      Skip nginx installation"
    echo "  --skip-certbot            Skip SSL certificate setup"
    echo "  --force                   Force overwrite existing configurations"
    echo "  --help                    Show this help message"
    echo ""
    echo "Examples:"
    echo "  # Basic setup with Let's Encrypt SSL"
    echo "  $0 --domain cryospy.example.com --email admin@example.com"
    echo ""
    echo "  # With proxy authentication"
    echo "  $0 --domain cryospy.example.com --email admin@example.com \\"
    echo "     --proxy-auth-header X-CryoSpy-Auth --proxy-auth-value secret-key-123"
}

# Function to parse command line arguments
parse_arguments() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            --domain)
                DOMAIN="$2"
                shift 2
                ;;
            --email)
                EMAIL="$2"
                shift 2
                ;;
            --capture-port)
                CAPTURE_SERVER_PORT="$2"
                shift 2
                ;;
            --https-port)
                HTTPS_PORT="$2"
                shift 2
                ;;
            --http-port)
                HTTP_PORT="$2"
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
            --skip-nginx-install)
                SKIP_NGINX_INSTALL=true
                shift
                ;;
            --skip-certbot)
                SKIP_CERTBOT=true
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
                echo -e "${RED}Unknown option: $1${NC}"
                echo "Use --help for usage information."
                exit 1
                ;;
        esac
    done
}

# Function to check if running as root
check_root() {
    if [[ $EUID -eq 0 ]]; then
        echo -e "${RED}This script should not be run as root for security reasons.${NC}"
        echo -e "${YELLOW}Please run as a regular user with sudo access.${NC}"
        exit 1
    fi
    
    # Check if user has sudo access
    if ! sudo -n true 2>/dev/null; then
        echo -e "${RED}This script requires sudo access. Please ensure your user is in the sudo group.${NC}"
        exit 1
    fi
}

# Function to detect package manager and install packages
install_packages() {
    local packages="$1"
    
    if command -v apt-get &> /dev/null; then
        echo -e "${BLUE}Installing packages with apt: $packages${NC}"
        sudo apt-get update
        sudo apt-get install -y $packages
    elif command -v dnf &> /dev/null; then
        echo -e "${BLUE}Installing packages with dnf: $packages${NC}"
        sudo dnf install -y $packages
    elif command -v yum &> /dev/null; then
        echo -e "${BLUE}Installing packages with yum: $packages${NC}"
        sudo yum install -y $packages
    elif command -v pacman &> /dev/null; then
        echo -e "${BLUE}Installing packages with pacman: $packages${NC}"
        sudo pacman -S --noconfirm $packages
    elif command -v zypper &> /dev/null; then
        echo -e "${BLUE}Installing packages with zypper: $packages${NC}"
        sudo zypper install -y $packages
    else
        echo -e "${RED}No supported package manager found. Please install manually: $packages${NC}"
        exit 1
    fi
}

# Function to install nginx
install_nginx() {
    if [[ "$SKIP_NGINX_INSTALL" == true ]]; then
        echo -e "${YELLOW}Skipping nginx installation${NC}"
        return
    fi
    
    echo -e "${GREEN}📥 Installing nginx...${NC}"
    
    if command -v nginx &> /dev/null; then
        echo -e "${YELLOW}Nginx is already installed${NC}"
        if [[ "$FORCE" != true ]]; then
            read -p "Nginx is already installed. Continue? (y/N): " -n 1 -r
            echo
            if [[ ! $REPLY =~ ^[Yy]$ ]]; then
                exit 1
            fi
        fi
    else
        install_packages "nginx"
    fi
    
    # Enable and start nginx
    sudo systemctl enable nginx
    sudo systemctl start nginx
    
    echo -e "${GREEN}✅ Nginx installed and started${NC}"
}

# Function to install certbot
install_certbot() {
    if [[ "$SKIP_CERTBOT" == true ]]; then
        echo -e "${YELLOW}Skipping certbot installation${NC}"
        return
    fi
    
    echo -e "${GREEN}📥 Installing certbot...${NC}"
    
    if command -v apt-get &> /dev/null; then
        install_packages "certbot python3-certbot-nginx"
    elif command -v dnf &> /dev/null; then
        install_packages "certbot python3-certbot-nginx"
    elif command -v yum &> /dev/null; then
        # Enable EPEL repository for CentOS/RHEL
        sudo yum install -y epel-release
        install_packages "certbot python3-certbot-nginx"
    elif command -v pacman &> /dev/null; then
        install_packages "certbot certbot-nginx"
    elif command -v zypper &> /dev/null; then
        install_packages "certbot python3-certbot-nginx"
    else
        echo -e "${RED}Cannot install certbot automatically. Please install manually.${NC}"
        exit 1
    fi
    
    echo -e "${GREEN}✅ Certbot installed${NC}"
}

# Function to create nginx configuration (before SSL setup)
create_nginx_config() {
    echo -e "${GREEN}📝 Creating nginx configuration...${NC}"
    
    local config_file="/etc/nginx/sites-available/cryospy"
    local sites_available_dir="/etc/nginx/sites-available"
    local sites_enabled_dir="/etc/nginx/sites-enabled"
    
    # Create sites-available and sites-enabled directories if they don't exist (for non-Debian systems)
    sudo mkdir -p "$sites_available_dir" "$sites_enabled_dir"
    
    # Generate proxy authentication logic if configured
    local proxy_auth_config=""
    if [[ -n "$PROXY_AUTH_HEADER" && -n "$PROXY_AUTH_VALUE" ]]; then
        # Convert header name to nginx variable format (lowercase, hyphens to underscores)
        local nginx_header_var="\$http_$(echo "$PROXY_AUTH_HEADER" | tr '[:upper:]' '[:lower:]' | tr '-' '_')"
        proxy_auth_config="        # Require proxy authentication header
        if ($nginx_header_var != \"$PROXY_AUTH_VALUE\") {
            return 401 \"Unauthorized - Invalid or missing proxy authentication\";
        }"
    fi
    
    # Create nginx configuration (certbot will add SSL configuration automatically)
    sudo tee "$config_file" > /dev/null <<EOF
# CryoSpy Nginx Configuration
# This configuration exposes ONLY the capture-server API to the internet
# The dashboard remains accessible only locally
# SSL configuration will be added automatically by certbot

# Rate limiting
limit_req_zone \$binary_remote_addr zone=api:10m rate=10r/s;
limit_req_zone \$binary_remote_addr zone=upload:10m rate=2r/s;

# Upstream for capture-server
upstream capture_server {
    server 127.0.0.1:$CAPTURE_SERVER_PORT;
}

# HTTP server (certbot will modify this to add SSL and redirects)
server {
    listen $HTTP_PORT;
    server_name $DOMAIN;
    
    # Security headers
    add_header X-Frame-Options DENY;
    add_header X-Content-Type-Options nosniff;
    add_header X-XSS-Protection "1; mode=block";
    
    # Security: Only allow capture-server API endpoints
    # Block access to dashboard and other services
    
    # Health check endpoint
    location = /health {
        limit_req zone=api burst=5 nodelay;
$proxy_auth_config
        
        proxy_pass http://capture_server;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
    }
    
    # API endpoints for clip upload (requires authentication)
    location /api/ {
        # Apply rate limiting for uploads
        limit_req zone=upload burst=10 nodelay;
$proxy_auth_config
        
        # Increase timeout and body size for video uploads
        client_max_body_size 100M;
        proxy_read_timeout 300s;
        proxy_connect_timeout 10s;
        proxy_send_timeout 300s;
        
        proxy_pass http://capture_server;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
        
        # Security headers for API
        add_header X-Robots-Tag "noindex, nofollow";
    }
    
    # Block all other requests (including dashboard)
    location / {
        return 403 "Access denied.";
        add_header Content-Type text/plain;
    }
    
    # Block common attack vectors
    location ~ /\. {
        return 403;
    }
    
    location ~ /admin {
        return 403;
    }
    
    location ~ /dashboard {
        return 403;
    }
}
EOF
    
    # Enable the site
    sudo ln -sf "$config_file" "$sites_enabled_dir/cryospy"
    
    # Remove default nginx site if it exists
    sudo rm -f "$sites_enabled_dir/default"
    
    # Test nginx configuration
    if sudo nginx -t; then
        echo -e "${GREEN}✅ Nginx configuration created and validated${NC}"
    else
        echo -e "${RED}❌ Nginx configuration validation failed${NC}"
        exit 1
    fi
    
    # Reload nginx
    sudo systemctl reload nginx
}

# Function to setup SSL with certbot
setup_ssl() {
    if [[ "$SKIP_CERTBOT" == true ]]; then
        echo -e "${YELLOW}Skipping SSL certificate setup${NC}"
        return
    fi
    
    if [[ -z "$EMAIL" ]]; then
        echo -e "${RED}Email is required for Let's Encrypt SSL certificate setup${NC}"
        echo "Please provide an email address with --email parameter"
        exit 1
    fi
    
    echo -e "${GREEN}🔒 Setting up SSL certificate with Let's Encrypt...${NC}"
    
    # Run certbot with nginx plugin (automatically modifies nginx config)
    if sudo certbot --nginx -d "$DOMAIN" --email "$EMAIL" --agree-tos --non-interactive --redirect; then
        echo -e "${GREEN}✅ SSL certificate obtained and nginx configured automatically${NC}"
        
        # Set up automatic renewal (certbot usually sets this up automatically, but let's ensure it)
        if ! sudo crontab -l 2>/dev/null | grep -q "certbot renew"; then
            # Add renewal command
            (sudo crontab -l 2>/dev/null; echo "0 12 * * * /usr/bin/certbot renew --quiet") | sudo crontab -
            echo -e "${GREEN}✅ Auto-renewal configured${NC}"
        fi
        
        # Add HSTS header manually since certbot doesn't add it
        local config_file="/etc/nginx/sites-available/cryospy"
        if ! sudo grep -q "Strict-Transport-Security" "$config_file"; then
            # Add HSTS header to the HTTPS server block
            sudo sed -i '/listen.*ssl/a\    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;' "$config_file"
            sudo systemctl reload nginx
            echo -e "${GREEN}✅ HSTS header added for enhanced security${NC}"
        fi
    else
        echo -e "${RED}❌ SSL certificate setup failed${NC}"
        echo -e "${YELLOW}Please check that:${NC}"
        echo "  1. Your domain $DOMAIN points to this server's public IP"
        echo "  2. Port 80 is open in your firewall (configured automatically for certbot)"
        echo "  3. The domain is accessible from the internet"
        echo "  4. No other process is blocking port 80"
        echo "  5. Nginx is running and configuration is valid"
        exit 1
    fi
}

# Function to configure firewall for certbot (HTTP only)
configure_firewall_for_certbot() {
    echo -e "${GREEN}🔥 Configuring firewall for certbot (opening port 80)...${NC}"
    
    if command -v ufw &> /dev/null; then
        echo -e "${BLUE}Configuring UFW firewall...${NC}"
        
        # Enable UFW if not already enabled
        sudo ufw --force enable
        
        # Allow SSH (important!)
        sudo ufw allow ssh
        
        # Allow HTTP for certbot challenge
        sudo ufw allow $HTTP_PORT/tcp
        
        echo -e "${GREEN}✅ UFW firewall configured for certbot${NC}"
        sudo ufw status
    elif command -v firewall-cmd &> /dev/null; then
        echo -e "${BLUE}Configuring firewalld...${NC}"
        
        # Allow HTTP for certbot challenge
        sudo firewall-cmd --permanent --add-port=$HTTP_PORT/tcp
        sudo firewall-cmd --reload
        
        echo -e "${GREEN}✅ Firewalld configured for certbot${NC}"
    else
        echo -e "${YELLOW}No recognized firewall found. Please manually open port $HTTP_PORT (HTTP) for certbot${NC}"
    fi
}

# Function to configure firewall for HTTPS service
configure_firewall_for_https() {
    echo -e "${GREEN}🔥 Configuring firewall for HTTPS service (opening port $HTTPS_PORT)...${NC}"
    
    if command -v ufw &> /dev/null; then
        echo -e "${BLUE}Adding HTTPS port to UFW firewall...${NC}"
        
        # Allow HTTPS
        sudo ufw allow $HTTPS_PORT/tcp
        
        echo -e "${GREEN}✅ UFW firewall configured for HTTPS${NC}"
        sudo ufw status
    elif command -v firewall-cmd &> /dev/null; then
        echo -e "${BLUE}Adding HTTPS port to firewalld...${NC}"
        
        # Allow HTTPS
        sudo firewall-cmd --permanent --add-port=$HTTPS_PORT/tcp
        sudo firewall-cmd --reload
        
        echo -e "${GREEN}✅ Firewalld configured for HTTPS${NC}"
    else
        echo -e "${YELLOW}No recognized firewall found. Please manually open port $HTTPS_PORT (HTTPS)${NC}"
    fi
}

# Function to validate inputs
validate_inputs() {
    if [[ -z "$DOMAIN" ]]; then
        echo -e "${RED}Domain is required. Use --domain parameter.${NC}"
        echo "Use --help for usage information."
        exit 1
    fi
    
    # Validate domain format
    if [[ ! "$DOMAIN" =~ ^([a-zA-Z0-9]+(-[a-zA-Z0-9]+)*\.)+[a-zA-Z]{2,}$ ]]; then
        echo -e "${RED}Invalid domain format: $DOMAIN${NC}"
        exit 1
    fi
    
    # Validate ports
    if [[ ! "$CAPTURE_SERVER_PORT" =~ ^[0-9]+$ ]] || [[ "$CAPTURE_SERVER_PORT" -lt 1 ]] || [[ "$CAPTURE_SERVER_PORT" -gt 65535 ]]; then
        echo -e "${RED}Invalid capture server port: $CAPTURE_SERVER_PORT${NC}"
        exit 1
    fi
    
    # Check if proxy auth is properly configured
    if [[ -n "$PROXY_AUTH_HEADER" && -z "$PROXY_AUTH_VALUE" ]] || [[ -z "$PROXY_AUTH_HEADER" && -n "$PROXY_AUTH_VALUE" ]]; then
        echo -e "${RED}Both --proxy-auth-header and --proxy-auth-value must be provided together${NC}"
        exit 1
    fi
}

# Main execution
main() {
    echo -e "${CYAN}🔒 CryoSpy Nginx Reverse Proxy Setup for Linux${NC}"
    echo -e "${CYAN}===============================================${NC}"
    echo ""
    
    # Parse command line arguments
    parse_arguments "$@"
    
    # Validate inputs
    validate_inputs
    
    # Check permissions
    check_root
    
    # Display configuration
    echo -e "${YELLOW}Configuration:${NC}"
    echo "  Domain: $DOMAIN"
    echo "  Capture Server Port: $CAPTURE_SERVER_PORT"
    echo "  HTTPS Port: $HTTPS_PORT"
    echo "  HTTP Port: $HTTP_PORT"
    if [[ -n "$PROXY_AUTH_HEADER" && -n "$PROXY_AUTH_VALUE" ]]; then
        echo "  Proxy Auth: $PROXY_AUTH_HEADER = ***configured***"
    else
        echo "  Proxy Auth: not configured (endpoints will rely on app-level auth only)"
    fi
    if [[ -n "$EMAIL" ]]; then
        echo "  Email (for SSL): $EMAIL"
    fi
    echo ""
    
    if [[ "$FORCE" != true ]]; then
        read -p "Continue with installation? (y/N): " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            echo "Installation cancelled."
            exit 0
        fi
    fi
    
    # Install components
    install_nginx
    install_certbot
    
    # Configure firewall for certbot (port 80 only)
    configure_firewall_for_certbot
    
    # Create nginx configuration
    create_nginx_config
    
    # Setup SSL certificates (certbot will automatically modify nginx config)
    setup_ssl
    
    # Configure firewall for HTTPS service (port 443)
    configure_firewall_for_https
    
    # Final message
    echo ""
    echo -e "${GREEN}🎉 Setup Complete!${NC}"
    echo -e "${GREEN}==================${NC}"
    echo ""
    echo -e "${YELLOW}Configuration Summary:${NC}"
    echo "✅ Nginx installed and configured"
    echo "✅ SSL certificate obtained and configured"
    echo "✅ HTTPS-only configuration activated"
    echo "✅ Automatic certificate renewal configured"
    echo ""
    echo -e "${YELLOW}Next steps:${NC}"
    echo "1. Ensure your CryoSpy capture-server is running on port $CAPTURE_SERVER_PORT"
    echo "2. Your capture-server API is now available at:"
    echo -e "   ${CYAN}https://$DOMAIN/api/${NC}"
    echo -e "   ${CYAN}https://$DOMAIN/health${NC}"
    
    if [[ -n "$PROXY_AUTH_HEADER" && -n "$PROXY_AUTH_VALUE" ]]; then
        echo ""
        echo -e "${YELLOW}All endpoints require proxy authentication header:${NC}"
        echo -e "   ${CYAN}$PROXY_AUTH_HEADER: $PROXY_AUTH_VALUE${NC}"
    fi
    
    echo ""
    echo -e "${YELLOW}Security notes:${NC}"
    echo "• Dashboard is NOT exposed (remains local-only)"
    echo "• Only /api/ and /health endpoints are accessible"
    echo "• Rate limiting is configured for API endpoints"
    echo "• HTTPS is enforced (HTTP automatically redirects to HTTPS)"
    echo "• SSL certificate will auto-renew via certbot"
    
    echo ""
    echo -e "${YELLOW}To manage nginx:${NC}"
    echo -e "   ${CYAN}sudo systemctl start nginx${NC}"
    echo -e "   ${CYAN}sudo systemctl stop nginx${NC}"
    echo -e "   ${CYAN}sudo systemctl restart nginx${NC}"
    echo -e "   ${CYAN}sudo systemctl status nginx${NC}"
    
    echo ""
    echo -e "${YELLOW}To check SSL certificate:${NC}"
    echo -e "   ${CYAN}sudo certbot certificates${NC}"
    echo -e "   ${CYAN}sudo certbot renew --dry-run${NC}"
}

# Run main function with all arguments
main "$@"