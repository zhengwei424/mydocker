package container

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"mydocker/vars"
	"os"
	"os/exec"
	"path"
	"strings"
)

// create a overlay filesystem as container root workspace
func NewWorkSpace(volume, imageName, containerName string) {
	CreateLowerDir(imageName, containerName)
	CreateUpperDir(containerName)
	CreateWorkDir(containerName)
	CreateMountPoint(containerName, imageName)
	if volume != "" {
		volumePaths := volumePathExtract(volume)
		length := len(volumePaths)
		if length == 2 && volumePaths[0] != "" && volumePaths[1] != "" {
			MountVolume(containerName, volumePaths)
			log.Infof("volumePaths: %v", volumePaths)
		} else {
			log.Errorf("volume parameters input is not correct.")
		}
	}
}

func CreateLowerDir(imageName, containerName string) {
	lowerdirPath := path.Join(fmt.Sprintf(vars.LowerDir, containerName), imageName)
	busyboxtarPath := path.Join(vars.ImagesDir, imageName+".tar")
	exist, err := PathExists(lowerdirPath)
	if err != nil {
		log.Infof("Fail to jude whether %s exists. %v", lowerdirPath, err)
	}
	if !exist {
		if err := os.MkdirAll(lowerdirPath, 0755); err != nil {
			log.Errorf("Mkdir %s error: %v", lowerdirPath, err)
		}

		if _, err := exec.Command("tar", "-xvf", busyboxtarPath, "-C", lowerdirPath).CombinedOutput(); err != nil {
			log.Errorf("Untar %s error: %v", lowerdirPath, err)
		}
	}
}

func CreateUpperDir(containerName string) {
	upperdirPath := fmt.Sprintf(vars.UpperDir, containerName)
	if err := os.MkdirAll(upperdirPath, 0777); err != nil {
		log.Errorf("Mkdir %s error: %v", upperdirPath, err)
	}
}

func CreateWorkDir(containerName string) {
	workdirPath := fmt.Sprintf(vars.WorkDir, containerName)
	if err := os.MkdirAll(workdirPath, 0777); err != nil {
		log.Errorf("Mkdir %s error: %v", workdirPath, err)
	}
}

func CreateMountPoint(containerName, imageName string) {
	mntPath := fmt.Sprintf(vars.MntDir, containerName)
	if err := os.MkdirAll(mntPath, 0777); err != nil {
		log.Errorf("mkdir %s error. %v", mntPath, err)
	}

	/*
		在 overlay 文件系统中，lowerdir、upperdir、workdir 和 mntdir 是四个关键的目录，各自有不同的作用：

		lowerdir（底层目录）：
		Lower 层是 OverlayFS 的基础层，包含了底层的目录结构和文件内容。
		Lower 层中的文件和目录是只读的，不能进行修改。
		Lower 层中的内容会被 Upper 层中的内容覆盖和修改。

		upperdir（覆盖层目录）：
		Upper 层包含了对 Lower 层的修改和新增的文件。
		Upper 层中的文件和目录是可写的，可以进行修改和新增。
		Upper 层中的内容会覆盖 Lower 层中相同路径下的文件。

		workdir（工作目录）：
		Work 层是 OverlayFS 内部使用的临时工作目录，用来存放临时文件和修改的地方。
		在进行写操作时，OverlayFS 会在 Work 层中进行临时修改，然后再将修改同步到 Upper 层。
		Work 层在操作完成后会被清空，用于下一次写操作。

		mntdir（挂载目录）：
		mntdir 是 overlay 文件系统挂载的目标目录，也就是我们在系统中看到的最终的虚拟文件系统。mntdir 实际上是 lowerdir 和 upperdir 的合并视图，用户可以通过 mntdir 来访问 overlay 文件系统提供的文件和目录。
	*/

	mountOptions := "lowerdir=" + path.Join(fmt.Sprintf(vars.LowerDir, containerName), imageName) + ",upperdir=" + fmt.Sprintf(vars.UpperDir, containerName) + ",workdir=" + fmt.Sprintf(vars.WorkDir, containerName)
	cmd := exec.Command("mount", "-t", "overlay", "overlay", "-o", mountOptions, mntPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Errorf("CreateMountPoint error: %v", err)
	}
}

