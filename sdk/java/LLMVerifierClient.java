package com.llmverifier.sdk;

import java.util.*;
import java.util.concurrent.*;
import java.net.http.*;
import java.net.URI;
import com.fasterxml.jackson.databind.*;
import com.fasterxml.jackson.annotation.*;

/**
 * LLM Verifier Java SDK - Complete Implementation
 * Provides comprehensive access to the LLM Verifier API
 */
public class LLMVerifierClient {
    private final String baseUrl;
    private final String apiKey;
    private final HttpClient httpClient;
    private final ObjectMapper objectMapper;
    private final ExecutorService executor;
    
    private static final String DEFAULT_BASE_URL = "https://api.llmverifier.com";
    private static final int DEFAULT_TIMEOUT = 30;
    private static final int DEFAULT_MAX_RETRIES = 3;
    
    public LLMVerifierClient(String apiKey) {
        this(DEFAULT_BASE_URL, apiKey);
    }
    
    public LLMVerifierClient(String baseUrl, String apiKey) {
        this.baseUrl = baseUrl.endsWith("/") ? baseUrl.substring(0, baseUrl.length() - 1) : baseUrl;
        this.apiKey = apiKey;
        this.httpClient = HttpClient.newBuilder()
                .connectTimeout(Duration.ofSeconds(DEFAULT_TIMEOUT))
                .build();
        this.objectMapper = new ObjectMapper()
                .setSerializationInclusion(JsonInclude.Include.NON_NULL)
                .configure(DeserializationFeature.FAIL_ON_UNKNOWN_PROPERTIES, false);
        this.executor = Executors.newCachedThreadPool();
    }
    
    /**
     * Get all available models with their scores
     */
    public CompletableFuture<List<Model>> getModels() {
        return makeRequest("GET", "/api/models", null, ModelListResponse.class)
                .thenApply(response -> response.models);
    }
    
    /**
     * Get a specific model by ID
     */
    public CompletableFuture<Model> getModel(String modelId) {
        return makeRequest("GET", "/api/models/" + modelId, null, ModelResponse.class)
                .thenApply(response -> response.model);
    }
    
    /**
     * Get models by score range
     */
    public CompletableFuture<List<Model>> getModelsByScore(double minScore, double maxScore, int limit) {
        String query = String.format("?min_score=%.2f&max_score=%.2f&limit=%d", minScore, maxScore, limit);
        return makeRequest("GET", "/api/models/score-range" + query, null, ModelListResponse.class)
                .thenApply(response -> response.models);
    }
    
    /**
     * Verify a model with a prompt
     */
    public CompletableFuture<VerificationResult> verifyModel(String modelId, String prompt) {
        VerificationRequest request = new VerificationRequest(modelId, prompt);
        return makeRequest("POST", "/api/verify", request, VerificationResponse.class)
                .thenApply(response -> response.result);
    }
    
    /**
     * Batch verify multiple models
     */
    public CompletableFuture<List<VerificationResult>> batchVerify(List<BatchVerificationRequest> requests) {
        return makeRequest("POST", "/api/verify/batch", requests, BatchVerificationResponse.class)
                .thenApply(response -> response.results);
    }
    
    /**
     * Calculate model score
     */
    public CompletableFuture<ModelScore> calculateScore(String modelId, ScoreWeights weights) {
        CalculateScoreRequest request = new CalculateScoreRequest(modelId, weights);
        return makeRequest("POST", "/api/scoring/calculate", request, CalculateScoreResponse.class)
                .thenApply(response -> response.score);
    }
    
    /**
     * Get score history
     */
    public CompletableFuture<List<ModelScore>> getScoreHistory(String modelId, int limit) {
        String query = String.format("?model_id=%s&limit=%d", modelId, limit);
        return makeRequest("GET", "/api/scoring/history" + query, null, ScoreHistoryResponse.class)
                .thenApply(response -> response.scores);
    }
    
    /**
     * Get model rankings
     */
    public CompletableFuture<List<ModelRanking>> getRankings(String category, int limit) {
        String query = String.format("?category=%s&limit=%d", category, limit);
        return makeRequest("GET", "/api/scoring/rankings" + query, null, RankingsResponse.class)
                .thenApply(response -> response.rankings);
    }
    
    /**
     * Enterprise: LDAP Authentication
     */
    public CompletableFuture<AuthResult> ldapAuth(String username, String password) {
        LDAPAuthRequest request = new LDAPAuthRequest(username, password);
        return makeRequest("POST", "/api/enterprise/auth/ldap", request, AuthResponse.class)
                .thenApply(response -> response.result);
    }
    
    /**
     * Enterprise: SSO Authentication
     */
    public CompletableFuture<AuthResult> ssoAuth(String provider, String token) {
        SSOAuthRequest request = new SSOAuthRequest(provider, token);
        return makeRequest("POST", "/api/enterprise/auth/sso", request, AuthResponse.class)
                .thenApply(response -> response.result);
    }
    
