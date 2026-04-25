# SPDX-License-Identifier: Apache-2.0
# Copyright (C) 2026 The Coco Sandbox Authors

"""
Coco Python SDK - Agent-native sandbox runtime for AI agents.

Usage:
    from pycoco import Sandbox

    # Create sandbox
    sandbox = Sandbox.create(template="alpine", memory_mb=512)

    # Execute code
    result = sandbox.run_code("print('hello')")
    print(result.stdout)  # "hello\\n"

    # Fork for parallel exploration
    fork = sandbox.fork()

    # Create checkpoint
    sandbox.checkpoint("before-mutation")

    # Undo if needed
    sandbox.undo("before-mutation")

    # Hibernate during idle
    sandbox.hibernate()

    # Resume when needed
    sandbox.resume()

    # Cleanup
    sandbox.destroy()
"""

from pycoco.client import CocoClient
from pycoco.sandbox import Sandbox, ExecResult, CheckpointResult, HibernateResult, ReplayResult
from pycoco.exceptions import (
    CocoError,
    SandboxNotFoundError,
    SandboxStateError,
    CheckpointNotFoundError,
    ReplayNotFoundError,
    RateLimitError,
    AuthenticationError,
)

__version__ = "0.1.0"
__all__ = [
    "CocoClient",
    "Sandbox",
    "ExecResult",
    "CheckpointResult",
    "HibernateResult",
    "ReplayResult",
    "CocoError",
    "SandboxNotFoundError",
    "SandboxStateError",
    "CheckpointNotFoundError",
    "ReplayNotFoundError",
    "RateLimitError",
    "AuthenticationError",
]