// volume 挂载信息提取
func volumePathExtract(volume string) []string {
	var volumePaths []string
	volumePaths = strings.Split(volume, ":")
	return volumePaths
}

func MountVolume(containerName string, volumePaths []string) {
	hostPath := volumePaths[0]
	exist, _ := PathExists(hostPath)
	if !exist {
		if err := os.MkdirAll(hostPath, 0777); err != nil {
			log.Errorf("Mkdir hostPath %s error: %v", hostPath, err)
		}
	}

	containerVolumePath := fmt.Sprintf(vars.MntDir, containerName) + volumePaths[1]

	exist, _ = PathExists(containerVolumePath)
	if !exist {
		if err := os.Mkdir(containerVolumePath, 0777); err != nil {
			log.Errorf("Mkdir containerVolumePath %s error: %v", containerVolumePath, err)
		}
	}

	cmd := exec.Command("mount", "-t", "bind", "-o", "rbind", hostPath, containerVolumePath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Errorf("Mount volume %s error: %v", containerVolumePath, err)
	} else {
		log.Infof("Mount volume %s success", containerVolumePath)
	}
}

// delete the overlay filesystem while container exit
func DeleteWorkSpace(containerName, volume string) {
	if volume != "" {
		volumePaths := volumePathExtract(volume)
		length := len(volumePaths)
		if length == 2 && volumePaths[0] != "" && volumePaths[1] != "" {
			DeleteVolumeMountPoint(containerName, volumePaths)
		}
	}
	DeleteMountPoint(containerName)
	DeleteWorkDir(containerName)
	DeleteUpperDir(containerName)
	DeleteLowerDir(containerName)
}

func DeleteMountPoint(containerName string) {
	mntPath := fmt.Sprintf(vars.MntDir, containerName)
	//mountPoints := []string{"/proc", "/dev", mntPath}
	//for _, item := range mountPoints {
	//	if item == mntPath {
	//		time.Sleep(5 * time.Second)
	//	}
	//	err := syscall.Unmount(item, syscall.MNT_FORCE)
	//	if err != nil {
	//		log.Errorf("umount %s error: %v", item, err)
	//	}
	//}

	cmd := exec.Command("umount", mntPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Errorf("%v", err)
	}
	if err := os.RemoveAll(mntPath); err != nil {
		log.Errorf("Remove %s error: %v", mntPath, err)
	}
}

func DeleteVolumeMountPoint(containerName string, volume []string) {
	mntPath := fmt.Sprintf(vars.MntDir, containerName)
	containerPath := mntPath + volume[1]
	cmd := exec.Command("umount", containerPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Errorf("umount %s error: %v", containerPath, err)
	}
}

func DeleteWorkDir(containerName string) {
	workdirPath := fmt.Sprintf(vars.WorkDir, containerName)
	if err := os.RemoveAll(workdirPath); err != nil {
		log.Errorf("Remove %s error: %v", workdirPath, err)
	}
}

func DeleteUpperDir(containerName string) {
	upperdirPath := fmt.Sprintf(vars.UpperDir, containerName)
	if err := os.RemoveAll(upperdirPath); err != nil {
		log.Errorf("Remove %s error: %v", upperdirPath, err)
	}
}

func DeleteLowerDir(containerName string) {
	lowerdirPath := fmt.Sprintf(vars.LowerDir, containerName)
	if err := os.RemoveAll(lowerdirPath); err != nil {
		log.Errorf("Remove %s error: %v", lowerdirPath, err)
	}
}

func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
