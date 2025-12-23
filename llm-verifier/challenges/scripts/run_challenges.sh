#!/bin/bash
# Generic Challenge Runner
# Can run all challenges, specific challenges, or list available challenges
# Uses ONLY project binaries

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CHALLENGES_BANK="$SCRIPT_DIR/challenges_bank.json"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log() {
    echo "$(date '+%Y-%m-%d %H:%M:%S')] $*"
}

log_error() {
    echo -e "${RED}$(date '+%Y-%m-%d %H:%M:%S')] ERROR: $*${NC}"
}

log_info() {
    echo -e "${BLUE}$(date '+%Y-%m-%d %H:%M:%S')] INFO: $*${NC}"
}

log_success() {
    echo -e "${GREEN}$(date '+%Y-%m-%d %H:%M:%S')] SUCCESS: $*${NC}"
}

log_warning() {
    echo -e "${YELLOW}$(date '+%Y-%m-%d %H:%M:%S')] WARNING: $*${NC}"
}

# Display usage
usage() {
    cat << EOF
${GREEN}Generic Challenge Runner - LLM Verifier${NC}

${BLUE}Usage:${NC}
    $0 [OPTIONS] [CHALLENGE...]

${BLUE}Options:${NC}
    ${YELLOW}--list${NC}                      List all available challenges
    ${YELLOW}--all${NC}                       Run all challenges
    ${YELLOW}--platform PLATFORM${NC}          Use specific platform (cli, rest-api, tui, desktop, mobile, web)
    ${YELLOW}--platforms${NC}                 List available platforms
    ${YELLOW}--help, -h${NC}                Show this help message

${BLUE}Platforms:${NC}
    cli         - Command Line Interface (llm-verifier)
    rest-api    - REST API Server (llm-verifier-api)
    tui         - Terminal User Interface (llm-verifier-tui)
    desktop     - Desktop Application (llm-verifier-desktop)
    mobile      - Mobile Application (llm-verifier-mobile)
    web         - Web Application (llm-verifier-web)

${BLUE}Examples:${NC}
    $0 --list                                          # List all challenges
    $0 --all                                           # Run all challenges
    $0 provider_models_discovery                    # Run specific challenge
    $0 provider_models_discovery model_verification    # Run multiple challenges
    $0 --all --platform cli                            # Run all challenges using CLI
    $0 provider_models_discovery --platform cli            # Run specific challenge using CLI

${BLUE}Output:${NC}
    Results are stored in: ${YELLOW}challenges/<challenge_name>/YYYY/MM/DD/timestamp/${NC}
    Logs are stored in: ${YELLOW}challenges/<challenge_name>/YYYY/MM/DD/timestamp/logs/${NC}

EOF
}

# Check if challenges_bank.json exists
if [ ! -f "$CHALLENGES_BANK" ]; then
    log_error "Challenges bank not found: $CHALLENGES_BANK"
    exit 1
fi

# List all platforms
list_platforms() {
    log_info "Available Platforms:"
    echo ""
    cat "$CHALLENGES_BANK" | grep -A 200 '"platforms"' | grep -E '^\s+"[a-z-]+":' | sed 's/.*"\(.*\)".*/\1/' | while read platform; do
        echo -e "  ${GREEN}${platform}${NC}"
    done
}

