#!/bin/bash
# Container Runtime Detection and Wrapper Script for LLMsVerifier
# Supports: Docker, Podman, docker-compose, podman-compose
#
# Usage:
#   source scripts/container-runtime.sh
#   $CONTAINER_CMD <docker/podman commands>
#   $COMPOSE_CMD <docker-compose/podman-compose commands>
#
#   Or run directly:
#   ./scripts/container-runtime.sh build
#   ./scripts/container-runtime.sh start
#   ./scripts/container-runtime.sh stop

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Detect container runtime
detect_container_runtime() {
    if command -v docker &> /dev/null && docker info &> /dev/null 2>&1; then
        echo "docker"
    elif command -v podman &> /dev/null; then
        echo "podman"
    else
        echo "none"
    fi
}

# Detect compose command
detect_compose_cmd() {
    local runtime="$1"

    if [ "$runtime" = "docker" ]; then
        # Check for docker compose (v2) first, then docker-compose (v1)
        if docker compose version &> /dev/null 2>&1; then
            echo "docker compose"
        elif command -v docker-compose &> /dev/null; then
            echo "docker-compose"
        else
            echo "none"
        fi
    elif [ "$runtime" = "podman" ]; then
        # Check for podman-compose
        if command -v podman-compose &> /dev/null; then
            echo "podman-compose"
        else
            echo "none"
        fi
    else
        echo "none"
    fi
}

# Initialize runtime variables
init_container_runtime() {
    export CONTAINER_RUNTIME=$(detect_container_runtime)

    if [ "$CONTAINER_RUNTIME" = "none" ]; then
        echo -e "${RED}Error: No container runtime found (Docker or Podman)${NC}"
        echo "Please install Docker or Podman to continue."
        echo ""
        echo "Install Docker: https://docs.docker.com/get-docker/"
        echo "Install Podman: https://podman.io/getting-started/installation"
        return 1
    fi

    export CONTAINER_CMD="$CONTAINER_RUNTIME"
    export COMPOSE_CMD_RAW=$(detect_compose_cmd "$CONTAINER_RUNTIME")

    if [ "$COMPOSE_CMD_RAW" = "none" ]; then
        echo -e "${YELLOW}Warning: No compose tool found for $CONTAINER_RUNTIME${NC}"
        if [ "$CONTAINER_RUNTIME" = "podman" ]; then
            echo "Install podman-compose: pip install podman-compose"
        else
            echo "Install docker-compose: https://docs.docker.com/compose/install/"
        fi
        export COMPOSE_CMD=""
    else
        export COMPOSE_CMD="$COMPOSE_CMD_RAW"
    fi

    # Set Podman-specific environment variables for better Docker compatibility
    if [ "$CONTAINER_RUNTIME" = "podman" ]; then
        # Enable Docker compatibility socket
        export DOCKER_HOST="${DOCKER_HOST:-unix:///run/user/$(id -u)/podman/podman.sock}"
        # Use compatible network mode
        export PODMAN_USERNS="keep-id"
    fi

    return 0
}

# Print detected configuration
print_runtime_info() {
    echo -e "${BLUE}Container Runtime Configuration:${NC}"
    echo -e "  Runtime:     ${GREEN}$CONTAINER_RUNTIME${NC}"
    echo -e "  Compose:     ${GREEN}$COMPOSE_CMD${NC}"
    if [ "$CONTAINER_RUNTIME" = "podman" ]; then
        echo -e "  Docker Host: ${GREEN}$DOCKER_HOST${NC}"
    fi
    echo ""
}

# Wrapper function to run container commands
run_container() {
    $CONTAINER_CMD "$@"
}

# Wrapper function to run compose commands
run_compose() {
    if [ -z "$COMPOSE_CMD" ]; then
        echo -e "${RED}Error: No compose tool available${NC}"
        return 1
    fi
    $COMPOSE_CMD "$@"
}

# Check if Podman needs rootless setup
check_podman_setup() {
    if [ "$CONTAINER_RUNTIME" != "podman" ]; then
        return 0
    fi

    # Check if podman socket is running (rootless)
    if ! systemctl --user is-active podman.socket &> /dev/null; then
        echo -e "${YELLOW}Podman socket not running. Starting...${NC}"
        systemctl --user start podman.socket 2>/dev/null || true
    fi

    # Verify socket exists
    if [ ! -S "/run/user/$(id -u)/podman/podman.sock" ]; then
        echo -e "${YELLOW}Podman socket not found at expected location.${NC}"
        echo "Run: systemctl --user enable --now podman.socket"
    fi
}

