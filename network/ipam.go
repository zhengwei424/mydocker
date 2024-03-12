package network

import (
	"encoding/json"
	"fmt"
	"mydocker/vars"
	"net"
	"os"
	"path"
	"strings"
)

type IPAM struct {
	SubnetAllocationPath string
	Subnets              *map[string]string
}

var ipAllocator = &IPAM{
	SubnetAllocationPath: path.Join(vars.IPAMDir, "subnet.json"),
}

// 将subnet的json配置文件加载到结构体
func (ipam *IPAM) load() error {
	if _, err := os.Stat(ipam.SubnetAllocationPath); err != nil {
		if os.IsNotExist(err) {
			// 不存在则不需要加载
			return nil
		}
		return fmt.Errorf("get %s state error: %v", ipam.SubnetAllocationPath, err)
	}
	subnetConfigFile, err := os.Open(ipam.SubnetAllocationPath)
	if err != nil {
		return fmt.Errorf("open file %s error: %v", ipam.SubnetAllocationPath, err)
	}

	defer subnetConfigFile.Close()

	// 大文件转json
	//decoder := json.NewDecoder(subnetConfigFile)
	//for decoder.More() {
	//	if err := decoder.Decode(&ipam.Subnets); err != nil {
	//
	//	}
	//}
	subnetJson := make([]byte, 2000)
	n, err := subnetConfigFile.Read(subnetJson)
	if err != nil {
		return fmt.Errorf("read file %s to []byte error: %v", ipam.SubnetAllocationPath, err)
	}

	err = json.Unmarshal(subnetJson[:n], ipam.Subnets)
	if err != nil {
		return fmt.Errorf("json unmarshal subnet config error: %v", err)
	}
	return err
}

// 将subnet配置结构体写入json配置文件中
func (ipam *IPAM) dump() error {
	if _, err := os.Stat(vars.IPAMDir); err != nil {
		if os.IsNotExist(err) {
			os.MkdirAll(vars.IPAMDir, 0644)
		} else {
			return fmt.Errorf("get %s state error: %v", vars.IPAMDir, err)
		}
	}

	subnetConfigFile, err := os.OpenFile(ipam.SubnetAllocationPath, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return fmt.Errorf("open file %s error: %v", ipam.SubnetAllocationPath, err)
	}
	defer subnetConfigFile.Close()

	ipamConfigJson, err := json.Marshal(ipam.Subnets)
	if err != nil {
		return fmt.Errorf("json marshal subnet config error: %v", err)
	}

	_, err = subnetConfigFile.Write(ipamConfigJson)
	if err != nil {
		return fmt.Errorf("write subnet config bytes to file error: %v", err)
	}

	return nil
}

// 分配ip
func (ipam *IPAM) Allocate(subnet *net.IPNet) (ip net.IP, err error) {
	// 初始化(子网分配信息)
	ipam.Subnets = &map[string]string{}

	// 加载配置文件
	err = ipam.load()
	if err != nil {
		return nil, fmt.Errorf("load subnet config error: %v", err)
	}

	// ones表示网络前缀中连续的1的位数，bits表示网络前缀的总位数。例：
	// 假设有一个 IP 地址的网络前缀是 192.168.1.0/24，其中 /24 表示网络前缀的长度为 24 位。那么在这个例子中：
	// ones 将是 24，表示网络前缀中连续的 1 的位数为 24 位。
	// bits 将是 32，表示 IPv4 地址的总位数为 32 位。
	ones, bits := subnet.Mask.Size()

	// 表示以subnet.String()的值作为ipam.Subnets这个map的key来检索map的value
	if _, exist := (*ipam.Subnets)[subnet.String()]; !exist {
		(*ipam.Subnets)[subnet.String()] = strings.Repeat("0", 1<<uint8(bits-ones))
	}

	for c := range (*ipam.Subnets)[subnet.String()] {
		// 地址分配原理(bitmap算法，一个ip地址用一位来表示，将已分配的ip地址标记为1，未分配标记为0）
		if (*ipam.Subnets)[subnet.String()][c] == '0' {
			// string转byte，用于修改字符串中的某个字符
			ipalloc := []byte((*ipam.Subnets)[subnet.String()])
			ipalloc[c] = '1'
			(*ipam.Subnets)[subnet.String()] = string(ipalloc)
			ip = subnet.IP
			for t := uint(4); t > 0; t -= 1 {
				ip[4-t] += uint8(c >> ((t - 1) * 8))
			}

			// 第一个ip地址默认是0，由于iP地址从1开始分配，所以再加1
			ip[3] += 1
			break
		}
	}

	// 写入配置文件
	err = ipam.dump()

	return ip, err
}

// 释放ip
func (ipam *IPAM) Release(subnet *net.IPNet, ipaddr *net.IP) error {
	ipam.Subnets = &map[string]string{}

	err := ipam.load()
	if err != nil {
		return fmt.Errorf("load subnet config error: %v", err)
	}

	c := 0
	releaseIP := ipaddr.To4()
	releaseIP[3] -= 1
	for t := uint(4); t > 0; t -= 1 {
		c += int(releaseIP[t-1]-subnet.IP[t-1]) << ((4 - t) * 8)
	}

	ipalloc := []byte((*ipam.Subnets)[subnet.String()])
	ipalloc[c] = '0'
	(*ipam.Subnets)[subnet.String()] = string(ipalloc)

	err = ipam.dump()
	return err
}
