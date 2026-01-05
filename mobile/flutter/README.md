# LLM Verifier Mobile App

Official mobile application for the LLM Verifier platform, built with Flutter. Provides model verification, scoring, and monitoring capabilities on iOS and Android.

## Features

- **Model Verification**: Verify LLM models directly from your mobile device
- **Score Tracking**: View and compare model scores with detailed breakdowns
- **Provider Monitoring**: Real-time provider health status
- **Push Notifications**: Get alerts for verification results and provider issues
- **Offline Support**: Cache verified models for offline access
- **Biometric Auth**: Secure access with fingerprint/Face ID
- **Dark Mode**: Full dark mode support

## Requirements

- Flutter SDK 3.10.0 or higher
- Dart SDK 3.0.0 or higher
- iOS 12.0+ / Android API 21+

## Getting Started

### 1. Clone the Repository

```bash
git clone https://github.com/llm-verifier/llm-verifier.git
cd llm-verifier/mobile/flutter
```

### 2. Install Dependencies

```bash
flutter pub get
```

### 3. Configure Environment

Create a `.env` file:

```env
API_BASE_URL=http://localhost:8080
# For production: https://api.llmverifier.com
```

### 4. Run the App

```bash
# iOS
flutter run -d ios

# Android
flutter run -d android

# Web
flutter run -d chrome
```

## Project Structure

```
lib/
├── main.dart              # App entry point
├── core/
│   ├── config/            # App configuration
│   ├── services/          # API services
│   │   ├── api_service.dart
│   │   └── auth_service.dart
│   ├── models/            # Data models
│   ├── providers/         # State management
│   └── utils/             # Utilities
├── features/
│   ├── auth/              # Authentication screens
│   ├── models/            # Model listing & details
│   ├── verification/      # Verification flow
│   ├── providers/         # Provider management
│   └── settings/          # App settings
└── widgets/               # Reusable widgets
```

## Building for Production

### Android

```bash
# Generate signing key
keytool -genkey -v -keystore android/app/keystore.jks \
  -keyalg RSA -keysize 2048 -validity 10000 -alias llm-verifier

# Build APK
flutter build apk --release

# Build App Bundle
flutter build appbundle --release
```

### iOS

```bash
# Build for iOS
flutter build ios --release

# Open in Xcode for distribution
open ios/Runner.xcworkspace
```

## Configuration

### Firebase Setup

1. Create a Firebase project at [Firebase Console](https://console.firebase.google.com)
2. Download `google-services.json` (Android) and `GoogleService-Info.plist` (iOS)
3. Place in respective platform directories

### API Configuration

The app connects to the LLM Verifier API. Configure the base URL:

```dart
// lib/core/config/api_config.dart
class ApiConfig {
  static const String baseUrl = String.fromEnvironment(
    'API_BASE_URL',
    defaultValue: 'http://localhost:8080',
  );
}
```

## Testing

```bash
# Unit tests
flutter test

# Integration tests
flutter test integration_test/

# With coverage
flutter test --coverage
```

## Key Dependencies

| Package | Purpose |
|---------|---------|
| `http` / `dio` | HTTP client |
| `provider` | State management |
| `flutter_secure_storage` | Secure token storage |
| `local_auth` | Biometric authentication |
| `firebase_messaging` | Push notifications |
| `charts_flutter` | Score visualizations |
| `connectivity_plus` | Network state monitoring |

## State Management

The app uses Provider for state management:

```dart
// Access model state
final models = context.watch<ModelProvider>().models;

// Trigger verification
context.read<VerificationProvider>().verifyModel(modelId);
```

## API Integration

```dart
// Services communicate with LLM Verifier API
final apiService = ApiService();

// Get models
final models = await apiService.getModels();

// Verify model
final result = await apiService.verifyModel(modelId);

// Get score
final score = await apiService.getModelScore(modelId);
```

## Security

- **Secure Storage**: JWT tokens stored using FlutterSecureStorage with platform encryption
- **Certificate Pinning**: SSL pinning for API requests (production)
- **Biometric Auth**: Optional biometric lock for app access
- **No Logging**: Sensitive data never logged

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Run tests: `flutter test`
5. Submit a pull request

## License

MIT License - see LICENSE file for details.

## Links

- [LLM Verifier Documentation](https://llm-verifier.dev/docs)
- [API Reference](https://llm-verifier.dev/docs/api-reference)
- [Flutter Documentation](https://docs.flutter.dev)
