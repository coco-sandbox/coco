# SPDX-License-Identifier: Apache-2.0
# Copyright (C) 2026 The Coco Sandbox Authors

"""Coco API client."""

import json
import urllib.request
import urllib.error
from typing import Any


class CocoClient:
    """
    Low-level HTTP client for the Coco API.

    This client handles authentication, request/response serialization,
    and error handling for all Coco API operations.
    """

    def __init__(
        self,
        base_url: str = "http://localhost:4747",
        api_key: str = None,
        timeout: int = 30,
    ):
        """
        Initialize Coco API client.

        Args:
            base_url: Base URL of the Coco API server.
            api_key: API key for authentication.
            timeout: Request timeout in seconds.
        """
        self.base_url = base_url.rstrip("/")
        self.api_key = api_key
        self.timeout = timeout

    def _headers(self) -> dict[str, str]:
        """Build request headers."""
        headers = {
            "Content-Type": "application/json",
            "Accept": "application/json",
        }
        if self.api_key:
            headers["X-API-Key"] = self.api_key
        return headers

    def _request(
        self,
        method: str,
        path: str,
        body: dict = None,
        params: dict = None,
    ) -> dict[str, Any]:
        """
        Make an HTTP request to the Coco API.

        Args:
            method: HTTP method (GET, POST, PATCH, DELETE).
            path: API path (e.g., "/v1/sandboxes").
            body: Request body dict.
            params: Query parameters.

        Returns:
            Parsed JSON response.

        Raises:
            urllib.error.HTTPError: On HTTP errors.
        """
        url = f"{self.base_url}{path}"
        if params:
            query = "&".join(f"{k}={v}" for k, v in params.items())
            url = f"{url}?{query}"

        data = json.dumps(body).encode("utf-8") if body else None
        req = urllib.request.Request(
            url,
            data=data,
            headers=self._headers(),
            method=method,
        )

        try:
            with urllib.request.urlopen(req, timeout=self.timeout) as resp:
                return json.loads(resp.read().decode("utf-8"))
        except urllib.error.HTTPError as e:
            body = e.read().decode("utf-8")
            try:
                error_data = json.loads(body)
                raise CocoAPIError.from_response(e.code, error_data) from e
            except (json.JSONDecodeError, ValueError):
                raise CocoAPIError(e.code, "unknown", body) from e

    def get(self, path: str, params: dict = None) -> dict[str, Any]:
        """GET request."""
        return self._request("GET", path, params=params)

    def post(self, path: str, body: dict = None) -> dict[str, Any]:
        """POST request."""
        return self._request("POST", path, body=body)

    def patch(self, path: str, body: dict = None) -> dict[str, Any]:
        """PATCH request."""
        return self._request("PATCH", path, body=body)

    def delete(self, path: str) -> dict[str, Any]:
        """DELETE request."""
        return self._request("DELETE", path)

    def post_stream(self, path: str, body: dict = None) -> list[str]:
        """
        Make a streaming POST request to the Coco API.

        Args:
            path: API path (e.g., "/v1/sandboxes/{id}/exec").
            body: Request body dict.

        Returns:
            List of response lines (for streaming endpoints).

        Raises:
            urllib.error.HTTPError: On HTTP errors.
        """
        url = f"{self.base_url}{path}"
        data = json.dumps(body).encode("utf-8") if body else None
        req = urllib.request.Request(
            url,
            data=data,
            headers=self._headers(),
            method="POST",
        )

        try:
            with urllib.request.urlopen(req, timeout=self.timeout) as resp:
                lines = []
                for line in resp:
                    decoded = line.decode("utf-8")
                    lines.append(decoded)
                    if b"\n" not in line and not line.strip():
                        break
                return lines
        except urllib.error.HTTPError as e:
            body = e.read().decode("utf-8")
            try:
                error_data = json.loads(body)
                raise CocoAPIError.from_response(e.code, error_data) from e
            except (json.JSONDecodeError, ValueError):
                raise CocoAPIError(e.code, "unknown", body) from e


class CocoAPIError(urllib.error.HTTPError):
    """Coco API error with structured error response."""

    def __init__(self, code: int, error_code: str, message: str, details: str = None):
        super().__init__(
            None,  # url
            code,  # code
            message,  # msg
            {},  # hdrs
            None,  # fp
        )
        self.error_code = error_code
        self.error_message = message
        self.details = details

    @classmethod
    def from_response(cls, code: int, data: dict) -> "CocoAPIError":
        """Create from parsed JSON error response."""
        error = data.get("error", {})
        return cls(
            code=code,
            error_code=error.get("code", "unknown"),
            message=error.get("message", "Unknown error"),
            details=error.get("details"),
        )

    def __str__(self) -> str:
        return f"CocoAPIError({self.code}, {self.error_code}): {self.error_message}"


# Import exceptions at module level for convenience
from coco.exceptions import (
    CocoError,
    SandboxNotFoundError,
    SandboxStateError,
    CheckpointNotFoundError,
    ReplayNotFoundError,
    RateLimitError,
    AuthenticationError,
    ExecTimeoutError,
    ForkDepthExceededError,
    HibernateError,
    ResumeError,
)