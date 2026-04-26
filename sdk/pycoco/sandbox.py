# SPDX-License-Identifier: Apache-2.0
# Copyright (C) 2026 The Coco Sandbox Authors

"""Coco Python SDK - Agent-native sandbox runtime"""

import requests
import asyncio
import time
from typing import Optional, List, Callable

class SandboxError(Exception):
    pass

class Sandbox:
    """High-level sandbox wrapper with context manager support"""

    def __init__(self, id: str, base_url: str = "http://localhost:4747"):
        self.id = id
        self.base_url = base_url
        self._closed = False

    @classmethod
    async def create(cls, template: str = "python-3.11", memory_mb: int = 512,
                    vcpus: int = 2, name: Optional[str] = None,
                    labels: Optional[dict] = None) -> "Sandbox":
        """Create and start a new sandbox"""
        payload = {
            "template": template,
            "memory_mb": memory_mb,
            "vcpus": vcpus,
            "name": name or f"sandbox-{int(time.time())}",
            "labels": labels or {},
        }

        async with requests.AsyncClient() as client:
            resp = await client.post(f"{self.base_url}/v1/sandboxes", json=payload)
            if resp.status_code != 201:
                raise SandboxError(f"Failed to create sandbox: {resp.text}")

            data = resp.json()
            return cls(id=data["sandbox"]["id"], base_url=self.base_url)

    async def exec(self, command: str, timeout: int = 30,
                  streaming: Optional[Callable] = None) -> dict:
        """Execute a command in the sandbox"""
        if self._closed:
            raise SandboxError("Sandbox has been closed")

        payload = {
            "command": command,
            "timeout_ms": timeout * 1000,
            "streaming": streaming is not None,
        }

        if streaming:
            return await self._exec_streaming(payload, streaming)
        else:
            return await self._exec_sync(payload)

    async def _exec_sync(self, payload: dict) -> dict:
        async with requests.AsyncClient() as client:
            resp = await client.post(
                f"{self.base_url}/v1/sandboxes/{self.id}/exec",
                json=payload
            )
            if resp.status_code != 200:
                raise SandboxError(f"Exec failed: {resp.text}")
            return resp.json()

    async def _exec_streaming(self, payload: dict, callback: Callable):
        async with requests.AsyncClient() as client:
            async with client.stream(
                "POST",
                f"{self.base_url}/v1/sandboxes/{self.id}/streaming-exec",
                json=payload
            ) as resp:
                async for line in resp.iter_lines():
                    if line.startswith("data: "):
                        import json
                        data = json.loads(line[6:])
                        callback(data)

    async def pause(self):
        async with requests.AsyncClient() as client:
            resp = await client.post(f"{self.base_url}/v1/sandboxes/{self.id}/pause")
            if resp.status_code != 200:
                raise SandboxError(f"Pause failed: {resp.text}")

    async def resume(self):
        async with requests.AsyncClient() as client:
            resp = await client.post(f"{self.base_url}/v1/sandboxes/{self.id}/resume")
            if resp.status_code != 200:
                raise SandboxError(f"Resume failed: {resp.text}")

    async def fork(self) -> "Sandbox":
        async with requests.AsyncClient() as client:
            resp = await client.post(f"{self.base_url}/v1/sandboxes/{self.id}/fork")
            if resp.status_code != 201:
                raise SandboxError(f"Fork failed: {resp.text}")
            data = resp.json()
            return Sandbox(id=data["sandbox"]["id"], base_url=self.base_url)

    async def checkpoint(self, name: str, description: str = "") -> dict:
        async with requests.AsyncClient() as client:
            resp = await client.post(
                f"{self.base_url}/v1/sandboxes/{self.id}/checkpoints",
                json={"name": name, "description": description}
            )
            if resp.status_code != 201:
                raise SandboxError(f"Checkpoint failed: {resp.text}")
            return resp.json()

    async def hibernate(self):
        async with requests.AsyncClient() as client:
            resp = await client.post(f"{self.base_url}/v1/sandboxes/{self.id}/hibernate")
            if resp.status_code != 200:
                raise SandboxError(f"Hibernate failed: {resp.text}")

    async def close(self):
        if self._closed:
            return
        async with requests.AsyncClient() as client:
            resp = await client.delete(f"{self.base_url}/v1/sandboxes/{self.id}")
            self._closed = True
            if resp.status_code not in (200, 204):
                raise SandboxError(f"Close failed: {resp.text}")

    async def __aenter__(self):
        return self

    async def __aexit__(self, *args):
        await self.close()