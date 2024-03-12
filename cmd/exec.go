package cmd

import (
	"encoding/json"
	"fmt"
	log "github.com/sirupsen/logrus"
	"mydocker/container"
	"mydocker/vars"
	"os"
	"os/exec"
	"path"
	"strings"
)

func ExecContainer(containerName string, cmdArray []string) {
	pid, err := getContainerPidByName(containerName)
	if err != nil {
		log.Errorf("Exec container getContainerPidByName %s error %v", containerName, err)
	}
	log.Infof("container pid %s", pid)

	cmdStr := strings.Join(cmdArray, ",")

	// 通过linux的nsenster命令进入进程的namespace
	cmd := exec.Command("nsenter", "-t", pid, "--all", cmdStr)

	// 如果使用了os.setEnv(xxx)，则 --> 不知道这个os是不是容器内的os，待验证
	//cmd.Env = append(os.Environ(), getEnvByPid(pid)...)
	cmd.Env = getEnvByPid(pid)
	os.Environ()
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Println("Error running nsenter command: ", err)
	}
}

func getContainerPidByName(containerName string) (string, error) {
	dirUrl := fmt.Sprintf(vars.DefaultInfoLocation, containerName)
	configFilePath := path.Join(dirUrl, vars.ConfigName)
	contentBytes, err := os.ReadFile(configFilePath)
	if err != nil {
		return "", err
	}

	var containerInfo container.ContainerInfo
	containerInfo = container.ContainerInfo{}
	if err := json.Unmarshal(contentBytes, &containerInfo); err != nil {
		return "", err
	}
	return containerInfo.Pid, nil
}

func getEnvByPid(pid string) []string {
	path := fmt.Sprintf("/proc/%s/environ", pid)
	contentBytes, err := os.ReadFile(path)
	if err != nil {
		log.Errorf("Read file %s error %v", path, err)
		return nil
	}
	// 该文件内的环境变量默认分隔符是\0(NULL字符),unicode表示为\u0000
	envs := strings.Split(string(contentBytes), "\u0000")
	return envs
}
