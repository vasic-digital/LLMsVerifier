import 'dart:convert';
import 'package:http/http.dart' as http;
import '../models/models.dart';

class ApiService {
  final String baseUrl;
  final String? authToken;

  ApiService({required this.baseUrl, this.authToken});

  Map<String, String> get _headers => {
        'Content-Type': 'application/json',
        if (authToken != null) 'Authorization': 'Bearer $authToken',
      };

  Future<List<Model>> getModels() async {
    try {
      final response = await http.get(
        Uri.parse('$baseUrl/api/v1/models'),
        headers: _headers,
      );

      if (response.statusCode == 200) {
        final data = json.decode(response.body);
        final models = data['models'] as List;
        return models.map((model) => Model.fromJson(model)).toList();
      } else {
        throw Exception('Failed to load models: ${response.statusCode}');
      }
    } catch (e) {
      throw Exception('Network error: $e');
    }
  }

  Future<Model> getModel(String id) async {
    try {
      final response = await http.get(
        Uri.parse('$baseUrl/api/v1/models/$id'),
        headers: _headers,
      );

      if (response.statusCode == 200) {
        final data = json.decode(response.body);
        return Model.fromJson(data);
      } else {
        throw Exception('Failed to load model: ${response.statusCode}');
      }
    } catch (e) {
      throw Exception('Network error: $e');
    }
  }

  Future<List<VerificationResult>> getVerificationResults({
    int limit = 50,
    int offset = 0,
  }) async {
    try {
      final response = await http.get(
        Uri.parse('$baseUrl/api/v1/verifications?limit=$limit&offset=$offset'),
        headers: _headers,
      );

      if (response.statusCode == 200) {
        final data = json.decode(response.body);
        final results = data['results'] as List;
        return results.map((result) => VerificationResult.fromJson(result)).toList();
      } else {
        throw Exception('Failed to load verification results: ${response.statusCode}');
      }
    } catch (e) {
      throw Exception('Network error: $e');
    }
  }

  Future<DashboardStats> getDashboardStats() async {
    try {
      final response = await http.get(
        Uri.parse('$baseUrl/api/v1/dashboard/stats'),
        headers: _headers,
      );

      if (response.statusCode == 200) {
        final data = json.decode(response.body);
        return DashboardStats.fromJson(data);
      } else {
        throw Exception('Failed to load dashboard stats: ${response.statusCode}');
      }
    } catch (e) {
      throw Exception('Network error: $e');
    }
  }

  Future<Map<String, dynamic>> startVerification(String modelId) async {
    try {
      final response = await http.post(
        Uri.parse('$baseUrl/api/v1/verifications'),
        headers: _headers,
        body: json.encode({
          'model_id': modelId,
          'verification_type': 'full',
        }),
      );

      if (response.statusCode == 201) {
        return json.decode(response.body);
      } else {
        throw Exception('Failed to start verification: ${response.statusCode}');
      }
    } catch (e) {
      throw Exception('Network error: $e');
    }
  }

  Future<Map<String, dynamic>> login(String username, String password) async {
    try {
      final response = await http.post(
        Uri.parse('$baseUrl/api/v1/auth/login'),
        headers: {'Content-Type': 'application/json'},
        body: json.encode({
          'username': username,
          'password': password,
        }),
      );

      if (response.statusCode == 200) {
        return json.decode(response.body);
      } else {
        throw Exception('Login failed: ${response.statusCode}');
      }
    } catch (e) {
      throw Exception('Network error: $e');
    }
  }

  Future<Map<String, dynamic>> exportConfig(String format) async {
    try {
      final response = await http.post(
        Uri.parse('$baseUrl/api/v1/config/export'),
        headers: _headers,
        body: json.encode({
          'format': format,
          'include_api_key': false,
        }),
      );

      if (response.statusCode == 200) {
        return json.decode(response.body);
      } else {
        throw Exception('Export failed: ${response.statusCode}');
      }
    } catch (e) {
      throw Exception('Network error: $e');
    }
  }
}