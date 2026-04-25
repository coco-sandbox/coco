// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package visor

import (
	"encoding/binary"
	"fmt"
	"strings"
)

// Protocol constants — must match src/cocovisor/main.zig
const (
	ReqBoot         = 1
	ReqExec         = 2
	ReqDestroy      = 3
	ReqPause        = 4
	ReqResume       = 5
	ReqGetState     = 6
	ReqFork         = 7
	ReqHibernate    = 8
	ReqResumeHibernated = 9

	RespOK       = 100
	RespBoot    = 101
	RespExec    = 102
	RespDestroy = 103
	RespError   = 199
)

// Binary protocol frame format:
// ┌──────────┬──────────┬─────────────────────┐
// │ kind (4B)│ size (4B)│ payload (size B)   │
// └──────────┴──────────┴─────────────────────┘

var nativeEndian = binary.LittleEndian

// Frame represents a binary protocol frame
type Frame struct {
	Kind    uint32
	Size    uint32
	Payload []byte
}

// Pack serializes a frame to bytes
func (f *Frame) Pack() []byte {
	buf := make([]byte, 8+len(f.Payload))
	nativeEndian.PutUint32(buf[0:4], f.Kind)
	nativeEndian.PutUint32(buf[4:8], f.Size)
	copy(buf[8:], f.Payload)
	return buf
}

// Unpack deserializes bytes to a frame
func (f *Frame) Unpack(data []byte) error {
	if len(data) < 8 {
		return fmt.Errorf("frame too short: %d bytes", len(data))
	}
	f.Kind = nativeEndian.Uint32(data[0:4])
	f.Size = nativeEndian.Uint32(data[4:8])
	if int(f.Size) > len(data)-8 {
		return fmt.Errorf("payload size mismatch: %d > %d", f.Size, len(data)-8)
	}
	f.Payload = data[8 : 8+f.Size]
	return nil
}

// NewBootRequest creates a boot request frame
func NewBootRequest(sandboxID, rootfs string, memoryMB, vcpus int) *Frame {
	sandboxIDBytes := []byte(sandboxID)
	rootfsBytes := []byte(rootfs)

	payload := make([]byte, 36+len(sandboxIDBytes)+len(rootfsBytes))
	nativeEndian.PutUint32(payload[0:4], uint32(len(rootfsBytes)))   // rootfs_path_len
	nativeEndian.PutUint32(payload[4:8], uint32(memoryMB))          // memory_mb
	nativeEndian.PutUint32(payload[8:12], uint32(vcpus))           // vcpu_count
	nativeEndian.PutUint32(payload[12:16], 0)                      // kernel_path_len
	nativeEndian.PutUint32(payload[16:20], 0)                      // initrd_path_len
	nativeEndian.PutUint32(payload[20:24], uint32(len(sandboxIDBytes))) // sandbox_id_len
	nativeEndian.PutUint32(payload[24:28], 4747)                    // vsock_port
	nativeEndian.PutUint32(payload[28:32], 0)                      // padding
	nativeEndian.PutUint32(payload[32:36], 0)                      // padding
	copy(payload[36:], sandboxIDBytes)
	copy(payload[36+len(sandboxIDBytes):], rootfsBytes)

	return &Frame{Kind: 1, Size: uint32(len(payload)), Payload: payload}
}

// NewForkRequest creates a fork request frame
func NewForkRequest(sandboxID string) *Frame {
	idBytes := []byte(sandboxID)
	nameBytes := []byte("fork-" + sandboxID)

	payload := make([]byte, 8+len(idBytes)+len(nameBytes))
	nativeEndian.PutUint32(payload[0:4], uint32(len(idBytes)))
	nativeEndian.PutUint32(payload[4:8], uint32(len(nameBytes)))
	copy(payload[8:], idBytes)
	copy(payload[8+len(idBytes):], nameBytes)

	return &Frame{Kind: 7, Size: uint32(len(payload)), Payload: payload}
}

// ParseBootResponse parses a boot response frame
func ParseBootResponse(f *Frame) (vsockCID, pid uint32, err error) {
	if f.Kind != 101 {
		return 0, 0, fmt.Errorf("expected RESP_BOOT (101), got %d", f.Kind)
	}
	if len(f.Payload) < 12 {
		return 0, 0, fmt.Errorf("boot response payload too short: %d", len(f.Payload))
	}
	vsockCID = nativeEndian.Uint32(f.Payload[0:4])
	pid = nativeEndian.Uint32(f.Payload[4:8])
	state := nativeEndian.Uint32(f.Payload[8:12])
	if state != 2 {
		return 0, 0, fmt.Errorf("VM not running, state=%d", state)
	}
	return vsockCID, pid, nil
}

