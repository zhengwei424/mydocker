package network

import (
	"encoding/json"
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
	"mydocker/container"
	"mydocker/vars"
	"net"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"text/tabwriter"
)

type Net struct {
	Name   string
	IpNet  *net.IPNet
	Driver string
}

// Dump config
func (nw *Net) dump(dumpPath string) error {
	if _, err := os.Stat(dumpPath); err != nil {
		if os.IsNotExist(err) {
			os.MkdirAll(dumpPath, 0644)
		} else {
			return fmt.Errorf("%s stat error: %v", dumpPath, err)
		}
	}

	nwPath := path.Join(dumpPath, nw.Name)
	// os.O_TRUNC表示如果文件存在，则清空文件内容
	nwFile, err := os.OpenFile(nwPath, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return fmt.Errorf("open file %s error: %v", nwPath, err)
	}

	defer nwFile.Close()

	nwJson, err := json.Marshal(nw)

	if err != nil {
		return fmt.Errorf("json Marshal error: %v", err)
	}

	_, err = nwFile.Write(nwJson)
	if err != nil {
		return fmt.Errorf("write file error: %v", err)
	}
	return nil
}

// Remove network
func (nw *Net) remove(dumpPath string) error {
	p := path.Join(dumpPath, nw.Name)
	if _, err := os.Stat(p); err != nil {
		if os.IsNotExist(err) {
			return nil
		} else {
			return fmt.Errorf("%s stat error: %v", p, err)
		}
	} else {
		return os.Remove(p)
	}
}

func (nw *Net) load(networkConfigPath string) error {
	nwConfigFile, err := os.Open(networkConfigPath)
	defer nwConfigFile.Close()
	if err != nil {
		return fmt.Errorf("open file %s error: %v", networkConfigPath, err)
	}

	nwJson := make([]byte, 2000)
	n, err := nwConfigFile.Read(nwJson)
	if err != nil {
		return fmt.Errorf("read file %s to []byte error: %v", networkConfigPath, err)
	}

	err = json.Unmarshal(nwJson[:n], nw)
	if err != nil {
		log.Errorf("json unmarshal *Network error: %v", err)
	}
	return nil
}

var (
	drivers  = map[string]NetworkDriver{}
	networks = map[string]*Net{}
)

// 初始化全局变量drivers和networks;加载(如果存在)所有的网络驱动配置到全局变量networks(网桥bridge等)
func Init() {
	networks = make(map[string]*Net)
	drivers = make(map[string]NetworkDriver)
	// 目前只实现了bridge驱动，所有只初始化bridge，如果有其他driver，都需要在这里初始化！！！！！！！
	drivers["bridge"] = &Bridge{}

	if _, err := os.Stat(vars.NetworkDir); err != nil {
		if os.IsNotExist(err) {
			os.MkdirAll(vars.NetworkDir, 0644)
		} else {
			log.Errorf("get %s stat error: %v", vars.NetworkDir, err)
		}
	}

	// 遍历vars.NetworkDir目录下的文件
	filepath.Walk(vars.NetworkDir, func(Path string, info os.FileInfo, err error) error {
		//if strings.HasSuffix(nwPath, "/") {
		//	return nil
		//}
		f, err := os.Stat(Path)
		if f.IsDir() {
			return nil
		}

		_, nwName := path.Split(Path)
		nw := &Net{
			Name: nwName,
		}

		if err := nw.load(Path); err != nil {
			return fmt.Errorf("error load network: %s", err)
		}

		nw.IpNet.IP = nw.IpNet.IP.To4()
		networks[nwName] = nw

		return nil
	})

}

func CreateNetwork(driver, subnet, name string) error {
	_, cidr, _ := net.ParseCIDR(subnet)
	ip, err := ipAllocator.Allocate(cidr)
	if err != nil {
		return err
	}
	cidr.IP = ip

	nw, err := drivers[driver].Create(cidr.String(), name)
	if err != nil {
		return err
	}

	return nw.dump(vars.NetworkDir)
}

func ListNetwork() {
	w := tabwriter.NewWriter(os.Stdout, 12, 1, 3, ' ', 0)
	fmt.Fprint(w, "NAME\tIpRange\tDriver\n")
	for _, nw := range networks {
		fmt.Fprintf(w, "%s\t%s\t%s\n",
			nw.Name,
			nw.IpNet.String(),
			nw.Driver,
		)
	}
	if err := w.Flush(); err != nil {
		fmt.Errorf("Flush error %v", err)
		return
	}
}