    /**
     * Enterprise: Get user roles
     */
    public CompletableFuture<List<String>> getUserRoles(String userId) {
        return makeRequest("GET", "/api/enterprise/users/" + userId + "/roles", null, UserRolesResponse.class)
                .thenApply(response -> response.roles);
    }
    
    /**
     * Enterprise: Check permissions
     */
    public CompletableFuture<Boolean> checkPermission(String userId, String permission) {
        CheckPermissionRequest request = new CheckPermissionRequest(userId, permission);
        return makeRequest("POST", "/api/enterprise/permissions/check", request, PermissionResponse.class)
                .thenApply(response -> response.hasPermission);
    }
    
    /**
     * Make HTTP request with retry logic
     */
    private <T> CompletableFuture<T> makeRequest(String method, String path, Object body, Class<T> responseType) {
        return CompletableFuture.supplyAsync(() -> {
            int retries = 0;
            Exception lastException = null;
            
            while (retries < DEFAULT_MAX_RETRIES) {
                try {
                    HttpRequest.Builder requestBuilder = HttpRequest.newBuilder()
                            .uri(URI.create(baseUrl + path))
                            .timeout(Duration.ofSeconds(DEFAULT_TIMEOUT))
                            .header("Authorization", "Bearer " + apiKey)
                            .header("Content-Type", "application/json")
                            .header("Accept", "application/json");
                    
                    if (body != null) {
                        String jsonBody = objectMapper.writeValueAsString(body);
                        requestBuilder.method(method, HttpRequest.BodyPublishers.ofString(jsonBody));
                    } else {
                        requestBuilder.method(method, HttpRequest.BodyPublishers.noBody());
                    }
                    
                    HttpResponse<String> response = httpClient.send(
                            requestBuilder.build(), 
                            HttpResponse.BodyHandlers.ofString()
                    );
                    
                    if (response.statusCode() >= 200 && response.statusCode() < 300) {
                        return objectMapper.readValue(response.body(), responseType);
                    } else if (response.statusCode() >= 500) {
                        // Retry on server errors
                        retries++;
                        Thread.sleep(1000 * retries); // Exponential backoff
                        continue;
                    } else {
                        throw new LLMVerifierException(
                                "API request failed: " + response.statusCode() + " - " + response.body()
                        );
                    }
                } catch (Exception e) {
                    lastException = e;
                    retries++;
                    if (retries < DEFAULT_MAX_RETRIES) {
                        try {
                            Thread.sleep(1000 * retries);
                        } catch (InterruptedException ie) {
                            Thread.currentThread().interrupt();
                            throw new LLMVerifierException("Request interrupted", ie);
                        }
                    }
                }
            }
            
            throw new LLMVerifierException("Request failed after " + DEFAULT_MAX_RETRIES + " retries", lastException);
        }, executor);
    }
    
    /**
     * Close the client and release resources
     */
    public void close() {
        executor.shutdown();
        try {
            if (!executor.awaitTermination(5, TimeUnit.SECONDS)) {
                executor.shutdownNow();
            }
        } catch (InterruptedException e) {
            executor.shutdownNow();
            Thread.currentThread().interrupt();
        }
    }
    
    // Request/Response DTOs
    
    @JsonIgnoreProperties(ignoreUnknown = true)
    public static class Model {
        public String id;
        public String name;
        public String provider;
        public double overallScore;
        public String scoreSuffix;
        public Map<String, Double> scores;
        public boolean isActive;
        public Date createdAt;
        public Date updatedAt;
    }
    
    @JsonIgnoreProperties(ignoreUnknown = true)
    public static class ModelScore {
        public String modelId;
        public String modelName;
        public double score;
        public String scoreSuffix;
        public ScoreComponents components;
        public Date timestamp;
    }
    
    @JsonIgnoreProperties(ignoreUnknown = true)
    public static class ScoreComponents {
        public double responseSpeed;
        public double modelEfficiency;
        public double costEffectiveness;
        public double capability;
        public double recency;
    }
    
    @JsonIgnoreProperties(ignoreUnknown = true)
    public static class ScoreWeights {
        public double responseSpeed = 0.25;
        public double modelEfficiency = 0.20;
        public double costEffectiveness = 0.25;
        public double capability = 0.20;
        public double recency = 0.10;
    }
    
    @JsonIgnoreProperties(ignoreUnknown = true)
    public static class VerificationResult {
        public String id;
        public String modelId;
        public String prompt;
        public String response;
        public double score;
        public String scoreSuffix;
        public boolean success;
        public String error;
        public Date timestamp;
        public long duration;
    }
    
    @JsonIgnoreProperties(ignoreUnknown = true)
    public static class ModelRanking {
        public int rank;
        public String modelId;
        public String modelName;
        public double score;
        public String scoreSuffix;
        public String category;
    }
    
    @JsonIgnoreProperties(ignoreUnknown = true)
    public static class AuthResult {
        public String token;
        public String refreshToken;
        public String userId;
        public String username;
        public List<String> roles;
        public Date expiresAt;
    }
    
