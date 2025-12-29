#!/usr/bin/env python3
"""
Ultimate Challenge Monitor - Real-time OpenCode Configuration Updater

This script continuously monitors the ultimate challenge log file and automatically
updates the OpenCode configuration when new providers or models are discovered.
"""

import os
import sys
import time
import json
import logging
from datetime import datetime
from pathlib import Path
from typing import Dict, Any, Optional
import signal
import threading
from watchdog.observers import Observer
from watchdog.events import FileSystemEventHandler

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)

class ChallengeLogHandler(FileSystemEventHandler):
    """File system event handler for challenge log changes"""
    
    def __init__(self, callback):
        self.callback = callback
        self.last_modified = 0
        
    def on_modified(self, event):
        if event.is_directory:
            return
            
        if event.src_path.endswith('.log'):
            # Check if file was actually modified (not just accessed)
            try:
                current_modified = os.path.getmtime(event.src_path)
                if current_modified > self.last_modified:
                    self.last_modified = current_modified
                    logger.info(f"Log file modified: {event.src_path}")
                    self.callback()
            except Exception as e:
                logger.error(f"Error checking file modification: {e}")

class UltimateChallengeMonitor:
    """Real-time monitor for ultimate challenge progress"""
    
    def __init__(self, log_file: str, output_file: str, env_file: str = ".env"):
        self.log_file = log_file
        self.output_file = output_file
        self.env_file = env_file
        self.running = False
        self.observer = None
        self.last_log_size = 0
        self.generation_count = 0
        
    def start(self):
        """Start monitoring the challenge log file"""
        if self.running:
            return
            
        self.running = True
        logger.info(f"Starting Ultimate Challenge Monitor")
        logger.info(f"Monitoring: {self.log_file}")
        logger.info(f"Output: {self.output_file}")
        
        # Initial generation
        self.generate_config()
        
        # Set up file system watcher
        self.setup_file_watcher()
        
        # Set up signal handlers for graceful shutdown
        signal.signal(signal.SIGINT, self.signal_handler)
        signal.signal(signal.SIGTERM, self.signal_handler)
        
        logger.info("Monitor started. Press Ctrl+C to stop.")
        
        try:
            while self.running:
                time.sleep(1)
        except KeyboardInterrupt:
            logger.info("Received interrupt signal")
        finally:
            self.stop()
    
    def stop(self):
        """Stop monitoring"""
        logger.info("Stopping Ultimate Challenge Monitor")
        self.running = False
        
        if self.observer:
            self.observer.stop()
            self.observer.join()
            
        logger.info("Monitor stopped")
    
    def signal_handler(self, signum, frame):
        """Handle shutdown signals"""
        logger.info(f"Received signal {signum}")
        self.stop()
        sys.exit(0)
    
    def setup_file_watcher(self):
        """Set up file system watcher for the log file"""
        if not os.path.exists(self.log_file):
            logger.warning(f"Log file does not exist: {self.log_file}")
            return
            
        # Get directory and filename
        log_dir = os.path.dirname(self.log_file) or '.'
        log_filename = os.path.basename(self.log_file)
        
        # Create event handler
        event_handler = ChallengeLogHandler(self.generate_config)
        
        # Create observer
        self.observer = Observer()
        self.observer.schedule(event_handler, log_dir, recursive=False)
        
        # Start observer in separate thread
        self.observer.start()
        logger.info(f"File watcher started for {log_dir}/{log_filename}")
    
    def generate_config(self):
        """Generate configuration from current log state"""
        logger.info("Generating configuration from challenge logs...")
        
        try:
            # Import the optimized generator
            from generate_ultimate_opencode_optimized import (
                OptimizedChallengeLogParser, 
                OptimizedOpenCodeConfigGenerator, 
                validate_opencode_config
            )
            
            # Parse challenge logs
            parser = OptimizedChallengeLogParser(self.log_file)
            challenge_data = parser.parse_log_file()
            
            if not challenge_data:
                logger.error("No data extracted from challenge logs")
                return False
            
            logger.info(f"Extracted {challenge_data['total_providers']} providers with {challenge_data['total_models']} models")
            
            # Generate configuration
            generator = OptimizedOpenCodeConfigGenerator(self.env_file)
            config = generator.generate_config(challenge_data)
            
            # Validate configuration
            if not validate_opencode_config(config):
                logger.error("Configuration validation failed")
                return False
            
            # Create backup of previous configuration
            if os.path.exists(self.output_file):
                backup_file = f"{self.output_file}.{datetime.now().strftime('%Y%m%d_%H%M%S')}.backup"
                try:
                    os.rename(self.output_file, backup_file)
                    logger.info(f"Created backup: {backup_file}")
                except Exception as e:
                    logger.warning(f"Could not create backup: {e}")
            
            # Save configuration
            with open(self.output_file, 'w', encoding='utf-8') as f:
                json.dump(config, f, indent=2, sort_keys=True)
            
            # Set restrictive permissions
            os.chmod(self.output_file, 0o600)
            
            self.generation_count += 1
            
            logger.info(f"Configuration saved to: {self.output_file}")
            logger.info(f"Generation #{self.generation_count} completed successfully")
            logger.info(f"Total providers: {len(config.get('provider', {}))}")
            logger.info(f"Total models: {sum(len(p.get('models', {})) for p in config.get('provider', {}).values())}")
            
            # Display summary
            self.display_generation_summary(config, challenge_data)
            
            return True
            
        except Exception as e:
            logger.error(f"Error generating configuration: {e}")
            return False
    
    def display_generation_summary(self, config: Dict[str, Any], challenge_data: Dict[str, Any]):
        """Display a summary of the generated configuration"""
        providers = config.get("provider", {})
        total_providers = len(providers)
        total_models = sum(len(p.get("models", {})) for p in providers.values())
        
        metadata = config.get("metadata", {})
        verified_providers = metadata.get("verified_providers", 0)
        registered_providers = metadata.get("registered_providers", 0)
        
        logger.info("=" * 60)
        logger.info(f"CONFIGURATION GENERATION #{self.generation_count} SUMMARY")
        logger.info("=" * 60)
        logger.info(f"Total Providers: {total_providers}")
        logger.info(f"Registered Providers: {registered_providers}")
        logger.info(f"Verified Providers: {verified_providers}")
        logger.info(f"Total Models: {total_models}")
        logger.info(f"Generated at: {metadata.get('generated_at', 'unknown')}")
        logger.info(f"Challenge-based Discovery: {metadata.get('challenge_based', False)}")
        
        # Show discovered providers
        discovered = challenge_data.get('discovered_providers', [])
        logger.info(f"Discovered Providers ({len(discovered)}): {', '.join(sorted(discovered)[:15])}{'...' if len(discovered) > 15 else ''}")
        
        logger.info("=" * 60)

