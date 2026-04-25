// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

/**
 * Coco SDK exceptions.
 */

export class CocoError extends Error {
  constructor(
    public readonly message: string,
    public readonly code: string = "internal_error",
    public readonly details: string | null = null
  ) {
    super(message);
    this.name = "CocoError";
  }
}

export class SandboxNotFoundError extends CocoError {
  constructor(public readonly sandboxId: string) {
    super(`Sandbox not found: ${sandboxId}`, "sandbox_not_found", sandboxId);
    this.name = "SandboxNotFoundError";
  }
}

export class SandboxStateError extends CocoError {
  constructor(
    public readonly sandboxId: string,
    public readonly currentState: string,
    expectedStates?: string[]
  ) {
    let msg = `Sandbox ${sandboxId} is in state '${currentState}'`;
    if (expectedStates) {
      msg += `, expected one of: ${expectedStates.join(", ")}`;
    }
    super(msg, "invalid_state", `current=${currentState}`);
    this.name = "SandboxStateError";
  }
}

export class CheckpointNotFoundError extends CocoError {
  constructor(public readonly checkpointId: string, sandboxId?: string) {
    const msg = sandboxId
      ? `Checkpoint ${checkpointId} not found in sandbox ${sandboxId}`
      : `Checkpoint not found: ${checkpointId}`;
    super(msg, "checkpoint_not_found", checkpointId);
    this.name = "CheckpointNotFoundError";
  }
}

export class ReplayNotFoundError extends CocoError {
  constructor(public readonly replayId: string) {
    super(`Replay not found: ${replayId}`, "replay_not_found", replayId);
    this.name = "ReplayNotFoundError";
  }
}

export class RateLimitError extends CocoError {
  constructor(public readonly retryAfter: number = 60) {
    super(`Rate limit exceeded, retry after ${retryAfter} seconds`, "rate_limited", String(retryAfter));
    this.name = "RateLimitError";
  }
}

export class AuthenticationError extends CocoError {
  constructor(message: string = "Authentication failed") {
    super(message, "unauthorized");
    this.name = "AuthenticationError";
  }
}

export class ExecTimeoutError extends CocoError {
  constructor(public readonly timeoutMs: number) {
    super(`Execution timed out after ${timeoutMs}ms`, "exec_timeout", String(timeoutMs));
    this.name = "ExecTimeoutError";
  }
}

export class ForkDepthExceededError extends CocoError {
  constructor(public readonly depth: number, public readonly maxDepth: number) {
    super(`Fork depth ${depth} exceeds maximum ${maxDepth}`, "fork_depth_exceeded", `depth=${depth}, max=${maxDepth}`);
    this.name = "ForkDepthExceededError";
  }
}

export class HibernateError extends CocoError {
  constructor(public readonly sandboxId: string, message?: string) {
    const msg = message ? `Failed to hibernate sandbox ${sandboxId}: ${message}` : `Failed to hibernate sandbox ${sandboxId}`;
    super(msg, "hibernate_failed");
    this.name = "HibernateError";
  }
}

export class ResumeError extends CocoError {
  constructor(public readonly sandboxId: string, message?: string) {
    const msg = message ? `Failed to resume sandbox ${sandboxId}: ${message}` : `Failed to resume sandbox ${sandboxId}`;
    super(msg, "resume_failed");
    this.name = "ResumeError";
  }
}
