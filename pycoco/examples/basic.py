#!/usr/bin/env python3
# SPDX-License-Identifier: Apache-2.0
# Copyright (C) 2026 The Coco Sandbox Authors
"""
Basic usage examples for the Coco Python SDK (pycoco)

Usage:
    pip install pycoco
    python examples/basic.py
"""

import asyncio
import sys
import os

# Add parent directory to path for local development
sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from pycoco import Sandbox, CocoClient
from pycoco.exceptions import CocoError


async def main():
    print("=== Coco Python SDK Examples ===\n")

    # Example 1: Health Check
    print("1. Health Check")
    try:
        client = CocoClient("http://localhost:4747")
        health = await client.health()
        print(f"   Healthy: {health.healthy}")
        print(f"   Version: {health.version}")
    except CocoError as e:
        print(f"   Error: {e}")
    print()

    # Example 2: Create Sandbox
    print("2. Create Sandbox")
    try:
        sandbox = await Sandbox.create(
            template="alpine",
            memory_mb=512,
            vcpus=2,
            name="example-sandbox",
        )
        print(f"   Created: {sandbox.id}")
        print(f"   State: {sandbox.state}")
        print(f"   Vsock CID: {sandbox.vsock_cid}")
    except CocoError as e:
        print(f"   Error: {e}")
        return
    print()

    # Wait for sandbox to be running
    print("3. Wait for sandbox to be running...")
    for _ in range(10):
        await asyncio.sleep(1)
        await sandbox.refresh()
        if sandbox.state == "running":
            print(f"   Sandbox is now running!")
            break
    print()

    # Example 4: Run Code
    print("4. Run Code in Sandbox")
    try:
        result = await sandbox.run_code("echo 'Hello from Coco!'")
        print(f"   stdout: {result.stdout.strip()}")
        print(f"   exit_code: {result.exit_code}")
    except CocoError as e:
        print(f"   Error: {e}")
    print()

    # Example 5: Fork Sandbox
    print("5. Fork Sandbox")
    try:
        forked = await sandbox.fork()
        print(f"   Forked: {forked.id}")
        print(f"   Parent: {sandbox.id}")
        await forked.destroy()
        print(f"   Fork destroyed")
    except CocoError as e:
        print(f"   Error: {e}")
    print()

    # Example 6: Create Checkpoint
    print("6. Create Checkpoint")
    try:
        checkpoint = await sandbox.checkpoint("before-test")
        print(f"   Checkpoint: {checkpoint.id}")
        print(f"   Name: {checkpoint.name}")
    except CocoError as e:
        print(f"   Error: {e}")
    print()

    # Example 7: Hibernate and Resume
    print("7. Hibernate Sandbox")
    try:
        result = await sandbox.hibernate()
        print(f"   State: {result.state}")
        print(f"   Duration: {result.duration_ms}ms")
    except CocoError as e:
        print(f"   Error: {e}")
    print()

    print("8. Resume Sandbox")
    try:
        await sandbox.resume()
        print(f"   Resumed! State: {sandbox.state}")
    except CocoError as e:
        print(f"   Error: {e}")
    print()

    # Cleanup
    print("9. Cleanup - Destroy Sandbox")
    await sandbox.destroy()
    print(f"   Destroyed: {sandbox.id}")
    print()

    print("All examples completed!")


if __name__ == "__main__":
    asyncio.run(main())
