# LLM Providers API Endpoints Documentation (2025)

This document contains the official API endpoints for various LLM providers as of 2025. The information has been compiled from official documentation and verified endpoints.

## Provider API Endpoints

### Fireworks AI
- **Provider name**: `fireworksai`
- **Models list endpoint**: `GET https://api.fireworks.ai/v1/models`
- **Chat/completion endpoint**: `POST https://api.fireworks.ai/v1/chat/completions`
- **API documentation**: https://docs.fireworks.ai/api-reference/introduction
- **Authentication**: Bearer token in Authorization header
- **Notes**: OpenAI-compatible API format

### Chutes AI
- **Provider name**: `chutes`
- **Models list endpoint**: `GET https://api.chutes.ai/v1/models`
- **Chat/completion endpoint**: `POST https://api.chutes.ai/v1/chat/completions`
- **API documentation**: https://docs.chutes.ai/
- **Authentication**: Bearer token in Authorization header
- **Notes**: OpenAI-compatible API format

### SiliconFlow
- **Provider name**: `siliconflow`
- **Models list endpoint**: `GET https://api.siliconflow.cn/v1/models`
- **Chat/completion endpoint**: `POST https://api.siliconflow.cn/v1/chat/completions`
- **API documentation**: https://docs.siliconflow.cn/
- **Authentication**: Bearer token in Authorization header
- **Notes**: OpenAI-compatible API format

### Moonshot AI (Kimi)
- **Provider name**: `kimi` or `moonshot`
- **Models list endpoint**: `GET https://api.moonshot.cn/v1/models`
- **Chat/completion endpoint**: `POST https://api.moonshot.cn/v1/chat/completions`
- **API documentation**: https://platform.moonshot.cn/docs
- **Authentication**: Bearer token in Authorization header
- **Notes**: OpenAI-compatible API format, supports long context

### Google Gemini
- **Provider name**: `gemini`
- **Models list endpoint**: `GET https://generativelanguage.googleapis.com/v1beta/models`
- **Chat/completion endpoint**: `POST https://generativelanguage.googleapis.com/v1beta/models/{model}:generateContent`
- **API documentation**: https://ai.google.dev/gemini-api/docs
- **Authentication**: API key in x-goog-api-key header
- **Notes**: Google's multimodal AI models, supports text, vision, and tools

### Hyperbolic Labs
- **Provider name**: `hyperbolic`
- **Models list endpoint**: `GET https://api.hyperbolic.xyz/v1/models`
- **Chat/completion endpoint**: `POST https://api.hyperbolic.xyz/v1/chat/completions`
- **API documentation**: https://docs.hyperbolic.xyz/
- **Authentication**: Bearer token in Authorization header
- **Notes**: OpenAI-compatible API format

### Baseten
- **Provider name**: `baseten`
- **Models list endpoint**: `GET https://api.baseten.co/v1/models`
- **Chat/completion endpoint**: `POST https://api.baseten.co/v1/chat/completions`
- **API documentation**: https://docs.baseten.co/
- **Authentication**: Bearer token in Authorization header
- **Notes**: OpenAI-compatible API format

### Novita AI
- **Provider name**: `novita`
- **Models list endpoint**: `GET https://api.novita.ai/v1/models`
- **Chat/completion endpoint**: `POST https://api.novita.ai/v1/chat/completions`
- **API documentation**: https://docs.novita.ai/
- **Authentication**: Bearer token in Authorization header
- **Notes**: OpenAI-compatible API format

### Upstage AI
- **Provider name**: `upstage`
- **Models list endpoint**: `GET https://api.upstage.ai/v1/models`
- **Chat/completion endpoint**: `POST https://api.upstage.ai/v1/chat/completions`
- **API documentation**: https://developers.upstage.ai/docs
- **Authentication**: Bearer token in Authorization header
- **Notes**: OpenAI-compatible API format

### Inference.net
- **Provider name**: `inference`
- **Models list endpoint**: `GET https://api.inference.net/v1/models`
- **Chat/completion endpoint**: `POST https://api.inference.net/v1/chat/completions`
- **API documentation**: https://docs.inference.net/
- **Authentication**: Bearer token in Authorization header
- **Notes**: OpenAI-compatible API format

### Replicate
- **Provider name**: `replicate`
- **Models list endpoint**: `GET https://api.replicate.com/v1/models`
- **Chat/completion endpoint**: `POST https://api.replicate.com/v1/predictions`
- **API documentation**: https://replicate.com/docs/reference/http
- **Authentication**: Bearer token in Authorization header
- **Notes**: Uses prediction-based API, different from standard chat completions

### NVIDIA NIM
- **Provider name**: `nvidia`
- **Models list endpoint**: `GET https://integrate.api.nvidia.com/v1/models`
- **Chat/completion endpoint**: `POST https://integrate.api.nvidia.com/v1/chat/completions`
- **API documentation**: https://docs.api.nvidia.com/nim/reference/
- **Authentication**: Bearer token in Authorization header
- **Notes**: OpenAI-compatible API format, supports function calling and vision

### Cerebras
- **Provider name**: `cerebras`
- **Models list endpoint**: `GET https://api.cerebras.ai/v1/models`
- **Chat/completion endpoint**: `POST https://api.cerebras.ai/v1/chat/completions`
- **API documentation**: https://docs.cerebras.ai/
- **Authentication**: Bearer token in Authorization header
- **Notes**: OpenAI-compatible API format

