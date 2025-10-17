#!/bin/bash
#
# Install script for git hooks
#
# This script installs the pre-commit hook and any other hooks
# into the .git/hooks directory.
#

set -e  # Exit on any error

# Color definitions for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}ℹ${NC} $1"
}

log_success() {
    echo -e "${GREEN}✓${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}⚠${NC} $1"
}

log_error() {
    echo -e "${RED}✗${NC} $1"
}

# Check if we're in a git repository
check_git_repo() {
    if ! git rev-parse --git-dir &> /dev/null; then
        log_error "Not in a git repository!"
        log_info "Please run this script from the root of a git repository."
        exit 1
    fi
}

# Install a single hook
install_hook() {
    local hook_name=$1
    local source_path="hooks/$hook_name"
    local target_path=".git/hooks/$hook_name"
    
    if [ ! -f "$source_path" ]; then
        log_error "Source hook not found: $source_path"
        return 1
    fi
    
    # Check if target already exists
    if [ -f "$target_path" ]; then
        log_warning "Hook already exists: $target_path"
        read -p "Do you want to overwrite it? (y/N): " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            log_info "Skipping $hook_name"
            return 0
        fi
    fi
    
    # Copy and make executable
    cp "$source_path" "$target_path"
    chmod +x "$target_path"
    
    log_success "Installed $hook_name hook"
    return 0
}

# Uninstall a single hook
uninstall_hook() {
    local hook_name=$1
    local target_path=".git/hooks/$hook_name"
    
    if [ -f "$target_path" ]; then
        rm "$target_path"
        log_success "Uninstalled $hook_name hook"
    else
        log_info "Hook not installed: $hook_name"
    fi
}

# List installed hooks
list_hooks() {
    log_info "Installed git hooks:"
    
    local hooks_dir=".git/hooks"
    local found_hooks=0
    
    for hook in "$hooks_dir"/*; do
        if [ -f "$hook" ] && [ -x "$hook" ]; then
            local hook_name=$(basename "$hook")
            # Skip sample hooks
            if [[ "$hook_name" != *.sample ]]; then
                echo "  - $hook_name"
                found_hooks=$((found_hooks + 1))
            fi
        fi
    done
    
    if [ $found_hooks -eq 0 ]; then
        echo "  (none)"
    fi
}

# Main function
main() {
    local command=${1:-install}
    
    log_info "🦊 Git Hooks Manager for Foxi"
    echo
    
    case "$command" in
        install)
            check_git_repo
            log_info "Installing git hooks..."
            echo
            
            # List of hooks to install
            local hooks=(
                "pre-commit"
                # Add more hooks here as needed
            )
            
            local installed=0
            for hook in "${hooks[@]}"; do
                if install_hook "$hook"; then
                    installed=$((installed + 1))
                fi
            done
            
            echo
            if [ $installed -gt 0 ]; then
                log_success "Successfully installed $installed hook(s)!"
                echo
                log_info "The hooks will now run automatically during git operations."
                log_info "To temporarily skip hooks, use: git commit --no-verify"
            else
                log_warning "No hooks were installed."
            fi
            ;;
            
        uninstall)
            check_git_repo
            log_info "Uninstalling git hooks..."
            echo
            
            local hooks=(
                "pre-commit"
                # Add more hooks here as needed
            )
            
            for hook in "${hooks[@]}"; do
                uninstall_hook "$hook"
            done
            
            echo
            log_success "Hook uninstallation complete!"
            ;;
            
        list)
            check_git_repo
            list_hooks
            ;;
            
        help|--help|-h)
            echo "Usage: $0 [command]"
            echo
            echo "Commands:"
            echo "  install     Install git hooks (default)"
            echo "  uninstall   Remove git hooks"
            echo "  list        List installed hooks"
            echo "  help        Show this help message"
            echo
            echo "Examples:"
            echo "  $0                # Install hooks"
            echo "  $0 install        # Install hooks"
            echo "  $0 uninstall      # Remove hooks"
            echo "  $0 list           # List installed hooks"
            ;;
            
        *)
            log_error "Unknown command: $command"
            log_info "Run '$0 help' for usage information."
            exit 1
            ;;
    esac
}

# Run main function
main "$@"