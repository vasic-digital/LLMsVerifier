#!/bin/bash

# LLM Verifier Release Automation Script
# Handles automated releases, changelogs, and version management

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Get current version from git tags
get_current_version() {
    git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0"
}

# Get next version based on conventional commits
get_next_version() {
    local current_version=$1
    local bump_type=$2

    # Remove 'v' prefix if present
    current_version=${current_version#v}

    IFS='.' read -r major minor patch <<< "$current_version"

    case $bump_type in
        major)
            major=$((major + 1))
            minor=0
            patch=0
            ;;
        minor)
            minor=$((minor + 1))
            patch=0
            ;;
        patch|*)
            patch=$((patch + 1))
            ;;
    esac

    echo "v$major.$minor.$patch"
}

# Analyze commits to determine version bump
analyze_commits() {
    local since_tag=$1

    # Check for breaking changes
    if git log "$since_tag..HEAD" --grep="BREAKING CHANGE" --oneline | grep -q "BREAKING CHANGE"; then
        echo "major"
        return
    fi

    # Check for feat commits (minor version)
    if git log "$since_tag..HEAD" --grep="^feat:" --oneline | head -1 | grep -q "feat:"; then
        echo "minor"
        return
    fi

    # Default to patch
    echo "patch"
}

# Generate changelog
generate_changelog() {
    local version=$1
    local previous_tag=$2

    log_info "Generating changelog for $version..."

    cat > CHANGELOG.md.new << EOF
# Changelog

## [$version] - $(date +%Y-%m-%d)

EOF

    # Add sections for different types of changes
    local sections=("### 🚀 Features" "### 🐛 Bug Fixes" "### 📚 Documentation" "### 🎨 Style" "### ♻️ Refactor" "### ⚡ Performance" "### ✅ Tests" "### 🔧 Build" "### 🔒 Security")

    for section in "${sections[@]}"; do
        local section_content=""
        case $section in
            "### 🚀 Features")
                section_content=$(git log "$previous_tag..HEAD" --grep="^feat:" --pretty=format:"* %s (%h)" || true)
                ;;
            "### 🐛 Bug Fixes")
                section_content=$(git log "$previous_tag..HEAD" --grep="^fix:" --pretty=format:"* %s (%h)" || true)
                ;;
            "### 📚 Documentation")
                section_content=$(git log "$previous_tag..HEAD" --grep="^docs:" --pretty=format:"* %s (%h)" || true)
                ;;
            "### 🎨 Style")
                section_content=$(git log "$previous_tag..HEAD" --grep="^style:" --pretty=format:"* %s (%h)" || true)
                ;;
            "### ♻️ Refactor")
                section_content=$(git log "$previous_tag..HEAD" --grep="^refactor:" --pretty=format:"* %s (%h)" || true)
                ;;
            "### ⚡ Performance")
                section_content=$(git log "$previous_tag..HEAD" --grep="^perf:" --pretty=format:"* %s (%h)" || true)
                ;;
            "### ✅ Tests")
                section_content=$(git log "$previous_tag..HEAD" --grep="^test:" --pretty=format:"* %s (%h)" || true)
                ;;
            "### 🔧 Build")
                section_content=$(git log "$previous_tag..HEAD" --grep="^build:" --pretty=format:"* %s (%h)" || true)
                ;;
            "### 🔒 Security")
                section_content=$(git log "$previous_tag..HEAD" --grep="^security:" --pretty=format:"* %s (%h)" || true)
                ;;
        esac

        if [ -n "$section_content" ]; then
            echo "$section" >> CHANGELOG.md.new
            echo "$section_content" >> CHANGELOG.md.new
            echo "" >> CHANGELOG.md.new
        fi
    done

    # Add breaking changes section
    local breaking_changes=$(git log "$previous_tag..HEAD" --grep="BREAKING CHANGE" --pretty=format:"* %s (%h)" || true)
    if [ -n "$breaking_changes" ]; then
        echo "### 💥 Breaking Changes" >> CHANGELOG.md.new
        echo "$breaking_changes" >> CHANGELOG.md.new
        echo "" >> CHANGELOG.md.new
    fi

    # Append existing changelog
    if [ -f CHANGELOG.md ]; then
        echo "" >> CHANGELOG.md.new
        cat CHANGELOG.md >> CHANGELOG.md.new
    fi

    mv CHANGELOG.md.new CHANGELOG.md
    log_success "Changelog generated"
}

# Update version in code
update_version() {
    local version=$1

    log_info "Updating version to $version..."

    # Update version.go if it exists
    if [ -f "version.go" ]; then
        sed -i "s/Version.*=.*\".*\"/Version = \"$version\"/g" version.go
    fi

    # Update package.json if it exists
    if [ -f "package.json" ]; then
        # This would require jq or similar for JSON editing
        log_warning "Please manually update version in package.json"
    fi

    log_success "Version updated"
}

