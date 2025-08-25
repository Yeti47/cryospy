# install-client-windows.ps1
# PowerShell script to install CryoSpy Capture Client on Windows

param(
    [Parameter(Mandatory=$false)]
    [switch]$SkipDependencies,
    
    [Parameter(Mandatory=$false)]
    [switch]$InstallAsService,
    
    [Parameter(Mandatory=$false)]
    [switch]$SetupFirewall,
    
    [Parameter(Mandatory=$false)]
    [string]$ServerUrl = "",
    
    [Parameter(Mandatory=$false)]
    [string]$ClientId = "",
    
    [Parameter(Mandatory=$false)]
    [string]$ClientSecret = "",
    
    [Parameter(Mandatory=$false)]
    [string]$ProxyAuthHeader = "",
    
    [Parameter(Mandatory=$false)]
    [string]$ProxyAuthValue = "",
    
    [Parameter(Mandatory=$false)]
    [switch]$Force
)

# Ensure running as Administrator for service installation
if ($InstallAsService -and -NOT ([Security.Principal.WindowsPrincipal] [Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole] "Administrator")) {
    Write-Host "Service installation requires Administrator privileges." -ForegroundColor Red
    Write-Host "Please run PowerShell as Administrator to install as a service." -ForegroundColor Yellow
    exit 1
}

Write-Host "📹 CryoSpy Capture Client - Windows Installation" -ForegroundColor Cyan
Write-Host "================================================" -ForegroundColor Cyan
Write-Host ""

# Function to check if FFmpeg is available in PATH (optional runtime dependency)
function Test-FFmpegAvailable {
    try {
        $null = Get-Command ffmpeg -ErrorAction Stop
        return $true
    } catch {
        return $false
    }
}

# Function to check if Chocolatey is installed
function Test-ChocolateyInstalled {
    try {
        Get-Command choco -ErrorAction Stop | Out-Null
        return $true
    } catch {
        return $false
    }
}

# Function to install Chocolatey
function Install-Chocolatey {
    Write-Host "📦 Installing Chocolatey package manager..." -ForegroundColor Green
    
    try {
        Set-ExecutionPolicy Bypass -Scope Process -Force
        [System.Net.ServicePointManager]::SecurityProtocol = [System.Net.ServicePointManager]::SecurityProtocol -bor 3072
        Invoke-Expression ((New-Object System.Net.WebClient).DownloadString('https://community.chocolatey.org/install.ps1'))
        
        # Refresh PATH
        $env:PATH = [System.Environment]::GetEnvironmentVariable("PATH", "Machine") + ";" + [System.Environment]::GetEnvironmentVariable("PATH", "User")
        
        Write-Host "✅ Chocolatey installed successfully" -ForegroundColor Green
        return $true
    } catch {
        Write-Error "Failed to install Chocolatey: $($_.Exception.Message)"
        return $false
    }
}

# Function to install optional runtime dependencies
function Install-OptionalDependencies {
    Write-Host "📥 Installing optional runtime dependencies..." -ForegroundColor Green
    
    $needsFFmpeg = -not (Test-FFmpegAvailable)
    
    if (-not $needsFFmpeg) {
        Write-Host "ℹ️  All optional dependencies are already available" -ForegroundColor Cyan
        return $true
    }
    
    # Check if Chocolatey is available for FFmpeg installation
    if (-not (Test-ChocolateyInstalled)) {
        Write-Host "ℹ️  Installing Chocolatey to install FFmpeg (optional runtime dependency)" -ForegroundColor Cyan
        if (-not (Install-Chocolatey)) {
            Write-Warning "Could not install Chocolatey. FFmpeg installation skipped."
            Write-Host "ℹ️  FFmpeg is optional and only needed for certain video processing features" -ForegroundColor Yellow
            return $true
        }
    }
    
    # Install FFmpeg via Chocolatey (optional runtime dependency)
    if ($needsFFmpeg) {
        Write-Host "  • Installing FFmpeg (optional runtime dependency)" -ForegroundColor White
        try {
            choco install ffmpeg -y
            if ($LASTEXITCODE -ne 0) {
                throw "Chocolatey FFmpeg installation failed"
            }
            Write-Host "    ✅ FFmpeg installed successfully" -ForegroundColor Green
        } catch {
            Write-Warning "Failed to install FFmpeg: $($_.Exception.Message)"
            Write-Host "ℹ️  FFmpeg is optional and only needed for certain video processing features" -ForegroundColor Yellow
        }
    }
    
    Write-Host "✅ Optional dependency installation completed" -ForegroundColor Green
    return $true
}

