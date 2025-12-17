#!/bin/bash
# LLM Verifier Database Backup Script
# This script creates automated backups of the SQLite database with encryption

set -euo pipefail

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
BACKUP_DIR="${BACKUP_DIR:-${PROJECT_ROOT}/backups}"
DATABASE_PATH="${DATABASE_PATH:-${PROJECT_ROOT}/data/llm-verifier.db}"
ENCRYPTION_KEY="${DATABASE_ENCRYPTION_KEY:-}"
RETENTION_DAYS="${RETENTION_DAYS:-30}"
COMPRESSION_LEVEL="${COMPRESSION_LEVEL:-6}"

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

# Validate database
validate_database() {
    if [ ! -f "$DATABASE_PATH" ]; then
        log_error "Database file not found: $DATABASE_PATH"
        exit 1
    fi

    # Check if database is valid SQLite
    if ! sqlite3 "$DATABASE_PATH" "PRAGMA integrity_check;" &> /dev/null; then
        log_error "Database integrity check failed"
        exit 1
    fi

    log_info "Database validation successful"
}

# Create backup directory
create_backup_dir() {
    if [ ! -d "$BACKUP_DIR" ]; then
        mkdir -p "$BACKUP_DIR"
        chmod 700 "$BACKUP_DIR"
        log_info "Created backup directory: $BACKUP_DIR"
    fi
}

# Generate backup filename
generate_backup_filename() {
    local timestamp=$(date '+%Y%m%d_%H%M%S')
    local hostname=$(hostname -s)
    echo "${BACKUP_DIR}/llm-verifier-backup-${hostname}-${timestamp}.db"
}

# Create database backup
create_backup() {
    local backup_file="$1"
    local temp_file="${backup_file}.tmp"

    log_info "Creating database backup: $backup_file"

    # Create SQLite backup
    if ! sqlite3 "$DATABASE_PATH" ".backup '$temp_file'"; then
        log_error "Failed to create SQLite backup"
        rm -f "$temp_file"
        exit 1
    fi

    # Encrypt backup if encryption key is provided
    if [ -n "$ENCRYPTION_KEY" ]; then
        log_info "Encrypting backup file"
        local encrypted_file="${backup_file}.enc"

        if ! openssl enc -aes-256-cbc -salt -in "$temp_file" -out "$encrypted_file" -k "$ENCRYPTION_KEY"; then
            log_error "Failed to encrypt backup"
            rm -f "$temp_file"
            exit 1
        fi

        rm -f "$temp_file"
        temp_file="$encrypted_file"
    fi

    # Compress backup
    log_info "Compressing backup file"
    if ! gzip -${COMPRESSION_LEVEL} "$temp_file"; then
        log_error "Failed to compress backup"
        rm -f "$temp_file"
        exit 1
    fi

    local final_file="${temp_file}.gz"
    mv "$final_file" "$backup_file"

    # Set secure permissions
    chmod 600 "$backup_file"

    log_success "Backup created successfully: $backup_file"
    echo "$backup_file"
}

# Verify backup integrity
verify_backup() {
    local backup_file="$1"

    log_info "Verifying backup integrity: $backup_file"

    # Decompress and decrypt if needed
    local temp_file="${backup_file}.verify"

    if [ -n "$ENCRYPTION_KEY" ]; then
        # Encrypted backup
        local decrypted_file="${temp_file}.dec"
        if ! gzip -d -c "$backup_file" | openssl enc -d -aes-256-cbc -in /dev/stdin -out "$decrypted_file" -k "$ENCRYPTION_KEY"; then
            log_error "Failed to decrypt backup for verification"
            rm -f "$decrypted_file"
            return 1
        fi
        temp_file="$decrypted_file"
    else
        # Unencrypted backup
        if ! gzip -d -c "$backup_file" > "$temp_file"; then
            log_error "Failed to decompress backup for verification"
            return 1
        fi
    fi

    # Verify SQLite integrity
    if ! sqlite3 "$temp_file" "PRAGMA integrity_check;" | grep -q "ok"; then
        log_error "Backup integrity check failed"
        rm -f "$temp_file"
        return 1
    fi

    rm -f "$temp_file"
    log_success "Backup verification successful"
    return 0
}

# Clean up old backups
cleanup_old_backups() {
    log_info "Cleaning up backups older than $RETENTION_DAYS days"

    local deleted_count=0
    local cutoff_date=$(date -d "$RETENTION_DAYS days ago" '+%Y%m%d')

    # Find and delete old backups
    while IFS= read -r -d '' backup_file; do
        local file_date=$(basename "$backup_file" | sed -n 's/.*backup-.*-\([0-9]\{8\}\)_[0-9]\{6\}.db/\1/p')

        if [ -n "$file_date" ] && [ "$file_date" -lt "$cutoff_date" ]; then
            log_info "Deleting old backup: $backup_file"
            rm -f "$backup_file"
            ((deleted_count++))
        fi
    done < <(find "$BACKUP_DIR" -name "llm-verifier-backup-*.db" -print0)

    if [ $deleted_count -gt 0 ]; then
        log_success "Cleaned up $deleted_count old backup(s)"
    else
        log_info "No old backups to clean up"
    fi
}

# Generate backup report
generate_report() {
    local backup_file="$1"
    local start_time="$2"
    local end_time=$(date '+%Y-%m-%d %H:%M:%S')
    local backup_size=$(stat -f%z "$backup_file" 2>/dev/null || stat -c%s "$backup_file")
    local duration=$(( $(date -d "$end_time" '+%s') - $(date -d "$start_time" '+%s') ))

    cat << EOF > "${backup_file}.report"
LLM Verifier Database Backup Report
====================================

Backup Details:
- Database: $DATABASE_PATH
- Backup File: $backup_file
- Backup Size: $(numfmt --to=iec-i --suffix=B $backup_size)
- Start Time: $start_time
- End Time: $end_time
- Duration: ${duration}s
- Encrypted: $([ -n "$ENCRYPTION_KEY" ] && echo "Yes" || echo "No")
- Compressed: Yes (gzip level $COMPRESSION_LEVEL)

Database Statistics:
$(sqlite3 "$DATABASE_PATH" << 'SQL'
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
- Backup Script: $(basename "$0")
- Working Directory: $(pwd)

Backup Verification: PASSED
EOF

    log_info "Backup report generated: ${backup_file}.report"
}

# Main backup function
main() {
    local start_time=$(date '+%Y-%m-%d %H:%M:%S')

    log_info "Starting LLM Verifier database backup"
    log_info "Database: $DATABASE_PATH"
    log_info "Backup Directory: $BACKUP_DIR"

    # Validate environment
    check_dependencies
    validate_database
    create_backup_dir

    # Generate backup filename
    local backup_file=$(generate_backup_filename)

    # Create backup
    backup_file=$(create_backup "$backup_file")

    # Verify backup
    if ! verify_backup "$backup_file"; then
        log_error "Backup verification failed"
        rm -f "$backup_file"
        exit 1
    fi

    # Generate report
    generate_report "$backup_file" "$start_time"

    # Cleanup old backups
    cleanup_old_backups

    log_success "Backup completed successfully"
    log_info "Backup file: $backup_file"
    log_info "Report file: ${backup_file}.report"

    # Output for automation
    echo "BACKUP_FILE=$backup_file"
}

# Run main function
main "$@"