# List all challenges
list_challenges() {
    log_info "Available Challenges:"
    echo ""
    
    CHALLENGE_COUNT=$(cat "$CHALLENGES_BANK" | grep -c '"id":')
    
    for i in $(seq 1 $CHALLENGE_COUNT); do
        CHALLENGE=$(cat "$CHALLENGES_BANK" | python3 -c "import sys, json; data=json.load(sys.stdin); print(json.dumps(data['challenges'][$i-1], indent=2))" 2>/dev/null || echo "Error parsing JSON")
        
        ID=$(echo "$CHALLENGE" | grep '"id"' | head -1 | sed 's/.*"\(.*\)".*/\1/')
        NAME=$(echo "$CHALLENGE" | grep '"name"' | head -1 | sed 's/.*"\(.*\)".*/\1/')
        DESC=$(echo "$CHALLENGE" | grep '"description"' | head -1 | sed 's/.*"\(.*\)".*/\1/')
        PLATFORMS=$(echo "$CHALLENGE" | grep '"platforms"' | head -1 | sed 's/.*\[\(.*\)].*/\1/' | sed 's/"//g')
        DURATION=$(echo "$CHALLENGE" | grep '"estimated_duration"' | head -1 | sed 's/.*"\(.*\)".*/\1/')
        
        echo -e "${GREEN}$ID${NC}"
        echo -e "  ${BLUE}Name:${NC}        $NAME"
        echo -e "  ${BLUE}Description:${NC} $DESC"
        echo -e "  ${BLUE}Platforms:${NC}    $PLATFORMS"
        echo -e "  ${BLUE}Duration:${NC}    $DURATION"
        echo ""
    done
    
    echo -e "${BLUE}Total Challenges: $CHALLENGE_COUNT${NC}"
}

# Get challenge details
get_challenge_details() {
    CHALLENGE_ID="$1"
    
    cat "$CHALLENGES_BANK" | python3 -c "
import sys, json
data = json.load(sys.stdin)
for challenge in data['challenges']:
    if challenge['id'] == '$CHALLENGE_ID':
        print(json.dumps(challenge, indent=2))
        sys.exit(0)
" 2>/dev/null || log_error "Challenge not found: $CHALLENGE_ID"
}

# Check if binary exists
check_binary() {
    BINARY="$1"
    
    if [ ! -f "$PROJECT_ROOT/$BINARY" ]; then
        log_error "Binary not found: $PROJECT_ROOT/$BINARY"
        return 1
    fi
    
    if [ ! -x "$PROJECT_ROOT/$BINARY" ]; then
        log_error "Binary not executable: $PROJECT_ROOT/$BINARY"
        return 1
    fi
    
    return 0
}

# Create challenge directory structure
create_challenge_directory() {
    CHALLENGE_NAME="$1"
    PLATFORM="$2"
    
    TIMESTAMP=$(date +%s)
    YEAR=$(date +%Y)
    MONTH=$(date +%m)
    DAY=$(date +%d)
    
    CHALLENGE_DIR="$SCRIPT_DIR/${CHALLENGE_NAME}/$YEAR/$MONTH/$DAY"
    
    if [ "$PLATFORM" != "" ]; then
        CHALLENGE_DIR="${CHALLENGE_DIR}_${PLATFORM}"
    fi
    
    CHALLENGE_DIR="${CHALLENGE_DIR}/${TIMESTAMP}"
    
    mkdir -p "$CHALLENGE_DIR/logs"
    mkdir -p "$CHALLENGE_DIR/results"
    
    echo "$CHALLENGE_DIR"
}

