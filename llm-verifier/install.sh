#!/bin/bash

# LLM Verifier Installation Script
# This script downloads and installs LLM Verifier for your platform

set -e

VERSION="latest"
INSTALL_DIR="/usr/local/bin"
SERVICE_NAME="llm-verifier"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Detect platform
detect_platform() {
    OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    ARCH=$(uname -m | tr '[:upper:]' '[:lower:]')
    
    case $OS in
        linux)
            PLATFORM="linux"
            ;;
        darwin)
            PLATFORM="darwin"
            ;;
        windows|cygwin|mingw)
            PLATFORM="windows"
            ;;
        *)
            print_error "Unsupported OS: $OS"
            exit 1
            ;;
    esac
    
    case $ARCH in
        x86_64|amd64)
            ARCH="amd64"
            ;;
        aarch64|arm64)
            ARCH="arm64"
            ;;
        armv7l)
            ARCH="arm"
            ;;
        *)
            print_error "Unsupported architecture: $ARCH"
            exit 1
            ;;
    esac
    
    BINARY_NAME="llm-verifier-${PLATFORM}-${ARCH}"
    if [ "$PLATFORM" = "windows" ]; then
        BINARY_NAME="${BINARY_NAME}.exe"
    fi
    
    print_status "Detected platform: $PLATFORM-$ARCH"
}

# Check dependencies
check_dependencies() {
    print_status "Checking dependencies..."
    
    # Check for curl
    if ! command -v curl &> /dev/null; then
        print_error "curl is required but not installed. Please install curl first."
        exit 1
    fi
    
    # Check for systemd (Linux only)
    if [ "$PLATFORM" = "linux" ] && ! command -v systemctl &> /dev/null; then
        print_warning "systemd not found. Service installation will be skipped."
    fi
}

# Download binary
download_binary() {
    print_status "Downloading LLM Verifier $VERSION..."
    
    RELEASE_URL="https://github.com/vasic-digital/LLMsVerifier/releases/download/$VERSION"
    
    if [ "$VERSION" = "latest" ]; then
        RELEASE_URL="https://github.com/vasic-digital/LLMsVerifier/releases/latest/download"
    fi
    
    DOWNLOAD_URL="${RELEASE_URL}/${BINARY_NAME}"
    
    # Create temporary directory
    TMP_DIR=$(mktemp -d)
    cd "$TMP_DIR"
    
    # Download binary
    if curl -fsSL -o "$BINARY_NAME" "$DOWNLOAD_URL"; then
        print_status "Download completed successfully"
    else
        print_error "Failed to download LLM Verifier"
        rm -rf "$TMP_DIR"
        exit 1
    fi
    
    # Verify binary (optional checksum check would go here)
    
    # Make executable
    chmod +x "$BINARY_NAME"
    
    # Move to install directory
    if [ -w "$INSTALL_DIR" ]; then
        mv "$BINARY_NAME" "$INSTALL_DIR/"
    else
        print_warning "No write access to $INSTALL_DIR. Installing to ~/.local/bin"
        mkdir -p "$HOME/.local/bin"
        mv "$BINARY_NAME" "$HOME/.local/bin/"
        INSTALL_DIR="$HOME/.local/bin"
    fi
    
    # Cleanup
    cd /
    rm -rf "$TMP_DIR"
    
    print_status "Installation completed successfully"
}

# Create configuration
create_config() {
    CONFIG_DIR="$HOME/.config/llm-verifier"
    DATA_DIR="$HOME/.local/share/llm-verifier"
    
    print_status "Creating configuration..."
    
    # Create directories
    mkdir -p "$CONFIG_DIR"
    mkdir -p "$DATA_DIR"
    mkdir -p "$DATA_DIR/logs"
    
    # Create default config
    cat > "$CONFIG_DIR/config.yaml" << CONFIG_EOF
api:
  port: 8080
  rate_limit: 1000
  enable_cors: true
  
database:
  path: "$DATA_DIR/llm-verifier.db"
  
logging:
  level: "info"
  file: "$DATA_DIR/logs/llm-verifier.log"
  max_size: "100MB"
  max_backups: 5
  
security:
  jwt_secret: "$(openssl rand -base64 32 2>/dev/null || echo 'change-this-secret-in-production')"
  session_timeout: "24h"
  
notifications:
  enabled: true
  webhook_url: ""
  
monitoring:
  enabled: true
  metrics_port: 9090
CONFIG_EOF
    
    print_status "Configuration created at $CONFIG_DIR/config.yaml"
}

# Create systemd service
create_service() {
    if [ "$PLATFORM" != "linux" ] || ! command -v systemctl &> /dev/null; then
        print_warning "Skipping service creation (not supported on this platform)"
        return
    fi
    
    print_status "Creating systemd service..."
    
    SERVICE_FILE="/etc/systemd/system/${SERVICE_NAME}.service"
    EXECUTABLE="$INSTALL_DIR/$BINARY_NAME"
    
    sudo tee "$SERVICE_FILE" > /dev/null << SERVICE_EOF
[Unit]
Description=LLM Verifier
After=network.target

[Service]
Type=simple
User=$USER
ExecStart=$EXECUTABLE api
Restart=on-failure
RestartSec=5
Environment=LLM_VERIFIER_CONFIG_PATH=$HOME/.config/llm-verifier/config.yaml

[Install]
WantedBy=multi-user.target
SERVICE_EOF
    
    sudo systemctl daemon-reload
    sudo systemctl enable $SERVICE_NAME
    
    print_status "Service created. Start with: sudo systemctl start $SERVICE_NAME"
}

# Main installation flow
main() {
    echo "LLM Verifier Installation Script"
    echo "================================"
    echo
    
    # Parse command line arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            --version)
                VERSION="$2"
                shift 2
                ;;
            --install-dir)
                INSTALL_DIR="$2"
                shift 2
                ;;
            --help|-h)
                echo "Usage: $0 [options]"
                echo "Options:"
                echo "  --version VERSION    Install specific version (default: latest)"
                echo "  --install-dir DIR    Install directory (default: /usr/local/bin)"
                echo "  --help              Show this help message"
                exit 0
                ;;
            *)
                print_error "Unknown option: $1"
                exit 1
                ;;
        esac
    done
    
    detect_platform
    check_dependencies
    download_binary
    create_config
    create_service
    
    echo
    print_status "Installation completed successfully!"
    echo
    echo "Next steps:"
    echo "1. Start the service: sudo systemctl start $SERVICE_NAME"
    echo "2. Check status: sudo systemctl status $SERVICE_NAME"
    echo "3. View logs: sudo journalctl -u $SERVICE_NAME -f"
    echo "4. Access web interface: http://localhost:8080"
    echo
    echo "Configuration: $HOME/.config/llm-verifier/config.yaml"
    echo "Logs: $HOME/.local/share/llm-verifier/logs/"
}

# Run main function with all arguments
main "$@"
