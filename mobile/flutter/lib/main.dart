import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import 'package:firebase_core/firebase_core.dart';
import 'package:firebase_messaging/firebase_messaging.dart';
import 'package:flutter_local_notifications/flutter_local_notifications.dart';

import 'core/services/api_service.dart';
import 'core/services/auth_service.dart';
import 'core/services/notification_service.dart';
import 'core/services/offline_service.dart';
import 'core/providers/auth_provider.dart';
import 'core/providers/verification_provider.dart';
import 'core/providers/models_provider.dart';
import 'core/routes/app_routes.dart';
import 'core/themes/app_theme.dart';
import 'features/auth/screens/login_screen.dart';
import 'features/dashboard/screens/dashboard_screen.dart';
import 'core/widgets/splash_screen.dart';

const AndroidNotificationChannel channel = AndroidNotificationChannel(
  'high_importance_channel',
  'High Importance Notifications',
  description: 'This channel is used for important notifications.',
  importance: Importance.high,
);

final FlutterLocalNotificationsPlugin flutterLocalNotificationsPlugin =
    FlutterLocalNotificationsPlugin();

Future<void> _firebaseMessagingBackgroundHandler(RemoteMessage message) async {
  await Firebase.initializeApp();
  print('Handling a background message: ${message.messageId}');
}

void main() async {
  WidgetsFlutterBinding.ensureInitialized();
  
  await Firebase.initializeApp();
  FirebaseMessaging.onBackgroundMessage(_firebaseMessagingBackgroundHandler);
  
  await flutterLocalNotificationsPlugin
      .resolvePlatformSpecificImplementation<AndroidFlutterLocalNotificationsPlugin>()
      ?.createNotificationChannel(channel);
  
  await NotificationService.initialize();
  
  runApp(MyApp());
}

class MyApp extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    return MultiProvider(
      providers: [
        Provider(create: (_) => ApiService()),
        Provider(create: (_) => OfflineService()),
        ChangeNotifierProvider(
          create: (context) => AuthProvider(
            apiService: context.read<ApiService>(),
          ),
        ),
        ChangeNotifierProvider(
          create: (context) => ModelsProvider(
            apiService: context.read<ApiService>(),
            offlineService: context.read<OfflineService>(),
          ),
        ),
        ChangeNotifierProvider(
          create: (context) => VerificationProvider(
            apiService: context.read<ApiService>(),
            offlineService: context.read<OfflineService>(),
          ),
        ),
      ],
      child: MaterialApp(
        title: 'LLM Verifier',
        theme: AppTheme.lightTheme,
        darkTheme: AppTheme.darkTheme,
        themeMode: ThemeMode.system,
        debugShowCheckedModeBanner: false,
        initialRoute: AppRoutes.splash,
        routes: AppRoutes.routes,
        home: SplashScreen(),
      ),
    );
  }
}