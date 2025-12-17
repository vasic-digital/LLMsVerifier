#!/bin/bash
# LLM Verifier Database Restore Script
# This script restores the SQLite database from encrypted/compressed backups

set -euo pipefail

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
BACKUP_DIR="${BACKUP_DIR:-${PROJECT_ROOT}/backups}"
DATABASE_PATH="${DATABASE_PATH:-${PROJECT_ROOT}/data/llm-verifier.db}"
ENCRYPTION_KEY="${DATABASE_ENCRYPTION_KEY:-}"
RESTORE_DIR="${RESTORE_DIR:-${PROJECT_ROOT}/data}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $(date '+%Y-%m-%d %H:%M:%S') - $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $(date '+%Y-%m-%d %H:%M:%S') - $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $(date '+%Y-%m-%d %H:%M:%S') - $1" >&2
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $(date '+%Y-%m-%d %H:%M:%S') - $1"
}

# Validate dependencies
check_dependencies() {
    local missing_deps=()

    if ! command -v sqlite3 &> /dev/null; then
        missing_deps+=("sqlite3")
    fi

    if ! command -v openssl &> /dev/null; then
        missing_deps+=("openssl")
    fi

    if ! command -v gzip &> /dev/null; then
        missing_deps+=("gzip")
    fi

    if [ ${#missing_deps[@]} -ne 0 ]; then
        log_error "Missing required dependencies: ${missing_deps[*]}"
        exit 1
    fi
}

# Validate backup file
validate_backup_file() {
    local backup_file="$1"

    if [ ! -f "$backup_file" ]; then
        log_error "Backup file not found: $backup_file"
        exit 1
    fi

    # Check file extension
    if [[ "$backup_file" != *.db ]]; then
        log_error "Invalid backup file extension. Expected .db"
        exit 1
    fi

    # Check if file is readable
    if [ ! -r "$backup_file" ]; then
        log_error "Backup file is not readable: $backup_file"
        exit 1
    fi

    log_info "Backup file validation successful: $backup_file"
}

# Create restore directory
create_restore_dir() {
    if [ ! -d "$RESTORE_DIR" ]; then
        mkdir -p "$RESTORE_DIR"
        log_info "Created restore directory: $RESTORE_DIR"
    fi
}

# Extract backup
extract_backup() {
    local backup_file="$1"
    local temp_file="${backup_file}.extract"

    log_info "Extracting backup file: $backup_file"

    # Decompress backup
    if ! gzip -d -c "$backup_file" > "$temp_file"; then
        log_error "Failed to decompress backup file"
        rm -f "$temp_file"
        exit 1
    fi

    # Decrypt backup if encryption key is provided
    if [ -n "$ENCRYPTION_KEY" ]; then
        log_info "Decrypting backup file"
        local decrypted_file="${temp_file}.dec"

        if ! openssl enc -d -aes-256-cbc -in "$temp_file" -out "$decrypted_file" -k "$ENCRYPTION_KEY"; then
            log_error "Failed to decrypt backup"
            rm -f "$temp_file"
            exit 1
        fi

        rm -f "$temp_file"
        temp_file="$decrypted_file"
    fi

    echo "$temp_file"
}

# Validate extracted database
validate_extracted_database() {
    local db_file="$1"

    log_info "Validating extracted database: $db_file"

    # Check if file is a valid SQLite database
    if ! sqlite3 "$db_file" "PRAGMA integrity_check;" | grep -q "ok"; then
        log_error "Extracted database integrity check failed"
        return 1
    fi

    # Check if required tables exist
    local required_tables=("models" "providers" "verification_results" "issues" "events")
    for table in "${required_tables[@]}"; do
        if ! sqlite3 "$db_file" "SELECT name FROM sqlite_master WHERE type='table' AND name='$table';" | grep -q "$table"; then
            log_error "Required table '$table' not found in database"
            return 1
        fi
    done

    log_success "Database validation successful"
    return 0
}

# Create backup of current database
backup_current_database() {
    local current_db="$1"
    local backup_suffix="${2:-pre-restore-$(date '+%Y%m%d_%H%M%S')}"

    if [ -f "$current_db" ]; then
        local backup_file="${current_db}.${backup_suffix}"

        log_info "Creating backup of current database: $backup_file"

        if ! cp "$current_db" "$backup_file"; then
            log_error "Failed to backup current database"
            exit 1
        fi

        log_success "Current database backed up: $backup_file"
        echo "$backup_file"
    else
        log_info "No existing database to backup"
        echo ""
    fi
}

# Perform database restore
perform_restore() {
    local source_db="$1"
    local target_db="$2"

    log_info "Performing database restore"
    log_info "Source: $source_db"
    log_info "Target: $target_db"

    # Stop any running processes that might be using the database
    # This is a simple approach - in production, you might want to use proper process management
    log_warn "Ensure no processes are currently using the database"

    # Create target directory if it doesn't exist
    local target_dir=$(dirname "$target_db")
    mkdir -p "$target_dir"

    # Perform the restore using SQLite .restore command
    if ! sqlite3 "$target_db" ".restore '$source_db'"; then
        log_error "Database restore failed"
        return 1
    fi

    # Set proper permissions
    chmod 600 "$target_db"

    log_success "Database restore completed successfully"
    return 0
}

# Verify restore
verify_restore() {
    local restored_db="$1"
    local original_backup="$2"

    log_info "Verifying restore integrity"

    # Run integrity check
    if ! sqlite3 "$restored_db" "PRAGMA integrity_check;" | grep -q "ok"; then
        log_error "Restored database integrity check failed"
        return 1
    fi

    # Compare some basic statistics (this is optional and depends on your needs)
    log_info "Restore verification completed"
    return 0
}

# Generate restore report
generate_restore_report() {
    local backup_file="$1"
    local restored_db="$2"
    local start_time="$3"
    local end_time=$(date '+%Y-%m-%d %H:%M:%S')
    local duration=$(( $(date -d "$end_time" '+%s') - $(date -d "$start_time" '+%s') ))

    local report_file="${restored_db}.restore-report"

    cat << EOF > "$report_file"
LLM Verifier Database Restore Report
=====================================

Restore Details:
- Backup File: $backup_file
- Restored Database: $restored_db
- Start Time: $start_time
- End Time: $end_time
- Duration: ${duration}s
- Encrypted: $([ -n "$ENCRYPTION_KEY" ] && echo "Yes" || echo "No")

Database Statistics After Restore:
$(sqlite3 "$restored_db" << 'SQL'
.mode list
SELECT 'Models: ' || COUNT(*) FROM models;
SELECT 'Providers: ' || COUNT(*) FROM providers;
SELECT 'Verifications: ' || COUNT(*) FROM verification_results;
SELECT 'Issues: ' || COUNT(*) FROM issues;
SELECT 'Events: ' || COUNT(*) FROM events;
SQL
)

System Information:
- Hostname: $(hostname)
- User: $(whoami)
- Restore Script: $(basename "$0")
- Working Directory: $(pwd)

Restore Verification: PASSED
EOF

    log_info "Restore report generated: $report_file"
}

# List available backups
list_backups() {
    log_info "Available backups in $BACKUP_DIR:"

    if [ ! -d "$BACKUP_DIR" ]; then
        log_error "Backup directory does not exist: $BACKUP_DIR"
        exit 1
    fi

    local count=0
    while IFS= read -r -d '' backup_file; do
        local size=$(stat -f%z "$backup_file" 2>/dev/null || stat -c%s "$backup_file")
        local mtime=$(stat -f%Sm -t "%Y-%m-%d %H:%M:%S" "$backup_file" 2>/dev/null || stat -c"%y" "$backup_file" | cut -d'.' -f1)
        echo "  $backup_file ($(numfmt --to=iec-i --suffix=B $size)) - $mtime"
        ((count++))
    done < <(find "$BACKUP_DIR" -name "llm-verifier-backup-*.db" -print0 | sort -z)

    if [ $count -eq 0 ]; then
        log_info "No backup files found"
    else
        log_info "Found $count backup file(s)"
    fi
}

# Show usage
usage() {
    cat << EOF
LLM Verifier Database Restore Script

Usage:
  $0 [OPTIONS] BACKUP_FILE

Options:
  -l, --list              List available backups
  -t, --target PATH       Target database path (default: $DATABASE_PATH)
  -b, --backup-dir DIR    Backup directory (default: $BACKUP_DIR)
  -r, --restore-dir DIR   Restore directory (default: $RESTORE_DIR)
  -k, --key KEY           Encryption key
  -h, --help              Show this help

Examples:
  $0 --list
  $0 /path/to/backup.db
  $0 --target /custom/path/db.sqlite /path/to/backup.db

EOF
}

# Main restore function
main() {
    local backup_file=""
    local list_only=false

    # Parse command line arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            -l|--list)
                list_only=true
                shift
                ;;
            -t|--target)
                DATABASE_PATH="$2"
                shift 2
                ;;
            -b|--backup-dir)
                BACKUP_DIR="$2"
                shift 2
                ;;
            -r|--restore-dir)
                RESTORE_DIR="$2"
                shift 2
                ;;
            -k|--key)
                ENCRYPTION_KEY="$2"
                shift 2
                ;;
            -h|--help)
                usage
                exit 0
                ;;
            -*)
                log_error "Unknown option: $1"
                usage
                exit 1
                ;;
            *)
                backup_file="$1"
                shift
                ;;
        esac
    done

    # Handle list operation
    if [ "$list_only" = true ]; then
        list_backups
        exit 0
    fi

    # Validate backup file argument
    if [ -z "$backup_file" ]; then
        log_error "Backup file is required"
        usage
        exit 1
    fi

    local start_time=$(date '+%Y-%m-%d %H:%M:%S')

    log_info "Starting LLM Verifier database restore"
    log_info "Backup File: $backup_file"
    log_info "Target Database: $DATABASE_PATH"

    # Validate environment
    check_dependencies
    validate_backup_file "$backup_file"
    create_restore_dir

    # Extract backup
    local extracted_db=$(extract_backup "$backup_file")

    # Validate extracted database
    if ! validate_extracted_database "$extracted_db"; then
        rm -f "$extracted_db"
        exit 1
    fi

    # Backup current database if it exists
    local current_backup=$(backup_current_database "$DATABASE_PATH")

    # Perform restore
    if ! perform_restore "$extracted_db" "$DATABASE_PATH"; then
        # Restore the backup if restore failed
        if [ -n "$current_backup" ]; then
            log_warn "Restoring original database from backup"
            cp "$current_backup" "$DATABASE_PATH"
        fi
        rm -f "$extracted_db"
        exit 1
    fi

    # Verify restore
    if ! verify_restore "$DATABASE_PATH" "$backup_file"; then
        log_error "Restore verification failed"
        # Attempt to restore from backup
        if [ -n "$current_backup" ]; then
            log_warn "Restoring original database from backup"
            cp "$current_backup" "$DATABASE_PATH"
        fi
        rm -f "$extracted_db"
        exit 1
    fi

    # Generate report
    generate_restore_report "$backup_file" "$DATABASE_PATH" "$start_time"

    # Clean up
    rm -f "$extracted_db"

    log_success "Database restore completed successfully"
    log_info "Restored database: $DATABASE_PATH"

    if [ -n "$current_backup" ]; then
        log_info "Original database backed up: $current_backup"
    fi

    # Output for automation
    echo "RESTORED_DATABASE=$DATABASE_PATH"
}

# Run main function
main "$@"