# Build image with appropriate runtime
build_image() {
    local tag="${1:-llm-verifier:latest}"
    local dockerfile="${2:-Dockerfile}"
    local context="${3:-.}"

    echo -e "${BLUE}Building image: $tag${NC}"

    if [ "$CONTAINER_RUNTIME" = "podman" ]; then
        podman build -f "$dockerfile" -t "$tag" "$context"
    else
        docker build -f "$dockerfile" -t "$tag" "$context"
    fi
}

# Start services with compose
start_services() {
    local profile="${1:-}"

    if [ -z "$COMPOSE_CMD" ]; then
        echo -e "${RED}Error: No compose tool available${NC}"
        return 1
    fi

    echo -e "${BLUE}Starting LLM Verifier services...${NC}"

    if [ -n "$profile" ]; then
        $COMPOSE_CMD --profile "$profile" up -d
    else
        $COMPOSE_CMD up -d
    fi
}

# Stop services with compose
stop_services() {
    if [ -z "$COMPOSE_CMD" ]; then
        echo -e "${RED}Error: No compose tool available${NC}"
        return 1
    fi

    echo -e "${BLUE}Stopping LLM Verifier services...${NC}"
    $COMPOSE_CMD down
}

# View logs
view_logs() {
    local service="${1:-}"

    if [ -z "$COMPOSE_CMD" ]; then
        echo -e "${RED}Error: No compose tool available${NC}"
        return 1
    fi

    if [ -n "$service" ]; then
        $COMPOSE_CMD logs -f "$service"
    else
        $COMPOSE_CMD logs -f
    fi
}

# Check container/pod status
check_status() {
    if [ -z "$COMPOSE_CMD" ]; then
        echo -e "${RED}Error: No compose tool available${NC}"
        return 1
    fi

    $COMPOSE_CMD ps
}

# Run the container directly (without compose)
run_container_direct() {
    local tag="${1:-llm-verifier:latest}"
    local port="${2:-8080}"

    echo -e "${BLUE}Running LLM Verifier container on port $port...${NC}"

    if [ "$CONTAINER_RUNTIME" = "podman" ]; then
        podman run -d --name llm-verifier -p "$port:8080" \
            -v ./data:/app/data:Z \
            -v ./exports:/app/exports:Z \
            -v ./config.yaml:/app/config.yaml:ro,Z \
            "$tag"
    else
        docker run -d --name llm-verifier -p "$port:8080" \
            -v ./data:/app/data \
            -v ./exports:/app/exports \
            -v ./config.yaml:/app/config.yaml:ro \
            "$tag"
    fi
}

# Show usage
show_usage() {
    echo "LLM Verifier Container Runtime Script"
    echo ""
    echo "Usage: $0 {build|start|stop|logs|status|run|detect}"
    echo ""
    echo "Commands:"
    echo "  build [tag] [dockerfile] [context] - Build container image"
    echo "  start [profile]                    - Start services with compose"
    echo "  stop                               - Stop services"
    echo "  logs [service]                     - View logs"
    echo "  status                             - Check service status"
    echo "  run [tag] [port]                   - Run container directly"
    echo "  detect                             - Show detected runtime"
    echo ""
    echo "Environment Variables:"
    echo "  CONTAINER_RUNTIME - Set to 'docker' or 'podman' to force runtime"
    echo "  DOCKER_HOST       - Docker/Podman socket path"
}

# Main initialization when sourced
if [ "${BASH_SOURCE[0]}" = "${0}" ]; then
    # Script is being run directly
    init_container_runtime || exit 1

    # If no arguments, show info
    if [ $# -eq 0 ]; then
        print_runtime_info
        show_usage
        exit 0
    fi

    # Process commands
    case "$1" in
        build)
            shift
            build_image "$@"
            ;;
        start)
            shift
            start_services "$@"
            ;;
        stop)
            stop_services
            ;;
        logs)
            shift
            view_logs "$@"
            ;;
        status)
            check_status
            ;;
        run)
            shift
            run_container_direct "$@"
            ;;
        detect)
            print_runtime_info
            ;;
        *)
            show_usage
            exit 1
            ;;
    esac
else
    # Script is being sourced
    init_container_runtime
fi
