# SPDX-License-Identifier: Apache-2.0
# Copyright (C) 2026 The Coco Sandbox Authors

"""Coco Sandbox high-level API."""

from __future__ import annotations

import json
import time
from typing import Any

from coco.client import CocoClient
from coco.exceptions import (
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
    CocoAPIError,
)


class ExecResult:
    """Result of a code execution."""

    def __init__(
        self,
        stdout: str = "",
        stderr: str = "",
        exit_code: int = 0,
        duration_ms: int = 0,
    ):
        self.stdout = stdout
        self.stderr = stderr
        self.exit_code = exit_code
        self.duration_ms = duration_ms

    def __repr__(self) -> str:
        return f"ExecResult(exit_code={self.exit_code}, duration_ms={self.duration_ms})"


class Sandbox:
    """
    A Coco sandbox instance.

    Sandboxes are isolated execution environments with support for:
    - Code execution
    - Forking (cloning the sandbox)
    - Checkpointing (snapshotting state)
    - Hibernate/Resume (persist to disk, restore)
    - Replay (record and replay execution)
    """

    def __init__(
        self,
        client: CocoClient,
        sandbox_id: str,
        data: dict[str, Any] = None,
    ):
        """
        Initialize a Sandbox wrapper.

        Typically not called directly; use Sandbox.create() or
        Sandbox.get() class methods instead.
        """
        self._client = client
        self._id = sandbox_id
        self._data = data or {}

    @classmethod
    def create(
        cls,
        template: str = "alpine",
        memory_mb: int = 512,
        vcpus: int = 2,
        name: str = None,
        labels: dict[str, str] = None,
        client: CocoClient = None,
    ) -> Sandbox:
        """
        Create a new sandbox.

        Args:
            template: OS template to use (default: "alpine").
            memory_mb: Memory limit in MB (default: 512).
            vcpus: Number of virtual CPUs (default: 2).
            name: Optional sandbox name.
            labels: Optional key-value labels.
            client: CocoClient instance. Uses default if None.

        Returns:
            A Sandbox instance.

        Raises:
            RateLimitError: If rate limit is exceeded.
            AuthenticationError: If authentication fails.
        """
        if client is None:
            client = CocoClient()

        body = {
            "template": template,
            "memory_mb": memory_mb,
            "vcpus": vcpus,
        }
        if name:
            body["name"] = name
        if labels:
            body["labels"] = labels

        try:
            resp = client.post("/v1/sandboxes", body)
            data = resp.get("sandbox", {})
            return cls(client, data["id"], data)
        except CocoAPIError as e:
            if e.code == 429:
                raise RateLimitError() from e
            if e.code == 401:
                raise AuthenticationError(e.error_message) from e
            raise

    @classmethod
    def get(cls, sandbox_id: str, client: CocoClient = None) -> Sandbox:
        """
        Get an existing sandbox by ID.

        Args:
            sandbox_id: The sandbox ID.
            client: CocoClient instance. Uses default if None.

        Returns:
            A Sandbox instance.

        Raises:
            SandboxNotFoundError: If sandbox doesn't exist.
        """
        if client is None:
            client = CocoClient()

        try:
            resp = client.get(f"/v1/sandboxes/{sandbox_id}")
            data = resp.get("sandbox", {})
            return cls(client, sandbox_id, data)
        except CocoAPIError as e:
            if e.code == 404:
                raise SandboxNotFoundError(sandbox_id) from e
            raise

    @classmethod
    def list(
        cls,
        state: str = None,
        label_key: str = None,
        label_value: str = None,
        offset: int = 0,
        limit: int = 100,
        client: CocoClient = None,
    ) -> list[Sandbox]:
        """
        List sandboxes with optional filtering.

        Args:
            state: Filter by sandbox state.
            label_key: Filter by label key.
            label_value: Filter by label value (requires label_key).
            offset: Pagination offset.
            limit: Pagination limit.
            client: CocoClient instance.

        Returns:
            List of Sandbox instances.
        """
        if client is None:
            client = CocoClient()

        params = {"offset": offset, "limit": limit}
        if state:
            params["state"] = state
        if label_key:
            params["label_key"] = label_key
            if label_value:
                params["label_value"] = label_value

        resp = client.get("/v1/sandboxes", params)
        items = resp.get("items", [])
        return [cls(client, item["id"], item) for item in items]

    @property
    def id(self) -> str:
        """Sandbox ID."""
        return self._id

    @property
    def state(self) -> str:
        """Current sandbox state."""
        return self._data.get("state", "unknown")

    @property
    def name(self) -> str:
        """Sandbox name."""
        return self._data.get("name", "")

    @property
    def created_at(self) -> str:
        """Creation timestamp (ISO 8601)."""
        return self._data.get("created_at", "")

    def refresh(self) -> None:
        """Refresh sandbox state from server."""
        try:
            resp = self._client.get(f"/v1/sandboxes/{self._id}")
            self._data = resp.get("sandbox", {})
        except CocoAPIError as e:
            if e.code == 404:
                raise SandboxNotFoundError(self._id) from e
            raise

    def _require_state(self, expected: list[str]) -> None:
        """Verify sandbox is in an expected state."""
        if self.state not in expected:
            raise SandboxStateError(self._id, self.state, expected)

    def run_code(
        self,
        code: str,
        timeout_ms: int = 30000,
        working_dir: str = "/tmp",
        env: dict[str, str] = None,
    ) -> ExecResult:
        """
        Execute code in the sandbox.

        Args:
            code: Code to execute.
            timeout_ms: Execution timeout in milliseconds.
            working_dir: Working directory for execution.
            env: Environment variables.

        Returns:
            ExecResult with stdout, stderr, exit_code.

        Raises:
            SandboxStateError: If sandbox is not running.
            ExecTimeoutError: If execution times out.
        """
        self._require_state(["running"])

        body = {
            "cmd": "sh",
            "args": ["-c", code],
            "timeout_ms": timeout_ms,
            "working_dir": working_dir,
        }
        if env:
            body["env"] = [f"{k}={v}" for k, v in env.items()]

        # Exec returns a streaming response - parse each line
        stdout_chunks = []
        stderr_chunks = []
        exit_code = 0

        try:
            resp = self._client.post_stream(f"/v1/sandboxes/{self._id}/exec", body)
            for line in resp:
                if not line.strip():
                    continue
                try:
                    chunk = json.loads(line)
                    stream_type = chunk.get("stream_type")
                    if stream_type == 1:  # stdout
                        stdout_chunks.append(chunk.get("data", ""))
                    elif stream_type == 2:  # stderr
                        stderr_chunks.append(chunk.get("data", ""))
                    elif stream_type == 3:  # exit
                        exit_code = chunk.get("exit_code", 0)
                        if chunk.get("error"):
                            stderr_chunks.append(chunk.get("error"))
                except json.JSONDecodeError:
                    continue
            return ExecResult(
                stdout="".join(stdout_chunks),
                stderr="".join(stderr_chunks),
                exit_code=exit_code,
                duration_ms=0,
            )
        except CocoAPIError as e:
            if e.code == 400 and "timeout" in e.error_message.lower():
                raise ExecTimeoutError(timeout_ms) from e
            raise

    def pause(self) -> None:
        """
        Pause the sandbox (freeze execution).

        Raises:
            SandboxStateError: If sandbox is not running.
        """
        self._require_state(["running"])
        self._client.post(f"/v1/sandboxes/{self._id}/pause")
        self._data["state"] = "paused"

    def resume(self) -> None:
        """
        Resume a paused sandbox.

        Raises:
            SandboxStateError: If sandbox is not paused.
            ResumeError: If resume operation fails.
        """
        self._require_state(["paused"])
        try:
            resp = self._client.post(f"/v1/sandboxes/{self._id}/resume")
            self._data["state"] = resp.get("state", "running")
        except CocoAPIError as e:
            if e.code == 400:
                raise ResumeError(self._id, e.error_message) from e
            raise

    def hibernate(self, hibernate_path: str = None) -> HibernateResult:
        """
        Hibernate the sandbox to disk.

        This suspends the sandbox state to persistent storage,
        allowing fast resume later.

        Args:
            hibernate_path: Optional custom hibernate path.

        Returns:
            HibernateResult with hibernate details.

        Raises:
            SandboxStateError: If sandbox is not running.
            HibernateError: If hibernate fails.
        """
        self._require_state(["running"])

        body = {}
        if hibernate_path:
            body["hibernate_path"] = hibernate_path

        try:
            resp = self._client.post(f"/v1/sandboxes/{self._id}/hibernate", body)
            self._data["state"] = "hibernated"
            return HibernateResult(
                id=resp.get("id", self._id),
                state=resp.get("state", "hibernated"),
                hibernate_path=resp.get("hibernate_path", ""),
                size_bytes=resp.get("size_bytes", 0),
                duration_ms=resp.get("duration_ms", 0),
            )
        except CocoAPIError as e:
            raise HibernateError(self._id, e.error_message) from e

    def fork(self) -> Sandbox:
        """
        Fork (clone) the sandbox.

        Creates a new sandbox that is a copy of this one,
        useful for parallel hypothesis exploration.

        Returns:
            A new Sandbox instance (the fork).

        Raises:
            SandboxStateError: If sandbox is not running.
            ForkDepthExceededError: If fork depth limit is exceeded.
        """
        self._require_state(["running"])

        try:
            resp = self._client.post(f"/v1/sandboxes/{self._id}/fork")
            fork_data = resp.get("sandbox", {})
            return Sandbox(self._client, fork_data["id"], fork_data)
        except CocoAPIError as e:
            if e.code == 400 and "fork" in e.error_message.lower():
                raise ForkDepthExceededError(
                    self._data.get("fork_depth", 0),
                    16,  # max fork depth
                ) from e
            raise

    def checkpoint(self, name: str = None) -> CheckpointResult:
        """
        Create a checkpoint (snapshot) of the sandbox.

        Args:
            name: Optional checkpoint name.

        Returns:
            CheckpointResult with checkpoint details.

        Raises:
            SandboxStateError: If sandbox is not running.
        """
        self._require_state(["running"])

        body = {}
        if name:
            body["name"] = name

        resp = self._client.post(f"/v1/sandboxes/{self._id}/checkpoints", body)
        data = resp.get("checkpoint", {})
        return CheckpointResult(
            id=data.get("id", ""),
            name=data.get("name", name or ""),
            sandbox_id=self._id,
            created_at=data.get("created_at", ""),
            path=data.get("path", ""),
            size_bytes=data.get("size_bytes", 0),
        )

    def list_checkpoints(self) -> list[CheckpointResult]:
        """
        List all checkpoints for this sandbox.

        Returns:
            List of CheckpointResult.
        """
        resp = self._client.get(f"/v1/sandboxes/{self._id}/checkpoints")
        items = resp.get("checkpoints", [])
        return [
            CheckpointResult(
                id=cp.get("id", ""),
                name=cp.get("name", ""),
                sandbox_id=self._id,
                created_at=cp.get("created_at", ""),
                path=cp.get("path", ""),
                size_bytes=cp.get("size_bytes", 0),
            )
            for cp in items
        ]

    def restore(self, checkpoint_id: str = None) -> None:
        """
        Restore sandbox to a checkpoint.

        Args:
            checkpoint_id: Checkpoint ID to restore. Uses latest if None.

        Raises:
            CheckpointNotFoundError: If checkpoint doesn't exist.
            SandboxStateError: If sandbox is not in a restorable state.
        """
        self._require_state(["stopped", "error"])

        body = {}
        if checkpoint_id:
            body["checkpoint_id"] = checkpoint_id

        self._client.post(f"/v1/sandboxes/{self._id}/restore", body)
        self._data["state"] = "running"

    def undo(self, checkpoint_name: str = None) -> CheckpointResult:
        """
        Undo to a named checkpoint.

        Args:
            checkpoint_name: Checkpoint name to undo to. Uses earliest if None.

        Returns:
            CheckpointResult for the restored checkpoint.

        Raises:
            CheckpointNotFoundError: If no checkpoints exist.
            SandboxStateError: If sandbox is not in a restorable state.
        """
        self._require_state(["running", "paused", "stopped", "error"])

        body = {}
        if checkpoint_name:
            body["name"] = checkpoint_name

        resp = self._client.post(f"/v1/sandboxes/{self._id}/undo", body)
        cp_data = resp.get("checkpoint", {})
        return CheckpointResult(
            id=cp_data.get("id", ""),
            name=cp_data.get("name", checkpoint_name or ""),
            sandbox_id=self._id,
            created_at=cp_data.get("created_at", ""),
            path=cp_data.get("path", ""),
            size_bytes=cp_data.get("size_bytes", 0),
        )

    def redo(self) -> CheckpointResult:
        """
        Redo (restore to next checkpoint in chain).

        Returns:
            CheckpointResult for the restored checkpoint.

        Raises:
            SandboxStateError: If no redo is available.
        """
        self._require_state(["running", "paused", "stopped", "error"])

        resp = self._client.post(f"/v1/sandboxes/{self._id}/redo", body={})
        cp_data = resp.get("checkpoint", {})
        return CheckpointResult(
            id=cp_data.get("id", ""),
            name=cp_data.get("name", ""),
            sandbox_id=self._id,
            created_at=cp_data.get("created_at", ""),
            path=cp_data.get("path", ""),
            size_bytes=cp_data.get("size_bytes", 0),
        )

    def replay_start(self, name: str = None) -> ReplayResult:
        """
        Start recording a replay session.

        Args:
            name: Optional replay name.

        Returns:
            ReplayResult for the started replay.
        """
        body = {}
        if name:
            body["name"] = name

        resp = self._client.post(f"/v1/sandboxes/{self._id}/replay/start", body)
        replay_data = resp.get("replay", {})
        return ReplayResult(
            id=replay_data.get("id", ""),
            name=replay_data.get("name", name or ""),
            sandbox_id=self._id,
            state=replay_data.get("state", "recording"),
        )

    def replay_stop(self) -> ReplayResult:
        """
        Stop the current replay recording.

        Returns:
            ReplayResult for the stopped replay.
        """
        resp = self._client.post(f"/v1/sandboxes/{self._id}/replay/stop")
        replay_data = resp.get("replay", {})
        return ReplayResult(
            id=replay_data.get("id", ""),
            name=replay_data.get("name", ""),
            sandbox_id=self._id,
            state=replay_data.get("state", "stopped"),
        )

    def destroy(self) -> None:
        """
        Destroy the sandbox.

        This stops and cleans up all resources associated with
        the sandbox.
        """
        self._client.delete(f"/v1/sandboxes/{self._id}")
        self._data["state"] = "stopped"


