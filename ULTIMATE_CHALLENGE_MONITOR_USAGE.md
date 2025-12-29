# Ultimate Challenge Monitor - Usage Guide

## Overview

The Ultimate Challenge Monitor is a real-time monitoring system that continuously watches the ultimate challenge log file and automatically updates the OpenCode configuration when new providers or models are discovered.

## Features

- **Real-time Monitoring**: Automatically detects changes in the challenge log file
- **Automatic Configuration Updates**: Regenerates OpenCode configuration when new data is discovered
- **Progress Tracking**: Maintains history of discovered providers and models
- **Backup System**: Automatically creates backups of previous configurations
- **Security**: Sets restrictive file permissions (600) on generated configurations
- **Daemon Mode**: Can run as a background service

## Quick Start

### 1. Generate Initial Configuration

```bash
# Generate the initial configuration from current challenge logs
python3 generate_ultimate_opencode_optimized.py \
    --log-file ultimate_challenge_complete.log \
    --env-file .env \
    --output ultimate_opencode_final.json \
    --validate \
    --pretty \
    --backup
```

### 2. Start Monitoring

```bash
# Start monitoring in foreground (recommended for testing)
python3 monitor_ultimate_challenge.py \
    --log-file ultimate_challenge_complete.log \
    --env-file .env \
    --output ultimate_opencode_final.json

# Or start as daemon (background process)
python3 monitor_ultimate_challenge.py \
    --log-file ultimate_challenge_complete.log \
    --env-file .env \
    --output ultimate_opencode_final.json \
    --daemon
```

### 3. Check Status

```bash
# Check if daemon is running
cat /tmp/ultimate_challenge_monitor.pid

# View recent logs
tail -f ultimate_challenge_complete.log

# Check generated configuration
ls -la ultimate_opencode_final.json*
```

## Command Line Options

### generate_ultimate_opencode_optimized.py

| Option | Description | Default |
|--------|-------------|---------|
| `--log-file` | Challenge log file to parse | `ultimate_challenge_complete.log` |
| `--env-file` | Environment file with API keys | `.env` |
| `--output` | Output configuration file | `ultimate_opencode_final.json` |
| `--validate` | Validate generated configuration | `False` |
| `--pretty` | Pretty print JSON output | `False` |
| `--backup` | Create backup of previous configuration | `False` |
| `--progress` | Show progress for large files | `False` |

### monitor_ultimate_challenge.py

| Option | Description | Default |
|--------|-------------|---------|
| `--log-file` | Challenge log file to monitor | `ultimate_challenge_complete.log` |
| `--env-file` | Environment file with API keys | `.env` |
| `--output` | Output configuration file | `ultimate_opencode_final.json` |
| `--daemon` | Run as daemon (background process) | `False` |
| `--pid-file` | PID file for daemon mode | `/tmp/ultimate_challenge_monitor.pid` |

## Usage Examples

### Basic Monitoring

```bash
# Start monitoring the challenge log
python3 monitor_ultimate_challenge.py

# The monitor will:
# 1. Generate initial configuration
# 2. Watch for changes in ultimate_challenge_complete.log
# 3. Automatically update ultimate_opencode_final.json when new data is found
# 4. Create backups of previous configurations
```

### Monitoring with Custom Files

```bash
# Use custom log file and output location
python3 monitor_ultimate_challenge.py \
    --log-file /var/log/challenge/ultimate_challenge.log \
    --env-file /secure/location/.env \
    --output /config/ultimate_opencode_final.json
```

### Daemon Mode

```bash
# Start as daemon
python3 monitor_ultimate_challenge.py --daemon

# Check if running
ps aux | grep monitor_ultimate_challenge

# Stop daemon (find PID from pidfile)
kill $(cat /tmp/ultimate_challenge_monitor.pid)
```

### Manual Configuration Update

```bash
# Force a manual update (useful if log file changed while monitor wasn't running)
python3 generate_ultimate_opencode_optimized.py \
    --log-file ultimate_challenge_complete.log \
    --env-file .env \
    --output ultimate_opencode_final.json \
    --validate \
    --pretty \
    --backup
```

## Monitoring Output

When the monitor detects changes, you'll see output like:

