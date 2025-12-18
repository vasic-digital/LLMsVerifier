import 'package:flutter/foundation.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import '../services/api_service.dart';

class AuthProvider with ChangeNotifier {
  final FlutterSecureStorage _storage = const FlutterSecureStorage();
  final ApiService _apiService;

  bool _isAuthenticated = false;
  String? _token;
  String? _username;
  String? _email;
  bool _isLoading = false;

  AuthProvider() : _apiService = ApiService(baseUrl: 'http://localhost:8080') {
    _loadStoredAuth();
  }

  bool get isAuthenticated => _isAuthenticated;
  String? get token => _token;
  String? get username => _username;
  String? get email => _email;
  bool get isLoading => _isLoading;

  Future<void> _loadStoredAuth() async {
    try {
      final storedToken = await _storage.read(key: 'auth_token');
      final storedUsername = await _storage.read(key: 'username');
      final storedEmail = await _storage.read(key: 'email');

      if (storedToken != null && storedUsername != null) {
        _token = storedToken;
        _username = storedUsername;
        _email = storedEmail;
        _isAuthenticated = true;
        notifyListeners();
      }
    } catch (e) {
      debugPrint('Error loading stored auth: $e');
    }
  }

  Future<void> login(String username, String password) async {
    _isLoading = true;
    notifyListeners();

    try {
      final result = await _apiService.login(username, password);
      final token = result['token'] as String;

      _token = token;
      _username = username;
      _isAuthenticated = true;

      // Store securely
      await _storage.write(key: 'auth_token', value: token);
      await _storage.write(key: 'username', value: username);
      await _storage.write(key: 'email', value: '');

      notifyListeners();
    } catch (e) {
      _isAuthenticated = false;
      rethrow;
    } finally {
      _isLoading = false;
      notifyListeners();
    }
  }

  Future<void> logout() async {
    _isLoading = true;
    notifyListeners();

    try {
      // Clear stored credentials
      await _storage.delete(key: 'auth_token');
      await _storage.delete(key: 'username');
      await _storage.delete(key: 'email');

      _token = null;
      _username = null;
      _email = null;
      _isAuthenticated = false;

      notifyListeners();
    } catch (e) {
      debugPrint('Error during logout: $e');
    } finally {
      _isLoading = false;
      notifyListeners();
    }
  }

  ApiService getApiService() {
    return ApiService(baseUrl: 'http://localhost:8080', authToken: _token);
  }

  Future<void> updateProfile(String newUsername, String newEmail) async {
    _isLoading = true;
    notifyListeners();

    try {
      // Update local state
      _username = newUsername;
      _email = newEmail;

      // Store securely
      await _storage.write(key: 'username', value: newUsername);
      await _storage.write(key: 'email', value: newEmail);

      notifyListeners();
    } catch (e) {
      debugPrint('Error updating profile: $e');
    } finally {
      _isLoading = false;
      notifyListeners();
    }
  }
}