# Function to show manual installation instructions for optional dependencies
function Show-ManualInstallInstructions {
    Write-Host ""
    Write-Host "📋 Optional Runtime Dependencies" -ForegroundColor Yellow
    Write-Host "================================" -ForegroundColor Yellow
    Write-Host ""
    Write-Host "The following dependencies are optional for enhanced functionality:" -ForegroundColor White
    Write-Host ""
    Write-Host "FFmpeg (optional - for advanced video processing):" -ForegroundColor White
    Write-Host "   - Download from: https://ffmpeg.org/download.html#build-windows" -ForegroundColor Cyan
    Write-Host "   - Extract to C:\\ffmpeg" -ForegroundColor Cyan
    Write-Host "   - Add C:\\ffmpeg\\bin to your PATH" -ForegroundColor Cyan
    Write-Host ""
    Write-Host "Alternatively, use Chocolatey:" -ForegroundColor White
    Write-Host "   choco install ffmpeg" -ForegroundColor Cyan
    Write-Host ""
    Write-Host "NOTE: All OpenCV dependencies are bundled with the release package." -ForegroundColor Green
    Write-Host "No additional installations are required for basic functionality." -ForegroundColor Green
    Write-Host ""
    Read-Host "Press Enter to continue..."
}

# Function to create configuration file
function Create-ConfigFile {
    param(
        [string]$ServerUrl,
        [string]$ClientId,
        [string]$ClientSecret,
        [string]$ProxyAuthHeader,
        [string]$ProxyAuthValue
    )
    
    Write-Host "📝 Creating client configuration..." -ForegroundColor Green
    
    # Check if config.json already exists
    if (Test-Path "config.json" -and -not $Force) {
        $overwrite = Read-Host "config.json already exists. Overwrite? (y/N)"
        if ($overwrite -ne 'y' -and $overwrite -ne 'Y') {
            Write-Host "⏭️  Skipping configuration file creation" -ForegroundColor Yellow
            return
        }
    }
    
    # Prompt for missing values
    if (-not $ServerUrl) {
        $ServerUrl = Read-Host "Enter server URL (e.g., http://your-server-ip:8081 or https://yourdomain.com)"
        if (-not $ServerUrl) {
            $ServerUrl = "http://localhost:8081"
        }
    }
    
    if (-not $ClientId) {
        $ClientId = Read-Host "Enter client ID (obtain from server admin)"
        if (-not $ClientId) {
            $ClientId = "your-client-id"
        }
    }
    
    if (-not $ClientSecret) {
        $ClientSecret = Read-Host "Enter client secret (obtain from server admin)"
        if (-not $ClientSecret) {
            $ClientSecret = "your-client-secret"
        }
    }
    
    # Ask about proxy authentication if not provided
    if (-not $ProxyAuthHeader -and -not $ProxyAuthValue) {
        $useProxyAuth = Read-Host "Configure proxy authentication for reverse proxy? (y/N)"
        if ($useProxyAuth -eq 'y' -or $useProxyAuth -eq 'Y') {
            $ProxyAuthHeader = Read-Host "Enter proxy auth header name (e.g., X-Proxy-Auth)"
            $ProxyAuthValue = Read-Host "Enter proxy auth header value"
        }
    }
    
    # Create configuration object
    $config = @{
        client_id = $ClientId
        client_secret = $ClientSecret
        server_url = $ServerUrl
        camera_device = "/dev/video0"
        buffer_size = 5
        settings_sync_seconds = 300
        server_timeout_seconds = 30
    }
    
    # Add proxy auth fields if provided
    if ($ProxyAuthHeader -and $ProxyAuthValue) {
        $config.proxy_auth_header = $ProxyAuthHeader
        $config.proxy_auth_value = $ProxyAuthValue
    } else {
        $config.proxy_auth_header = ""
        $config.proxy_auth_value = ""
    }
    
    try {
        $config | ConvertTo-Json -Depth 10 | Out-File -FilePath "config.json" -Encoding UTF8
        Write-Host "✅ Configuration file created: config.json" -ForegroundColor Green
        
        if ($ProxyAuthHeader -and $ProxyAuthValue) {
            Write-Host "🔐 Proxy authentication configured for defense-in-depth security" -ForegroundColor Green
        }
        
        return $true
    } catch {
        Write-Error "Failed to create configuration file: $($_.Exception.Message)"
        return $false
    }
}

