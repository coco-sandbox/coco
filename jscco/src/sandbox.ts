// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

/**
 * High-level Sandbox API for the Coco SDK.
 */

import { CocoClient, CocoAPIError } from "./client.js";
import {
  SandboxNotFoundError,
  SandboxStateError,
  CheckpointNotFoundError,
  RateLimitError,
  AuthenticationError,
  ExecTimeoutError,
  ForkDepthExceededError,
  HibernateError,
  ResumeError,
} from "./exceptions.js";

export interface ExecResult {
  stdout: string;
  stderr: string;
  exitCode: number;
  durationMs: number;
}

export interface CheckpointResult {
  id: string;
  name: string;
  sandboxId: string;
  createdAt: string;
  path: string;
  sizeBytes: number;
}

export interface HibernateResult {
  id: string;
  state: string;
  hibernatePath: string;
  sizeBytes: number;
  durationMs: number;
}

export interface ReplayResult {
  id: string;
  name: string;
  sandboxId: string;
  state: string;
  startTime: string;
}

export interface SandboxData {
  id: string;
  name: string;
  state: string;
  created_at: string;
  template?: string;
  memory_mb?: number;
  vcpus?: number;
  labels?: Record<string, string>;
  fork_depth?: number;
  [key: string]: any;
}

export class Sandbox {
  constructor(
    private client: CocoClient,
    public readonly id: string,
    private data: SandboxData = {} as SandboxData
  ) {}

  get state(): string {
    return this.data.state || "unknown";
  }

  get name(): string {
    return this.data.name || "";
  }

  get createdAt(): string {
    return this.data.created_at || "";
  }

  static async create(options: {
    template?: string;
    memoryMb?: number;
    vcpus?: number;
    name?: string;
    labels?: Record<string, string>;
    client?: CocoClient;
  } = {}): Promise<Sandbox> {
    const {
      template = "alpine",
      memoryMb = 512,
      vcpus = 2,
      name,
      labels,
      client = new CocoClient(),
    } = options;

    const body: Record<string, any> = {
      template,
      memory_mb: memoryMb,
      vcpus,
    };
    if (name) body.name = name;
    if (labels) body.labels = labels;

    try {
      const resp = await client.post<{ sandbox: SandboxData }>("/v1/sandboxes", body);
      return new Sandbox(client, resp.sandbox.id, resp.sandbox);
    } catch (e: any) {
      if (e.statusCode === 429) throw new RateLimitError();
      if (e.statusCode === 401) throw new AuthenticationError(e.errorMessage);
      throw e;
    }
  }

  static async get(sandboxId: string, client?: CocoClient): Promise<Sandbox> {
    client ||= new CocoClient();
    try {
      const resp = await client.get<{ sandbox: SandboxData }>(`/v1/sandboxes/${sandboxId}`);
      return new Sandbox(client, sandboxId, resp.sandbox);
    } catch (e: any) {
      if (e.statusCode === 404) throw new SandboxNotFoundError(sandboxId);
      throw e;
    }
  }

  static async list(options: {
    state?: string;
    labelKey?: string;
    labelValue?: string;
    offset?: number;
    limit?: number;
    client?: CocoClient;
  } = {}): Promise<Sandbox[]> {
    const {
      state: stateFilter,
      labelKey,
      labelValue,
      offset = 0,
      limit = 100,
      client = new CocoClient(),
    } = options;

    const params: Record<string, string | number> = { offset, limit };
    if (stateFilter) params.state = stateFilter;
    if (labelKey) params.label_key = labelKey;
    if (labelValue) params.label_value = labelValue;

    const resp = await client.get<{ items: SandboxData[] }>("/v1/sandboxes", params);
    return resp.items.map((item) => new Sandbox(client, item.id, item));
  }

  async refresh(): Promise<void> {
    try {
      const resp = await this.client.get<{ sandbox: SandboxData }>(`/v1/sandboxes/${this.id}`);
      this.data = resp.sandbox;
    } catch (e: any) {
      if (e.statusCode === 404) throw new SandboxNotFoundError(this.id);
      throw e;
    }
  }

  private requireState(expected: string[]): void {
    if (!expected.includes(this.state)) {
      throw new SandboxStateError(this.id, this.state, expected);
    }
  }

  async runCode(
    code: string,
    options: {
      timeoutMs?: number;
      workingDir?: string;
      env?: Record<string, string>;
    } = {}
  ): Promise<ExecResult> {
    this.requireState(["running"]);

    const { timeoutMs = 30000, workingDir = "/tmp", env } = options;

    const body: Record<string, any> = {
      cmd: "sh",
      args: ["-c", code],
      timeout_ms: timeoutMs,
      working_dir: workingDir,
    };
    if (env) {
      body.env = Object.entries(env).map(([k, v]) => `${k}=${v}`);
    }

    try {
      const lines = await this.client.postStream(`/v1/sandboxes/${this.id}/exec`, body);
      const stdoutChunks: string[] = [];
      const stderrChunks: string[] = [];
      let exitCode = 0;

      for (const line of lines) {
        if (!line.trim()) continue;
        try {
          const chunk = JSON.parse(line);
          switch (chunk.stream_type) {
            case 1:
              stdoutChunks.push(chunk.data || "");
              break;
            case 2:
              stderrChunks.push(chunk.data || "");
              break;
            case 3:
              exitCode = chunk.exit_code ?? 0;
              if (chunk.error) stderrChunks.push(chunk.error);
              break;
          }
        } catch {
          // ignore parse errors
        }
      }

      return {
        stdout: stdoutChunks.join(""),
        stderr: stderrChunks.join(""),
        exitCode,
        durationMs: 0,
      };
    } catch (e: any) {
      if (e.statusCode === 400 && e.errorMessage.toLowerCase().includes("timeout")) {
        throw new ExecTimeoutError(timeoutMs);
      }
      throw e;
    }
  }

