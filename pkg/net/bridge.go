package net

import (
	"fmt"
	"net"
)

type Bridge struct {
	Name    string
	IP      net.IP
	Netmask net.IPMask
	MTU     int
}

func NewBridge(name string, ip string, netmask string) (*Bridge, error) {
	bridgeIP := net.ParseIP(ip)
	if bridgeIP == nil {
		return nil, fmt.Errorf("invalid IP address: %s", ip)
	}

	mask := net.ParseIP(netmask)
	if mask == nil {
		return nil, fmt.Errorf("invalid netmask: %s", netmask)
	}

	return &Bridge{
		Name:    name,
		IP:      bridgeIP,
		Netmask: net.IPMask(mask),
		MTU:     1500,
	}, nil
}

func (b *Bridge) Create() error {
	return nil
}

func (b *Bridge) Delete() error {
	return nil
}

func (b *Bridge) AddMember(iface string) error {
	return nil
}

func (b *Bridge) RemoveMember(iface string) error {
	return nil
}

func (b *Bridge) SetUp() error {
	return nil
}

func (b *Bridge) SetDown() error {
	return nil
}

func (b *Bridge) SetMTU(mtu int) error {
	b.MTU = mtu
	return nil
}
