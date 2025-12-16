"""
LLM Verifier Python SDK
A Python client library for the LLM Verifier REST API
"""

import requests
from typing import Optional, List, Dict, Any


class LLMVerifierClient:
    """
    Python client for the LLM Verifier REST API

    Example:
        client = LLMVerifierClient("http://localhost:8080")
        auth = client.login("admin", "password")
        models = client.get_models(limit=10)
        print(f"Found {len(models)} models")
    """

    def __init__(self, base_url: str, api_key: Optional[str] = None):
        """
        Initialize the client

        Args:
            base_url: Base URL of the LLM Verifier API
            api_key: Optional JWT token for authentication
        """
        self.base_url = base_url.rstrip("/")
        self.api_key = api_key
        self.session = requests.Session()

    def login(self, username: str, password: str) -> Dict[str, Any]:
        """
        Authenticate user and get JWT token

        Args:
            username: Username
            password: Password

        Returns:
            Auth response with token and user info
        """
        response = self._post(
            "/auth/login", {"username": username, "password": password}
        )

        # Set token for future requests
        self.api_key = response["token"]

        return response

    def get_models(
        self,
        limit: Optional[int] = None,
        offset: Optional[int] = None,
        provider: Optional[str] = None,
    ) -> List[Dict[str, Any]]:
        """
        Get all models with optional filtering

        Args:
            limit: Maximum number of results
            offset: Pagination offset
            provider: Filter by provider name

        Returns:
            List of model dictionaries
        """
        params = {}
        if limit:
            params["limit"] = str(limit)
        if offset:
            params["offset"] = str(offset)
        if provider:
            params["provider"] = provider

        response = self._get("/api/v1/models", params)
        return response if isinstance(response, list) else []

    def get_model(self, model_id: int) -> Dict[str, Any]:
        """
        Get a specific model by ID

        Args:
            model_id: Model ID

        Returns:
            Model dictionary
        """
        return self._get(f"/api/v1/models/{model_id}")

    def create_model(self, model_data: Dict[str, Any]) -> Dict[str, Any]:
        """
        Create a new model (admin only)

        Args:
            model_data: Model data dictionary

        Returns:
            Created model dictionary
        """
        return self._post("/api/v1/models", model_data)

    def update_model(self, model_id: int, updates: Dict[str, Any]) -> Dict[str, Any]:
        """
        Update an existing model (admin only)

        Args:
            model_id: Model ID
            updates: Fields to update

        Returns:
            Updated model dictionary
        """
        return self._put(f"/api/v1/models/{model_id}", updates)

    def delete_model(self, model_id: int) -> None:
        """
        Delete a model (admin only)

        Args:
            model_id: Model ID
        """
        self._delete(f"/api/v1/models/{model_id}")

    def verify_model(self, model_id: str) -> Dict[str, Any]:
        """
        Trigger verification for a specific model

        Args:
            model_id: Model identifier

        Returns:
            Verification result dictionary
        """
        return self._post(f"/api/v1/models/{model_id}/verify", {"model_id": model_id})

    def get_verification_results(
        self, limit: Optional[int] = None, offset: Optional[int] = None
    ) -> List[Dict[str, Any]]:
        """
        Get verification results

        Args:
            limit: Maximum number of results
            offset: Pagination offset

        Returns:
            List of verification result dictionaries
        """
        params = {}
        if limit:
            params["limit"] = str(limit)
        if offset:
            params["offset"] = str(offset)

        response = self._get("/api/v1/verification-results", params)
        return response if isinstance(response, list) else []

    def get_providers(self) -> List[Dict[str, Any]]:
        """
        Get all providers

        Returns:
            List of provider dictionaries
        """
        response = self._get("/api/v1/providers")
        return response if isinstance(response, list) else []

    def get_provider(self, provider_id: int) -> Dict[str, Any]:
        """
        Get a specific provider by ID

        Args:
            provider_id: Provider ID

        Returns:
            Provider dictionary
        """
        return self._get(f"/api/v1/providers/{provider_id}")

    def get_health(self) -> Dict[str, Any]:
        """
        Get system health status

        Returns:
            Health status dictionary
        """
        return self._get("/health")

    def get_system_info(self) -> Dict[str, Any]:
        """
        Get system information

        Returns:
            System info dictionary
        """
        return self._get("/api/v1/system/info")

    # Private HTTP helper methods

    def _get(self, endpoint: str, params: Optional[Dict[str, str]] = None) -> Any:
        """Make GET request"""
        url = self.base_url + endpoint
        headers = self._get_headers()

        response = self.session.get(url, headers=headers, params=params)
        return self._handle_response(response)

    def _post(self, endpoint: str, data: Dict[str, Any]) -> Any:
        """Make POST request"""
        url = self.base_url + endpoint
        headers = self._get_headers()

        response = self.session.post(url, headers=headers, json=data)
        return self._handle_response(response)

    def _put(self, endpoint: str, data: Dict[str, Any]) -> Any:
        """Make PUT request"""
        url = self.base_url + endpoint
        headers = self._get_headers()

        response = self.session.put(url, headers=headers, json=data)
        return self._handle_response(response)

    def _delete(self, endpoint: str) -> None:
        """Make DELETE request"""
        url = self.base_url + endpoint
        headers = self._get_headers()

        response = self.session.delete(url, headers=headers)
        if not response.ok:
            raise Exception(f"HTTP {response.status_code}: {response.text}")

    def _get_headers(self) -> Dict[str, str]:
        """Get request headers"""
        headers = {"Content-Type": "application/json"}
        if self.api_key:
            headers["Authorization"] = f"Bearer {self.api_key}"
        return headers

    def _handle_response(self, response: requests.Response) -> Any:
        """Handle API response"""
        if not response.ok:
            raise Exception(f"HTTP {response.status_code}: {response.text}")

        try:
            return response.json()
        except:
            return response.text


# Example usage:
#
# client = LLMVerifierClient("http://localhost:8080")
# auth = client.login("admin", "password")
# print(f"Logged in as: {auth['user']['username']}")
#
# models = client.get_models(limit=10)
# print(f"Found {len(models)} models")
#
# health = client.get_health()
# print(f"System status: {health['status']}")