```
2025-12-29 13:43:51,588 - INFO - Configuration saved to: ultimate_opencode_final.json
2025-12-29 13:43:51,588 - INFO - Generation #1 completed successfully
2025-12-29 13:43:51,588 - INFO - Total providers: 32
2025-12-29 13:43:51,588 - INFO - Total models: 9
2025-12-29 13:43:51,588 - INFO - ============================================================
2025-12-29 13:43:51,588 - INFO - CONFIGURATION GENERATION #1 SUMMARY
2025-12-29 13:43:51,588 - INFO - ============================================================
2025-12-29 13:43:51,588 - INFO - Total Providers: 32
2025-12-29 13:43:51,588 - INFO - Registered Providers: 32
2025-12-29 13:43:51,588 - INFO - Verified Providers: 0
2025-12-29 13:43:51,588 - INFO - Total Models: 9
2025-12-29 13:43:51,588 - INFO - Generated at: 2025-12-29T13:43:51.587076
2025-12-29 13:43:51,588 - INFO - Challenge-based Discovery: True
2025-12-29 13:43:51,588 - INFO - Configuration Version: 2.0-optimized
2025-12-29 13:43:51,588 - INFO - Discovered Providers (32): anthropic, baseten, cerebras, chutes, cloudflare, codestral, deepseek, fireworks, gemini, groq...
2025-12-29 13:43:51,588 - INFO - ============================================================
```

## Generated Configuration

The monitor generates a comprehensive OpenCode configuration that includes:

- **32+ Providers**: All discovered LLM providers with API keys
- **Model Groups**: Premium, balanced, fast, and free model categories
- **Feature Flags**: Streaming, tool calling, vision, embeddings, MCP, LSP, ACP support
- **Security Warnings**: Embedded warnings about API keys
- **Metadata**: Generation timestamp, version, discovery statistics
- **Backup Files**: Previous configurations are automatically backed up

## Security Features

- **File Permissions**: Generated configurations have 600 permissions (owner read/write only)
- **API Key Protection**: Real API keys are embedded with security warnings
- **Backup System**: Previous configurations are preserved
- **Safe Defaults**: Unknown providers get placeholder API keys

## Troubleshooting

### Monitor Not Detecting Changes

1. Check if the log file exists and is being written to:
   ```bash
   ls -la ultimate_challenge_complete.log
   tail -f ultimate_challenge_complete.log
   ```

2. Verify file permissions:
   ```bash
   chmod 644 ultimate_challenge_complete.log
   ```

3. Check if the challenge is still running:
   ```bash
   ps aux | grep ultimate
   ```

### Configuration Not Updating

1. Check monitor logs:
   ```bash
   # If running in foreground, check terminal output
   # If running as daemon, check system logs
   journalctl -f | grep monitor_ultimate
   ```

2. Force manual update:
   ```bash
   python3 generate_ultimate_opencode_optimized.py --validate --pretty --backup
   ```

### Permission Issues

1. Ensure write permissions for output directory:
   ```bash
   chmod 755 .
   ls -la ultimate_opencode_final.json
   ```

2. Check .env file permissions:
   ```bash
   chmod 600 .env
   ```

## Integration with OpenCode

The generated configuration can be used directly with OpenCode:

```bash
# Copy to OpenCode config directory
cp ultimate_opencode_final.json ~/.opencode/config.json

# Or use with OpenCode CLI
opencode --config ultimate_opencode_final.json

# Or set environment variable
export OPENCODE_CONFIG=ultimate_opencode_final.json
```

## Performance

- **Efficient Parsing**: Optimized for large log files (>1MB)
- **Chunked Processing**: Handles files up to several GB efficiently
- **Memory Efficient**: Uses streaming for large files
- **Fast Updates**: Typical update time < 1 second

## Dependencies

```bash
# Install required packages
pip install watchdog

# For daemon mode
pip install python-daemon
```

## Files Generated

- `ultimate_opencode_final.json` - Main configuration file
- `ultimate_opencode_final.json.YYYYMMDD_HHMMSS.backup` - Backup files
- `ultimate_challenge_complete.log.progress.json` - Progress tracking data
- `/tmp/ultimate_challenge_monitor.pid` - PID file (daemon mode)

## Best Practices

1. **Start with Validation**: Always use `--validate` flag for initial generation
2. **Enable Backups**: Use `--backup` to preserve previous configurations
3. **Monitor Logs**: Keep an eye on the challenge log for activity
4. **Check Permissions**: Ensure proper file permissions (600 for config files)
5. **Use Daemon Mode**: For production, use daemon mode with proper logging
6. **Regular Updates**: Even if monitor is running, occasionally force manual updates

## Advanced Usage

### Custom Model Groups

Edit the generated configuration to add custom model groups:

```json
"model_groups": {
    "custom_group": ["gpt-4", "claude-3-sonnet"],
    "enterprise": ["gpt-4-turbo", "claude-3-opus"]
}
```

### Provider-Specific Configuration

Add custom settings for specific providers:

```json
"provider": {
    "openai": {
        "options": {
            "apiKey": "your-key-here",
            "baseURL": "https://api.openai.com/v1",
            "custom_setting": "value"
        }
    }
}
```

### Feature Toggles

Enable/disable specific features:

```json
"features": {
    "streaming": true,
    "tool_calling": false,
    "vision": true
}
```

This monitoring system ensures your OpenCode configuration stays up-to-date with the latest challenge discoveries while maintaining security and providing comprehensive provider and model coverage.