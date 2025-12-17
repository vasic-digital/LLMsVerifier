class Model {
  final String id;
  final String name;
  final String provider;
  final String category;
  final String description;
  final double overallScore;
  final double codeCapability;
  final double responsiveness;
  final bool supportsVision;
  final bool supportsAudio;
  final bool supportsVideo;
  final bool supportsReasoning;
  final List<String> capabilities;
  final List<String> tags;
  final DateTime lastVerified;

  Model({
    required this.id,
    required this.name,
    required this.provider,
    required this.category,
    required this.description,
    required this.overallScore,
    required this.codeCapability,
    required this.responsiveness,
    required this.supportsVision,
    required this.supportsAudio,
    required this.supportsVideo,
    required this.supportsReasoning,
    required this.capabilities,
    required this.tags,
    required this.lastVerified,
  });

  factory Model.fromJson(Map<String, dynamic> json) {
    return Model(
      id: json['id'] ?? '',
      name: json['name'] ?? '',
      provider: json['provider'] ?? '',
      category: json['category'] ?? 'general',
      description: json['description'] ?? '',
      overallScore: (json['overall_score'] ?? 0.0).toDouble(),
      codeCapability: (json['code_capability_score'] ?? 0.0).toDouble(),
      responsiveness: (json['responsiveness_score'] ?? 0.0).toDouble(),
      supportsVision: json['supports_vision'] ?? false,
      supportsAudio: json['supports_audio'] ?? false,
      supportsVideo: json['supports_video'] ?? false,
      supportsReasoning: json['supports_reasoning'] ?? false,
      capabilities: List<String>.from(json['capabilities'] ?? []),
      tags: List<String>.from(json['tags'] ?? []),
      lastVerified: json['last_verified'] != null
          ? DateTime.parse(json['last_verified'])
          : DateTime.now(),
    );
  }
}

class VerificationResult {
  final String id;
  final String modelId;
  final String status;
  final double overallScore;
  final double codeCapability;
  final double responsiveness;
  final double reliability;
  final double featureRichness;
  final Map<String, dynamic> details;
  final DateTime createdAt;

  VerificationResult({
    required this.id,
    required this.modelId,
    required this.status,
    required this.overallScore,
    required this.codeCapability,
    required this.responsiveness,
    required this.reliability,
    required this.featureRichness,
    required this.details,
    required this.createdAt,
  });

  factory VerificationResult.fromJson(Map<String, dynamic> json) {
    return VerificationResult(
      id: json['id'] ?? '',
      modelId: json['model_id'] ?? '',
      status: json['status'] ?? 'pending',
      overallScore: (json['overall_score'] ?? 0.0).toDouble(),
      codeCapability: (json['code_capability_score'] ?? 0.0).toDouble(),
      responsiveness: (json['responsiveness_score'] ?? 0.0).toDouble(),
      reliability: (json['reliability_score'] ?? 0.0).toDouble(),
      featureRichness: (json['feature_richness_score'] ?? 0.0).toDouble(),
      details: json['details'] ?? {},
      createdAt: json['created_at'] != null
          ? DateTime.parse(json['created_at'])
          : DateTime.now(),
    );
  }
}

class DashboardStats {
  final int totalModels;
  final int verifiedModels;
  final int topPerformers;
  final double averageScore;
  final int recentVerifications;

  DashboardStats({
    required this.totalModels,
    required this.verifiedModels,
    required this.topPerformers,
    required this.averageScore,
    required this.recentVerifications,
  });

  factory DashboardStats.fromJson(Map<String, dynamic> json) {
    return DashboardStats(
      totalModels: json['total_models'] ?? 0,
      verifiedModels: json['verified_models'] ?? 0,
      topPerformers: json['top_performers'] ?? 0,
      averageScore: (json['average_score'] ?? 0.0).toDouble(),
      recentVerifications: json['recent_verifications'] ?? 0,
    );
  }
}