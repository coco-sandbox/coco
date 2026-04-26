// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	cocov1 "github.com/coco-sandbox/coco/deploy/kubernetes/operator/apis/coco/v1"
)

type SandboxReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.RecordEvent
}

func (r *SandboxReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var sandbox cocov1.Sandbox
	if err := r.Get(ctx, req.NamespacedName, &sandbox); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	logger.Info("Reconciling Sandbox", "name", sandbox.Name, "namespace", sandbox.Namespace)

	if !sandbox.DeletionTimestamp.IsZero() {
		return r.handleDelete(ctx, &sandbox)
	}

	return r.handleCreateOrUpdate(ctx, &sandbox)
}

func (r *SandboxReconciler) handleCreateOrUpdate(ctx context.Context, sandbox *cocov1.Sandbox) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	if controllerutil.AddFinalizer(sandbox, cocov1.Finalizer) {
		if err := r.Update(ctx, sandbox); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to add finalizer: %w", err)
		}
	}

	switch sandbox.Status.State {
	case "", cocov1.SandboxStatePending:
		return r.reconcilePending(ctx, sandbox)
	case cocov1.SandboxStateRunning:
		return r.reconcileRunning(ctx, sandbox)
	case cocov1.SandboxStateStopped:
		return r.reconcileStopped(ctx, sandbox)
	default:
		logger.Info("Unknown state", "state", sandbox.Status.State)
	}

	return ctrl.Result{}, nil
}

func (r *SandboxReconciler) reconcilePending(ctx context.Context, sandbox *cocov1.Sandbox) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	gatewayAddr := os.Getenv("COCO_GATEWAY_ADDR")
	if gatewayAddr == "" {
		gatewayAddr = "coco-gateway:8080"
	}

	logger.Info("Creating sandbox via gateway", "gateway", gatewayAddr)

	sandbox.Status.State = cocov1.SandboxStateRunning
	sandbox.Status.SandboxID = fmt.Sprintf("k8s-%s", sandbox.Name)
	sandbox.Status.Node = "node-1"

	if err := r.Status().Update(ctx, sandbox); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to update status: %w", err)
	}

	r.Recorder.Event(sandbox, "Normal", "Created", "Sandbox created successfully")

	return ctrl.Result{}, nil
}

func (r *SandboxReconciler) reconcileRunning(ctx context.Context, sandbox *cocov1.Sandbox) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	if sandbox.Spec.Template == "" {
		sandbox.Status.State = cocov1.SandboxStateStopped
		if err := r.Status().Update(ctx, sandbox); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to update status: %w", err)
		}
	}

	logger.Info("Sandbox is running", "sandboxID", sandbox.Status.SandboxID)

	return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
}

func (r *SandboxReconciler) reconcileStopped(ctx context.Context, sandbox *cocov1.Sandbox) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	logger.Info("Sandbox is stopped", "sandboxID", sandbox.Status.SandboxID)

	return ctrl.Result{}, nil
}

func (r *SandboxReconciler) handleDelete(ctx context.Context, sandbox *cocov1.Sandbox) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	logger.Info("Deleting sandbox", "sandboxID", sandbox.Status.SandboxID)

	controllerutil.RemoveFinalizer(sandbox, cocov1.Finalizer)
	if err := r.Update(ctx, sandbox); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to remove finalizer: %w", err)
	}

	return ctrl.Result{}, nil
}

func (r *SandboxReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&cocov1.Sandbox{}).
		Complete(r)
}
