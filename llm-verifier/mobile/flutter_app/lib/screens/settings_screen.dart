import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../providers/auth_provider.dart';
import '../providers/settings_provider.dart';

class SettingsScreen extends StatelessWidget {
  const SettingsScreen({super.key});

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Settings'),
      ),
      body: ListView(
        children: [
          ListTile(
            leading: const Icon(Icons.person),
            title: const Text('User Profile'),
            subtitle: Consumer<AuthProvider>(
              builder: (context, auth, _) => Text(auth.username ?? 'Not logged in'),
            ),
            onTap: () {
              _showProfileDialog(context);
            },
          ),
          const Divider(),
          Consumer<SettingsProvider>(
            builder: (context, settings, _) => ListTile(
              leading: const Icon(Icons.palette),
              title: const Text('Theme'),
              subtitle: Text(settings.getThemeDisplayName()),
              onTap: () => _showThemeDialog(context, settings),
            ),
          ),
          Consumer<SettingsProvider>(
            builder: (context, settings, _) => ListTile(
              leading: const Icon(Icons.language),
              title: const Text('Language'),
              subtitle: Text(settings.getLanguageName()),
              onTap: () => _showLanguageDialog(context, settings),
            ),
          ),
          const Divider(),
          ListTile(
            leading: const Icon(Icons.notifications),
            title: const Text('Notifications'),
            subtitle: const Text('Manage notification preferences'),
            onTap: () => _showNotificationSettings(context),
          ),
          ListTile(
            leading: const Icon(Icons.backup),
            title: const Text('Backup & Sync'),
            subtitle: Consumer<SettingsProvider>(
              builder: (context, settings, _) => Text(
                settings.autoBackupEnabled
                  ? 'Auto backup: ${settings.backupFrequency}'
                  : 'Auto backup disabled',
              ),
            ),
            onTap: () => _showBackupSettings(context),
          ),
          const Divider(),
          ListTile(
            leading: const Icon(Icons.info),
            title: const Text('About'),
            subtitle: const Text('Version 1.0.0'),
            onTap: () => _showAboutDialog(context),
          ),
          ListTile(
            leading: const Icon(Icons.logout),
            title: const Text('Logout'),
            onTap: () => _confirmLogout(context),
          ),
        ],
      ),
    );
  }

  void _showThemeDialog(BuildContext context, SettingsProvider settings) {
    showDialog(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text('Select Theme'),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            RadioListTile<ThemeMode>(
              title: const Text('System'),
              subtitle: const Text('Follow system settings'),
              value: ThemeMode.system,
              groupValue: settings.themeMode,
              onChanged: (value) {
                settings.setThemeMode(value!);
                Navigator.pop(context);
              },
            ),
            RadioListTile<ThemeMode>(
              title: const Text('Light'),
              subtitle: const Text('Always use light theme'),
              value: ThemeMode.light,
              groupValue: settings.themeMode,
              onChanged: (value) {
                settings.setThemeMode(value!);
                Navigator.pop(context);
              },
            ),
            RadioListTile<ThemeMode>(
              title: const Text('Dark'),
              subtitle: const Text('Always use dark theme'),
              value: ThemeMode.dark,
              groupValue: settings.themeMode,
              onChanged: (value) {
                settings.setThemeMode(value!);
                Navigator.pop(context);
              },
            ),
          ],
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(context),
            child: const Text('Cancel'),
          ),
        ],
      ),
    );
  }

  void _showLanguageDialog(BuildContext context, SettingsProvider settings) {
    showDialog(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text('Select Language'),
        content: SizedBox(
          width: double.maxFinite,
          child: ListView.builder(
            shrinkWrap: true,
            itemCount: SettingsProvider.supportedLanguages.length,
            itemBuilder: (context, index) {
              final entry = SettingsProvider.supportedLanguages.entries.elementAt(index);
              return RadioListTile<String>(
                title: Text(entry.value),
                value: entry.key,
                groupValue: settings.locale,
                onChanged: (value) {
                  settings.setLocale(value!);
                  Navigator.pop(context);
                  ScaffoldMessenger.of(context).showSnackBar(
                    SnackBar(content: Text('Language changed to ${entry.value}')),
                  );
                },
              );
            },
          ),
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(context),
            child: const Text('Cancel'),
          ),
        ],
      ),
    );
  }

  void _showNotificationSettings(BuildContext context) {
    showModalBottomSheet(
      context: context,
      builder: (context) => Consumer<SettingsProvider>(
        builder: (context, settings, _) => Padding(
          padding: const EdgeInsets.all(16),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              const Text(
                'Notification Settings',
                style: TextStyle(fontSize: 20, fontWeight: FontWeight.bold),
              ),
              const SizedBox(height: 16),
              SwitchListTile(
                title: const Text('Push Notifications'),
                subtitle: const Text('Receive push notifications'),
                value: settings.pushNotificationsEnabled,
                onChanged: (value) => settings.setPushNotifications(value),
              ),
              SwitchListTile(
                title: const Text('Email Notifications'),
                subtitle: const Text('Receive email notifications'),
                value: settings.emailNotificationsEnabled,
                onChanged: (value) => settings.setEmailNotifications(value),
              ),
              SwitchListTile(
                title: const Text('Verification Alerts'),
                subtitle: const Text('Get alerts for verification results'),
                value: settings.verificationAlertsEnabled,
                onChanged: (value) => settings.setVerificationAlerts(value),
              ),
              const SizedBox(height: 16),
            ],
          ),
        ),
      ),
    );
  }

  void _showBackupSettings(BuildContext context) {
    showModalBottomSheet(
      context: context,
      builder: (context) => Consumer<SettingsProvider>(
        builder: (context, settings, _) => Padding(
          padding: const EdgeInsets.all(16),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              const Text(
                'Backup & Sync',
                style: TextStyle(fontSize: 20, fontWeight: FontWeight.bold),
              ),
              const SizedBox(height: 16),
              SwitchListTile(
                title: const Text('Auto Backup'),
                subtitle: const Text('Automatically backup your data'),
                value: settings.autoBackupEnabled,
                onChanged: (value) => settings.setAutoBackup(value),
              ),
              if (settings.autoBackupEnabled) ...[
                ListTile(
                  title: const Text('Backup Frequency'),
                  subtitle: Text(settings.backupFrequency),
                  trailing: const Icon(Icons.chevron_right),
                  onTap: () => _showFrequencyDialog(context, settings),
                ),
              ],
              ListTile(
                title: const Text('Last Backup'),
                subtitle: Text(
                  settings.lastBackup != null
                    ? _formatDate(settings.lastBackup!)
                    : 'Never',
                ),
              ),
              const SizedBox(height: 16),
              SizedBox(
                width: double.infinity,
                child: ElevatedButton.icon(
                  onPressed: () async {
                    Navigator.pop(context);
                    ScaffoldMessenger.of(context).showSnackBar(
                      const SnackBar(content: Text('Backup started...')),
                    );
                    await settings.performBackup();
                    if (context.mounted) {
                      ScaffoldMessenger.of(context).showSnackBar(
                        const SnackBar(content: Text('Backup completed!')),
                      );
                    }
                  },
                  icon: const Icon(Icons.backup),
                  label: const Text('Backup Now'),
                ),
              ),
              const SizedBox(height: 16),
            ],
          ),
        ),
      ),
    );
  }

  void _showFrequencyDialog(BuildContext context, SettingsProvider settings) {
    Navigator.pop(context); // Close the bottom sheet first
    showDialog(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text('Backup Frequency'),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            RadioListTile<String>(
              title: const Text('Daily'),
              value: 'daily',
              groupValue: settings.backupFrequency,
              onChanged: (value) {
                settings.setBackupFrequency(value!);
                Navigator.pop(context);
              },
            ),
            RadioListTile<String>(
              title: const Text('Weekly'),
              value: 'weekly',
              groupValue: settings.backupFrequency,
              onChanged: (value) {
                settings.setBackupFrequency(value!);
                Navigator.pop(context);
              },
            ),
            RadioListTile<String>(
              title: const Text('Monthly'),
              value: 'monthly',
              groupValue: settings.backupFrequency,
              onChanged: (value) {
                settings.setBackupFrequency(value!);
                Navigator.pop(context);
              },
            ),
          ],
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(context),
            child: const Text('Cancel'),
          ),
        ],
      ),
    );
  }

  String _formatDate(DateTime date) {
    return '${date.year}-${date.month.toString().padLeft(2, '0')}-${date.day.toString().padLeft(2, '0')} '
           '${date.hour.toString().padLeft(2, '0')}:${date.minute.toString().padLeft(2, '0')}';
  }

  void _showAboutDialog(BuildContext context) {
    showDialog(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text('LLM Verifier'),
        content: const Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text('Version: 1.0.0'),
            SizedBox(height: 8),
            Text('A comprehensive system for testing, evaluating, and benchmarking Large Language Models.'),
            SizedBox(height: 16),
            Text('Features:'),
            Text('• Multi-provider support'),
            Text('• Real-time verification'),
            Text('• Performance scoring'),
            Text('• Export configurations'),
          ],
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(context),
            child: const Text('Close'),
          ),
        ],
      ),
    );
  }

  void _showProfileDialog(BuildContext context) {
    final authProvider = context.read<AuthProvider>();
    final usernameController = TextEditingController(text: authProvider.username);
    final emailController = TextEditingController(text: authProvider.email);

    showDialog(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text('Edit Profile'),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            TextField(
              controller: usernameController,
              decoration: const InputDecoration(
                labelText: 'Username',
                prefixIcon: Icon(Icons.person),
              ),
            ),
            const SizedBox(height: 16),
            TextField(
              controller: emailController,
              decoration: const InputDecoration(
                labelText: 'Email',
                prefixIcon: Icon(Icons.email),
              ),
              keyboardType: TextInputType.emailAddress,
            ),
          ],
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(context),
            child: const Text('Cancel'),
          ),
          TextButton(
            onPressed: () async {
              if (usernameController.text.isNotEmpty &&
                  emailController.text.isNotEmpty) {
                await authProvider.updateProfile(
                  usernameController.text.trim(),
                  emailController.text.trim(),
                );
                if (context.mounted) {
                  Navigator.pop(context);
                  ScaffoldMessenger.of(context).showSnackBar(
                    const SnackBar(content: Text('Profile updated successfully')),
                  );
                }
              }
            },
            child: const Text('Save'),
          ),
        ],
      ),
    );
  }

  void _confirmLogout(BuildContext context) {
    showDialog(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text('Logout'),
        content: const Text('Are you sure you want to logout?'),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(context),
            child: const Text('Cancel'),
          ),
          TextButton(
            onPressed: () async {
              Navigator.pop(context); // Close dialog
              await context.read<AuthProvider>().logout();
            },
            style: TextButton.styleFrom(foregroundColor: Colors.red),
            child: const Text('Logout'),
          ),
        ],
      ),
    );
  }
}