class ChallengeProgressTracker:
    """Track challenge progress over time"""
    
    def __init__(self, log_file: str):
        self.log_file = log_file
        self.progress_file = f"{log_file}.progress.json"
        self.progress_data = self.load_progress()
    
    def load_progress(self) -> Dict[str, Any]:
        """Load previous progress data"""
        if os.path.exists(self.progress_file):
            try:
                with open(self.progress_file, 'r') as f:
                    return json.load(f)
            except Exception as e:
                logger.warning(f"Could not load progress data: {e}")
        
        return {
            "created_at": datetime.now().isoformat(),
            "updates": [],
            "total_providers_discovered": 0,
            "total_models_discovered": 0
        }
    
    def save_progress(self):
        """Save progress data"""
        try:
            with open(self.progress_file, 'w') as f:
                json.dump(self.progress_data, f, indent=2)
        except Exception as e:
            logger.error(f"Could not save progress data: {e}")
    
    def record_update(self, providers: int, models: int):
        """Record a progress update"""
        update = {
            "timestamp": datetime.now().isoformat(),
            "providers": providers,
            "models": models
        }
        
        self.progress_data["updates"].append(update)
        self.progress_data["total_providers_discovered"] = providers
        self.progress_data["total_models_discovered"] = models
        
        self.save_progress()
        
        # Log progress if there are previous updates
        if len(self.progress_data["updates"]) > 1:
            prev_update = self.progress_data["updates"][-2]
            provider_diff = providers - prev_update["providers"]
            model_diff = models - prev_update["models"]
            
            if provider_diff > 0 or model_diff > 0:
                logger.info(f"Progress update: +{provider_diff} providers, +{model_diff} models")

def main():
    """Main function"""
    import argparse
    
    parser = argparse.ArgumentParser(description="Ultimate Challenge Monitor - Real-time OpenCode Configuration Updater")
    parser.add_argument("--log-file", default="ultimate_challenge_complete.log", 
                       help="Challenge log file to monitor")
    parser.add_argument("--env-file", default=".env", 
                       help="Environment file with API keys")
    parser.add_argument("--output", default="ultimate_opencode_final.json", 
                       help="Output configuration file")
    parser.add_argument("--daemon", action="store_true", 
                       help="Run as daemon (background process)")
    parser.add_argument("--pid-file", default="/tmp/ultimate_challenge_monitor.pid", 
                       help="PID file for daemon mode")
    
    args = parser.parse_args()
    
    # Check if running as daemon
    if args.daemon:
        import daemon
        
        logger.info("Starting in daemon mode...")
        
        # Create daemon context
        with daemon.DaemonContext(
            pidfile=daemon.pidfile.PIDLockFile(args.pid_file),
            working_directory=os.getcwd()
        ):
            monitor = UltimateChallengeMonitor(args.log_file, args.output, args.env_file)
            monitor.start()
    else:
        # Run in foreground
        monitor = UltimateChallengeMonitor(args.log_file, args.output, args.env_file)
        monitor.start()

if __name__ == "__main__":
    main()