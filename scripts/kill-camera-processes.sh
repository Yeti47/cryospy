#!/bin/bash

# kill-camera-processes.sh
# Script to terminate processes that are currently accessing the webcam
# This is useful for testing when you need to ensure the camera is available

set -e

echo "🔍 Checking for processes using the webcam..."

# Function to check if a command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Function to kill processes using video devices
kill_video_processes() {
    local killed_any=false
    
    # Check for processes using /dev/video* devices
    for video_device in /dev/video*; do
        if [[ -e "$video_device" ]]; then
            echo "Checking processes using $video_device..."
            
            # Use lsof to find processes using the video device
            if command_exists lsof; then
                local pids=$(lsof "$video_device" 2>/dev/null | awk 'NR>1 {print $2}' | sort -u)
                
                if [[ -n "$pids" ]]; then
                    echo "Found processes using $video_device:"
                    lsof "$video_device" 2>/dev/null | awk 'NR>1 {printf "  PID %s: %s (%s)\n", $2, $1, $9}'
                    
                    for pid in $pids; do
                        if kill -0 "$pid" 2>/dev/null; then
                            local cmd=$(ps -p "$pid" -o comm= 2>/dev/null || echo "unknown")
                            local args=$(ps -p "$pid" -o args= 2>/dev/null || echo "")
                            
                            # Ask for confirmation unless force mode is enabled
                            if [[ "${FORCE:-}" != "true" ]]; then
                                echo "Process details:"
                                echo "  PID: $pid"
                                echo "  Command: $cmd"
                                echo "  Args: $args"
                                read -p "❓ Kill this process? [y/N] " -n 1 -r
                                echo
                                if [[ ! $REPLY =~ ^[Yy]$ ]]; then
                                    echo "⏭️  Skipping process $pid"
                                    continue
                                fi
                            fi
                            
                            echo "Killing process $pid ($cmd)..."
                            kill -TERM "$pid" 2>/dev/null || true
                            killed_any=true
                            
                            # Give it a moment to terminate gracefully
                            sleep 1
                            
                            # Force kill if still running
                            if kill -0 "$pid" 2>/dev/null; then
                                echo "🔥 Force killing process $pid..."
                                kill -KILL "$pid" 2>/dev/null || true
                            fi
                        fi
                    done
                else
                    echo "  No processes found using $video_device"
                fi
            else
                echo "  ⚠️  lsof not available - cannot detect processes using $video_device"
            fi
        fi
    done
    
    if [[ "$killed_any" == true ]]; then
        return 0
    else
        return 1
    fi
}

# Function to list video devices
list_video_devices() {
    echo "📹 Available video devices:"
    local found_devices=false
    
    for video_device in /dev/video*; do
        if [[ -e "$video_device" ]]; then
            found_devices=true
            echo "  $video_device"
            
            # Try to get device info if v4l2-ctl is available
            if command_exists v4l2-ctl; then
                local device_name=$(v4l2-ctl --device="$video_device" --info 2>/dev/null | grep "Card type" | cut -d':' -f2 | xargs 2>/dev/null || echo "Unknown")
                echo "    Device: $device_name"
                
                # Show if device is currently in use
                if command_exists lsof; then
                    local users=$(lsof "$video_device" 2>/dev/null | wc -l)
                    if [[ $users -gt 1 ]]; then
                        echo "    Status: ⚠️  In use by $((users-1)) process(es)"
                    else
                        echo "    Status: ✅ Available"
                    fi
                fi
            fi
        fi
    done
    
    if [[ "$found_devices" == false ]]; then
        echo "  ❌ No video devices found"
    fi
}

# Main execution
main() {
    local force=false
    local verbose=false
    
    # Parse command line arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            -f|--force)
                force=true
                export FORCE=true
                shift
                ;;
            -v|--verbose)
                verbose=true
                shift
                ;;
            -h|--help)
                echo "Usage: $0 [OPTIONS]"
                echo ""
                echo "Kill processes that are currently accessing the webcam."
                echo ""
                echo "Options:"
                echo "  -f, --force    Force kill without asking for confirmation"
                echo "  -v, --verbose  Show verbose output"
                echo "  -h, --help     Show this help message"
                echo ""
                echo "Examples:"
                echo "  $0              # Interactive mode"
                echo "  $0 --force      # Force kill all camera processes"
                echo "  $0 --verbose    # Show detailed output"
                exit 0
                ;;
            *)
                echo "Unknown option: $1"
                echo "Use --help for usage information."
                exit 1
                ;;
        esac
    done
    
    echo "🎥 CryoSpy Camera Process Killer"
    echo "================================"
    
    if [[ "$force" == true ]]; then
        echo "⚠️  Force mode enabled - will kill processes without confirmation"
    fi
    
    # Check if lsof is available
    if ! command_exists lsof; then
        echo "⚠️  Warning: lsof not found. Install it for better process detection:"
        echo "   Ubuntu/Debian: sudo apt install lsof"
        echo "   CentOS/RHEL:   sudo yum install lsof"
        echo "   macOS:         brew install lsof"
        echo ""
        echo "❌ Cannot detect camera processes without lsof. Exiting."
        exit 1
    fi
    
    # Kill processes using video devices directly
    if kill_video_processes; then
        echo ""
        echo "✅ Camera processes have been terminated."
        echo "⏳ Waiting a moment for devices to be released..."
        sleep 2
    else
        echo ""
        echo "✅ No camera processes found running."
    fi
    
    # List available video devices
    echo ""
    list_video_devices
    
    echo ""
    echo "🚀 Camera should now be available for CryoSpy capture client!"
}

# Run main function with all arguments
main "$@"
