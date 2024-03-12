package subsystems

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strconv"
)

// 定义结构体，并实现subsystem接口

type CpuSubSystem struct {
}

func (s *CpuSubSystem) Name() string {
	return "cpu"
}

// 将cpu资源限制写入cpu cgroup目录下的cpu.shares文件中
func (s *CpuSubSystem) Set(cgroupPath string, res *ResourceConfig) error {
	// 获取（新建）目录/sys/fs/cgroup/cpu,cpuacct
	if subsysCgroupPath, err := GetCgroupPath(s.Name(), cgroupPath, true); err == nil {
		if res.CpuShare != "" {
			// 将res.CpuShare配置写入到/sys/fs/cgroup/cpu,cpuacct/cpu.shares文件中
			if err := ioutil.WriteFile(path.Join(subsysCgroupPath, "cpu.shares"), []byte(res.CpuShare), 0644); err != nil {
				return fmt.Errorf("set cgroup share fail %v", err)
			}
		}
		return nil
	} else {
		return err
	}

}

// 将对应进程的pid写入cpu cgroup下的tasks文件中
func (s *CpuSubSystem) Apply(cgroupPath string, pid int) error {
	if subsysCgroupPath, err := GetCgroupPath(s.Name(), cgroupPath, false); err == nil {
		if err := ioutil.WriteFile(path.Join(subsysCgroupPath, "tasks"), []byte(strconv.Itoa(pid)), 0644); err != nil {
			return fmt.Errorf("set cgroup proc fail %v", err)
		}
		return nil
	} else {
		return err
	}
}

// 删除cpu cgroup目录
func (s *CpuSubSystem) Remove(cgroupPath string) error {
	if subsysCgroup, err := GetCgroupPath(s.Name(), cgroupPath, false); err == nil {
		// 删除目录
		return os.RemoveAll(subsysCgroup)
	} else {
		return err
	}
}
