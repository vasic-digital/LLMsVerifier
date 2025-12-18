# Production Build Artifacts

This directory contains production-ready binaries for multiple platforms.

## Build Information

- **Build Date**: $(date +%Y-%m-%d)
- **Git Commit**: $(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
- **Go Version**: $(go version | awk '{print $3}')
- **Build Flags**: -ldflags="-s -w" (stripped binaries)

## Available Binaries

| Platform | Architecture | Binary | Size |
|----------|-------------|---------|------|
| Linux | x86_64 (amd64) | llm-verifier-linux-amd64 | $(du -h build-artifacts/llm-verifier-linux-amd64 | cut -f1) |
| Linux | ARM64 (arm64) | llm-verifier-linux-arm64 | $(du -h build-artifacts/llm-verifier-linux-arm64 | cut -f1) |
| macOS | Intel (amd64) | llm-verifier-darwin-amd64 | $(du -h build-artifacts/llm-verifier-darwin-amd64 | cut -f1) |
| macOS | Apple Silicon (arm64) | llm-verifier-darwin-arm64 | $(du -h build-artifacts/llm-verifier-darwin-arm64 | cut -f1) |
| Windows | x86_64 (amd64) | llm-verifier-windows-amd64.exe | $(du -h build-artifacts/llm-verifier-windows-amd64.exe | cut -f1) |

## Usage

### Linux/macOS
```bash
chmod +x llm-verifier-<platform>-<arch>
./llm-verifier-<platform>-<arch> api --port 8080
```

### Windows
```cmd
llm-verifier-windows-amd64.exe api --port 8080
```

### Docker Alternative
```bash
docker run -p 8080:8080 ghcr.io/vasic-digital/llm-verifier:latest
```

## Verification

After deployment, verify the service:
```bash
# Health check
curl http://localhost:8080/health

# Version check
curl http://localhost:8080/version

# API documentation
open http://localhost:8080/swagger/index.html
```

## Security Notes

- All binaries are statically linked
- Symbols stripped for smaller size
- Production builds include security hardening
- Verify binary integrity before deployment

## Support

- Documentation: [../docs/](../docs/)
- Issues: [GitHub Issues](https://github.com/vasic-digital/LLMsVerifier/issues)
- Community: [Discussions](https://github.com/vasic-digital/LLMsVerifier/discussions)