  async pause(): Promise<void> {
    this.requireState(["running"]);
    await this.client.post(`/v1/sandboxes/${this.id}/pause`);
    this.data.state = "paused";
  }

  async resume(): Promise<void> {
    this.requireState(["paused"]);
    try {
      const resp = await this.client.post<{ state: string }>(`/v1/sandboxes/${this.id}/resume`);
      this.data.state = resp.state || "running";
    } catch (e: any) {
      if (e.statusCode === 400) throw new ResumeError(this.id, e.errorMessage);
      throw e;
    }
  }

  async hibernate(options: { hibernatePath?: string } = {}): Promise<HibernateResult> {
    this.requireState(["running"]);

    const body: Record<string, any> = {};
    if (options.hibernatePath) body.hibernate_path = options.hibernatePath;

    try {
      const resp = await this.client.post<any>(`/v1/sandboxes/${this.id}/hibernate`, body);
      this.data.state = "hibernated";
      return {
        id: resp.id || this.id,
        state: resp.state || "hibernated",
        hibernatePath: resp.hibernate_path || "",
        sizeBytes: resp.size_bytes || 0,
        durationMs: resp.duration_ms || 0,
      };
    } catch (e: any) {
      throw new HibernateError(this.id, e.errorMessage);
    }
  }

  async fork(): Promise<Sandbox> {
    this.requireState(["running"]);

    try {
      const resp = await this.client.post<{ sandbox: SandboxData }>(`/v1/sandboxes/${this.id}/fork`);
      return new Sandbox(this.client, resp.sandbox.id, resp.sandbox);
    } catch (e: any) {
      if (e.statusCode === 400 && e.errorMessage.toLowerCase().includes("fork")) {
        throw new ForkDepthExceededError(this.data.fork_depth || 0, 16);
      }
      throw e;
    }
  }

  async checkpoint(name?: string): Promise<CheckpointResult> {
    this.requireState(["running", "paused"]);

    const body: Record<string, any> = {};
    if (name) body.name = name;

    const resp = await this.client.post<{ checkpoint: any }>(`/v1/sandboxes/${this.id}/checkpoints`, body);
    const cp = resp.checkpoint;
    return {
      id: cp.id || "",
      name: cp.name || name || "",
      sandboxId: this.id,
      createdAt: cp.created_at || "",
      path: cp.path || "",
      sizeBytes: cp.size_bytes || 0,
    };
  }

  async listCheckpoints(): Promise<CheckpointResult[]> {
    const resp = await this.client.get<{ checkpoints: any[] }>(`/v1/sandboxes/${this.id}/checkpoints`);
    return (resp.checkpoints || []).map((cp) => ({
      id: cp.id || "",
      name: cp.name || "",
      sandboxId: this.id,
      createdAt: cp.created_at || "",
      path: cp.path || "",
      sizeBytes: cp.size_bytes || 0,
    }));
  }

  async restore(checkpointId?: string): Promise<void> {
    this.requireState(["stopped", "error"]);

    const body: Record<string, any> = {};
    if (checkpointId) body.checkpoint_id = checkpointId;

    await this.client.post(`/v1/sandboxes/${this.id}/restore`, body);
    this.data.state = "running";
  }

  async undo(checkpointName?: string): Promise<CheckpointResult> {
    this.requireState(["running", "paused", "stopped", "error"]);

    const body: Record<string, any> = {};
    if (checkpointName) body.name = checkpointName;

    const resp = await this.client.post<{ checkpoint: any }>(`/v1/sandboxes/${this.id}/undo`, body);
    const cp = resp.checkpoint;
    return {
      id: cp.id || "",
      name: cp.name || checkpointName || "",
      sandboxId: this.id,
      createdAt: cp.created_at || "",
      path: cp.path || "",
      sizeBytes: cp.size_bytes || 0,
    };
  }

  async redo(): Promise<CheckpointResult> {
    this.requireState(["running", "paused", "stopped", "error"]);

    const resp = await this.client.post<{ checkpoint: any }>(`/v1/sandboxes/${this.id}/redo`, {});
    const cp = resp.checkpoint;
    return {
      id: cp.id || "",
      name: cp.name || "",
      sandboxId: this.id,
      createdAt: cp.created_at || "",
      path: cp.path || "",
      sizeBytes: cp.size_bytes || 0,
    };
  }

  async replayStart(name?: string): Promise<ReplayResult> {
    const body: Record<string, any> = {};
    if (name) body.name = name;

    const resp = await this.client.post<{ replay: any }>(`/v1/sandboxes/${this.id}/replay/start`, body);
    const replay = resp.replay;
    return {
      id: replay.id || "",
      name: replay.name || name || "",
      sandboxId: this.id,
      state: replay.state || "recording",
      startTime: replay.start_time || "",
    };
  }

  async replayStop(): Promise<ReplayResult> {
    const resp = await this.client.post<{ replay: any }>(`/v1/sandboxes/${this.id}/replay/stop`);
    const replay = resp.replay;
    return {
      id: replay.id || "",
      name: replay.name || "",
      sandboxId: this.id,
      state: replay.state || "stopped",
      startTime: replay.start_time || "",
    };
  }

  async destroy(): Promise<void> {
    await this.client.delete(`/v1/sandboxes/${this.id}`);
    this.data.state = "stopped";
  }
}