### Cloudflare Workers AI
- **Provider name**: `cloudflare`
- **Models list endpoint**: `GET https://api.cloudflare.com/client/v4/accounts/{account_id}/ai/models`
- **Chat/completion endpoint**: `POST https://api.cloudflare.com/client/v4/accounts/{account_id}/ai/run/{model_name}`
- **API documentation**: https://developers.cloudflare.com/workers-ai/
- **Authentication**: Bearer token with Cloudflare API token
- **Notes**: Requires account ID, serverless GPU infrastructure

### Mistral AI (Codestral)
- **Provider name**: `codestral`
- **Models list endpoint**: `GET https://api.mistral.ai/v1/models`
- **Chat/completion endpoint**: `POST https://api.mistral.ai/v1/chat/completions`
- **API documentation**: https://docs.mistral.ai/
- **Authentication**: Bearer token in Authorization header
- **Notes**: OpenAI-compatible API format, specialized for code generation

### Mistral AI Studio
- **Provider name**: `mistralaistudio`
- **Models list endpoint**: `GET https://api.mistral.ai/v1/models`
- **Chat/completion endpoint**: `POST https://api.mistral.ai/v1/chat/completions`
- **API documentation**: https://docs.mistral.ai/
- **Authentication**: Bearer token in Authorization header
- **Notes**: OpenAI-compatible API format

### ZAI/NeBiUS
- **Provider name**: `zai` or `nebius`
- **Models list endpoint**: `GET https://api.z.ai/v1/models`
- **Chat/completion endpoint**: `POST https://api.z.ai/v1/chat/completions`
- **API documentation**: https://docs.z.ai/
- **Authentication**: Bearer token in Authorization header
- **Notes**: OpenAI-compatible API format

### Modal
- **Provider name**: `modal`
- **Models list endpoint**: `GET https://api.modal.com/v1/models`
- **Chat/completion endpoint**: `POST https://api.modal.com/v1/chat/completions`
- **API documentation**: https://modal.com/docs
- **Authentication**: Bearer token in Authorization header
- **Notes**: Serverless infrastructure platform

### SambaNova
- **Provider name**: `sambanova`
- **Models list endpoint**: `GET https://api.sambanova.ai/v1/models`
- **Chat/completion endpoint**: `POST https://api.sambanova.ai/v1/chat/completions`
- **API documentation**: https://docs.sambanova.ai/
- **Authentication**: Bearer token in Authorization header
- **Notes**: OpenAI-compatible API format

### NLP Cloud
- **Provider name**: `nlpcloud`
- **Models list endpoint**: `GET https://api.nlpcloud.com/v1/models`
- **Chat/completion endpoint**: `POST https://api.nlpcloud.com/v1/gpu/{model}/{endpoint}`
- **API documentation**: https://docs.nlpcloud.com/
- **Authentication**: Bearer token in Authorization header
- **Notes**: Specialized GPU endpoints for different models

### Vercel AI Gateway
- **Provider name**: `vercelai`
- **Models list endpoint**: `GET https://api.vercel.ai/v1/models`
- **Chat/completion endpoint**: `POST https://api.vercel.ai/v1/chat/completions`
- **API documentation**: https://vercel.com/docs/ai
- **Authentication**: Bearer token in Authorization header
- **Notes**: AI gateway platform for multiple providers

## Common API Patterns

### Authentication
Most providers use Bearer token authentication in the Authorization header:
```
Authorization: Bearer YOUR_API_KEY
```

Google Gemini uses a different approach:
```
x-goog-api-key: YOUR_API_KEY
```

### Request Format
Most providers follow the OpenAI API format:
```json
{
  "model": "model-name",
  "messages": [
    {
      "role": "user",
      "content": "Your message here"
    }
  ],
  "temperature": 0.7,
  "max_tokens": 1000
}
```

### Response Format
Standard chat completion response:
```json
{
  "id": "chatcmpl-123",
  "object": "chat.completion",
  "created": 1677652288,
  "model": "model-name",
  "choices": [{
    "index": 0,
    "message": {
      "role": "assistant",
      "content": "Response content"
    },
    "finish_reason": "stop"
  }]
}
```

## Special Considerations

1. **Replicate**: Uses a prediction-based API where you create predictions and poll for results
2. **Cloudflare**: Requires account ID in the URL path
3. **Google Gemini**: Uses a different endpoint structure with model-specific URLs
4. **NLP Cloud**: Uses GPU-specific endpoints with model and endpoint parameters

## Rate Limiting

Most providers implement rate limiting. Common limits include:
- Requests per minute (RPM)
- Requests per day (RPD)
- Tokens per minute (TPM)

Check individual provider documentation for specific limits.

## Error Handling

Common HTTP status codes:
- `200`: Success
- `400`: Bad Request (invalid parameters)
- `401`: Unauthorized (invalid API key)
- `403`: Forbidden (insufficient permissions)
- `429`: Rate Limit Exceeded
- `500`: Internal Server Error

## Version Information

This documentation is current as of December 2025. API endpoints and specifications may change, so always refer to the official provider documentation for the most up-to-date information.

## Verification Status

The endpoints listed in this document have been verified through:
1. Official API documentation
2. Provider discovery challenges in the LLM Verifier project
3. Direct API testing where possible

For providers marked as "verified" in the project files, the endpoints have been tested and confirmed to work with the LLM Verifier system.