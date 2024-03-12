package cmd

import (
	"encoding/json"
	"fmt"
	log "github.com/sirupsen/logrus"
	"math/rand"
	"mydocker/cgroups"
	"mydocker/cgroups/subsystems"
	"mydocker/container"
	"mydocker/network"
	"mydocker/vars"
	"os"
	"path"
	"strconv"
	"strings"
	"time"
)

// 启动容器时，增加资源限制
func Run(tty bool, commandArray []string, res *subsystems.ResourceConfig, volume, containerName string, env []string, networkName string, portMapping []string) {
	// 生成容器ID
	containerID := randStringBytes(10)
	if containerName == "" {
		containerName = containerID
	}

	imageName := commandArray[0]

	// 提前建好目录
	os.MkdirAll(path.Join(vars.ContainersRootPath, containerName), 0755)
	dirs := []string{
		fmt.Sprintf(vars.LowerDir, containerName),
		fmt.Sprintf(vars.UpperDir, containerName),
		fmt.Sprintf(vars.WorkDir, containerName),
		fmt.Sprintf(vars.MntDir, containerName),
	}
	for _, dir := range dirs {
		exist, err := container.PathExists(dir)
		if !exist && err == nil {
			if err = os.MkdirAll(dir, 0755); err != nil {
				fmt.Printf("Mkdir %s error: %v\n", dir, err)
			}
		}
	}

	cmd, writePipe := container.NewParentProcess(tty, volume, containerName, imageName, env)
	if cmd == nil {
		log.Errorf("New parent process error")
		return
	}

	// exec.Command.Run()会阻塞当前程序，直到命令执行完成；exec.Command.Start()允许你在命令执行的同时，继续执行其他操作，符合容器运行情况。
	if err := cmd.Start(); err != nil {
		log.Error(err)
	}

	// 记录容器信息
	containerName, err := recordContainerInfo(cmd.Process.Pid, commandArray[1:], containerName, containerID, volume)
	if err != nil {
		log.Errorf("Record container info error: %v", err)
	}

	// 用mydocker-cgroup作为cgroup名称
	cgroupManager := cgroups.NewCgroupManager("mydocker-cgroup")
	cgroupManager.Resource = res

	// 退出run时销毁cgroup
	defer cgroupManager.Destory()
	cgroupManager.Set()
	cgroupManager.Apply(cmd.Process.Pid)

	if networkName != "" {
		network.Init()
		containerInfo := &container.ContainerInfo{
			Id:          containerID,
			Pid:         strconv.Itoa(cmd.Process.Pid),
			Name:        containerName,
			PortMapping: portMapping,
		}
		if err := network.Connect(networkName, containerInfo); err != nil {
			log.Errorf("connect network error: %v", err)
			return
		}
	}

	// 将命令行参数传入到writePipe中
	sendInitCommand(commandArray[1:], writePipe)

	if tty {
		err := cmd.Wait()
		if err != nil {
			log.Errorf("parent.Wait() error: %v\n", err)
		}

		// 如果tty方式，在退出时清理容器信息
		deleteContainerInfo(containerName)
		// 为什么不能用defer？？？？？？？？？？？？？？？？？
		container.DeleteWorkSpace(containerName, volume)

		os.Exit(0)
	}
}

func sendInitCommand(commandArray []string, writePipe *os.File) {
	command := strings.Join(commandArray, " ")
	log.Infof("command is %s", command)

	writePipe.WriteString(command)
	writePipe.Close()
}

// 记录容器相关信息
func recordContainerInfo(containerPID int, commandArray []string, containerName, containerID, volume string) (string, error) {
	// 以当前时间作为容器的创建时间
	createTime := time.Now().Format("2006-01-02 15:04:05")
	// 容器的命令
	command := strings.Join(commandArray, " ")

	// 生成容器信息
	containerInfo := container.ContainerInfo{
		Id:          containerID,
		Pid:         strconv.Itoa(containerPID),
		Name:        containerName,
		Command:     command,
		CreatedTime: createTime,
		Status:      vars.RUNNING,
		Volume:      volume,
	}

	// 将容器信息转换为json
	jsonBytes, err := json.Marshal(containerInfo)
	if err != nil {
		log.Errorf("Record containerInfo error: %v", err)
	}
	jsonstr := string(jsonBytes)

	// 创建当前容器存储容器信息的目录
	dir := fmt.Sprintf(vars.DefaultInfoLocation, containerName)
	if err := os.MkdirAll(dir, 0622); err != nil {
		log.Errorf("mkdir %s error: %v", dir, err)
		return "", nil
	}

	// 创建当前容器存储容器信息的文件
	fileName := path.Join(dir, vars.ConfigName)
	file, err := os.Create(fileName)
	defer file.Close()
	if err != nil {
		log.Errorf("Create file %s error %v", fileName, err)
		return "", nil
	}

	// 将容器信息写入到文件中
	if _, err := file.WriteString(jsonstr); err != nil {
		log.Errorf("File write string error %v", err)
		return "", err
	}

	return containerName, nil
}

// 清理指定容器的相关信息
func deleteContainerInfo(containerName string) {
	dir := fmt.Sprintf(vars.DefaultInfoLocation, containerName)
	if err := os.RemoveAll(dir); err != nil {
		log.Errorf("Remove %s error: %v", dir, err)
	}
}

// 容器ID生成器
func randStringBytes(n int) string {
	b := make([]string, n)
	for i := range b {
		b[i] = strconv.Itoa(rand.Intn(n))
	}
	return strings.Join(b, "")
}
