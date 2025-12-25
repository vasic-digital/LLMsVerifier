# Data Model: Extend LLM Providers Support

## Provider Entity (Extended)

**Purpose**: Store configuration and metadata for LLM providers, extended to include new providers.

**Fields**:
- id: int64 (primary key)
- name: string (unique, e.g., "Groq", "Together AI")
- endpoint: string (API base URL)
- api_key_encrypted: string (encrypted API key)
- description: string (human-readable description)
- website: string (provider's website)
- is_active: bool (whether provider is enabled)
- created_at: timestamp
- updated_at: timestamp

**Validation Rules**:
- name: required, unique, matches known provider names
- endpoint: required, valid URL
- api_key_encrypted: required for active providers

**Relationships**:
- Has many Models (one-to-many)
- Has many VerificationResults (one-to-many)
- Has one PricingInfo (one-to-one)
- Has one LimitsInfo (one-to-one)

## Model Entity (Extended)

**Purpose**: Store model metadata, extended with new models from additional providers.

**Fields**:
- id: int64 (primary key)
- provider_id: int64 (foreign key to Provider)
- model_id: string (provider's model identifier)
- name: string (display name)
- description: string (model description)
- version: string (model version)
- architecture: string (underlying architecture)
- max_input_tokens: int (maximum input tokens)
- max_output_tokens: int (maximum output tokens)
- supports_streaming: bool
- supports_reasoning: bool
- created_at: timestamp
- updated_at: timestamp

**Validation Rules**:
- provider_id: required, references valid provider
- model_id: required, unique per provider
- max_input_tokens, max_output_tokens: positive integers

**Relationships**:
- Belongs to Provider (many-to-one)
- Has many VerificationResults (one-to-many)

## VerificationResult Entity (Unchanged)

**Purpose**: Store results of model verification tests.

**Fields**: (unchanged from existing)
- id, model_id, provider_id, test_type, scores, evidence, etc.

## PricingInfo Entity (Extended)

**Purpose**: Store pricing information for providers, extended for new providers.

**Fields**:
- provider_id: int64 (primary key, foreign key)
- input_price_per_token: float64
- output_price_per_token: float64
- currency: string (default "USD")
- last_updated: timestamp

**Validation Rules**:
- provider_id: required, unique
- prices: non-negative floats

## LimitsInfo Entity (Extended)

**Purpose**: Store rate limits and quotas for providers, extended for new providers.

**Fields**:
- provider_id: int64 (primary key, foreign key)
- requests_per_minute: int
- requests_per_hour: int
- tokens_per_minute: int
- tokens_per_hour: int
- last_updated: timestamp

**Validation Rules**:
- provider_id: required, unique
- limits: positive integers

## State Transitions

**Provider States**:
- Inactive → Active (when API key configured and validated)
- Active → Inactive (when API key removed or invalid)

**Model States**:
- Unverified → Verified (after successful verification)
- Verified → Unverified (if verification fails repeatedly)

## Data Integrity Rules

- Cannot delete Provider with associated Models
- Cannot delete Model with associated VerificationResults
- API keys must be encrypted at rest
- All price/limit data must have last_updated timestamps