# Function to create startup scripts
function Create-StartupScripts {
    Write-Host "📝 Creating client startup scripts..." -ForegroundColor Green
    
    try {
        # Create start-client.bat
        $startScript = @"
@echo off
echo Starting CryoSpy Capture Client...
echo.
echo Make sure you have configured config.json with:
echo   - server_url (pointing to your CryoSpy server)
echo   - client_id and client_secret
echo   - camera_device (if different from default)
echo.
capture-client.exe
pause
"@
        $startScript | Out-File -FilePath "start-client.bat" -Encoding ASCII
        
        # Create stop-client.bat
        $stopScript = @"
@echo off
echo Stopping CryoSpy capture client...
taskkill /f /im capture-client.exe 2>nul
echo Capture client stopped.
pause
"@
        $stopScript | Out-File -FilePath "stop-client.bat" -Encoding ASCII
        
        Write-Host "✅ Created client startup scripts:" -ForegroundColor Green
        Write-Host "   - start-client.bat    (starts capture client)" -ForegroundColor White
        Write-Host "   - stop-client.bat     (stops capture client)" -ForegroundColor White
        Write-Host ""
        
        return $true
    } catch {
        Write-Error "Failed to create startup scripts: $($_.Exception.Message)"
        return $false
    }
}

# Function to create Windows service scripts
function Create-ServiceScripts {
    Write-Host "🔧 Creating Windows service management scripts..." -ForegroundColor Green
    
    try {
        # Create service installation script
        $installServiceScript = @"
@echo off
echo Installing CryoSpy Client Windows Service...
echo.
sc create "CryoSpy Capture Client" binPath= "%CD%\capture-client.exe" start= auto
echo.
echo Service created. Starting service...
sc start "CryoSpy Capture Client"
echo.
echo Service installed and started successfully!
pause
"@
        $installServiceScript | Out-File -FilePath "install-service.bat" -Encoding ASCII
        
        # Create service removal script
        $uninstallServiceScript = @"
@echo off
echo Removing CryoSpy Client Windows Service...
echo.
sc stop "CryoSpy Capture Client"
sc delete "CryoSpy Capture Client"
echo.
echo Service removed successfully!
pause
"@
        $uninstallServiceScript | Out-File -FilePath "uninstall-service.bat" -Encoding ASCII
        
        Write-Host "✅ Created service management scripts:" -ForegroundColor Green
        Write-Host "   - install-service.bat   (run as Administrator to install service)" -ForegroundColor White
        Write-Host "   - uninstall-service.bat (run as Administrator to remove service)" -ForegroundColor White
        Write-Host ""
        
        return $true
    } catch {
        Write-Error "Failed to create service scripts: $($_.Exception.Message)"
        return $false
    }
}

# Function to create firewall setup script
function Create-FirewallScript {
    Write-Host "🔥 Creating firewall configuration script..." -ForegroundColor Green
    
    try {
        $firewallScript = @"
@echo off
echo Configuring Windows Firewall for CryoSpy Client...
echo.
netsh advfirewall firewall add rule name="CryoSpy Capture Client" dir=in action=allow program="%CD%\capture-client.exe"
echo.
echo Firewall rules added successfully!
pause
"@
        $firewallScript | Out-File -FilePath "setup-firewall.bat" -Encoding ASCII
        
        Write-Host "✅ Created setup-firewall.bat (run as Administrator to configure firewall)" -ForegroundColor Green
        Write-Host ""
        
        return $true
    } catch {
        Write-Error "Failed to create firewall script: $($_.Exception.Message)"
        return $false
    }
}

# Function to configure Windows Firewall directly
function Configure-Firewall {
    Write-Host "🔥 Configuring Windows Firewall..." -ForegroundColor Green
    
    try {
        # Remove existing rules if they exist
        Remove-NetFirewallRule -DisplayName "CryoSpy Capture Client" -ErrorAction SilentlyContinue
        
        # Create new firewall rule
        $clientPath = Join-Path $PWD "capture-client.exe"
        New-NetFirewallRule -DisplayName "CryoSpy Capture Client" -Direction Inbound -Program $clientPath -Action Allow
        
        Write-Host "✅ Firewall rules configured successfully" -ForegroundColor Green
        return $true
    } catch {
        Write-Warning "Failed to configure firewall rules: $($_.Exception.Message)"
        Write-Host "Please run setup-firewall.bat as Administrator to configure firewall manually" -ForegroundColor Yellow
        return $false
    }
}

