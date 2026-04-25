// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors
/**
 * Basic usage examples for the Coco JavaScript/TypeScript SDK (jscco)
 *
 * Usage:
 *   npm install jscco
 *   npx ts-node examples/basic.ts
 */

import { Sandbox, CocoClient } from "../src/index.js";

async function main() {
  console.log("=== Coco JavaScript SDK Examples ===\n");

  const client = new CocoClient("http://localhost:4747");

  // Example 1: Health Check
  console.log("1. Health Check");
  try {
    const health = await client.health();
    console.log(`   Healthy: ${health.healthy}`);
    console.log(`   Version: ${health.version}`);
  } catch (e: any) {
    console.log(`   Error: ${e.message}`);
  }
  console.log();

  // Example 2: Create Sandbox
  console.log("2. Create Sandbox");
  let sandbox: Sandbox;
  try {
    sandbox = await Sandbox.create({
      template: "alpine",
      memoryMb: 512,
      vcpus: 2,
      name: "example-sandbox",
      client,
    });
    console.log(`   Created: ${sandbox.id}`);
    console.log(`   State: ${sandbox.state}`);
  } catch (e: any) {
    console.log(`   Error: ${e.message}`);
    return;
  }
  console.log();

  // Wait for sandbox to be running
  console.log("3. Wait for sandbox to be running...");
  for (let i = 0; i < 10; i++) {
    await new Promise((resolve) => setTimeout(resolve, 1000));
    await sandbox.refresh();
    if (sandbox.state === "running") {
      console.log("   Sandbox is now running!");
      break;
    }
  }
  console.log();

  // Example 4: Run Code
  console.log("4. Run Code in Sandbox");
  try {
    const result = await sandbox.runCode("echo 'Hello from Coco!'");
    console.log(`   stdout: ${result.stdout.trim()}`);
    console.log(`   exit_code: ${result.exitCode}`);
  } catch (e: any) {
    console.log(`   Error: ${e.message}`);
  }
  console.log();

  // Example 5: Fork Sandbox
  console.log("5. Fork Sandbox");
  try {
    const forked = await sandbox.fork();
    console.log(`   Forked: ${forked.id}`);
    console.log(`   Parent: ${sandbox.id}`);
    await forked.destroy();
    console.log("   Fork destroyed");
  } catch (e: any) {
    console.log(`   Error: ${e.message}`);
  }
  console.log();

  // Example 6: Create Checkpoint
  console.log("6. Create Checkpoint");
  try {
    const checkpoint = await sandbox.checkpoint("before-test");
    console.log(`   Checkpoint: ${checkpoint.id}`);
    console.log(`   Name: ${checkpoint.name}`);
  } catch (e: any) {
    console.log(`   Error: ${e.message}`);
  }
  console.log();

  // Example 7: Hibernate and Resume
  console.log("7. Hibernate Sandbox");
  try {
    const result = await sandbox.hibernate();
    console.log(`   State: ${result.state}`);
    console.log(`   Duration: ${result.durationMs}ms`);
  } catch (e: any) {
    console.log(`   Error: ${e.message}`);
  }
  console.log();

  console.log("8. Resume Sandbox");
  try {
    await sandbox.resume();
    console.log(`   Resumed! State: ${sandbox.state}`);
  } catch (e: any) {
    console.log(`   Error: ${e.message}`);
  }
  console.log();

  // Cleanup
  console.log("9. Cleanup - Destroy Sandbox");
  await sandbox.destroy();
  console.log(`   Destroyed: ${sandbox.id}`);
  console.log();

  console.log("All examples completed!");
}

main().catch(console.error);
