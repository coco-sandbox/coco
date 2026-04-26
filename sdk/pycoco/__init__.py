# SPDX-License-Identifier: Apache-2.0
# Copyright (C) 2026 The Coco Sandbox Authors

"""Coco Python SDK - Agent-native sandbox runtime"""

from .sandbox import Sandbox, SandboxError

__version__ = "0.2.0"
__all__ = ["Sandbox", "SandboxError"]