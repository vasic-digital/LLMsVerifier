import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import 'providers/auth_provider.dart';
import 'providers/verification_provider.dart';
import 'screens/dashboard_screen.dart';
import 'screens/login_screen.dart';
import 'screens/models_screen.dart';
import 'screens/settings_screen.dart';
import 'screens/verification_screen.dart';

void main() {
  runApp(const LLMVerifierApp());
}

class LLMVerifierApp extends StatelessWidget {
  const LLMVerifierApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MultiProvider(
      providers: [
        ChangeNotifierProvider(create: (_) => AuthProvider()),
        ChangeNotifierProvider(create: (_) => VerificationProvider()),
      ],
      child: MaterialApp(
        title: 'LLM Verifier',
        theme: ThemeData(
          colorScheme: ColorScheme.fromSeed(seedColor: Colors.blue),
          useMaterial3: true,
        ),
        darkTheme: ThemeData.dark(
          useMaterial3: true,
        ),
        themeMode: ThemeMode.system,
        initialRoute: '/',
        routes: {
          '/': (context) => const AuthWrapper(),
          '/dashboard': (context) => const DashboardScreen(),
          '/models': (context) => const ModelsScreen(),
          '/verification': (context) => const VerificationScreen(),
          '/settings': (context) => const SettingsScreen(),
        },
      ),
    );
  }
}

class AuthWrapper extends StatelessWidget {
  const AuthWrapper({super.key});

  @override
  Widget build(BuildContext context) {
    return Consumer<AuthProvider>(
      builder: (context, auth, _) {
        return auth.isAuthenticated ? const DashboardScreen() : const LoginScreen();
      },
    );
  }
}