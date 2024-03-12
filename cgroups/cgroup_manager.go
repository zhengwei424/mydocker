package cgroups

import (
	"mydocker/cgroups/subsystems"
)

type CgroupManager struct {
	// cgroup在hierarchy中的路径，相当于创建的cgroup目录相对于root cgroup目录的路径
	cgroupPath string
	// 资源配置
	Resource *subsystems.ResourceConfig
}

func NewCgroupManager(cgroupPath string) *CgroupManager {
	return &CgroupManager{
		cgroupPath: cgroupPath,
	}
}

// 统一资源限制设置
func (c *CgroupManager) Set() error {
	for _, subSysIns := range subsystems.SubSystemIns {
		return subSysIns.Set(c.cgroupPath, c.Resource)
	}
	return nil
}

// 统一pid设置
func (c *CgroupManager) Apply(pid int) error {
	for _, subSysIns := range subsystems.SubSystemIns {
		return subSysIns.Apply(c.cgroupPath, pid)
	}
	return nil
}

// 统一cgroup移除
func (c *CgroupManager) Destory() error {
	for _, subSysIns := range subsystems.SubSystemIns {
		return subSysIns.Remove(c.cgroupPath)
	}
	return nil
}
