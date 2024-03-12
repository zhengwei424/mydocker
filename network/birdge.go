package network

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
	"net"
	"os/exec"
	"strings"
)

type Bridge struct {
}

func (b *Bridge) Name() string {
	return "bridge"
}

// Create 创建网桥
//
// 参数 subnet: 指定ip/net,如: "192.168.1.11/24"
//
// 参数 bridgeName: 网桥名称
func (b *Bridge) Create(subnet, bridgeName string) (*Net, error) {
	ip, ipNet, _ := net.ParseCIDR(subnet)
	ipNet.IP = ip
	n := &Net{
		Name:   bridgeName,
		IpNet:  ipNet,
		Driver: b.Name(),
	}

	// 判断bridge是否存在
	_, err := net.InterfaceByName(bridgeName)
	if err == nil {
		return nil, fmt.Errorf("bridge %s already exists", bridgeName)
	} else if !strings.Contains(err.Error(), "no such network interface") {
		return nil, fmt.Errorf("get interface %s error: %v", bridgeName, err)
	}

	la := netlink.NewLinkAttrs()
	la.Name = bridgeName
	br := &netlink.Bridge{
		LinkAttrs: la,
	}

	if err := netlink.LinkAdd(br); err != nil {
		return nil, fmt.Errorf("bridge %s create failed: %v", bridgeName, err)
	}

	if err := setInterfaceIP(bridgeName, n.IpNet.String()); err != nil {
		return nil, fmt.Errorf("allocate ip address %s on bridge %s error: %v", n.IpNet.IP.String(), bridgeName, err)
	}

	if err := setInterfaceUp(bridgeName); err != nil {
		return nil, fmt.Errorf("set bridge %s up error: %v", bridgeName, err)
	}

	if err := setUpIptables(bridgeName, n.IpNet); err != nil {
		return nil, fmt.Errorf("set iptables for bridge %s error: %v", bridgeName, err)
	}

	return n, nil
}

func (b *Bridge) Delete(network Net) error {
	bridgeName := network.Name
	br, err := netlink.LinkByName(bridgeName)
	if err != nil {
		return fmt.Errorf("get bridge %s link error: %v", bridgeName, err)
	}
	err = netlink.LinkDel(br)
	if err != nil {
		return fmt.Errorf("delete %s`s link error: %v", bridgeName, err)
	}
	return err
}

func (b *Bridge) Connect(network *Net, endpoint *Endpoint) error {
	bridgeName := network.Name
	br, err := netlink.LinkByName(bridgeName)
	if err != nil {
		return fmt.Errorf("get bridge %s link error: %v", bridgeName, err)
	}

	la := netlink.NewLinkAttrs()
	la.Name = endpoint.ID[:5]
	la.MasterIndex = br.Attrs().Index

	endpoint.Device = netlink.Veth{
		LinkAttrs: la,
		PeerName:  "cif-" + endpoint.ID[:5],
	}

	if err = netlink.LinkAdd(&endpoint.Device); err != nil {
		return fmt.Errorf("add endpoint device %s error: %v", endpoint.Device.PeerName, err)
	}

	if err = netlink.LinkSetUp(&endpoint.Device); err != nil {
		return fmt.Errorf("enable endpoint device %s error: %v", endpoint.Device.PeerName, err)
	}

	return nil
}

func setUpIptables(bridgeName string, subnet *net.IPNet) error {
	iptablesCmd := fmt.Sprintf("-t nat -A POSTROUTING -s %s ! -o %s -j MASQUERADE", subnet.String(), bridgeName)
	cmd := exec.Command("iptables", strings.Split(iptablesCmd, " ")...)
	output, err := cmd.Output()
	if err != nil {
		log.Errorf("write iptables output: %s", string(output))
		return fmt.Errorf("iptables error: %v", err)
	}
	return nil
}
