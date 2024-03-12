package cmd

import (
	"encoding/json"
	"fmt"
	log "github.com/sirupsen/logrus"
	"mydocker/vars"
	"os"
	"path"
	"strconv"
	"syscall"
)

func StopContainer(containerName string) {
	// 获取容器进程
	pid, err := getContainerPidByName(containerName)
	if err != nil {
		log.Errorf("Get container pid by name %s error %v", containerName, err)
	}

	pidInt, err := strconv.Atoi(pid)
	if err != nil {
		log.Errorf("Conver pid from string to int error %v", err)
		return
	}

	// 杀死容器进程(附kill signal 列表）
	/*
		SIGHUP (1) - 挂起信号
		SIGINT (2) - 中断信号
		SIGQUIT (3) - 退出信号
		SIGILL (4) - 非法指令信号
		SIGTRAP (5) - 跟踪陷阱信号
		SIGABRT (6) - 中止信号
		SIGBUS (7) - 总线错误信号
		SIGFPE (8) - 浮点异常信号
		SIGKILL (9) - 强制终止信号
		SIGUSR1 (10) - 用户定义信号 1
		SIGSEGV (11) - 段错误信号
		SIGUSR2 (12) - 用户定义信号 2
		SIGPIPE (13) - 管道损坏信号
		SIGALRM (14) - 警报时钟信号
		SIGTERM (15) - 终止信号
		SIGSTKFLT (16) - 协处理器栈错误信号
		SIGCHLD (17) - 子进程退出信号
		SIGCONT (18) - 继续执行信号
		SIGSTOP (19) - 停止进程信号
		SIGTSTP (20) - 终端停止信号
		SIGTTIN (21) - 后台进程尝试读取控制终端信号
		SIGTTOU (22) - 后台进程尝试写控制终端信号
	*/
	if err := syscall.Kill(pidInt, syscall.SIGTERM); err != nil {
		log.Errorf("Stop container %s error %v", containerName, err)
	}

	// 修改容器运行状态信息
	containerInfo, err := getContainerInfo(containerName)
	if err != nil {
		log.Errorf("Get containerInfo from %s error %v", containerName, err)
		return
	}
	containerInfo.Status = vars.STOP
	containerInfo.Pid = ""
	newContentBytes, err := json.Marshal(containerInfo)
	if err != nil {
		log.Errorf("Json marshal %s error %v", containerName, err)
		return
	}

	dirUrl := fmt.Sprintf(vars.DefaultInfoLocation, containerName)
	configFilePath := path.Join(dirUrl, vars.ConfigName)
	if err := os.WriteFile(configFilePath, newContentBytes, 0622); err != nil {
		log.Errorf("Write file %s error %v", configFilePath, err)
	}
}
