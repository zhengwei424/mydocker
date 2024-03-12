package container

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"mydocker/vars"
	"os"
	"os/exec"
	"path"
	"syscall"
)

type ContainerInfo struct {
	Pid         string   `json:"pid"`         // 容器的init进程在主机上的pid
	Id          string   `json:"id"`          // 容器id
	Name        string   `json:"name"`        // 容器名称
	Command     string   `json:"command"`     // 容器的init运行命令
	CreatedTime string   `json:"createTime"`  // 容器创建时间
	Status      string   `json:"status"`      // 容器的状态
	Volume      string   `json:"volume"`      // 容器的数据卷
	PortMapping []string `json:"portMapping"` // 端口映射
}

// NewParentProcess 创建容器的父进程
func NewParentProcess(tty bool, volume, containerName, imageName string, env []string) (*exec.Cmd, *os.File) {
	//
	readPipe, writePipe, err := NewPipe()
	if err != nil {
		log.Errorf("New pipe error %v", err)
		return nil, nil
	}

	//
	cmd := exec.Command("/proc/self/exe", "init", containerName)
	cmd.Env = env
	cmd.SysProcAttr = &syscall.SysProcAttr{
		// 利用clone fork出来一个新进程，并使用namespace隔离
		Cloneflags: syscall.CLONE_NEWUTS | syscall.CLONE_NEWPID | syscall.CLONE_NEWNS | syscall.CLONE_NEWNET | syscall.CLONE_NEWIPC,
	}

	// 如果用户指定了-ti，则tty为true，需要将当前进程的输入输出导入到标准输入输出
	if tty {
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	} else {
		dirUrl := fmt.Sprintf(vars.DefaultInfoLocation, containerName)
		if err := os.MkdirAll(dirUrl, 0622); err != nil {
			log.Errorf("NewParentProcess mkdir %s error %v", dirUrl, err)
			return nil, nil
		}
		stdLogFilePath := path.Join(dirUrl, vars.ContainerLogFile)
		stdLogFile, err := os.Create(stdLogFilePath)
		if err != nil {
			log.Errorf("NewParentProcess create file %s error %v", stdLogFilePath, err)
		}
		// 将标准输出写入到日志文件
		cmd.Stdout = stdLogFile
	}

	// 传入pipe读取端
	cmd.ExtraFiles = []*os.File{readPipe}

	NewWorkSpace(volume, imageName, containerName)

	// 指定cmd工作目录
	cmd.Dir = fmt.Sprintf(vars.MntDir, containerName)

	// 返回cmd及pipe写入端
	return cmd, writePipe
}

func NewPipe() (*os.File, *os.File, error) {
	read, write, err := os.Pipe()
	if err != nil {
		return nil, nil, err
	}
	return read, write, nil
}
