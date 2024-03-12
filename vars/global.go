package vars

import "path"

var (
	RUNNING             string = "running"
	STOP                string = "stopped"
	RootPath            string = "/tmp/docker" // docker根目录
	ConfigName          string = "config.json"
	ContainerLogFile    string = "container.log"
	ContainersRootPath  string = path.Join(RootPath, "containers")     // 容器根目录
	NetworkRootPath     string = path.Join(RootPath, "network/")       // 网络根目录
	NetworkDir          string = path.Join(NetworkRootPath, "network") // 网络配置
	IPAMDir             string = path.Join(NetworkRootPath, "ipam")    // ipam配置
	ImagesDir           string = path.Join(RootPath, "images")
	DefaultInfoLocation string = path.Join(ContainersRootPath, "%s")
	LowerDir            string = path.Join(ContainersRootPath, "%s/lowerLayer") // overlay文件系统层
	UpperDir            string = path.Join(ContainersRootPath, "%s/upperLayer") // overlay文件系统层
	WorkDir             string = path.Join(ContainersRootPath, "%s/workLayer")  // overlay文件系统层
	MntDir              string = path.Join(ContainersRootPath, "%s/mnt")        // overlay文件系统层
)
