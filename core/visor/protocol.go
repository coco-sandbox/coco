// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package visor

import (
	"bytes"
	"encoding/binary"
	"strings"
)

// Protocol constants — must match cocovisor/src/cocovisor/main.zig
const (
	ReqBoot    uint32 = 1
	ReqExec    uint32 = 2
	ReqDestroy uint32 = 3
	ReqGetState uint32 = 6

	RespBoot     uint32 = 101
	RespExec     uint32 = 102
	RespDestroy  uint32 = 103
	RespGetState uint32 = 106
	RespError    uint32 = 199

	SockPath = "/run/coco/visor.sock"
)

// BootRequest is the binary payload sent for a Boot request.
// All length fields count bytes of the string data that follows the struct.
type BootRequest struct {
	RootfsPathLen  uint32
	MemoryMB       uint32
	VCPUs          uint32
	KernelPathLen  uint32
	InitrdPathLen  uint32
	SandboxIDLen   uint32
	VsockPort      uint32
	Padding        uint32
}

// BootResponse is the binary payload returned from a successful Boot.
type BootResponse struct {
	VsockCID uint32
	PID      uint32
	State    uint32
}

// ExecRequest is the binary payload sent for an Exec request.
type ExecRequest struct {
	CmdLen       uint32
	ArgsLen      uint32
	EnvLen       uint32
	WorkingDirLen uint32
}

// ExecChunk represents a chunk in the exec response stream.
type ExecChunk struct {
	StreamType uint32 // 1=stdout, 2=stderr, 3=exit
	DataLen    uint32
	ExitCode   uint32
	Data       []byte
}

// BuildBootFrame constructs a complete Boot request frame.
// Payload: BootRequest struct + sandbox_id bytes + rootfs_path bytes.
func BuildBootFrame(sandboxID, rootfsPath string, memoryMB, vcpus uint32, vsockPort uint32) ([]byte, error) {
	const kernelPath = "/boot/vmlinuz"
	const initrdPath = "/boot/initrd"

	req := BootRequest{
		RootfsPathLen: uint32(len(rootfsPath)),
		MemoryMB:      memoryMB,
		VCPUs:         vcpus,
		KernelPathLen: uint32(len(kernelPath)),
		InitrdPathLen: uint32(len(initrdPath)),
		SandboxIDLen:  uint32(len(sandboxID)),
		VsockPort:     vsockPort,
		Padding:       0,
	}

	var buf bytes.Buffer
	// Frame header will be written last; first write the struct
	if err := binary.Write(&buf, binary.LittleEndian, req); err != nil {
		return nil, err
	}
	buf.WriteString(sandboxID)
	buf.WriteString(rootfsPath)
	buf.WriteString(kernelPath)
	buf.WriteString(initrdPath)

	frame := buildFrame(ReqBoot, buf.Bytes())
	return frame, nil
}

// BuildExecFrame constructs a complete Exec request frame.
// Payload: ExecRequest struct + cmd bytes + args bytes + env bytes + working_dir bytes.
func BuildExecFrame(cmd string, args, env []string, workingDir string) ([]byte, error) {
	// Build args string (space-separated)
	argsStr := joinStrings(args, "\x00")
	envStr := joinEnv(env)
	wdStr := workingDir

	// Handle empty working dir
	if wdStr == "" {
		wdStr = "/"
	}

	req := ExecRequest{
		CmdLen:       uint32(len(cmd)),
		ArgsLen:      uint32(len(argsStr)),
		EnvLen:       uint32(len(envStr)),
		WorkingDirLen: uint32(len(wdStr)),
	}

	var buf bytes.Buffer
	if err := binary.Write(&buf, binary.LittleEndian, req); err != nil {
		return nil, err
	}
	buf.WriteString(cmd)
	buf.WriteString(argsStr)
	buf.WriteString(envStr)
	buf.WriteString(wdStr)

	frame := buildFrame(ReqExec, buf.Bytes())
	return frame, nil
}

// buildFrame prepends [kind:u32][size:u32] to the payload.
func buildFrame(kind uint32, payload []byte) []byte {
	size := uint32(len(payload))
	frame := make([]byte, 8+size)
	binary.LittleEndian.PutUint32(frame[0:4], kind)
	binary.LittleEndian.PutUint32(frame[4:8], size)
	copy(frame[8:], payload)
	return frame
}

// ParseBootResponse extracts VsockCID, PID, and State from a Boot response frame.
func ParseBootResponse(frame []byte) (*BootResponse, error) {
	if len(frame) < 8 {
		return nil, ErrShortFrame
	}
	kind := binary.LittleEndian.Uint32(frame[0:4])
	if kind != RespBoot {
		return nil, ErrUnexpectedKind
	}
	size := binary.LittleEndian.Uint32(frame[4:8])
	if size < 12 || len(frame) < 8+int(size) {
		return nil, ErrShortFrame
	}
	resp := &BootResponse{
		VsockCID: binary.LittleEndian.Uint32(frame[8:12]),
		PID:      binary.LittleEndian.Uint32(frame[12:16]),
		State:    binary.LittleEndian.Uint32(frame[16:20]),
	}
	return resp, nil
}

// ReadExecChunk reads a single Exec response chunk from the socket.
// Returns the chunk and whether the stream has ended (stream_type == 3).
func ReadExecChunk(frame []byte) (*ExecChunk, error) {
	if len(frame) < 8 {
		return nil, ErrShortFrame
	}
	kind := binary.LittleEndian.Uint32(frame[0:4])
	if kind != RespExec {
		return nil, ErrUnexpectedKind
	}
	size := binary.LittleEndian.Uint32(frame[4:8])
	if size < 12 || len(frame) < 8+int(size) {
		return nil, ErrShortFrame
	}

	chunk := &ExecChunk{
		StreamType: binary.LittleEndian.Uint32(frame[8:12]),
		DataLen:    binary.LittleEndian.Uint32(frame[12:16]),
		ExitCode:   binary.LittleEndian.Uint32(frame[16:20]),
	}
	dataStart := 20
	if int(chunk.DataLen) <= int(size)-12 {
		chunk.Data = frame[dataStart : dataStart+int(chunk.DataLen)]
	}
	return chunk, nil
}

func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	var b strings.Builder
	for i, s := range strs {
		if i > 0 {
			b.WriteString(sep)
		}
		b.WriteString(s)
	}
	return b.String()
}

func joinEnv(env []string) string {
	if len(env) == 0 {
		return ""
	}
	var b strings.Builder
	for i, e := range env {
		if i > 0 {
			b.WriteByte('\x00')
		}
		b.WriteString(e)
	}
	return b.String()
}
