import 'package:flutter/foundation.dart';
import '../models/models.dart';
import '../services/api_service.dart';

class VerificationProvider with ChangeNotifier {
  ApiService? _apiService;
  bool _isLoading = false;
  List<Model> _allModels = [];
  List<Model> _topModels = [];
  List<VerificationResult> _recentResults = [];
  DashboardStats? _dashboardStats;

  bool get isLoading => _isLoading;
  List<Model> get allModels => _allModels;
  List<Model> get topModels => _topModels;
  List<VerificationResult> get recentResults => _recentResults;
  DashboardStats? get dashboardStats => _dashboardStats;

  void setApiService(ApiService apiService) {
    _apiService = apiService;
  }

  Future<void> loadDashboardData() async {
    if (_apiService == null) return;

    _isLoading = true;
    notifyListeners();

    try {
      // Load data in parallel
      final results = await Future.wait([
        loadModels(),
        loadRecentVerifications(),
        loadDashboardStats(),
      ]);

      // Update top models (top 5 by score)
      _topModels = _allModels
          .where((model) => model.overallScore > 0)
          .toList()
        ..sort((a, b) => b.overallScore.compareTo(a.overallScore))
        ..take(5).toList();

      notifyListeners();
    } catch (e) {
      debugPrint('Error loading dashboard data: $e');
    } finally {
      _isLoading = false;
      notifyListeners();
    }
  }

  Future<void> loadModels() async {
    if (_apiService == null) return;

    try {
      _allModels = await _apiService!.getModels();
      notifyListeners();
    } catch (e) {
      debugPrint('Error loading models: $e');
      // Keep existing data on error
    }
  }

  Future<void> loadRecentVerifications() async {
    if (_apiService == null) return;

    try {
      _recentResults = await _apiService!.getVerificationResults(limit: 20);
      notifyListeners();
    } catch (e) {
      debugPrint('Error loading recent verifications: $e');
    }
  }

  Future<void> loadDashboardStats() async {
    if (_apiService == null) return;

    try {
      _dashboardStats = await _apiService!.getDashboardStats();
      notifyListeners();
    } catch (e) {
      debugPrint('Error loading dashboard stats: $e');
    }
  }

  Future<Map<String, dynamic>> startVerification(String modelId) async {
    if (_apiService == null) {
      throw Exception('API service not initialized');
    }

    _isLoading = true;
    notifyListeners();

    try {
      final result = await _apiService!.startVerification(modelId);
      await loadRecentVerifications(); // Refresh the list
      return result;
    } finally {
      _isLoading = false;
      notifyListeners();
    }
  }

  Future<Map<String, dynamic>> exportConfig(String format) async {
    if (_apiService == null) {
      throw Exception('API service not initialized');
    }

    try {
      return await _apiService!.exportConfig(format);
    } catch (e) {
      debugPrint('Error exporting config: $e');
      rethrow;
    }
  }

  Model? getModelById(String id) {
    try {
      return _allModels.firstWhere((model) => model.id == id);
    } catch (e) {
      return null;
    }
  }

  List<Model> getModelsByProvider(String provider) {
    return _allModels.where((model) => model.provider == provider).toList();
  }

  List<Model> getModelsByCategory(String category) {
    return _allModels.where((model) => model.category == category).toList();
  }

  double getAverageScore() {
    if (_allModels.isEmpty) return 0.0;

    final scoredModels = _allModels.where((model) => model.overallScore > 0);
    if (scoredModels.isEmpty) return 0.0;

    final total = scoredModels.fold<double>(0, (sum, model) => sum + model.overallScore);
    return total / scoredModels.length;
  }
}