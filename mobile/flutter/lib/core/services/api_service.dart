import 'package:http/http.dart' as http;
import 'dart:convert';

class ApiService {
  static const String baseUrl = 'http://localhost:8080/api/v1'; // Change for production
  final http.Client _client;

  ApiService({http.Client? client}) : _client = client ?? http.Client();

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
      return jsonDecode(response.body);
    } else {
      throw Exception('Login failed: ${response.statusCode}');
    }
  }

  Future<List<dynamic>> getModels() async {
    final response = await _client.get(
      Uri.parse('$baseUrl/models'),
      headers: {'Authorization': 'Bearer \${getToken()}'}, // TODO: Implement token management
    );

    if (response.statusCode == 200) {
      return jsonDecode(response.body)['data'];
    } else {
      throw Exception('Failed to load models');
    }
  }

  Future<Map<String, dynamic>> verifyModel(String modelId) async {
    final response = await _client.post(
      Uri.parse('$baseUrl/models/$modelId/verify'),
      headers: {'Authorization': 'Bearer \${getToken()}'},
    );

    if (response.statusCode == 200) {
      return jsonDecode(response.body);
    } else {
      throw Exception('Verification failed');
    }
  }

  String? getToken() {
    // TODO: Implement secure token storage
    return null;
  }
}