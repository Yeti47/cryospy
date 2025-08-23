# setup-nginx-proxy.ps1
# PowerShell script to set up nginx as a reverse proxy for CryoSpy capture-server
# This exposes ONLY the capture-server to the internet via HTTPS, keeping the dashboard local-only

param(
    [Parameter(Mandatory=$true)]
    [string]$Domain,
    
    [Parameter(Mandatory=$false)]
    [int]$CaptureServerPort = 8081,
    
    [Parameter(Mandatory=$false)]
    [int]$HttpsPort = 443,
    
    [Parameter(Mandatory=$false)]
    [int]$HttpPort = 80,
    
    [Parameter(Mandatory=$false)]
    [string]$NginxPath = "C:\nginx",
    
    [Parameter(Mandatory=$false)]
    [string]$CertPath = "C:\ssl\certs\$Domain",
    
    [Parameter(Mandatory=$false)]
    [string]$ProxyAuthHeader = "",
    
    [Parameter(Mandatory=$false)]
    [string]$ProxyAuthValue = "",
    
    [Parameter(Mandatory=$false)]
    [switch]$SkipNginxInstall,
    
    [Parameter(Mandatory=$false)]
    [switch]$Force
)

# Ensure running as Administrator
if (-NOT ([Security.Principal.WindowsPrincipal] [Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole] "Administrator")) {
    Write-Error "This script must be run as Administrator. Please restart PowerShell as Administrator."
    exit 1
}

Write-Host "🔒 CryoSpy Nginx Reverse Proxy Setup" -ForegroundColor Cyan
Write-Host "====================================" -ForegroundColor Cyan
Write-Host ""
Write-Host "Domain: $Domain" -ForegroundColor Yellow
Write-Host "Capture Server Port: $CaptureServerPort" -ForegroundColor Yellow
Write-Host "HTTPS Port: $HttpsPort" -ForegroundColor Yellow
Write-Host "HTTP Port: $HttpPort (redirect to HTTPS)" -ForegroundColor Yellow
Write-Host "Certificate Path: $CertPath" -ForegroundColor Yellow
if ($ProxyAuthHeader -and $ProxyAuthValue) {
    Write-Host "Proxy Auth: $ProxyAuthHeader = ***configured***" -ForegroundColor Yellow
} else {
    Write-Host "Proxy Auth: not configured (endpoints will rely on app-level auth only)" -ForegroundColor Red
}
Write-Host ""

