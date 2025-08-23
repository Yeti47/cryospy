# install-server-windows.ps1
# PowerShell script to install CryoSpy Server Components on Windows

param(
    [Parameter(Mandatory=$false)]
    [switch]$SkipDependencies,
    
    [Parameter(Mandatory=$false)]
    [switch]$InstallAsService,
    
    [Parameter(Mandatory=$false)]
    [switch]$SetupFirewall,
    
    [Parameter(Mandatory=$false)]
    [switch]$Force
)

# Ensure running as Administrator
if (-NOT ([Security.Principal.WindowsPrincipal] [Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole] "Administrator")) {
    Write-Host "This script requires Administrator privileges for best results." -ForegroundColor Yellow
    Write-Host "Some features may not work properly without admin privileges." -ForegroundColor Yellow
    Write-Host ""
    
    $continue = Read-Host "Continue anyway? (y/N)"
    if ($continue -ne 'y' -and $continue -ne 'Y') {
        exit 1
    }
}

Write-Host "🚀 CryoSpy Server Components - Windows Installation" -ForegroundColor Cyan
Write-Host "====================================================" -ForegroundColor Cyan
Write-Host ""

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

# Function to install dependencies
function Install-Dependencies {
    Write-Host "📥 Installing server dependencies..." -ForegroundColor Green
    Write-Host "This may take several minutes..." -ForegroundColor Gray
    
    try {
        # Install FFmpeg
        choco install ffmpeg -y
        
        if ($LASTEXITCODE -eq 0) {
            Write-Host "✅ Dependencies installed successfully" -ForegroundColor Green
            return $true
        } else {
            Write-Warning "Failed to install some dependencies via Chocolatey"
            return $false
        }
    } catch {
        Write-Error "Error installing dependencies: $($_.Exception.Message)"
        return $false
    }
}

# Function to show manual installation instructions
function Show-ManualInstallInstructions {
    Write-Host ""
    Write-Host "📋 Manual Installation Required" -ForegroundColor Yellow
    Write-Host "===============================" -ForegroundColor Yellow
    Write-Host ""
    Write-Host "Please install the following dependencies manually:" -ForegroundColor White
    Write-Host ""
    Write-Host "1. FFmpeg for Windows:" -ForegroundColor White
    Write-Host "   - Download from: https://ffmpeg.org/download.html#build-windows" -ForegroundColor Cyan
    Write-Host "   - Extract to C:\ffmpeg" -ForegroundColor Cyan
    Write-Host "   - Add C:\ffmpeg\bin to your PATH" -ForegroundColor Cyan
    Write-Host ""
    Write-Host "Alternatively, you can use vcpkg:" -ForegroundColor White
    Write-Host "   git clone https://github.com/Microsoft/vcpkg.git" -ForegroundColor Cyan
    Write-Host "   cd vcpkg" -ForegroundColor Cyan
    Write-Host "   .\bootstrap-vcpkg.bat" -ForegroundColor Cyan
    Write-Host "   .\vcpkg install ffmpeg" -ForegroundColor Cyan
    Write-Host ""
    Read-Host "Press Enter to continue..."
}

