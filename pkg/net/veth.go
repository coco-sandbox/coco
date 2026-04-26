package net

import (
	"fmt"
	"net"
)

type Veth struct {
	Name     string
	PeerName string
	MTU      int
	MAC      net.HardwareAddr
}

func NewVeth(name, peerName string) (*Veth, error) {
	return &Veth{
		Name:     name,
		PeerName: peerName,
		MTU:      1500,
	}, nil
}

func (v *Veth) Create() error {
	return nil
}

func (v *Veth) Delete() error {
	return nil
}

func (v *Veth) SetUp() error {
	return nil
}

func (v *Veth) SetDown() error {
	return nil
}

func (v *Veth) SetMTU(mtu int) error {
	v.MTU = mtu
	return nil
}

func (v *Veth) SetMAC(mac string) error {
	hw, err := net.ParseMAC(mac)
	if err != nil {
		return fmt.Errorf("invalid MAC address: %w", err)
	}
	v.MAC = hw
	return nil
}

func (v *Veth) SetPeer(ns string) error {
	return nil
}

func (v *Veth) MoveToNamespace(nsPath string) error {
	return nil
}

func (v *Veth) GetPeer() (*Veth, error) {
	return &Veth{Name: v.PeerName}, nil
}
