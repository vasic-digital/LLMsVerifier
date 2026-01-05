import 'package:http/http.dart' as http;
import 'dart:convert';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';

class ApiService {
  static const String baseUrl = 'http://localhost:8080/api/v1'; // Change for production
  final http.Client _client;
  final FlutterSecureStorage _storage;
  String? _cachedToken;

  ApiService({http.Client? client, FlutterSecureStorage? storage})
      : _client = client ?? http.Client(),
        _storage = storage ?? const FlutterSecureStorage();

  // Token Management
  Future<String?> getToken() async {
    if (_cachedToken != null) {
      return _cachedToken;
    }
    _cachedToken = await _storage.read(key: 'auth_token');
    return _cachedToken;
  }

  Future<void> setToken(String token) async {
    _cachedToken = token;
    await _storage.write(key: 'auth_token', value: token);
  }

  Future<void> clearToken() async {
    _cachedToken = null;
    await _storage.delete(key: 'auth_token');
  }

  Future<bool> hasValidToken() async {
    final token = await getToken();
    if (token == null || token.isEmpty) {
      return false;
    }
    // Check token expiration
    try {
      final parts = token.split('.');
      if (parts.length != 3) return false;

      final payload = json.decode(
        utf8.decode(base64Url.decode(base64Url.normalize(parts[1]))),
      );

      final exp = payload['exp'] as int?;
      if (exp == null) return true; // No expiration

      return DateTime.now().millisecondsSinceEpoch < exp * 1000;
    } catch (e) {
      return false;
    }
  }

  // Get authorization headers
  Future<Map<String, String>> _getAuthHeaders() async {
    final token = await getToken();
    return {
      'Content-Type': 'application/json',
      if (token != null) 'Authorization': 'Bearer $token',
    };
  }

  // Authentication
  Future<Map<String, dynamic>> login(String username, String password) async {
    final response = await _client.post(
      Uri.parse('$baseUrl/auth/login'),
      headers: {'Content-Type': 'application/json'},
      body: jsonEncode({
        'username': username,
        'password': password,
      }),
    );

    if (response.statusCode == 200) {
      final data = jsonDecode(response.body);
      // Store token from response
      if (data['data'] != null && data['data']['token'] != null) {
        await setToken(data['data']['token']);
      }
      return data;
    } else {
      throw Exception('Login failed: ${response.statusCode}');
    }
  }

  Future<void> logout() async {
    try {
      final headers = await _getAuthHeaders();
      await _client.post(
        Uri.parse('$baseUrl/auth/logout'),
        headers: headers,
      );
    } finally {
      await clearToken();
    }
  }

  Future<Map<String, dynamic>> refreshToken() async {
    final headers = await _getAuthHeaders();
    final response = await _client.post(
      Uri.parse('$baseUrl/auth/refresh'),
      headers: headers,
    );

    if (response.statusCode == 200) {
      final data = jsonDecode(response.body);
      if (data['data'] != null && data['data']['token'] != null) {
        await setToken(data['data']['token']);
      }
      return data;
    } else {
      throw Exception('Token refresh failed: ${response.statusCode}');
    }
  }

  // Models API
  Future<List<dynamic>> getModels() async {
    final headers = await _getAuthHeaders();
    final response = await _client.get(
      Uri.parse('$baseUrl/models'),
      headers: headers,
    );

    if (response.statusCode == 200) {
      return jsonDecode(response.body)['data'] ?? [];
    } else if (response.statusCode == 401) {
      // Token expired, try to refresh
      await refreshToken();
      return getModels();
    } else {
      throw Exception('Failed to load models: ${response.statusCode}');
    }
  }

  Future<Map<String, dynamic>> getModel(String modelId) async {
    final headers = await _getAuthHeaders();
    final response = await _client.get(
      Uri.parse('$baseUrl/models/$modelId'),
      headers: headers,
    );

    if (response.statusCode == 200) {
      return jsonDecode(response.body)['data'];
    } else {
      throw Exception('Failed to load model: ${response.statusCode}');
    }
  }

  // Verification API
  Future<Map<String, dynamic>> verifyModel(String modelId) async {
    final headers = await _getAuthHeaders();
    final response = await _client.post(
      Uri.parse('$baseUrl/models/$modelId/verify'),
      headers: headers,
    );

    if (response.statusCode == 200) {
      return jsonDecode(response.body);
    } else {
      throw Exception('Verification failed: ${response.statusCode}');
    }
  }

  Future<Map<String, dynamic>> getVerificationStatus(String verificationId) async {
    final headers = await _getAuthHeaders();
    final response = await _client.get(
      Uri.parse('$baseUrl/verifications/$verificationId'),
      headers: headers,
    );

    if (response.statusCode == 200) {
      return jsonDecode(response.body)['data'];
    } else {
      throw Exception('Failed to get verification status: ${response.statusCode}');
    }
  }

  Future<List<dynamic>> getVerificationHistory(String modelId) async {
    final headers = await _getAuthHeaders();
    final response = await _client.get(
      Uri.parse('$baseUrl/models/$modelId/verifications'),
      headers: headers,
    );

    if (response.statusCode == 200) {
      return jsonDecode(response.body)['data'] ?? [];
    } else {
      throw Exception('Failed to get verification history: ${response.statusCode}');
    }
  }

  // Scoring API
  Future<Map<String, dynamic>> getModelScore(String modelId) async {
    final headers = await _getAuthHeaders();
    final response = await _client.get(
      Uri.parse('$baseUrl/models/$modelId/score'),
      headers: headers,
    );

    if (response.statusCode == 200) {
      return jsonDecode(response.body)['data'];
    } else {
      throw Exception('Failed to get model score: ${response.statusCode}');
    }
  }

  // Provider API
  Future<List<dynamic>> getProviders() async {
    final headers = await _getAuthHeaders();
    final response = await _client.get(
      Uri.parse('$baseUrl/providers'),
      headers: headers,
    );

    if (response.statusCode == 200) {
      return jsonDecode(response.body)['data'] ?? [];
    } else {
      throw Exception('Failed to load providers: ${response.statusCode}');
    }
  }

  // User API
  Future<Map<String, dynamic>> getCurrentUser() async {
    final headers = await _getAuthHeaders();
    final response = await _client.get(
      Uri.parse('$baseUrl/auth/me'),
      headers: headers,
    );

    if (response.statusCode == 200) {
      return jsonDecode(response.body)['data'];
    } else {
      throw Exception('Failed to get current user: ${response.statusCode}');
    }
  }

  Future<Map<String, dynamic>> updateProfile(Map<String, dynamic> profileData) async {
    final headers = await _getAuthHeaders();
    final response = await _client.put(
      Uri.parse('$baseUrl/auth/profile'),
      headers: headers,
      body: jsonEncode(profileData),
    );

    if (response.statusCode == 200) {
      return jsonDecode(response.body)['data'];
    } else {
      throw Exception('Failed to update profile: ${response.statusCode}');
    }
  }

  // Dispose
  void dispose() {
    _client.close();
  }
}