# Function to create startup scripts
function Create-StartupScripts {
    Write-Host "📝 Creating server startup scripts..." -ForegroundColor Green
    
    try {
        # Create start-server.bat
        $startScript = @"
@echo off
echo Starting CryoSpy Server Components...
echo.
start "CryoSpy Capture Server" capture-server.exe
timeout /t 2 /nobreak >nul
start "CryoSpy Dashboard" dashboard.exe
echo Server components started!
echo Check the opened windows for any error messages.
echo Access the dashboard at: http://localhost:8080
pause
"@
        $startScript | Out-File -FilePath "start-server.bat" -Encoding ASCII
        
        # Create stop-server.bat
        $stopScript = @"
@echo off
echo Stopping CryoSpy server processes...
taskkill /f /im capture-server.exe 2>nul
taskkill /f /im dashboard.exe 2>nul
echo Server processes stopped.
pause
"@
        $stopScript | Out-File -FilePath "stop-server.bat" -Encoding ASCII
        
        Write-Host "✅ Created server startup scripts:" -ForegroundColor Green
        Write-Host "   - start-server.bat    (starts server components)" -ForegroundColor White
        Write-Host "   - stop-server.bat     (stops server processes)" -ForegroundColor White
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
echo Installing CryoSpy Server Windows Services...
echo.
sc create "CryoSpy Capture Server" binPath= "%CD%\capture-server.exe" start= auto
sc create "CryoSpy Dashboard" binPath= "%CD%\dashboard.exe" start= auto
echo.
echo Services created. Starting services...
sc start "CryoSpy Capture Server"
sc start "CryoSpy Dashboard"
echo.
echo Services installed and started successfully!
echo Access the dashboard at: http://localhost:8080
pause
"@
        $installServiceScript | Out-File -FilePath "install-services.bat" -Encoding ASCII
        
        # Create service removal script
        $uninstallServiceScript = @"
@echo off
echo Removing CryoSpy Server Windows Services...
echo.
sc stop "CryoSpy Capture Server"
sc stop "CryoSpy Dashboard"
sc delete "CryoSpy Capture Server"
sc delete "CryoSpy Dashboard"
echo.
echo Services removed successfully!
pause
"@
        $uninstallServiceScript | Out-File -FilePath "uninstall-services.bat" -Encoding ASCII
        
        Write-Host "✅ Created service management scripts:" -ForegroundColor Green
        Write-Host "   - install-services.bat   (run as Administrator to install services)" -ForegroundColor White
        Write-Host "   - uninstall-services.bat (run as Administrator to remove services)" -ForegroundColor White
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
echo Configuring Windows Firewall for CryoSpy Server...
echo.
netsh advfirewall firewall add rule name="CryoSpy Capture Server" dir=in action=allow program="%CD%\capture-server.exe"
netsh advfirewall firewall add rule name="CryoSpy Dashboard" dir=in action=allow program="%CD%\dashboard.exe"
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

# Function to create initial configuration
function Create-InitialConfig {
    Write-Host "📝 Creating initial server configuration..." -ForegroundColor Green
    
    # Get user's home directory and create cryospy folder
    $userHome = [Environment]::GetFolderPath("UserProfile")
    $cryospyDir = Join-Path $userHome "cryospy"
    $configPath = Join-Path $cryospyDir "config.json"
    
    # Create directory if it doesn't exist
    if (-not (Test-Path $cryospyDir)) {
        New-Item -ItemType Directory -Path $cryospyDir -Force | Out-Null
        Write-Host "✅ Created configuration directory: $cryospyDir" -ForegroundColor Green
    }
    
    # Check if config already exists
    if (Test-Path $configPath -and -not $Force) {
        $overwrite = Read-Host "Configuration file already exists at $configPath. Overwrite? (y/N)"
        if ($overwrite -ne 'y' -and $overwrite -ne 'Y') {
            Write-Host "⏭️  Skipping configuration file creation" -ForegroundColor Yellow
            return $configPath
        }
    }
    
    # Create logs directory path
    $logsPath = Join-Path $cryospyDir "logs"
    $dbPath = Join-Path $cryospyDir "cryospy.db"
    
    # Create initial configuration with Windows-appropriate paths
    $config = @{
        web_addr = "127.0.0.1"
        web_port = 8080
        capture_port = 8081
        database_path = $dbPath.Replace('\', '/')
        log_path = $logsPath.Replace('\', '/')
        log_level = "info"
        trusted_proxies = @{
            capture_server = @()
            dashboard = @()
        }
        storage_notification_settings = @{
            recipient = "admin@example.com"
            min_interval_minutes = 60
            warning_threshold = 0.8
        }
        motion_notification_settings = @{
            recipient = "admin@example.com"
            min_interval_minutes = 15
        }
        auth_event_settings = @{
            time_window_minutes = 60
            auto_disable_threshold = 10
            notification_recipient = "admin@example.com"
            notification_threshold = 5
            min_interval_minutes = 30
        }
        smtp_settings = @{
            host = "smtp.gmail.com"
            port = 587
            username = "your-email@gmail.com"
            password = "your-app-password"
            from_addr = "your-email@gmail.com"
        }
        streaming_settings = @{
            cache = @{
                enabled = $true
                max_size_bytes = 1073741824
            }
            look_ahead = 10
            width = 854
            height = 480
            video_bitrate = "1000k"
            video_codec = "libx264"
            frame_rate = 25
        }
    }
    
    try {
        # Convert to JSON and save
        $config | ConvertTo-Json -Depth 10 | Out-File -FilePath $configPath -Encoding UTF8
        Write-Host "✅ Configuration file created: $configPath" -ForegroundColor Green
        Write-Host "🔧 Please customize the configuration before starting the server:" -ForegroundColor Yellow
        Write-Host "   - Update email settings for notifications" -ForegroundColor White
        Write-Host "   - Configure trusted proxies if using reverse proxy" -ForegroundColor White
        Write-Host "   - Adjust storage and streaming settings as needed" -ForegroundColor White
        Write-Host ""
        
        return $configPath
    } catch {
        Write-Error "Failed to create configuration file: $($_.Exception.Message)"
        return $null
    }
}

# Function to configure Windows Firewall directly
function Configure-Firewall {
    Write-Host "🔥 Configuring Windows Firewall..." -ForegroundColor Green
    
    try {
        # Remove existing rules if they exist
        Remove-NetFirewallRule -DisplayName "CryoSpy Capture Server" -ErrorAction SilentlyContinue
        Remove-NetFirewallRule -DisplayName "CryoSpy Dashboard" -ErrorAction SilentlyContinue
        
        # Create new firewall rules
        $captureServerPath = Join-Path $PWD "capture-server.exe"
        $dashboardPath = Join-Path $PWD "dashboard.exe"
        
        New-NetFirewallRule -DisplayName "CryoSpy Capture Server" -Direction Inbound -Program $captureServerPath -Action Allow
        New-NetFirewallRule -DisplayName "CryoSpy Dashboard" -Direction Inbound -Program $dashboardPath -Action Allow
        
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
    # Install dependencies
    if (-not $SkipDependencies) {
        if (-not (Test-ChocolateyInstalled)) {
            $installChoco = $true
            if (-not $Force) {
                $response = Read-Host "Chocolatey not found. Install it? (Y/n)"
                $installChoco = ($response -eq '' -or $response -eq 'y' -or $response -eq 'Y')
            }
            
            if ($installChoco) {
                if (-not (Install-Chocolatey)) {
                    Show-ManualInstallInstructions
                }
            } else {
                Show-ManualInstallInstructions
            }
        }
        
        if (Test-ChocolateyInstalled) {
            if (-not (Install-Dependencies)) {
                Show-ManualInstallInstructions
            }
        }
    } else {
        Write-Host "⏭️  Skipping dependency installation" -ForegroundColor Yellow
    }
    
    # Create startup scripts
    Create-StartupScripts | Out-Null
    
    # Create service scripts
    Create-ServiceScripts | Out-Null
    
    # Create firewall script
    Create-FirewallScript | Out-Null
    
    # Create initial configuration
    $configPath = Create-InitialConfig
    
    # Configure firewall if requested
    if ($SetupFirewall) {
        Configure-Firewall | Out-Null
    }
    
    # Display completion message
    Write-Host "🎉 Server Installation Complete!" -ForegroundColor Green
    Write-Host "=================================" -ForegroundColor Green
    Write-Host ""
    Write-Host "Next steps:" -ForegroundColor Yellow
    Write-Host "1. Configuration file created at: $configPath" -ForegroundColor White
    Write-Host "   Please customize it before starting the server (especially email settings)" -ForegroundColor White
    Write-Host "2. Choose how to start CryoSpy Server:" -ForegroundColor White
    Write-Host ""
    Write-Host "    Option A - Manual startup:" -ForegroundColor White
    Write-Host "      - Double-click start-server.bat to start server components" -ForegroundColor Cyan
    Write-Host ""
    
    if ($InstallAsService) {
        Write-Host "    Option B - Windows Services:" -ForegroundColor White
        Write-Host "      - Run install-services.bat as Administrator" -ForegroundColor Cyan
        Write-Host "      - Services will start automatically on boot" -ForegroundColor Cyan
    } else {
        Write-Host "    Option B - Windows Services:" -ForegroundColor White
        Write-Host "      - Run install-services.bat as Administrator" -ForegroundColor Cyan
        Write-Host "      - Services will start automatically on boot" -ForegroundColor Cyan
    }
    
    Write-Host ""
    Write-Host "3. Access the web dashboard at: http://localhost:8080 (local access only)" -ForegroundColor White
    Write-Host "4. Use stop-server.bat to stop server processes when needed" -ForegroundColor White
    Write-Host "5. To expose capture-server securely to the internet:" -ForegroundColor White
    Write-Host "   - Run: PowerShell -ExecutionPolicy Bypass -File setup-nginx-proxy.ps1 -Domain yourdomain.com" -ForegroundColor Cyan
    Write-Host "   - This will configure nginx as a reverse proxy with SSL support" -ForegroundColor Cyan
    Write-Host ""
    Write-Host "SECURITY NOTE: The dashboard is only accessible locally for security." -ForegroundColor Yellow
    Write-Host "The nginx proxy script safely exposes only the capture-server API." -ForegroundColor Yellow
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