// ParseGetStateResponse parses a get state response frame
func ParseGetStateResponse(f *Frame) (state, pid, vsockCID uint32, err error) {
	if f.Kind != 106 {
		return 0, 0, 0, fmt.Errorf("expected RESP_GET_STATE (106), got %d", f.Kind)
	}
	if len(f.Payload) < 12 {
		return 0, 0, 0, fmt.Errorf("get_state payload too short: %d", len(f.Payload))
	}
	state = nativeEndian.Uint32(f.Payload[0:4])
	pid = nativeEndian.Uint32(f.Payload[4:8])
	vsockCID = nativeEndian.Uint32(f.Payload[8:12])
	return state, pid, vsockCID, nil
}

// ParseForkResponse parses a fork response frame
func ParseForkResponse(f *Frame) (childVsockCID, childPID, durationMS uint32, err error) {
	if f.Kind != 107 {
		return 0, 0, 0, fmt.Errorf("expected RESP_FORK (107), got %d", f.Kind)
	}
	if len(f.Payload) < 12 {
		return 0, 0, 0, fmt.Errorf("fork response payload too short: %d", len(f.Payload))
	}
	childVsockCID = nativeEndian.Uint32(f.Payload[0:4])
	childPID = nativeEndian.Uint32(f.Payload[4:8])
	durationMS = nativeEndian.Uint32(f.Payload[8:12])
	return childVsockCID, childPID, durationMS, nil
}

// BuildBootFrame builds a boot request frame for the visor protocol
func BuildBootFrame(sandboxID, rootfs string, memoryMB, vcpus, vsockPort uint32) ([]byte, error) {
	idBytes := []byte(sandboxID)
	rootfsBytes := []byte(rootfs)

	// Header: rootfs_len(4) + memory_mb(4) + vcpus(4) + kernel_len(4) + initrd_len(4) + id_len(4) + vsock_port(4) + padding(8) = 36 bytes
	headerSize := 36
	payload := make([]byte, headerSize+len(idBytes)+len(rootfsBytes))

	nativeEndian.PutUint32(payload[0:4], uint32(len(rootfsBytes)))
	nativeEndian.PutUint32(payload[4:8], memoryMB)
	nativeEndian.PutUint32(payload[8:12], vcpus)
	nativeEndian.PutUint32(payload[12:16], 0)          // kernel_path_len (empty)
	nativeEndian.PutUint32(payload[16:20], 0)          // initrd_path_len (empty)
	nativeEndian.PutUint32(payload[20:24], uint32(len(idBytes)))
	nativeEndian.PutUint32(payload[24:28], vsockPort)
	nativeEndian.PutUint32(payload[28:32], 0)          // padding
	nativeEndian.PutUint32(payload[32:36], 0)          // padding

	offset := headerSize
	copy(payload[offset:], idBytes)
	offset += len(idBytes)
	copy(payload[offset:], rootfsBytes)

	// Frame: [kind:4][size:4][payload]
	frame := make([]byte, 8+len(payload))
	nativeEndian.PutUint32(frame[0:4], 1) // ReqBoot
	nativeEndian.PutUint32(frame[4:8], uint32(len(payload)))
	copy(frame[8:], payload)
	return frame, nil
}

// BuildExecFrame builds an exec request frame for the visor protocol
func BuildExecFrame(cmd string, args []string, env []string, workingDir string) ([]byte, error) {
	cmdBytes := []byte(cmd)
	argsJoined := strings.Join(args, " ")
	argsBytes := []byte(argsJoined)
	envJoined := strings.Join(env, "\n")
	envBytes := []byte(envJoined)
	wdBytes := []byte(workingDir)

	// Header: cmd_len(4) + args_len(4) + env_len(4) + wd_len(4) + padding(4) = 20 bytes
	headerSize := 20
	payload := make([]byte, headerSize+len(cmdBytes)+len(argsBytes)+len(envBytes)+len(wdBytes))

	nativeEndian.PutUint32(payload[0:4], uint32(len(cmdBytes)))
	nativeEndian.PutUint32(payload[4:8], uint32(len(argsBytes)))
	nativeEndian.PutUint32(payload[8:12], uint32(len(envBytes)))
	nativeEndian.PutUint32(payload[12:16], uint32(len(wdBytes)))
	nativeEndian.PutUint32(payload[16:20], 0) // padding

	offset := headerSize
	copy(payload[offset:], cmdBytes)
	offset += len(cmdBytes)
	copy(payload[offset:], argsBytes)
	offset += len(argsBytes)
	copy(payload[offset:], envBytes)
	offset += len(envBytes)
	copy(payload[offset:], wdBytes)

	frame := make([]byte, 8+len(payload))
	nativeEndian.PutUint32(frame[0:4], 2) // ReqExec
	nativeEndian.PutUint32(frame[4:8], uint32(len(payload)))
	copy(frame[8:], payload)
	return frame, nil
}