# SPDX-License-Identifier: Apache-2.0
# Copyright (C) 2026 The Coco Sandbox Authors

"""Coco SDK exceptions."""


class CocoError(Exception):
    """Base exception for all Coco SDK errors."""

    def __init__(self, message: str, code: str = "internal_error", details: str = None):
        super().__init__(message)
        self.message = message
        self.code = code
        self.details = details


class SandboxNotFoundError(CocoError):
    """Raised when a sandbox is not found."""

    def __init__(self, sandbox_id: str):
        super().__init__(
            message=f"Sandbox not found: {sandbox_id}",
            code="sandbox_not_found",
            details=sandbox_id,
        )


class SandboxStateError(CocoError):
    """Raised when a sandbox is in an invalid state for the operation."""

    def __init__(self, sandbox_id: str, current_state: str, expected_states: list[str] = None):
        msg = f"Sandbox {sandbox_id} is in state '{current_state}'"
        if expected_states:
            msg += f", expected one of: {', '.join(expected_states)}"
        super().__init__(
            message=msg,
            code="invalid_state",
            details=f"current={current_state}",
        )


class CheckpointNotFoundError(CocoError):
    """Raised when a checkpoint is not found."""

    def __init__(self, checkpoint_id: str, sandbox_id: str = None):
        msg = f"Checkpoint not found: {checkpoint_id}"
        if sandbox_id:
            msg = f"Checkpoint {checkpoint_id} not found in sandbox {sandbox_id}"
        super().__init__(
            message=msg,
            code="checkpoint_not_found",
            details=checkpoint_id,
        )


class ReplayNotFoundError(CocoError):
    """Raised when a replay is not found."""

    def __init__(self, replay_id: str):
        super().__init__(
            message=f"Replay not found: {replay_id}",
            code="replay_not_found",
            details=replay_id,
        )


class RateLimitError(CocoError):
    """Raised when rate limit is exceeded."""

    def __init__(self, retry_after: int = 60):
        super().__init__(
            message=f"Rate limit exceeded, retry after {retry_after} seconds",
            code="rate_limited",
            details=str(retry_after),
        )
        self.retry_after = retry_after


class AuthenticationError(CocoError):
    """Raised when authentication fails."""

    def __init__(self, message: str = "Authentication failed"):
        super().__init__(
            message=message,
            code="unauthorized",
        )


class ExecTimeoutError(CocoError):
    """Raised when execution times out."""

    def __init__(self, timeout_ms: int):
        super().__init__(
            message=f"Execution timed out after {timeout_ms}ms",
            code="exec_timeout",
            details=str(timeout_ms),
        )


class ForkDepthExceededError(CocoError):
    """Raised when fork depth limit is exceeded."""

    def __init__(self, depth: int, max_depth: int):
        super().__init__(
            message=f"Fork depth {depth} exceeds maximum {max_depth}",
            code="fork_depth_exceeded",
            details=f"depth={depth}, max={max_depth}",
        )


class HibernateError(CocoError):
    """Raised when hibernate operation fails."""

    def __init__(self, sandbox_id: str, message: str = None):
        msg = f"Failed to hibernate sandbox {sandbox_id}"
        if message:
            msg += f": {message}"
        super().__init__(
            message=msg,
            code="hibernate_failed",
        )


class ResumeError(CocoError):
    """Raised when resume operation fails."""

    def __init__(self, sandbox_id: str, message: str = None):
        msg = f"Failed to resume sandbox {sandbox_id}"
        if message:
            msg += f": {message}"
        super().__init__(
            message=msg,
            code="resume_failed",
        )