package cmd

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"mydocker/container"
	"mydocker/vars"
	"os"
)

func RemoveContainer(containerName string) {
	containerInfo, err := getContainerInfo(containerName)
	if err != nil {
		log.Errorf("Get container %s info error %v", containerName, err)
		return
	}

	if containerInfo.Status == vars.RUNNING {
		log.Errorf("Could not remove running container")
	}

	// 移除挂载
	container.DeleteWorkSpace(containerName, containerInfo.Volume)

	dirUrl := fmt.Sprintf(vars.DefaultInfoLocation, containerName)
	// 容器的运行目录都应该放在这个目录下面，这样就能完全清理干净。涉及overlay文件系统的umount和remove
	if err := os.RemoveAll(dirUrl); err != nil {
		log.Errorf("Remove file %s error %v", dirUrl, err)
		return
	}

}
