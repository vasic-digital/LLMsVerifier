import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';

class SettingsProvider with ChangeNotifier {
  final FlutterSecureStorage _storage = const FlutterSecureStorage();

  // Theme settings
  ThemeMode _themeMode = ThemeMode.system;
  ThemeMode get themeMode => _themeMode;

  // Language settings
  String _locale = 'en';
  String get locale => _locale;

  static const Map<String, String> supportedLanguages = {
    'en': 'English',
    'es': 'Spanish',
    'fr': 'French',
    'de': 'German',
    'zh': 'Chinese',
    'ja': 'Japanese',
  };

  // Notification settings
  bool _pushNotificationsEnabled = true;
  bool _emailNotificationsEnabled = true;
  bool _verificationAlertsEnabled = true;

  bool get pushNotificationsEnabled => _pushNotificationsEnabled;
  bool get emailNotificationsEnabled => _emailNotificationsEnabled;
  bool get verificationAlertsEnabled => _verificationAlertsEnabled;

  // Backup settings
  bool _autoBackupEnabled = false;
  String _backupFrequency = 'weekly';
  DateTime? _lastBackup;

  bool get autoBackupEnabled => _autoBackupEnabled;
  String get backupFrequency => _backupFrequency;
  DateTime? get lastBackup => _lastBackup;

  SettingsProvider() {
    _loadSettings();
  }

  Future<void> _loadSettings() async {
    try {
      // Load theme
      final storedTheme = await _storage.read(key: 'theme_mode');
      if (storedTheme != null) {
        _themeMode = ThemeMode.values.firstWhere(
          (mode) => mode.name == storedTheme,
          orElse: () => ThemeMode.system,
        );
      }

      // Load language
      final storedLocale = await _storage.read(key: 'locale');
      if (storedLocale != null) {
        _locale = storedLocale;
      }

      // Load notification settings
      final pushEnabled = await _storage.read(key: 'push_notifications');
      _pushNotificationsEnabled = pushEnabled != 'false';

      final emailEnabled = await _storage.read(key: 'email_notifications');
      _emailNotificationsEnabled = emailEnabled != 'false';

      final alertsEnabled = await _storage.read(key: 'verification_alerts');
      _verificationAlertsEnabled = alertsEnabled != 'false';

      // Load backup settings
      final autoBackup = await _storage.read(key: 'auto_backup');
      _autoBackupEnabled = autoBackup == 'true';

      final frequency = await _storage.read(key: 'backup_frequency');
      if (frequency != null) {
        _backupFrequency = frequency;
      }

      final lastBackupStr = await _storage.read(key: 'last_backup');
      if (lastBackupStr != null) {
        _lastBackup = DateTime.tryParse(lastBackupStr);
      }

      notifyListeners();
    } catch (e) {
      debugPrint('Error loading settings: $e');
    }
  }

  // Theme methods
  Future<void> setThemeMode(ThemeMode mode) async {
    _themeMode = mode;
    await _storage.write(key: 'theme_mode', value: mode.name);
    notifyListeners();
  }

  // Language methods
  Future<void> setLocale(String locale) async {
    if (supportedLanguages.containsKey(locale)) {
      _locale = locale;
      await _storage.write(key: 'locale', value: locale);
      notifyListeners();
    }
  }

  String getLanguageName() {
    return supportedLanguages[_locale] ?? 'English';
  }

  // Notification methods
  Future<void> setPushNotifications(bool enabled) async {
    _pushNotificationsEnabled = enabled;
    await _storage.write(key: 'push_notifications', value: enabled.toString());
    notifyListeners();
  }

  Future<void> setEmailNotifications(bool enabled) async {
    _emailNotificationsEnabled = enabled;
    await _storage.write(key: 'email_notifications', value: enabled.toString());
    notifyListeners();
  }

  Future<void> setVerificationAlerts(bool enabled) async {
    _verificationAlertsEnabled = enabled;
    await _storage.write(key: 'verification_alerts', value: enabled.toString());
    notifyListeners();
  }

  // Backup methods
  Future<void> setAutoBackup(bool enabled) async {
    _autoBackupEnabled = enabled;
    await _storage.write(key: 'auto_backup', value: enabled.toString());
    notifyListeners();
  }

  Future<void> setBackupFrequency(String frequency) async {
    _backupFrequency = frequency;
    await _storage.write(key: 'backup_frequency', value: frequency);
    notifyListeners();
  }

  Future<void> performBackup() async {
    // Simulate backup operation
    await Future.delayed(const Duration(seconds: 2));
    _lastBackup = DateTime.now();
    await _storage.write(key: 'last_backup', value: _lastBackup!.toIso8601String());
    notifyListeners();
  }

  String getThemeDisplayName() {
    switch (_themeMode) {
      case ThemeMode.light:
        return 'Light';
      case ThemeMode.dark:
        return 'Dark';
      case ThemeMode.system:
        return 'System';
    }
  }
}