# Build release artifacts
build_artifacts() {
    local version=$1

    log_info "Building release artifacts for $version..."

    # Create build directory
    mkdir -p release

    # Build for multiple platforms
    GOOS=linux GOARCH=amd64 go build -ldflags "-X main.Version=$version" -o "release/llm-verifier-linux-amd64" ./cmd
    GOOS=darwin GOARCH=amd64 go build -ldflags "-X main.Version=$version" -o "release/llm-verifier-darwin-amd64" ./cmd
    GOOS=darwin GOARCH=arm64 go build -ldflags "-X main.Version=$version" -o "release/llm-verifier-darwin-arm64" ./cmd
    GOOS=windows GOARCH=amd64 go build -ldflags "-X main.Version=$version" -o "release/llm-verifier-windows-amd64.exe" ./cmd

    # Create archives
    cd release
    tar -czf "llm-verifier-$version-linux-amd64.tar.gz" llm-verifier-linux-amd64
    tar -czf "llm-verifier-$version-darwin-amd64.tar.gz" llm-verifier-darwin-amd64
    tar -czf "llm-verifier-$version-darwin-arm64.tar.gz" llm-verifier-darwin-arm64
    zip "llm-verifier-$version-windows-amd64.zip" llm-verifier-windows-amd64.exe

    cd ..
    log_success "Release artifacts built"
}

# Create GitHub release
create_github_release() {
    local version=$1
    local previous_tag=$2

    if ! command -v gh &> /dev/null; then
        log_warning "GitHub CLI not found, skipping GitHub release creation"
        return
    fi

    log_info "Creating GitHub release..."

    # Generate release notes
    local release_notes="## Changes

$(git log "$previous_tag..HEAD" --pretty=format:"* %s (%h)" | head -20)

### Installation

\`\`\`bash
# Download the appropriate binary for your platform
wget https://github.com/your-org/llm-verifier/releases/download/$version/llm-verifier-$version-linux-amd64.tar.gz
tar -xzf llm-verifier-$version-linux-amd64.tar.gz
./llm-verifier-linux-amd64 --help
\`\`\`

### Docker

\`\`\`bash
docker run -p 8080:8080 ghcr.io/your-org/llm-verifier:$version
\`\`\`
"

    # Create release
    gh release create "$version" \
        --title "Release $version" \
        --notes "$release_notes" \
        release/*.tar.gz release/*.zip

    log_success "GitHub release created"
}

# Validate release
validate_release() {
    local version=$1

    log_info "Validating release..."

    # Run tests
    if ! make test-all; then
        log_error "Tests failed"
        exit 1
    fi

    # Run security checks
    if ! make security-scan; then
        log_error "Security checks failed"
        exit 1
    fi

    # Check if tag exists
    if git tag | grep -q "^$version$"; then
        log_error "Tag $version already exists"
        exit 1
    fi

    log_success "Release validation passed"
}

# Main release function
main() {
    local bump_type=""
    local custom_version=""
    local skip_validation=false
    local dry_run=false

    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            --major)
                bump_type="major"
                shift
                ;;
            --minor)
                bump_type="minor"
                shift
                ;;
            --patch)
                bump_type="patch"
                shift
                ;;
            --version=*)
                custom_version="${1#*=}"
                shift
                ;;
            --skip-validation)
                skip_validation=true
                shift
                ;;
            --dry-run)
                dry_run=true
                shift
                ;;
            --help)
                echo "Usage: $0 [OPTIONS]"
                echo ""
                echo "Options:"
                echo "  --major           Major version bump"
                echo "  --minor           Minor version bump"
                echo "  --patch           Patch version bump"
                echo "  --version=VER     Custom version"
                echo "  --skip-validation Skip validation checks"
                echo "  --dry-run         Show what would be done"
                echo "  --help            Show this help"
                exit 0
                ;;
            *)
                log_error "Unknown option: $1"
                exit 1
                ;;
        esac
    done

    cd "$PROJECT_ROOT"

    # Get current version
    local current_version=$(get_current_version)
    log_info "Current version: $current_version"

    # Determine next version
    local next_version
    if [ -n "$custom_version" ]; then
        next_version="$custom_version"
    elif [ -n "$bump_type" ]; then
        next_version=$(get_next_version "$current_version" "$bump_type")
    else
        # Auto-detect from commits
        bump_type=$(analyze_commits "$current_version")
        next_version=$(get_next_version "$current_version" "$bump_type")
    fi

    log_info "Next version: $next_version (bump: $bump_type)"

    if [ "$dry_run" = true ]; then
        echo "DRY RUN - Would perform the following actions:"
        echo "1. Validate release"
        echo "2. Generate changelog"
        echo "3. Update version to $next_version"
        echo "4. Build release artifacts"
        echo "5. Create git tag $next_version"
        echo "6. Create GitHub release"
        exit 0
    fi

    # Validate release
    if [ "$skip_validation" = false ]; then
        validate_release "$next_version"
    fi

    # Generate changelog
    generate_changelog "$next_version" "$current_version"

    # Update version
    update_version "$next_version"

    # Build artifacts
    build_artifacts "$next_version"

    # Commit changes
    log_info "Committing changes..."
    git add .
    git commit -m "chore: release $next_version

- Update version to $next_version
- Update changelog
- Build release artifacts"

    # Create tag
    log_info "Creating git tag..."
    git tag -a "$next_version" -m "Release $next_version"

    # Push changes
    log_info "Pushing to remote..."
    git push origin main
    git push origin "$next_version"

    # Create GitHub release
    create_github_release "$next_version" "$current_version"

    log_success "Release $next_version completed successfully!"
    log_info "Artifacts available in: release/"
    log_info "GitHub Release: https://github.com/your-org/llm-verifier/releases/tag/$next_version"
}

# Run main function with all arguments
main "$@"