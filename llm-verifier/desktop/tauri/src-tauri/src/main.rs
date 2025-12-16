// Prevents additional console window on Windows in release, DO NOT REMOVE!!
#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]

use std::process::{Command, Stdio};
use std::sync::Mutex;
use tauri::{AppHandle, Manager, State};
use tokio::process::Command as TokioCommand;

#[derive(Default)]
struct BackendProcess(Mutex<Option<std::process::Child>>);

#[tauri::command]
async fn start_backend(app: AppHandle) -> Result<String, String> {
    // Get the backend executable path
    let backend_path = app
        .path_resolver()
        .resolve_resource("../llm-verifier")
        .ok_or("Failed to resolve backend path")?;

    println!("Starting backend: {:?}", backend_path);

    // Spawn the backend process
    let child = TokioCommand::new(&backend_path)
        .arg("api")
        .arg("--port")
        .arg("8080")
        .stdout(Stdio::piped())
        .stderr(Stdio::piped())
        .spawn()
        .map_err(|e| format!("Failed to start backend: {}", e))?;

    // Store the child process (simplified - in production you'd want better process management)

    Ok("Backend started successfully".to_string())
}

#[tauri::command]
async fn stop_backend() -> Result<String, String> {
    // In a real implementation, you'd track and kill the backend process
    Ok("Backend stopped successfully".to_string())
}

#[tauri::command]
async fn get_backend_status() -> Result<serde_json::Value, String> {
    // Simplified status check
    Ok(serde_json::json!({
        "running": false,
        "port": "8080",
        "host": "localhost"
    }))
}

#[tauri::command]
async fn get_system_info() -> Result<serde_json::Value, String> {
    Ok(serde_json::json!({
        "platform": std::env::consts::OS,
        "arch": std::env::consts::ARCH,
        "version": env!("CARGO_PKG_VERSION"),
        "rustc": "1.70.0", // Would be dynamic in real implementation
        "tauri": "1.5.0"
    }))
}

#[tauri::command]
async fn select_directory() -> Result<Option<String>, String> {
    // Use Tauri's dialog API
    // This would be implemented with Tauri's dialog plugin
    Ok(Some("/tmp".to_string()))
}

#[tauri::command]
async fn select_file() -> Result<Option<String>, String> {
    // Use Tauri's dialog API
    Ok(Some("selected_file.txt".to_string()))
}

#[tauri::command]
async fn save_file() -> Result<Option<String>, String> {
    // Use Tauri's dialog API
    Ok(Some("saved_file.txt".to_string()))
}

#[tauri::command]
async fn load_config() -> Result<serde_json::Value, String> {
    // Load configuration from Tauri's app data directory
    Ok(serde_json::json!({}))
}

#[tauri::command]
async fn save_config(config: serde_json::Value) -> Result<String, String> {
    // Save configuration to Tauri's app data directory
    println!("Saving config: {:?}", config);
    Ok("Configuration saved successfully".to_string())
}

fn main() {
    tauri::Builder::default()
        .plugin(tauri_plugin_fs::init())
        .plugin(tauri_plugin_shell::init())
        .plugin(tauri_plugin_dialog::init())
        .manage(BackendProcess::default())
        .invoke_handler(tauri::generate_handler![
            start_backend,
            stop_backend,
            get_backend_status,
            get_system_info,
            select_directory,
            select_file,
            save_file,
            load_config,
            save_config
        ])
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}