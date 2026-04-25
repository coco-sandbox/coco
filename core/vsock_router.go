// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
)

// VsockPeer represents a remote vsock endpoint in the cluster
type VsockPeer struct {
	NodeID   string `json:"node_id"`
	NodeAddr string `json:"node_addr"`
	CID      uint32 `json:"cid"`
	State    string `json:"state"` // "connected", "disconnected", "error"
}

// VsockConnection represents an active cross-node vsock connection
type VsockConnection struct {
	LocalCID   uint32 `json:"local_cid"`
	RemoteCID  uint32 `json:"remote_cid"`
	RemoteNode string `json:"remote_node"`
	State      string `json:"state"`
}

// VsockRouter manages cross-node vsock communication
type VsockRouter struct {
	selfCID      uint32
	selfNodeID   string
	peers        map[string]*VsockPeer
	connections map[uint32]*VsockConnection
	listener     net.Listener // TCP proxy listener for cross-node transport
	mu           sync.RWMutex
	stopCh       chan struct{}
}

// NewVsockRouter creates a new vsock router for cross-node communication
func NewVsockRouter(selfCID uint32, selfNodeID string) *VsockRouter {
	return &VsockRouter{
		selfCID:      selfCID,
		selfNodeID:   selfNodeID,
		peers:        make(map[string]*VsockPeer),
		connections: make(map[uint32]*VsockConnection),
		stopCh:       make(chan struct{}),
	}
}

// Start begins the vsock router
func (vr *VsockRouter) Start() error {
	log.Printf("[vsock-router] Starting vsock router for node %s (CID: %d)", vr.selfNodeID, vr.selfCID)

	// Start TCP proxy for cross-node vsock transport
	// In production, this would use RDMA or a high-performance TCP proxy
	addr := fmt.Sprintf(":%d", 4747+int(vr.selfCID))
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Printf("[vsock-router] Failed to start TCP proxy: %v", err)
		return err
	}
	vr.listener = listener

	go vr.acceptLoop()
	return nil
}

// Stop shuts down the vsock router
func (vr *VsockRouter) Stop() {
	close(vr.stopCh)
	if vr.listener != nil {
		vr.listener.Close()
	}
}

// =============================================================================
// Peer Management
// =============================================================================

// RegisterPeer adds a new peer to the routing table
func (vr *VsockRouter) RegisterPeer(peer *VsockPeer) error {
	vr.mu.Lock()
	defer vr.mu.Unlock()

	if _, exists := vr.peers[peer.NodeID]; exists {
		return fmt.Errorf("peer %s already registered", peer.NodeID)
	}

	vr.peers[peer.NodeID] = peer
	log.Printf("[vsock-router] Registered peer %s (CID: %d, addr: %s)", peer.NodeID, peer.CID, peer.NodeAddr)
	return nil
}

// UnregisterPeer removes a peer from the routing table
func (vr *VsockRouter) UnregisterPeer(nodeID string) error {
	vr.mu.Lock()
	defer vr.mu.Unlock()

	if _, exists := vr.peers[nodeID]; !exists {
		return fmt.Errorf("peer %s not found", nodeID)
	}

	delete(vr.peers, nodeID)
	log.Printf("[vsock-router] Unregistered peer %s", nodeID)
	return nil
}

// GetPeer returns a specific peer
func (vr *VsockRouter) GetPeer(nodeID string) (*VsockPeer, error) {
	vr.mu.RLock()
	defer vr.mu.RUnlock()

	peer, exists := vr.peers[nodeID]
	if !exists {
		return nil, fmt.Errorf("peer %s not found", nodeID)
	}
	return peer, nil
}

// ListPeers returns all known peers
func (vr *VsockRouter) ListPeers() []*VsockPeer {
	vr.mu.RLock()
	defer vr.mu.RUnlock()

	result := make([]*VsockPeer, 0, len(vr.peers))
	for _, p := range vr.peers {
		result = append(result, p)
	}
	return result
}

// =============================================================================
// Connection Management
// =============================================================================