# Run specific challenge
run_challenge() {
    CHALLENGE_ID="$1"
    PLATFORM="${2:-cli}"
    
    log_info "Running challenge: $CHALLENGE_ID"
    log_info "Platform: $PLATFORM"
    
    # Get challenge details
    CHALLENGE=$(get_challenge_details "$CHALLENGE_ID")
    
    if [ -z "$CHALLENGE" ]; then
        log_error "Challenge not found: $CHALLENGE_ID"
        return 1
    fi
    
    # Extract challenge info
    CHALLENGE_NAME=$(echo "$CHALLENGE" | python3 -c "import sys, json; data=json.load(sys.stdin); print(data['name'])" 2>/dev/null)
    SCRIPT_PATH=$(echo "$CHALLENGE" | python3 -c "import sys, json; data=json.load(sys.stdin); print(data.get('script', ''))" 2>/dev/null)
    BINARY=$(echo "$CHALLENGE" | python3 -c "import sys, json; data=json.load(sys.stdin); print(data.get('binary', 'llm-verifier'))" 2>/dev/null)
    DEPS=$(echo "$CHALLENGE" | python3 -c "import sys, json; data=json.load(sys.stdin); deps=data.get('dependencies', []); print(' '.join(deps))" 2>/dev/null)
    
    # Check dependencies
    if [ ! -z "$DEPS" ]; then
        log_warning "This challenge has dependencies: $DEPS"
        log_info "Please run dependent challenges first"
        read -p "Continue anyway? (y/N) " -n 1 -r
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            log_info "Skipping challenge: $CHALLENGE_ID"
            return 0
        fi
    fi
    
    # Check if binary exists
    if ! check_binary "$BINARY"; then
        log_error "Binary not available for this platform"
        return 1
    fi
    
    # Create challenge directory
    CHALLENGE_DIR=$(create_challenge_directory "$CHALLENGE_ID" "$PLATFORM")
    LOG_FILE="$CHALLENGE_DIR/logs/challenge.log"
    CMD_LOG_FILE="$CHALLENGE_DIR/logs/commands.log"
    
    log_info "Challenge directory: $CHALLENGE_DIR"
    log_info "Log file: $LOG_FILE"
    
    # Log challenge start
    log "========================================" | tee -a "$LOG_FILE"
    log "CHALLENGE: $CHALLENGE_NAME" | tee -a "$LOG_FILE"
    log "ID: $CHALLENGE_ID" | tee -a "$LOG_FILE"
    log "Platform: $PLATFORM" | tee -a "$LOG_FILE"
    log "Binary: $PROJECT_ROOT/$BINARY" | tee -a "$LOG_FILE"
    log "========================================" | tee -a "$LOG_FILE"
    
    # Execute challenge script if exists
    if [ -f "$SCRIPT_DIR/$SCRIPT_PATH" ]; then
        log_info "Executing challenge script: $SCRIPT_PATH"
        bash "$SCRIPT_DIR/$SCRIPT_PATH" "$CHALLENGE_DIR" "$PLATFORM" 2>&1 | tee -a "$LOG_FILE"
    else
        log_error "Challenge script not found: $SCRIPT_PATH"
        return 1
    fi
    
    log_success "Challenge completed: $CHALLENGE_ID"
    log_info "Results: $CHALLENGE_DIR/results"
}

# Run all challenges
run_all_challenges() {
    PLATFORM="${1:-cli}"
    
    log_info "Running all challenges on platform: $PLATFORM"
    echo ""
    
    CHALLENGE_COUNT=$(cat "$CHALLENGES_BANK" | grep -c '"id":')
    
    for i in $(seq 1 $CHALLENGE_COUNT); do
        CHALLENGE_ID=$(cat "$CHALLENGES_BANK" | python3 -c "import sys, json; data=json.load(sys.stdin); print(data['challenges'][$i-1]['id'])" 2>/dev/null)
        run_challenge "$CHALLENGE_ID" "$PLATFORM"
        echo ""
    done
    
    log_success "All challenges completed on platform: $PLATFORM"
}

# Main script logic
main() {
    if [ $# -eq 0 ]; then
        usage
        exit 0
    fi
    
    # Parse arguments
    CHALLENGES=()
    PLATFORM="cli"
    LIST_CHALLENGES=false
    RUN_ALL=false
    LIST_PLATFORMS=false
    
    while [ $# -gt 0 ]; do
        case "$1" in
            --list)
                LIST_CHALLENGES=true
                shift
                ;;
            --all)
                RUN_ALL=true
                shift
                ;;
            --platform)
                PLATFORM="$2"
                shift 2
                ;;
            --platforms)
                LIST_PLATFORMS=true
                shift
                ;;
            --help|-h)
                usage
                exit 0
                ;;
            -*)
                log_error "Unknown option: $1"
                usage
                exit 1
                ;;
            *)
                CHALLENGES+=("$1")
                shift
                ;;
        esac
    done
    
    # Execute requested action
    if [ "$LIST_PLATFORMS" = true ]; then
        list_platforms
    elif [ "$LIST_CHALLENGES" = true ]; then
        list_challenges
    elif [ "$RUN_ALL" = true ]; then
        run_all_challenges "$PLATFORM"
    elif [ ${#CHALLENGES[@]} -gt 0 ]; then
        for challenge in "${CHALLENGES[@]}"; do
            run_challenge "$challenge" "$PLATFORM"
        done
    else
        usage
        exit 1
    fi
}

main "$@"
