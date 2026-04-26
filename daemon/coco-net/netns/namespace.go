package netns

import (
	"fmt"
	"net"
	"os"
	"sync"

	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
)

type Namespace struct {
	mu       sync.RWMutex
	ID       string
	Handle   netns.NsHandle
	Interfaces map[string]*NetInterface
	CreatedAt  string
}

type NetInterface struct {
	Name     string
	Link     netlink.Link
	IPs      []net.IP
	MAC      string
	Type     string
}

func New(id string) (*Namespace, error) {
	ns, err := netns.NewNamed(id)
	if err != nil {
		return nil, fmt.Errorf("failed to create namespace: %w", err)
	}

	n := &Namespace{
		ID:         id,
		Handle:     ns,
		Interfaces: make(map[string]*NetInterface),
		CreatedAt:  fmt.Sprintf("%d", os.Getpid()),
	}

	return n, nil
}

func Get(id string) (*Namespace, error) {
	ns, err := netns.GetFromName(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get namespace: %w", err)
	}

	n := &Namespace{
		ID:         id,
		Handle:     ns,
		Interfaces: make(map[string]*NetInterface),
	}

	return n, nil
}

func (n *Namespace) Close() error {
	if n.Handle > 0 {
		return n.Handle.Close()
	}
	return nil
}

func (n *Namespace) Enter() error {
	return netns.Set(n.Handle)
}

func (n *Namespace) AddVeth(hostName, guestName string) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	hostLink := &netlink.Veth{
		LinkAttrs: netlink.LinkAttrs{
			Name: hostName,
		},
		PeerName: guestName,
	}

	if err := netlink.LinkAdd(hostLink); err != nil {
		return fmt.Errorf("failed to add veth pair: %w", err)
	}

	if err := netlink.LinkSetUp(hostLink); err != nil {
		return fmt.Errorf("failed to set veth up: %w", err)
	}

	n.Interfaces[hostName] = &NetInterface{
		Name: hostName,
		Link: hostLink,
		Type: "veth",
	}

	return nil
}

func (n *Namespace) SetMaster(ifaceName, bridgeName string) error {
	link, err := netlink.LinkByName(ifaceName)
	if err != nil {
		return fmt.Errorf("failed to get interface: %w", err)
	}

	bridge, err := netlink.LinkByName(bridgeName)
	if err != nil {
		return fmt.Errorf("failed to get bridge: %w", err)
	}

	if err := netlink.LinkSetMaster(link, bridge); err != nil {
		return fmt.Errorf("failed to set master: %w", err)
	}

	return nil
}

func (n *Namespace) SetIP(ifaceName string, ip net.IP, mask int) error {
	link, err := netlink.LinkByName(ifaceName)
	if err != nil {
		return fmt.Errorf("failed to get interface: %w", err)
	}

	addr := &netlink.Addr{
		IPNet: &net.IPNet{
			IP:   ip,
			Mask: net.CIDRMask(mask, 32),
		},
	}

	if err := netlink.AddrAdd(link, addr); err != nil {
		return fmt.Errorf("failed to set IP: %w", err)
	}

	n.mu.Lock()
	if iface, ok := n.Interfaces[ifaceName]; ok {
		iface.IPs = append(iface.IPs, ip)
	}
	n.mu.Unlock()

	return nil
}

func (n *Namespace) ListInterfaces() []*NetInterface {
	n.mu.RLock()
	defer n.mu.RUnlock()

	interfaces := make([]*NetInterface, 0, len(n.Interfaces))
	for _, iface := range n.Interfaces {
		interfaces = append(interfaces, iface)
	}

	return interfaces
}

func (n *Namespace) GetInterface(name string) (*NetInterface, bool) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	iface, ok := n.Interfaces[name]
	return iface, ok
}

func (n *Namespace) DeleteInterface(name string) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	link, err := netlink.LinkByName(name)
	if err != nil {
		return fmt.Errorf("failed to get interface: %w", err)
	}

	if err := netlink.LinkDel(link); err != nil {
		return fmt.Errorf("failed to delete interface: %w", err)
	}

	delete(n.Interfaces, name)

	return nil
}
