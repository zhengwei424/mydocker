package network

import (
	"github.com/vishvananda/netlink"
	"net"
)

type Endpoint struct {
	ID          string           `json:"id"` // fmt.Sprintf("%s-%s", cinfo.Id, networkName),
	Device      netlink.Veth     `json:"dev"`
	IPAddress   net.IP           `json:"ip"`
	MacAddress  net.HardwareAddr `json:"mac"`
	Network     *Net
	PortMapping []string
}

type NetworkDriver interface {
	Name() string
	Create(subnet string, name string) (*Net, error)
	Delete(network Net) error
	Connect(network *Net, endpoint *Endpoint) error
	//Disconnect(network Net, endpoint *Endpoint) error
}
