import 'package:flutter/material.dart';
import 'api_service.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';

class AuthService {
  final ApiService _apiService;
  final FlutterSecureStorage _storage;

  AuthService(this._apiService) : _storage = const FlutterSecureStorage();

  Future<bool> login(String username, String password) async {
    try {
      final response = await _apiService.login(username, password);
      final token = response['data']['token'];

      // Store token securely
      await _storage.write(key: 'auth_token', value: token);
      await _storage.write(key: 'user_data', value: jsonEncode(response['data']['user']));

      return true;
    } catch (e) {
      return false;
    }
  }

  Future<void> logout() async {
    await _storage.delete(key: 'auth_token');
    await _storage.delete(key: 'user_data');
  }

  Future<String?> getToken() async {
    return await _storage.read(key: 'auth_token');
  }

  Future<Map<String, dynamic>?> getUserData() async {
    final userData = await _storage.read(key: 'user_data');
    return userData != null ? jsonDecode(userData) : null;
  }

  Future<bool> isLoggedIn() async {
    final token = await getToken();
    return token != null && token.isNotEmpty;
  }
}