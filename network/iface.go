package network

import (
	"fmt"
	"github.com/vishvananda/netlink"
	"time"
)

func setInterfaceUp(linkName string) error {
	l, err := netlink.LinkByName(linkName)
	if err != nil {
		return fmt.Errorf("get bridge %s link error: %v", linkName, err)
	}

	if err := netlink.LinkSetUp(l); err != nil {
		return fmt.Errorf("enable link %s error: %v", linkName, err)
	}

	return nil
}

func setInterfaceIP(linkName string, ipNet string) error {
	retries := 2
	l := new(netlink.Link)
	var err error

	for i := 0; i < retries; i++ {
		*l, err = netlink.LinkByName(linkName)
		if err == nil {
			break
		}
		time.Sleep(2 * time.Second)
	}

	if err != nil {
		return fmt.Errorf("get link %s error: %v", linkName, err)
	}

	ipnet, err := netlink.ParseIPNet(ipNet)
	if err != nil {
		return fmt.Errorf("parse ipnet error: %v", err)
	}

	addr := &netlink.Addr{IPNet: ipnet}
	return netlink.AddrAdd(*l, addr)
}
