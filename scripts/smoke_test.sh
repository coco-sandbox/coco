#!/bin/bash
# SPDX-License-Identifier: Apache-2.0
# Copyright (C) 2026 The Coco Sandbox Authors
# Smoke test script - validates basic functionality

set -e

COCO_CORE="${COCO_CORE:-./coco-core}"
COCO_CTL="${COCO_CTL:-./cococtl}"
API="${API:-http://localhost:4747}"
ID="${ID:-sb_test}"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

pass() { echo -e "${GREEN}✓${NC} $1"; }
fail() { echo -e "${RED}✗${NC} $1"; exit 1; }
info() { echo -e "${YELLOW}→${NC} $1"; }

info "Starting smoke test..."

# Start coco-core in background
if ! curl -s "$API/health" > /dev/null 2>&1; then
    info "Starting coco-core..."
    $COCO_CORE &
    sleep 2
fi

# Test 1: Health check
info "Test: Health endpoint"
HEALTH=$(curl -s "$API/health")
echo "$HEALTH" | grep -q '"healthy":true' && pass "Health check" || fail "Health check"

# Test 2: Create sandbox
info "Test: Create sandbox"
CREATE_RESP=$(curl -s -X POST "$API/v1/sandboxes" \
    -H "Content-Type: application/json" \
    -d '{"name":"test","template":"alpine"}')
echo "$CREATE_RESP" | grep -q '"state":"running"' && pass "Create sandbox" || fail "Create sandbox"

SB_ID=$(echo "$CREATE_RESP" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
info "Sandbox ID: $SB_ID"

# Test 3: Get sandbox
info "Test: Get sandbox"
GET_RESP=$(curl -s "$API/v1/sandboxes/$SB_ID")
echo "$GET_RESP" | grep -q "\"id\":\"$SB_ID\"" && pass "Get sandbox" || fail "Get sandbox"

# Test 4: List sandboxes
info "Test: List sandboxes"
LIST_RESP=$(curl -s "$API/v1/sandboxes")
echo "$LIST_RESP" | grep -q '"items"' && pass "List sandboxes" || fail "List sandboxes"

# Test 5: Fork sandbox
info "Test: Fork sandbox"
FORK_RESP=$(curl -s -X POST "$API/v1/sandboxes/$SB_ID/fork" \
    -H "Content-Type: application/json" \
    -d '{"name":"fork-test"}')
echo "$FORK_RESP" | grep -q '"parent_id"' && pass "Fork sandbox" || fail "Fork sandbox"

# Test 6: Checkpoint
info "Test: Create checkpoint"
CKPT_RESP=$(curl -s -X POST "$API/v1/sandboxes/$SB_ID/checkpoint" \
    -H "Content-Type: application/json" \
    -d '{"name":"test-checkpoint"}')
echo "$CKPT_RESP" | grep -q '"checkpoint_id"' && pass "Checkpoint" || fail "Checkpoint"

# Test 7: Hibernate
info "Test: Hibernate sandbox"
HIB_RESP=$(curl -s -X POST "$API/v1/sandboxes/$SB_ID/hibernate")
echo "$HIB_RESP" | grep -q '"state":"hibernated"' && pass "Hibernate" || fail "Hibernate"

# Test 8: Resume
info "Test: Resume sandbox"
RESUME_RESP=$(curl -s -X POST "$API/v1/sandboxes/$SB_ID/resume")
echo "$RESUME_RESP" | grep -q '"state":"running"' && pass "Resume" || fail "Resume"

# Test 9: Exec
info "Test: Exec command"
EXEC_RESP=$(curl -s -X POST "$API/v1/sandboxes/$SB_ID/exec" \
    -H "Content-Type: application/json" \
    -d '{"cmd":["echo","hello"]}')
echo "$EXEC_RESP" | grep -q '"exit_code":0' && pass "Exec" || fail "Exec"

# Test 10: Undo
info "Test: Undo to checkpoint"
UNDO_RESP=$(curl -s -X POST "$API/v1/sandboxes/$SB_ID/undo")
echo "$UNDO_RESP" | grep -q '"state":"running"' && pass "Undo" || fail "Undo"

# Test 11: Metrics
info "Test: Metrics endpoint"
METRICS=$(curl -s "$API/metrics")
echo "$METRICS" | grep -q "coco_sandbox_creates_total" && pass "Metrics" || fail "Metrics"

# Test 12: Ready probe
info "Test: Ready endpoint"
READY=$(curl -s "$API/ready")
echo "$READY" | grep -q '"ready":true' && pass "Ready" || fail "Ready"

# Test 13: Destroy sandbox
info "Test: Destroy sandbox"
DESTROY_RESP=$(curl -s -X DELETE "$API/v1/sandboxes/$SB_ID")
echo "$DESTROY_RESP" | grep -q '"success":true' && pass "Destroy sandbox" || fail "Destroy sandbox"

info ""
info "All smoke tests passed!"
echo ""
echo "Summary:"
echo "  - Health check       ✓"
echo "  - Sandbox CRUD       ✓"
echo "  - Fork               ✓"
echo "  - Checkpoint/Undo    ✓"
echo "  - Hibernate/Resume   ✓"
echo "  - Exec streaming     ✓"
echo "  - Metrics            ✓"
echo ""
echo "Coco Sandbox is working correctly."