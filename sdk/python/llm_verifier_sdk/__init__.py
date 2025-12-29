"""
LLM Verifier Python SDK

A comprehensive Python SDK for the LLM Verifier platform, providing easy access
to model verification, scoring, and management capabilities.
"""

import requests
import json
from typing import Dict, List, Optional, Any, Union
from dataclasses import dataclass
from datetime import datetime
import logging

logger = logging.getLogger(__name__)


@dataclass
class Model:
    """Represents a verified LLM model"""

    id: str
    name: str
    provider: str
    verified: bool
    score: float
    features: Dict[str, Any]
    capabilities: List[str]
    metadata: Dict[str, Any]


@dataclass
class VerificationResult:
    """Result of a model verification"""

    model_id: str
    success: bool
    score: float
    capabilities: List[str]
    timestamp: datetime
    details: Dict[str, Any]


class LLMVerifierClient:
    """
    Main client for interacting with the LLM Verifier API

    Args:
        base_url: Base URL of the LLM Verifier API
        api_key: API key for authentication
        timeout: Request timeout in seconds
    """

    def __init__(
        self,
        base_url: str = "http://localhost:8080",
        api_key: Optional[str] = None,
        timeout: int = 30,
    ):
        self.base_url = base_url.rstrip("/")
        self.api_key = api_key
        self.timeout = timeout
        self.session = requests.Session()

        if api_key:
            self.session.headers.update(
                {
                    "Authorization": f"Bearer {api_key}",
                    "Content-Type": "application/json",
                }
            )

    def _make_request(self, method: str, endpoint: str, **kwargs) -> Dict[str, Any]:
        """Make HTTP request to API"""
        url = f"{self.base_url}/api/v1{endpoint}"
        kwargs.setdefault("timeout", self.timeout)

        try:
            response = self.session.request(method, url, **kwargs)
            response.raise_for_status()
            return response.json()
        except requests.exceptions.RequestException as e:
            logger.error(f"API request failed: {e}")
            raise

    def login(self, username: str, password: str) -> Dict[str, Any]:
        """Authenticate and get access token"""
        data = {"username": username, "password": password}
        response = self._make_request("POST", "/auth/login", json=data)

        if response.get("success"):
            token = response["data"]["token"]
            self.session.headers["Authorization"] = f"Bearer {token}"

        return response

    def get_models(
        self, provider: Optional[str] = None, verified_only: bool = False
    ) -> List[Model]:
        """Get list of available models"""
        params = {}
        if provider:
            params["provider"] = provider
        if verified_only:
            params["verified_only"] = "true"

        response = self._make_request("GET", "/models", params=params)

        models = []
        for model_data in response.get("data", []):
            model = Model(
                id=model_data["id"],
                name=model_data["name"],
                provider=model_data["provider"],
                verified=model_data.get("verified", False),
                score=model_data.get("score", 0.0),
                features=model_data.get("features", {}),
                capabilities=model_data.get("capabilities", []),
                metadata=model_data,
            )
            models.append(model)

        return models

    def verify_model(self, model_id: str) -> VerificationResult:
        """Trigger verification for a specific model"""
        response = self._make_request("POST", f"/models/{model_id}/verify")

        data = response.get("data", {})
        return VerificationResult(
            model_id=model_id,
            success=data.get("success", False),
            score=data.get("score", 0.0),
            capabilities=data.get("capabilities", []),
            timestamp=datetime.fromisoformat(
                data.get("timestamp", datetime.now().isoformat())
            ),
            details=data,
        )

    def get_model_details(self, model_id: str) -> Model:
        """Get detailed information about a specific model"""
        response = self._make_request("GET", f"/models/{model_id}")
        model_data = response.get("data", {})

        return Model(
            id=model_data["id"],
            name=model_data["name"],
            provider=model_data["provider"],
            verified=model_data.get("verified", False),
            score=model_data.get("score", 0.0),
            features=model_data.get("features", {}),
            capabilities=model_data.get("capabilities", []),
            metadata=model_data,
        )

    def get_providers(self) -> List[Dict[str, Any]]:
        """Get list of available providers"""
        response = self._make_request("GET", "/providers")
        return response.get("data", [])

    def get_verification_stats(self) -> Dict[str, Any]:
        """Get verification statistics and analytics"""
        response = self._make_request("GET", "/analytics/verification-stats")
        return response.get("data", {})

    def export_configuration(self, format: str = "json") -> Dict[str, Any]:
        """Export verified configuration"""
        params = {"format": format}
        response = self._make_request("GET", "/export/configuration", params=params)
        return response.get("data", {})


# Convenience functions
def create_client(
    base_url: str = "http://localhost:8080", api_key: Optional[str] = None
) -> LLMVerifierClient:
    """Create a new LLM Verifier client"""
    return LLMVerifierClient(base_url=base_url, api_key=api_key)


def quick_verify(
    model_id: str, api_key: str, base_url: str = "http://localhost:8080"
) -> VerificationResult:
    """Quick verification of a model"""
    client = LLMVerifierClient(base_url=base_url, api_key=api_key)
    return client.verify_model(model_id)
