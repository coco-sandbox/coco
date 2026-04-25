# SPDX-License-Identifier: Apache-2.0
# Copyright (C) 2026 The Coco Sandbox Authors

"""Tests for Coco Python SDK."""

import pytest


class TestExecResult:
    """Test ExecResult class."""

    def test_exec_result_creation(self):
        from coco.sandbox import ExecResult

        result = ExecResult(
            stdout="hello\n",
            stderr="",
            exit_code=0,
            duration_ms=42,
        )
        assert result.stdout == "hello\n"
        assert result.stderr == ""
        assert result.exit_code == 0
        assert result.duration_ms == 42

    def test_exec_result_repr(self):
        from coco.sandbox import ExecResult

        result = ExecResult(exit_code=0, duration_ms=100)
        assert "exit_code=0" in repr(result)
        assert "duration_ms=100" in repr(result)


class TestCheckpointResult:
    """Test CheckpointResult class."""

    def test_checkpoint_result_creation(self):
        from coco.sandbox import CheckpointResult

        cp = CheckpointResult(
            id="cp_abc123",
            name="before-test",
            sandbox_id="sb_xyz789",
            created_at="2026-04-26T00:00:00Z",
            path="/var/lib/coco/checkpoints/sb_xyz789/cp_abc123",
            size_bytes=64 * 1024 * 1024,
        )
        assert cp.id == "cp_abc123"
        assert cp.name == "before-test"
        assert cp.sandbox_id == "sb_xyz789"
        assert cp.size_bytes == 64 * 1024 * 1024


class TestHibernateResult:
    """Test HibernateResult class."""

    def test_hibernate_result_creation(self):
        from coco.sandbox import HibernateResult

        result = HibernateResult(
            id="sb_abc123",
            state="hibernated",
            hibernate_path="/var/lib/coco/hibernation/sb_abc123",
            size_bytes=512 * 1024 * 1024,
            duration_ms=1500,
        )
        assert result.state == "hibernated"
        assert result.size_bytes == 512 * 1024 * 1024
        assert result.duration_ms == 1500


class TestExceptions:
    """Test exception classes."""

    def test_coco_error(self):
        from coco.exceptions import CocoError

        err = CocoError("test error", "test_code", "details")
        assert err.message == "test error"
        assert err.code == "test_code"
        assert err.details == "details"

    def test_sandbox_not_found_error(self):
        from coco.exceptions import SandboxNotFoundError

        err = SandboxNotFoundError("sb_abc123")
        assert "sb_abc123" in str(err)
        assert err.code == "sandbox_not_found"

    def test_rate_limit_error(self):
        from coco.exceptions import RateLimitError

        err = RateLimitError(retry_after=30)
        assert err.retry_after == 30
        assert "30" in str(err)

    def test_exec_timeout_error(self):
        from coco.exceptions import ExecTimeoutError

        err = ExecTimeoutError(timeout_ms=5000)
        assert "5000" in str(err)
        assert err.code == "exec_timeout"

    def test_fork_depth_exceeded_error(self):
        from coco.exceptions import ForkDepthExceededError

        err = ForkDepthExceededError(depth=5, max_depth=4)
        assert "5" in str(err)
        assert "4" in str(err)


class TestCocoClient:
    """Test CocoClient class."""

    def test_client_init_defaults(self):
        from coco.client import CocoClient

        client = CocoClient()
        assert client.base_url == "http://localhost:4747"
        assert client.api_key is None
        assert client.timeout == 30

    def test_client_init_custom(self):
        from coco.client import CocoClient

        client = CocoClient(
            base_url="https://coco.example.com",
            api_key="test-key",
            timeout=60,
        )
        assert client.base_url == "https://coco.example.com"
        assert client.api_key == "test-key"
        assert client.timeout == 60

    def test_headers_without_key(self):
        from coco.client import CocoClient

        client = CocoClient()
        headers = client._headers()
        assert "Content-Type" in headers
        assert "Accept" in headers
        assert "X-API-Key" not in headers

    def test_headers_with_key(self):
        from coco.client import CocoClient

        client = CocoClient(api_key="my-key")
        headers = client._headers()
        assert headers["X-API-Key"] == "my-key"

    def test_client_has_post_stream(self):
        from coco.client import CocoClient

        client = CocoClient()
        assert hasattr(client, 'post_stream')
        assert callable(client.post_stream)


class TestCocoAPIError:
    """Test CocoAPIError class."""

    def test_from_response(self):
        from coco.client import CocoAPIError

        data = {
            "error": {
                "code": "sandbox_not_found",
                "message": "Sandbox not found",
                "details": "sb_abc123",
            }
        }
        err = CocoAPIError.from_response(404, data)
        assert err.code == 404
        assert err.error_code == "sandbox_not_found"
        assert err.error_message == "Sandbox not found"
        assert err.details == "sb_abc123"

    def test_str_representation(self):
        from coco.client import CocoAPIError

        err = CocoAPIError(404, "sandbox_not_found", "Sandbox not found")
        s = str(err)
        assert "404" in s
        assert "sandbox_not_found" in s
        assert "Sandbox not found" in s


class TestSandboxClass:
    """Test Sandbox class."""

    def test_sandbox_properties(self):
        from coco.client import CocoClient
        from coco.sandbox import Sandbox

        client = CocoClient()
        sb = Sandbox(client, "sb_abc123", {"state": "running", "name": "test"})
        assert sb.id == "sb_abc123"
        assert sb.state == "running"
        assert sb.name == "test"

    def test_sandbox_default_data(self):
        from coco.client import CocoClient
        from coco.sandbox import Sandbox

        client = CocoClient()
        sb = Sandbox(client, "sb_abc123")
        assert sb.state == "unknown"
        assert sb.name == ""

    def test_sandbox_created_at(self):
        from coco.client import CocoClient
        from coco.sandbox import Sandbox

        client = CocoClient()
        sb = Sandbox(client, "sb_abc123", {"created_at": "2026-04-26T00:00:00Z"})
        assert sb.created_at == "2026-04-26T00:00:00Z"

    def test_sandbox_require_state_raises(self):
        from coco.client import CocoClient
        from coco.sandbox import Sandbox
        from coco.exceptions import SandboxStateError

        client = CocoClient()
        sb = Sandbox(client, "sb_abc123", {"state": "stopped"})
        try:
            sb._require_state(["running"])
            assert False, "Should have raised"
        except SandboxStateError as e:
            assert e.sandbox_id == "sb_abc123"
            assert e.current_state == "stopped"