# Function to download and install nginx
function Install-Nginx {
    param([string]$InstallPath)
    
    Write-Host "📥 Installing nginx..." -ForegroundColor Green
    
    if (Test-Path $InstallPath) {
        if (-not $Force) {
            $choice = Read-Host "Nginx directory already exists at $InstallPath. Overwrite? (y/N)"
            if ($choice -ne 'y' -and $choice -ne 'Y') {
                Write-Host "Skipping nginx installation." -ForegroundColor Yellow
                return
            }
        }
        Remove-Item -Path $InstallPath -Recurse -Force
    }
    
    # Download nginx for Windows
    $nginxVersion = "1.24.0"
    $downloadUrl = "http://nginx.org/download/nginx-$nginxVersion.zip"
    $zipPath = "$env:TEMP\nginx-$nginxVersion.zip"
    
    Write-Host "Downloading nginx $nginxVersion..." -ForegroundColor Gray
    try {
        Invoke-WebRequest -Uri $downloadUrl -OutFile $zipPath -UseBasicParsing
    } catch {
        Write-Error "Failed to download nginx: $($_.Exception.Message)"
        exit 1
    }
    
    # Extract nginx
    Write-Host "Extracting nginx..." -ForegroundColor Gray
    try {
        Add-Type -AssemblyName System.IO.Compression.FileSystem
        [System.IO.Compression.ZipFile]::ExtractToDirectory($zipPath, "C:\")
        
        # Rename the extracted folder
        $extractedPath = "C:\nginx-$nginxVersion"
        if (Test-Path $extractedPath) {
            Move-Item -Path $extractedPath -Destination $InstallPath
        }
        
        Remove-Item -Path $zipPath -Force
    } catch {
        Write-Error "Failed to extract nginx: $($_.Exception.Message)"
        exit 1
    }
    
    Write-Host "✅ Nginx installed to $InstallPath" -ForegroundColor Green
}

# Function to create nginx configuration
function Create-NginxConfig {
    param(
        [string]$NginxPath,
        [string]$Domain,
        [int]$CaptureServerPort,
        [int]$HttpsPort,
        [int]$HttpPort,
        [string]$CertPath,
        [string]$ProxyAuthHeader,
        [string]$ProxyAuthValue,
        [string]$HealthApiKey
    )
    
    Write-Host "📝 Creating nginx configuration..." -ForegroundColor Green
    
    $confPath = "$NginxPath\conf\nginx.conf"
    
    # Generate proxy authentication logic if configured
    $proxyAuthConfig = ""
    if ($ProxyAuthHeader -and $ProxyAuthValue) {
        # Convert header name to nginx variable format (lowercase, hyphens to underscores)
        $nginxHeaderVar = "`$http_" + $ProxyAuthHeader.ToLower().Replace('-', '_')
        $proxyAuthConfig = @"
            # Require proxy authentication header
            if ($nginxHeaderVar != "$ProxyAuthValue") {
                return 401 "Unauthorized - Invalid or missing proxy authentication";
            }
"@
    }
    
    }
    
    $nginxConfig = @"
# CryoSpy Nginx Configuration
# This configuration exposes ONLY the capture-server API to the internet
# The dashboard remains accessible only locally

worker_processes auto;

events {
    worker_connections 1024;
}

http {
    include       mime.types;
    default_type  application/octet-stream;
    
    # Logging
    access_log logs/access.log;
    error_log logs/error.log;
    
    # Basic settings
    sendfile on;
    tcp_nopush on;
    tcp_nodelay on;
    keepalive_timeout 65;
    types_hash_max_size 2048;
    
    # Security headers
    add_header X-Frame-Options DENY;
    add_header X-Content-Type-Options nosniff;
    add_header X-XSS-Protection "1; mode=block";
    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;
    
    # Rate limiting
    limit_req_zone `$binary_remote_addr zone=api:10m rate=10r/s;
    limit_req_zone `$binary_remote_addr zone=upload:10m rate=2r/s;
    
    # Upstream for capture-server
    upstream capture_server {
        server 127.0.0.1:$CaptureServerPort;
    }
    
    # HTTP to HTTPS redirect
    server {
        listen $HttpPort;
        server_name $Domain;
        
        # Redirect all HTTP traffic to HTTPS
        return 301 https://`$server_name`$request_uri;
    }
    
    # HTTPS server for capture-server API only
    server {
        listen $HttpsPort ssl http2;
        server_name $Domain;
        
        # SSL Configuration
        # Update these paths to match your SSL certificate location
        ssl_certificate "$CertPath\fullchain.pem";
        ssl_certificate_key "$CertPath\privkey.pem";
        
        # Modern SSL configuration
        ssl_protocols TLSv1.2 TLSv1.3;
        ssl_ciphers ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256:ECDHE-ECDSA-AES256-GCM-SHA384:ECDHE-RSA-AES256-GCM-SHA384;
        ssl_prefer_server_ciphers off;
        ssl_session_cache shared:SSL:10m;
        ssl_session_timeout 10m;
        
        # OCSP stapling
        ssl_stapling on;
        ssl_stapling_verify on;
        
        # Security: Only allow capture-server API endpoints
        # Block access to dashboard and other services
        
        # Health check endpoint
        location = /health {
            limit_req zone=api burst=5 nodelay;
            $proxyAuthConfig
            proxy_pass http://capture_server;
            proxy_set_header Host `$host;
            proxy_set_header X-Real-IP `$remote_addr;
            proxy_set_header X-Forwarded-For `$proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto `$scheme;
        }
        
        # API endpoints for clip upload (requires authentication)
        location /api/ {
            # Apply rate limiting for uploads
            limit_req zone=upload burst=10 nodelay;
            $proxyAuthConfig
            
            # Increase timeout and body size for video uploads
            client_max_body_size 100M;
            proxy_read_timeout 300s;
            proxy_connect_timeout 10s;
            proxy_send_timeout 300s;
            
            proxy_pass http://capture_server;
            proxy_set_header Host `$host;
            proxy_set_header X-Real-IP `$remote_addr;
            proxy_set_header X-Forwarded-For `$proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto `$scheme;
            
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
}
"@

    try {
        $nginxConfig | Out-File -FilePath $confPath -Encoding UTF8 -Force
        Write-Host "✅ Nginx configuration created at $confPath" -ForegroundColor Green
    } catch {
        Write-Error "Failed to create nginx configuration: $($_.Exception.Message)"
        exit 1
    }
}

# Function to install nginx as Windows service
function Install-NginxService {
    param([string]$NginxPath)
    
    Write-Host "🔧 Installing nginx as Windows service..." -ForegroundColor Green
    
    # Check if nssm (Non-Sucking Service Manager) is available
    $nssmPath = "$NginxPath\nssm.exe"
    
    if (-not (Test-Path $nssmPath)) {
        Write-Host "Downloading NSSM (service manager)..." -ForegroundColor Gray
        try {
            $nssmUrl = "https://nssm.cc/release/nssm-2.24.zip"
            $nssmZip = "$env:TEMP\nssm.zip"
            Invoke-WebRequest -Uri $nssmUrl -OutFile $nssmZip -UseBasicParsing
            
            Add-Type -AssemblyName System.IO.Compression.FileSystem
            [System.IO.Compression.ZipFile]::ExtractToDirectory($nssmZip, $env:TEMP)
            
            Copy-Item "$env:TEMP\nssm-2.24\win64\nssm.exe" $nssmPath
            Remove-Item $nssmZip -Force
            Remove-Item "$env:TEMP\nssm-2.24" -Recurse -Force
        } catch {
            Write-Error "Failed to download NSSM: $($_.Exception.Message)"
            exit 1
        }
    }
    
    # Remove existing service if it exists
    $existingService = Get-Service -Name "CryoSpyNginx" -ErrorAction SilentlyContinue
    if ($existingService) {
        Write-Host "Removing existing nginx service..." -ForegroundColor Yellow
        & $nssmPath stop CryoSpyNginx
        & $nssmPath remove CryoSpyNginx confirm
    }
    
    # Install new service
    $nginxExe = "$NginxPath\nginx.exe"
    & $nssmPath install CryoSpyNginx $nginxExe
    & $nssmPath set CryoSpyNginx AppDirectory $NginxPath
    & $nssmPath set CryoSpyNginx DisplayName "CryoSpy Nginx Reverse Proxy"
    & $nssmPath set CryoSpyNginx Description "Nginx reverse proxy for CryoSpy capture-server"
    & $nssmPath set CryoSpyNginx Start SERVICE_AUTO_START
    
    Write-Host "✅ Nginx service installed" -ForegroundColor Green
}

# Function to configure Windows Firewall
function Configure-Firewall {
    param([int]$HttpPort, [int]$HttpsPort)
    
    Write-Host "🔥 Configuring Windows Firewall..." -ForegroundColor Green
    
    try {
        # Remove existing rules if they exist
        Remove-NetFirewallRule -DisplayName "CryoSpy Nginx HTTP" -ErrorAction SilentlyContinue
        Remove-NetFirewallRule -DisplayName "CryoSpy Nginx HTTPS" -ErrorAction SilentlyContinue
        
        # Create new firewall rules
        New-NetFirewallRule -DisplayName "CryoSpy Nginx HTTP" -Direction Inbound -Protocol TCP -LocalPort $HttpPort -Action Allow
        New-NetFirewallRule -DisplayName "CryoSpy Nginx HTTPS" -Direction Inbound -Protocol TCP -LocalPort $HttpsPort -Action Allow
        
        Write-Host "✅ Firewall rules configured for ports $HttpPort and $HttpsPort" -ForegroundColor Green
    } catch {
        Write-Warning "Failed to configure firewall rules: $($_.Exception.Message)"
        Write-Host "Please manually configure firewall rules for ports $HttpPort and $HttpsPort" -ForegroundColor Yellow
    }
}

# Function to check certificate path
function Test-CertificatePath {
    param([string]$CertPath)
    
    $fullchainPath = "$CertPath\fullchain.pem"
    $privkeyPath = "$CertPath\privkey.pem"
    
    Write-Host "🔍 Checking SSL certificates..." -ForegroundColor Green
    
    if (-not (Test-Path $fullchainPath)) {
        Write-Warning "Certificate not found at $fullchainPath"
        Write-Host "Please ensure you have SSL certificates at $CertPath" -ForegroundColor Yellow
        return $false
    }
    
    if (-not (Test-Path $privkeyPath)) {
        Write-Warning "Private key not found at $privkeyPath"
        return $false
    }
    
    Write-Host "✅ SSL certificates found" -ForegroundColor Green
    return $true
}

# Main execution
try {
    # Install nginx if not skipped
    if (-not $SkipNginxInstall) {
        Install-Nginx -InstallPath $NginxPath
    } elseif (-not (Test-Path $NginxPath)) {
        Write-Error "Nginx not found at $NginxPath and installation was skipped"
        exit 1
    }
    
    # Check certificate path
    if (-not (Test-CertificatePath -CertPath $CertPath)) {
        Write-Host ""
        Write-Host "⚠️  SSL certificates not found!" -ForegroundColor Yellow
        Write-Host "Please place your SSL certificates at: $CertPath" -ForegroundColor Yellow
        Write-Host "Required files:" -ForegroundColor Yellow
        Write-Host "  - fullchain.pem (certificate + intermediate chain)" -ForegroundColor White
        Write-Host "  - privkey.pem (private key)" -ForegroundColor White
        Write-Host ""
        Write-Host "You can use tools like Certify the Web, Let's Encrypt, or any other" -ForegroundColor Yellow
        Write-Host "SSL certificate provider to obtain these certificates." -ForegroundColor Yellow
        
        $continue = Read-Host "Continue anyway? The nginx config will be created but may not work until certificates are available (y/N)"
        if ($continue -ne 'y' -and $continue -ne 'Y') {
            exit 1
        }
    }
    
    # Create nginx configuration
    Create-NginxConfig -NginxPath $NginxPath -Domain $Domain -CaptureServerPort $CaptureServerPort -HttpsPort $HttpsPort -HttpPort $HttpPort -CertPath $CertPath -ProxyAuthHeader $ProxyAuthHeader -ProxyAuthValue $ProxyAuthValue
    
    # Install as Windows service
    Install-NginxService -NginxPath $NginxPath
    
    # Configure firewall
    Configure-Firewall -HttpPort $HttpPort -HttpsPort $HttpsPort
    
    Write-Host ""
    Write-Host "🎉 Setup Complete!" -ForegroundColor Green
    Write-Host "=================" -ForegroundColor Green
    Write-Host ""
    Write-Host "Next steps:" -ForegroundColor Yellow
    Write-Host "1. Ensure your CryoSpy capture-server is running on port $CaptureServerPort" -ForegroundColor White
    Write-Host "2. Configure DNS: Point $Domain to this server's public IP" -ForegroundColor White
    Write-Host "3. Obtain and install SSL certificates:" -ForegroundColor White
    Write-Host "   - Place fullchain.pem and privkey.pem in: $CertPath" -ForegroundColor Cyan
    Write-Host "   - You can use Certify the Web, Let's Encrypt, or any SSL provider" -ForegroundColor Cyan
    Write-Host "4. Start the nginx service:" -ForegroundColor White
    Write-Host "   Start-Service CryoSpyNginx" -ForegroundColor Cyan
    Write-Host ""
    Write-Host "Your capture-server API will be available at:" -ForegroundColor Yellow
    Write-Host "  https://$Domain/api/" -ForegroundColor Cyan
    if ($ProxyAuthHeader -and $ProxyAuthValue) {
        Write-Host "  https://$Domain/health (requires $ProxyAuthHeader header)" -ForegroundColor Cyan
        Write-Host ""
        Write-Host "All endpoints require proxy authentication header:" -ForegroundColor Yellow
        Write-Host "  $ProxyAuthHeader : $ProxyAuthValue" -ForegroundColor Cyan
    } else {
        Write-Host "  https://$Domain/health" -ForegroundColor Cyan
    }
    Write-Host ""
    Write-Host "Security notes:" -ForegroundColor Yellow
    Write-Host "• Dashboard is NOT exposed (remains local-only)" -ForegroundColor White
    Write-Host "• Only /api/ and /health endpoints are accessible" -ForegroundColor White
    Write-Host "• Rate limiting is configured for API endpoints" -ForegroundColor White
    Write-Host "• HTTPS is enforced (HTTP redirects to HTTPS)" -ForegroundColor White
    Write-Host ""
    Write-Host "To manage the service:" -ForegroundColor Yellow
    Write-Host "  Start-Service CryoSpyNginx" -ForegroundColor Cyan
    Write-Host "  Stop-Service CryoSpyNginx" -ForegroundColor Cyan
    Write-Host "  Restart-Service CryoSpyNginx" -ForegroundColor Cyan
    
} catch {
    Write-Error "Setup failed: $($_.Exception.Message)"
    Write-Host "Stack trace: $($_.ScriptStackTrace)" -ForegroundColor Red
    exit 1
}