# Main installation logic
try {
    # Check if this is a bundled release (has all required DLLs)
    $isReleasePackage = Test-Path "capture-client.exe"
    
    if (-not $isReleasePackage) {
        Write-Host "❌ capture-client.exe not found in current directory" -ForegroundColor Red
        Write-Host "Please run this script from the extracted release package directory." -ForegroundColor Yellow
        exit 1
    }
    
    Write-Host "✅ Found CryoSpy Capture Client release package" -ForegroundColor Green
    
    # Check if FFmpeg is available (optional runtime dependency)
    $hasFFmpegInPath = Test-FFmpegAvailable
    
    if ($hasFFmpegInPath) {
        Write-Host "✅ FFmpeg found in PATH (optional runtime dependency satisfied)" -ForegroundColor Green
    } else {
        Write-Host "ℹ️  FFmpeg not found in PATH (optional - only needed for advanced features)" -ForegroundColor Cyan
    }
    
    Write-Host ""
    
    # Install optional dependencies if requested
    if (-not $SkipDependencies -and -not $hasFFmpegInPath) {
        $installOptional = $true
        if (-not $Force) {
            $response = Read-Host "Install optional FFmpeg dependency? (Y/n)"
            $installOptional = ($response -eq '' -or $response -eq 'y' -or $response -eq 'Y')
        }
        
        if ($installOptional) {
            Install-OptionalDependencies
        } else {
            Write-Host "⏭️  Skipping optional dependency installation" -ForegroundColor Yellow
        }
    } elseif ($SkipDependencies) {
        Write-Host "⏭️  Skipping dependency installation (--SkipDependencies specified)" -ForegroundColor Yellow
    }
    
    Write-Host ""
    
    # Create configuration file
    Create-ConfigFile -ServerUrl $ServerUrl -ClientId $ClientId -ClientSecret $ClientSecret -ProxyAuthHeader $ProxyAuthHeader -ProxyAuthValue $ProxyAuthValue | Out-Null
    
    # Create startup scripts
    Create-StartupScripts | Out-Null
    
    # Create service scripts
    Create-ServiceScripts | Out-Null
    
    # Create firewall script
    Create-FirewallScript | Out-Null
    
    # Configure firewall if requested
    if ($SetupFirewall) {
        Configure-Firewall | Out-Null
    }
    
    # Display completion message
    Write-Host "🎉 Client Installation Complete!" -ForegroundColor Green
    Write-Host "=================================" -ForegroundColor Green
    Write-Host ""
    Write-Host "📋 Installation Summary:" -ForegroundColor Cyan
    Write-Host "  • CryoSpy Capture Client: ✅ Ready" -ForegroundColor White
    Write-Host "  • All OpenCV dependencies: ✅ Bundled in release" -ForegroundColor White
    if (Test-FFmpegAvailable) {
        Write-Host "  • FFmpeg (optional): ✅ Available" -ForegroundColor White
    } else {
        Write-Host "  • FFmpeg (optional): ⚠️  Not installed" -ForegroundColor Yellow
    }
    Write-Host ""
    Write-Host "IMPORTANT: Configuration Review!" -ForegroundColor Yellow
    Write-Host ""
    Write-Host "The client configuration has been created, but please review config.json:" -ForegroundColor White
    Write-Host ""
    Write-Host "1. Verify server_url points to your CryoSpy server" -ForegroundColor White
    Write-Host "2. Ensure client_id and client_secret are correct" -ForegroundColor White
    Write-Host "3. Configure camera_device if needed (Windows uses different format)" -ForegroundColor White
    Write-Host "4. Review proxy authentication settings if applicable" -ForegroundColor White
    Write-Host ""
    Write-Host "To start the client:" -ForegroundColor Yellow
    Write-Host ""
    Write-Host "    Option A - Manual startup:" -ForegroundColor White
    Write-Host "      - Double-click start-client.bat" -ForegroundColor Cyan
    Write-Host ""
    Write-Host "    Option B - Windows Service:" -ForegroundColor White
    Write-Host "      - Run install-service.bat as Administrator" -ForegroundColor Cyan
    Write-Host "      - Service will start automatically on boot" -ForegroundColor Cyan
    
    Write-Host ""
    Write-Host "Use stop-client.bat to stop the client when needed." -ForegroundColor White
    Write-Host ""
    Write-Host "For help and documentation, visit:" -ForegroundColor White
    Write-Host "https://github.com/Yeti47/cryospy" -ForegroundColor Cyan
    Write-Host ""
    
} catch {
    Write-Error "Installation failed: $($_.Exception.Message)"
    Write-Host "Stack trace: $($_.ScriptStackTrace)" -ForegroundColor Red
    exit 1
}

if (-not $Force) {
    Read-Host "Press Enter to exit..."
}