    // Request DTOs
    
    public static class VerificationRequest {
        public String modelId;
        public String prompt;
        
        public VerificationRequest(String modelId, String prompt) {
            this.modelId = modelId;
            this.prompt = prompt;
        }
    }
    
    public static class BatchVerificationRequest {
        public String modelId;
        public String prompt;
        
        public BatchVerificationRequest(String modelId, String prompt) {
            this.modelId = modelId;
            this.prompt = prompt;
        }
    }
    
    public static class CalculateScoreRequest {
        public String modelId;
        public ScoreWeights weights;
        
        public CalculateScoreRequest(String modelId, ScoreWeights weights) {
            this.modelId = modelId;
            this.weights = weights;
        }
    }
    
    public static class LDAPAuthRequest {
        public String username;
        public String password;
        
        public LDAPAuthRequest(String username, String password) {
            this.username = username;
            this.password = password;
        }
    }
    
    public static class SSOAuthRequest {
        public String provider;
        public String token;
        
        public SSOAuthRequest(String provider, String token) {
            this.provider = provider;
            this.token = token;
        }
    }
    
    public static class CheckPermissionRequest {
        public String userId;
        public String permission;
        
        public CheckPermissionRequest(String userId, String permission) {
            this.userId = userId;
            this.permission = permission;
        }
    }
    
    // Response DTOs
    
    @JsonIgnoreProperties(ignoreUnknown = true)
    private static class ModelListResponse {
        public List<Model> models;
    }
    
    @JsonIgnoreProperties(ignoreUnknown = true)
    private static class ModelResponse {
        public Model model;
    }
    
    @JsonIgnoreProperties(ignoreUnknown = true)
    private static class VerificationResponse {
        public VerificationResult result;
    }
    
    @JsonIgnoreProperties(ignoreUnknown = true)
    private static class BatchVerificationResponse {
        public List<VerificationResult> results;
    }
    
    @JsonIgnoreProperties(ignoreUnknown = true)
    private static class CalculateScoreResponse {
        public ModelScore score;
    }
    
    @JsonIgnoreProperties(ignoreUnknown = true)
    private static class ScoreHistoryResponse {
        public List<ModelScore> scores;
    }
    
    @JsonIgnoreProperties(ignoreUnknown = true)
    private static class RankingsResponse {
        public List<ModelRanking> rankings;
    }
    
    @JsonIgnoreProperties(ignoreUnknown = true)
    private static class AuthResponse {
        public AuthResult result;
    }
    
    @JsonIgnoreProperties(ignoreUnknown = true)
    private static class UserRolesResponse {
        public List<String> roles;
    }
    
    @JsonIgnoreProperties(ignoreUnknown = true)
    private static class PermissionResponse {
        public boolean hasPermission;
    }
    
    /**
     * Custom exception for LLM Verifier API errors
     */
    public static class LLMVerifierException extends RuntimeException {
        public LLMVerifierException(String message) {
            super(message);
        }
        
        public LLMVerifierException(String message, Throwable cause) {
            super(message, cause);
        }
    }
    
    /**
     * Builder for creating LLMVerifierClient instances
     */
    public static class Builder {
        private String baseUrl = DEFAULT_BASE_URL;
        private String apiKey;
        private int timeout = DEFAULT_TIMEOUT;
        private int maxRetries = DEFAULT_MAX_RETRIES;
        
        public Builder apiKey(String apiKey) {
            this.apiKey = apiKey;
            return this;
        }
        
        public Builder baseUrl(String baseUrl) {
            this.baseUrl = baseUrl;
            return this;
        }
        
        public Builder timeout(int timeout) {
            this.timeout = timeout;
            return this;
        }
        
        public Builder maxRetries(int maxRetries) {
            this.maxRetries = maxRetries;
            return this;
        }
        
        public LLMVerifierClient build() {
            if (apiKey == null || apiKey.isEmpty()) {
                throw new IllegalArgumentException("API key is required");
            }
            
            return new LLMVerifierClient(baseUrl, apiKey);
        }
    }
    
    /**
     * Example usage
     */
    public static void main(String[] args) {
        // Create client
        LLMVerifierClient client = new LLMVerifierClient.Builder()
                .apiKey("your-api-key")
                .baseUrl("https://api.llmverifier.com")
                .build();
        
        try {
            // Get models
            List<Model> models = client.getModels().join();
            System.out.println("Available models: " + models.size());
            
            // Verify a model
            VerificationResult result = client.verifyModel("gpt-4", "Test prompt").join();
            System.out.println("Verification result: " + result.score + " " + result.scoreSuffix);
            
            // Calculate score
            ScoreWeights weights = new ScoreWeights();
            ModelScore score = client.calculateScore("gpt-4", weights).join();
            System.out.println("Calculated score: " + score.score + " " + score.scoreSuffix);
            
        } finally {
            client.close();
        }
    }
}