func DeleteNetwork(networkName string) error {
	nw, ok := networks[networkName]
	if !ok {
		return fmt.Errorf("No Such Network: %s", networkName)
	}

	if err := ipAllocator.Release(nw.IpNet, &nw.IpNet.IP); err != nil {
		return fmt.Errorf("Error Remove Network gateway ip: %s", err)
	}

	if err := drivers[nw.Driver].Delete(*nw); err != nil {
		return fmt.Errorf("Error Remove Network DriverError: %s", err)
	}

	return nw.remove(vars.NetworkDir)
}

func enterContainerNetns(enLink *netlink.Link, cinfo *container.ContainerInfo) func() {
	f, err := os.OpenFile(fmt.Sprintf("/proc/%s/ns/net", cinfo.Pid), os.O_RDONLY, 0)
	if err != nil {
		log.Errorf("error get container net namespace, %v", err)
	}

	nsFD := f.Fd()
	runtime.LockOSThread()

	// 修改veth peer 另外一端移到容器的namespace中
	if err = netlink.LinkSetNsFd(*enLink, int(nsFD)); err != nil {
		log.Errorf("error set link netns , %v", err)
	}

	// 获取当前的网络namespace
	origns, err := netns.Get()
	if err != nil {
		log.Errorf("error get current netns, %v", err)
	}

	// 设置当前进程到新的网络namespace，并在函数执行完成之后再恢复到之前的namespace
	if err = netns.Set(netns.NsHandle(nsFD)); err != nil {
		log.Errorf("error set netns, %v", err)
	}
	return func() {
		netns.Set(origns)
		origns.Close()
		runtime.UnlockOSThread()
		f.Close()
	}
}

func configEndpointIpAddressAndRoute(ep *Endpoint, cinfo *container.ContainerInfo) error {
	peerLink, err := netlink.LinkByName(ep.Device.PeerName)
	if err != nil {
		return fmt.Errorf("fail config endpoint: %v", err)
	}

	defer enterContainerNetns(&peerLink, cinfo)()

	interfaceIP := *ep.Network.IpNet
	interfaceIP.IP = ep.IPAddress

	if err = setInterfaceIP(ep.Device.PeerName, interfaceIP.String()); err != nil {
		return fmt.Errorf("%v,%s", ep.Network, err)
	}

	if err = setInterfaceUp(ep.Device.PeerName); err != nil {
		return err
	}

	if err = setInterfaceUp("lo"); err != nil {
		return err
	}

	_, cidr, _ := net.ParseCIDR("0.0.0.0/0")

	defaultRoute := &netlink.Route{
		LinkIndex: peerLink.Attrs().Index,
		Gw:        ep.Network.IpNet.IP,
		Dst:       cidr,
	}

	if err = netlink.RouteAdd(defaultRoute); err != nil {
		return err
	}

	return nil
}

func configPortMapping(ep *Endpoint, cinfo *container.ContainerInfo) error {
	for _, pm := range ep.PortMapping {
		portMapping := strings.Split(pm, ":")
		if len(portMapping) != 2 {
			log.Errorf("port mapping format error, %v", pm)
			continue
		}
		iptablesCmd := fmt.Sprintf("-t nat -A PREROUTING -p tcp -m tcp --dport %s -j DNAT --to-destination %s:%s",
			portMapping[0], ep.IPAddress.String(), portMapping[1])
		cmd := exec.Command("iptables", strings.Split(iptablesCmd, " ")...)
		//err := cmd.Run()
		output, err := cmd.Output()
		if err != nil {
			log.Errorf("iptables Output, %v", output)
			continue
		}
	}
	return nil
}

func Connect(networkName string, cinfo *container.ContainerInfo) error {
	nw, ok := networks[networkName]
	if !ok {
		return fmt.Errorf("No Such Network: %s", networkName)
	}

	// 分配容器IP地址
	ip, err := ipAllocator.Allocate(nw.IpNet)
	if err != nil {
		return err
	}

	// 创建网络端点
	ep := &Endpoint{
		ID:          fmt.Sprintf("%s-%s", cinfo.Id, networkName),
		IPAddress:   ip,
		Network:     nw,
		PortMapping: cinfo.PortMapping,
	}
	// 调用网络驱动挂载和配置网络端点
	if err = drivers[nw.Driver].Connect(nw, ep); err != nil {
		return err
	}
	// 到容器的namespace配置容器网络设备IP地址
	if err = configEndpointIpAddressAndRoute(ep, cinfo); err != nil {
		return err
	}

	return configPortMapping(ep, cinfo)
}

func Disconnect(networkName string, cinfo *container.ContainerInfo) error {
	return nil
}
