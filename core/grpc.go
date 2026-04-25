// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)

// gRPC server type
type CocoGRPCServer struct {
	server *grpc.Server
	listener   net.Listener
}

// =============================================================================
// gRPC Server Setup
// =============================================================================

func setupGRPC(addr string) (*CocoGRPCServer, error) {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to listen on %s: %w", addr, err)
	}

	// Create gRPC server with options
	opts := []grpc.ServerOption{
		grpc.UnaryInterceptor(grpcLoggingUnaryInterceptor),
		grpc.StreamInterceptor(grpcLoggingStreamInterceptor),
		grpc.ConnectionTimeout(30 * time.Second),
	}

	grpcSrv := grpc.NewServer(opts...)

	server := &CocoGRPCServer{
		server: grpcSrv,
		listener:   listener,
	}

	// Register reflection service for debugging
	reflection.Register(grpcSrv)

	log.Printf("gRPC server listening on %s", addr)

	return server, nil
}

func (s *CocoGRPCServer) Serve() error {
	return s.server.Serve(s.listener)
}

func (s *CocoGRPCServer) GracefulShutdown() {
	s.server.GracefulStop()
}

// =============================================================================
// gRPC Interceptors
// =============================================================================

func grpcLoggingUnaryInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	start := time.Now()

	// Extract request ID from metadata if present
	md, _ := metadata.FromIncomingContext(ctx)
	requestIDs := md.Get("x-request-id")
	var requestID string
	if len(requestIDs) > 0 {
		requestID = requestIDs[0]
	}

	// Log request
	log.Printf("[gRPC] %s %s request_id=%s", info.FullMethod, time.Since(start), requestID)

	resp, err := handler(ctx, req)

	// Log response
	duration := time.Since(start)
	if err != nil {
		log.Printf("[gRPC] %s error=%v duration=%v", info.FullMethod, err, duration)
	} else {
		log.Printf("[gRPC] %s OK duration=%v", info.FullMethod, duration)
	}

	return resp, err
}

func grpcLoggingStreamInterceptor(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	start := time.Now()

	// Log stream start
	log.Printf("[gRPC] %s stream started", info.FullMethod)

	err := handler(srv, ss)

	duration := time.Since(start)
	if err != nil {
		log.Printf("[gRPC] %s stream error=%v duration=%v", info.FullMethod, err, duration)
	} else {
		log.Printf("[gRPC] %s stream completed duration=%v", info.FullMethod, duration)
	}

	return err
}

// =============================================================================
// Auth Interceptors
// =============================================================================

func grpcAuthUnaryInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	// Skip auth for health/ready endpoints
	method := info.FullMethod
	if method == "/coco.v1.SandboxService/Health" || method == "/coco.v1.SandboxService/Ready" {
		return handler(ctx, req)
	}

	// Validate API key from metadata
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing metadata")
	}

	apiKeys := md.Get("x-api-key")
	if len(apiKeys) == 0 {
		// Also check bearer token
		bearer := md.Get("authorization")
		if len(bearer) > 0 && len(bearer[0]) > 7 {
			apiKeys = []string{bearer[0][7:]} // Strip "Bearer "
		}
	}

	if len(apiKeys) == 0 {
		return nil, status.Error(codes.Unauthenticated, "missing API key")
	}

	hashedKey := hashAPIKey(apiKeys[0])
	apiKey, err := validateAPIKey(hashedKey)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "invalid API key: %v", err)
	}

	// Add auth info to context
	ctx = WithAPIKey(ctx, apiKey)
	ctx = WithTenantID(ctx, apiKey.TenantID)

	return handler(ctx, req)
}

// =============================================================================
// Rate Limit Interceptor
// =============================================================================

func grpcRateLimitUnaryInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	tenantID := TenantIDFromContext(ctx)
	if tenantID == "" {
		tenantID = "default"
	}

	limiter := getTenantLimiter(tenantID)
	allowed, retryAfter := limiter.Allow(tenantID)

	if !allowed {
		return nil, status.Errorf(codes.ResourceExhausted, "rate limit exceeded, retry after %d seconds", retryAfter)
	}

	return handler(ctx, req)
}

// =============================================================================
// Peer Info Helper
// =============================================================================

func getPeerAddr(ctx context.Context) string {
	if peer, ok := peer.FromContext(ctx); ok {
		return peer.Addr.String()
	}
	return "unknown"
}

// =============================================================================
// Stream Helper for Exec
// =============================================================================

type execStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (s *execStream) Context() context.Context {
	return s.ctx
}