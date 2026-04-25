// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

/**
 * Low-level HTTP client for the Coco API.
 */

import {
  CocoError,
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
} from "./exceptions.js";

export interface CocoAPIError extends Error {
  readonly statusCode: number;
  readonly errorCode: string;
  readonly errorMessage: string;
  readonly details: string | null;
}

export class CocoClient {
  constructor(
    public readonly baseUrl: string = "http://localhost:4747",
    private apiKey?: string,
    private timeout: number = 30
  ) {}

  private headers(): Record<string, string> {
    const headers: Record<string, string> = {
      "Content-Type": "application/json",
      Accept: "application/json",
    };
    if (this.apiKey) {
      headers["X-API-Key"] = this.apiKey;
    }
    return headers;
  }

  private buildUrl(path: string, params?: Record<string, string | number>): string {
    let url = `${this.baseUrl}${path}`;
    if (params) {
      const query = Object.entries(params)
        .map(([k, v]) => `${k}=${encodeURIComponent(String(v))}`)
        .join("&");
      url = `${url}?${query}`;
    }
    return url;
  }

  private async request<T>(
    method: string,
    path: string,
    body?: unknown,
    params?: Record<string, string | number>
  ): Promise<T> {
    const url = this.buildUrl(path, params);
    const options: RequestInit = {
      method,
      headers: this.headers(),
    };

    if (body !== undefined) {
      options.body = JSON.stringify(body);
    }

    try {
      const resp = await fetch(url, {
        ...options,
        signal: AbortSignal.timeout(this.timeout * 1000),
      });

      if (!resp.ok) {
        let errorData: any = {};
        try {
          errorData = await resp.json();
        } catch {
          // ignore parse errors
        }
        throw this.createError(resp.status, errorData);
      }

      if (resp.status === 204) {
        return {} as T;
      }
      return await resp.json();
    } catch (e) {
      if (e instanceof CocoError) {
        throw e;
      }
      throw new CocoError(
        `Request failed: ${(e as Error).message}`,
        "request_failed"
      );
    }
  }

  private createError(statusCode: number, data: any): CocoAPIError {
    const error = data?.error || {};
    const code = error.code || "unknown";
    const message = error.message || `HTTP ${statusCode}`;
    const details = error.details || null;

    const err: any = new Error(message);
    err.statusCode = statusCode;
    err.errorCode = code;
    err.errorMessage = message;
    err.details = details;

    return err as CocoAPIError;
  }

  async get<T>(path: string, params?: Record<string, string | number>): Promise<T> {
    return this.request<T>("GET", path, undefined, params);
  }

  async post<T>(path: string, body?: unknown): Promise<T> {
    return this.request<T>("POST", path, body);
  }

  async patch<T>(path: string, body?: unknown): Promise<T> {
    return this.request<T>("PATCH", path, body);
  }

  async delete<T>(path: string): Promise<T> {
    return this.request<T>("DELETE", path);
  }

  async postStream(path: string, body?: unknown): Promise<string[]> {
    const url = this.buildUrl(path);
    const options: RequestInit = {
      method: "POST",
      headers: this.headers(),
      body: body ? JSON.stringify(body) : undefined,
    };

    const resp = await fetch(url, {
      ...options,
      signal: AbortSignal.timeout(this.timeout * 1000),
    });

    if (!resp.ok) {
      let errorData: any = {};
      try {
        errorData = await resp.json();
      } catch {
        // ignore parse errors
      }
      throw this.createError(resp.status, errorData);
    }

    const lines: string[] = [];
    const reader = resp.body?.getReader();
    const decoder = new TextDecoder();

    if (reader) {
      let buffer = "";
      while (true) {
        const { done, value } = await reader.read();
        if (done) break;
        buffer += decoder.decode(value, { stream: true });
        const parts = buffer.split("\n");
        buffer = parts.pop() || "";
        for (const line of parts) {
          if (line.trim()) {
            lines.push(line);
          }
        }
      }
      if (buffer.trim()) {
        lines.push(buffer);
      }
    }

    return lines;
  }
}

export { CocoClient as default };
