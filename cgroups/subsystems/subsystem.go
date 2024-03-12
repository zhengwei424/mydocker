package subsystems

// 用于传递资源限制
type ResourceConfig struct {
	// 内存限制
	MemoryLimit string
	// cpu时间片权重限制
	CpuShare string
	// cpu核心数限制
	CpuSet string
}

// 将cgroup抽象为path
type SubSystem interface {
	Name() string
	Set(path string, res *ResourceConfig) error
	Apply(path string, pid int) error
	Remove(path string) error
}

// 定义一个全局的subsystem
var (
	SubSystemIns = []SubSystem{
		&CpuSubSystem{},
		&CpusetSubSystem{},
		&MemorySubSystem{},
	}
)
