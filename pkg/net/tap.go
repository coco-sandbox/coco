package net

import (
    "fmt"
    "net"
    "sync"
    "syscall"
    "unsafe"
)

const (
    TUNSETIFF = 0x400454ca
    IFF_TAP   = 0x0002
    IFF_NO_PI = 0x1000
)

type TAPManager struct {
    mu      sync.RWMutex
    devices map[string]*TAPDevice
}

type TAPDevice struct {
    Name string
    FD   int
    MAC  net.HardwareAddr
    IP   net.IP
}

func NewTAPManager() *TAPManager {
    return &TAPManager{
        devices: make(map[string]*TAPDevice),
    }
}

func (m *TAPManager) Create(name string) (*TAPDevice, error) {
    m.mu.Lock()
    defer m.mu.Unlock()

    if _, exists := m.devices[name]; exists {
        return nil, fmt.Errorf("device %s already exists", name)
    }

    // Open TAP character device
    tapFD, err := syscall.Open("/dev/net/tun", syscall.O_RDWR, 0)
    if err != nil {
        return nil, fmt.Errorf("failed to open /dev/net/tun: %w", err)
    }

    // Set IFF_TAP and IFF_NO_PI flags
    type ifreq struct {
        Name  [16]byte
        Flags uint16
    }
    req := ifreq{}
    copy(req.Name[:], name)
    req.Flags = IFF_TAP | IFF_NO_PI

    _, _, errno := syscall.Syscall(
        syscall.SYS_IOCTL,
        uintptr(tapFD),
        uintptr(TUNSETIFF),
        uintptr(unsafe.Pointer(&req)),
    )
    if errno != 0 {
        syscall.Close(tapFD)
        return nil, fmt.Errorf("TUNSETIFF failed: %v", errno)
    }

    // Generate MAC address
    mac := make(net.HardwareAddr, 6)
    mac[0] = 0x02 // Local admin bit
    mac[1] = 0x00
    mac[2] = 0x00
    mac[3] = 0x00
    mac[4] = byte(len(m.devices) >> 8)
    mac[5] = byte(len(m.devices) & 0xff)

    dev := &TAPDevice{
        Name: name,
        FD:   tapFD,
        MAC:  mac,
    }

    m.devices[name] = dev
    return dev, nil
}

func (m *TAPManager) Destroy(name string) error {
    m.mu.Lock()
    defer m.mu.Unlock()

    dev, exists := m.devices[name]
    if !exists {
        return fmt.Errorf("device %s not found", name)
    }

    syscall.Close(dev.FD)
    delete(m.devices, name)

    return nil
}

func (m *TAPManager) SetIP(name string, ipStr string, maskLen int) error {
    m.mu.Lock()
    defer m.mu.Unlock()

    dev, exists := m.devices[name]
    if !exists {
        return fmt.Errorf("device %s not found", name)
    }

    dev.IP = net.ParseIP(ipStr)
    return nil
}

func (m *TAPManager) Get(name string) (*TAPDevice, error) {
    m.mu.RLock()
    defer m.mu.RUnlock()

    dev, exists := m.devices[name]
    if !exists {
        return nil, fmt.Errorf("device %s not found", name)
    }
    return dev, nil
}