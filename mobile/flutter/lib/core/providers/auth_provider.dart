import 'package:flutter/material.dart';
import '../services/auth_service.dart';
import '../services/api_service.dart';

class AuthProvider with ChangeNotifier {
  final AuthService _authService;
  bool _isLoading = false;
  String? _error;
  Map<String, dynamic>? _userData;

  AuthProvider({required ApiService apiService})
      : _authService = AuthService(apiService);

  bool get isLoading => _isLoading;
  String? get error => _error;
  bool get isLoggedIn => _userData != null;
  Map<String, dynamic>? get userData => _userData;

  Future<bool> login(String username, String password) async {
    _isLoading = true;
    _error = null;
    notifyListeners();

    try {
      final success = await _authService.login(username, password);
      if (success) {
        _userData = await _authService.getUserData();
        _error = null;
      } else {
        _error = 'Login failed';
      }
      return success;
    } catch (e) {
      _error = e.toString();
      return false;
    } finally {
      _isLoading = false;
      notifyListeners();
    }
  }

  Future<void> logout() async {
    await _authService.logout();
    _userData = null;
    _error = null;
    notifyListeners();
  }

  Future<void> checkAuthStatus() async {
    final loggedIn = await _authService.isLoggedIn();
    if (loggedIn) {
      _userData = await _authService.getUserData();
    } else {
      _userData = null;
    }
    notifyListeners();
  }
}