#!/bin/bash

# kill-debug-processes.sh
# Script to terminate debug processes that start with "__debug_bin"
# This is useful for cleaning up Go debug binaries left running from debugging sessions

set -e

echo "🔍 Checking for debug processes..."

# Function to kill debug processes
kill_debug_processes() {
    local killed_any=false
    
    # Find all processes that start with "__debug_bin"
    local pids=$(pgrep -f "^__debug_bin" 2>/dev/null || true)
    
    if [[ -n "$pids" ]]; then
        echo "Found debug processes:"
        
        for pid in $pids; do
            if kill -0 "$pid" 2>/dev/null; then
                local cmd=$(ps -p "$pid" -o comm= 2>/dev/null || echo "unknown")
                local args=$(ps -p "$pid" -o args= 2>/dev/null || echo "")
                
                echo "  PID $pid: $cmd"
                echo "    Command: $args"
                
                # Ask for confirmation unless force mode is enabled
                if [[ "${FORCE:-}" != "true" ]]; then
                    read -p "❓ Kill this debug process? [y/N] " -n 1 -r
                    echo
                    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
                        echo "⏭️  Skipping process $pid"
                        continue
                    fi
                fi
                
                echo "🔪 Killing debug process $pid ($cmd)..."
                kill -TERM "$pid" 2>/dev/null || true
                killed_any=true
                
                # Give it a moment to terminate gracefully
                sleep 1
                
                # Force kill if still running
                if kill -0 "$pid" 2>/dev/null; then
                    echo "🔥 Force killing debug process $pid..."
                    kill -KILL "$pid" 2>/dev/null || true
                fi
            fi
        done
    else
        echo "  No debug processes found"
    fi
    
    if [[ "$killed_any" == true ]]; then
        return 0
    else
        return 1
    fi
}

# Main execution
main() {
    local force=false
    
    # Parse command line arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            -f|--force)
                force=true
                export FORCE=true
                shift
                ;;
            -h|--help)
                echo "Usage: $0 [OPTIONS]"
                echo ""
                echo "Kill debug processes that start with '__debug_bin'."
                echo ""
                echo "Options:"
                echo "  -f, --force    Force kill without asking for confirmation"
                echo "  -h, --help     Show this help message"
                echo ""
                echo "Examples:"
                echo "  $0              # Interactive mode"
                echo "  $0 --force      # Force kill all debug processes"
                exit 0
                ;;
            *)
                echo "Unknown option: $1"
                echo "Use --help for usage information."
                exit 1
                ;;
        esac
    done
    
    echo "🐛 CryoSpy Debug Process Killer"
    echo "==============================="
    
    if [[ "$force" == true ]]; then
        echo "⚠️  Force mode enabled - will kill processes without confirmation"
    fi
    
    # Kill debug processes
    if kill_debug_processes; then
        echo ""
        echo "✅ Debug processes have been terminated."
    else
        echo ""
        echo "✅ No debug processes found running."
    fi
    
    echo ""
    echo "🚀 Debug cleanup complete!"
}

# Run main function with all arguments
main "$@"