// Connect establishes a connection to a remote vsock endpoint
func (vr *VsockRouter) Connect(nodeID string, remoteCID uint32) (*VsockConnection, error) {
	vr.mu.RLock()
	_, exists := vr.peers[nodeID]
	vr.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("peer %s not found", nodeID)
	}

	conn := &VsockConnection{
		LocalCID:   vr.selfCID,
		RemoteCID:  remoteCID,
		RemoteNode: nodeID,
		State:      "connecting",
	}

	vr.mu.Lock()
	vr.connections[remoteCID] = conn
	vr.mu.Unlock()

	// In production, establish actual RDMA or TCP connection
	log.Printf("[vsock-router] Connecting to peer %s CID %d", nodeID, remoteCID)
	conn.State = "connected"

	return conn, nil
}

// Disconnect closes a connection
func (vr *VsockRouter) Disconnect(remoteCID uint32) error {
	vr.mu.Lock()
	defer vr.mu.Unlock()

	conn, exists := vr.connections[remoteCID]
	if !exists {
		return fmt.Errorf("connection %d not found", remoteCID)
	}

	conn.State = "disconnected"
	delete(vr.connections, remoteCID)
	log.Printf("[vsock-router] Disconnected from CID %d", remoteCID)
	return nil
}

// GetConnections returns all active connections
func (vr *VsockRouter) GetConnections() []*VsockConnection {
	vr.mu.RLock()
	defer vr.mu.RUnlock()

	result := make([]*VsockConnection, 0, len(vr.connections))
	for _, c := range vr.connections {
		result = append(result, c)
	}
	return result
}

// =============================================================================
// TCP Proxy Loop (Cross-Node Transport)
// =============================================================================

func (vr *VsockRouter) acceptLoop() {
	for {
		select {
		case <-vr.stopCh:
			return
		default:
			if vr.listener == nil {
				return
			}
			conn, err := vr.listener.Accept()
			if err != nil {
				continue
			}
			go vr.handleConnection(conn)
		}
	}
}

func (vr *VsockRouter) handleConnection(conn net.Conn) {
	// Read vsock frame header
	buf := make([]byte, 16)
	_, err := conn.Read(buf)
	if err != nil {
		conn.Close()
		return
	}

	// Parse header to get destination CID
	// Format: [version:1][family:1][src_cid:4][dst_cid:4][port:4][len:2]
	// For now, just log and close
	log.Printf("[vsock-router] Received connection, closing (not implemented)")
	conn.Close()
}

// =============================================================================
// API Handlers
// =============================================================================

// HandleVsockPeers handles /vsock/peers endpoint
func handleVsockPeers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		peers := vsockRouter.ListPeers()
		writeJSON(w, http.StatusOK, map[string]any{
			"peers":      peers,
			"total_count": len(peers),
		})
	case http.MethodPost:
		// Register a new peer
		var peer VsockPeer
		if err := decodeJSON(r.Body, &peer); err != nil {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("Invalid request: %v", err))
			return
		}
		if err := vsockRouter.RegisterPeer(&peer); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{"peer": &peer})
	default:
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// HandleVsockPeersByID handles /vsock/peers/{node_id} endpoint
func handleVsockPeersByID(w http.ResponseWriter, r *http.Request) {
	nodeID := r.URL.Path[len("/vsock/peers/"):]
	nodeID = strings.SplitN(nodeID, "?", 2)[0]

	peer, err := vsockRouter.GetPeer(nodeID)
	if err != nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("Peer %s not found", nodeID))
		return
	}

	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, map[string]any{"peer": peer})
	case http.MethodDelete:
		if err := vsockRouter.UnregisterPeer(nodeID); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"success": true})
	default:
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// HandleVsockConnections handles /vsock/connections endpoint
func handleVsockConnections(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	connections := vsockRouter.GetConnections()
	writeJSON(w, http.StatusOK, map[string]any{
		"connections": connections,
		"total_count": len(connections),
	})
}

// HandleVsockStatus handles /vsock/status endpoint
func handleVsockStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	vsockRouter.mu.RLock()
	status := map[string]any{
		"self_cid":       vsockRouter.selfCID,
		"self_node_id":   vsockRouter.selfNodeID,
		"peer_count":     len(vsockRouter.peers),
		"connection_count": len(vsockRouter.connections),
	}
	vsockRouter.mu.RUnlock()

	writeJSON(w, http.StatusOK, status)
}

// =============================================================================
// Global VsockRouter Instance
// =============================================================================

var vsockRouter *VsockRouter