class CheckpointResult:
    """Result of a checkpoint operation."""

    def __init__(
        self,
        id: str,
        name: str,
        sandbox_id: str,
        created_at: str = "",
        path: str = "",
        size_bytes: int = 0,
    ):
        self.id = id
        self.name = name
        self.sandbox_id = sandbox_id
        self.created_at = created_at
        self.path = path
        self.size_bytes = size_bytes

    def __repr__(self) -> str:
        return f"CheckpointResult(id={self.id}, name={self.name})"


class HibernateResult:
    """Result of a hibernate operation."""

    def __init__(
        self,
        id: str,
        state: str,
        hibernate_path: str = "",
        size_bytes: int = 0,
        duration_ms: int = 0,
    ):
        self.id = id
        self.state = state
        self.hibernate_path = hibernate_path
        self.size_bytes = size_bytes
        self.duration_ms = duration_ms

    def __repr__(self) -> str:
        return f"HibernateResult(state={self.state}, duration_ms={self.duration_ms})"


class ReplayResult:
    """Result of a replay operation."""

    def __init__(
        self,
        id: str,
        name: str,
        sandbox_id: str,
        state: str = "",
        start_time: str = "",
    ):
        self.id = id
        self.name = name
        self.sandbox_id = sandbox_id
        self.state = state
        self.start_time = start_time

    def __repr__(self) -> str:
        return f"ReplayResult(id={self.id}, state={self.state})"