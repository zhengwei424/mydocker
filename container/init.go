package container

import (
	"fmt"
	"github.com/moby/sys/mount"
	"github.com/moby/sys/mountinfo"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
	"io/ioutil"
	"mydocker/utils"
	"mydocker/vars"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

func RunContainerInitProcess(containerName string) error {
	commandArray := ReadUserCommand()
	if commandArray == nil || len(commandArray) == 0 {
		return fmt.Errorf("Run container get user command error, commandArray is nil")
	}

	setupMount(containerName)

	// 在PATH环境变量内搜索commandArray[0]，并返回绝对路径或者时一个相对于当前目录的相对路径
	path, err := exec.LookPath(commandArray[0])
	if err != nil {
		log.Errorf("Exec loop path error %v", err)
		return err
	}
	log.Infof("Find path %s", path)
	if err := syscall.Exec(path, commandArray, os.Environ()); err != nil {
		log.Errorf(err.Error())
	}
	return nil
}

func ReadUserCommand() []string {
	pipe := os.NewFile(uintptr(3), "pipe")
	defer pipe.Close()
	msg, err := ioutil.ReadAll(pipe)
	if err != nil {
		log.Errorf("init read pipe error %v", err)
		return nil
	}
	msgStr := string(msg)
	return strings.Split(msgStr, " ")
}

func setupMount(containerName string) {
	mnt := fmt.Sprintf(vars.MntDir, containerName)
	chroot(mnt)

	// mount proc
	defaultMountFlags := syscall.MS_NOEXEC | syscall.MS_NOSUID | syscall.MS_NODEV
	err := syscall.Mount("proc", "/proc", "proc", uintptr(defaultMountFlags), "")
	if err != nil {
		log.Errorf("mount proc error: %v", err)
	}
	err = syscall.Mount("tmpfs", "/dev", "tmpfs", syscall.MS_NOSUID|syscall.MS_STRICTATIME, "mode=755")
	if err != nil {
		log.Errorf("mount dev error: %v", err)
	}
}

// 切换根目录（直接引用docker-ce源码，是最重要最核心的一段代码！！！！！！！！！！！)
func chroot(path string) (err error) {
	// if the engine is running in a user namespace we need to use actual chroot
	if utils.RunningInUserNS() {
		return realChroot(path)
	}
	if err := unix.Unshare(unix.CLONE_NEWNS); err != nil {
		return fmt.Errorf("Error creating mount namespace before pivot: %v", err)
	}

	// Make everything in new ns slave.
	// Don't use `private` here as this could race where the mountns gets a
	//   reference to a mount and an unmount from the host does not propagate,
	//   which could potentially cause transient errors for other operations,
	//   even though this should be relatively small window here `slave` should
	//   not cause any problems.
	if err := mount.MakeRSlave("/"); err != nil {
		return err
	}

	if mounted, _ := mountinfo.Mounted(path); !mounted {
		//if err := mount.Mount(path, path, "bind", "rbind,rw"); err != nil {
		//	return realChroot(path)
		//}
		if err := syscall.Mount(path, path, "bind", syscall.MS_BIND|syscall.MS_REC, ""); err != nil {
			return realChroot(path)
		}
	}

	// setup oldRoot for pivot_root
	pivotDir, err := os.MkdirTemp(path, ".pivot_root")
	if err != nil {
		return fmt.Errorf("Error setting up pivot dir: %v", err)
	}

	var mounted bool
	defer func() {
		if mounted {
			// make sure pivotDir is not mounted before we try to remove it
			if errCleanup := unix.Unmount(pivotDir, unix.MNT_DETACH); errCleanup != nil {
				if err == nil {
					err = errCleanup
				}
				return
			}
		}

		errCleanup := os.Remove(pivotDir)
		// pivotDir doesn't exist if pivot_root failed and chroot+chdir was successful
		// because we already cleaned it up on failed pivot_root
		if errCleanup != nil && !os.IsNotExist(errCleanup) {
			errCleanup = fmt.Errorf("Error cleaning up after pivot: %v", errCleanup)
			if err == nil {
				err = errCleanup
			}
		}
	}()

	if err := unix.PivotRoot(path, pivotDir); err != nil {
		// If pivot fails, fall back to the normal chroot after cleaning up temp dir
		if err := os.Remove(pivotDir); err != nil {
			return fmt.Errorf("Error cleaning up after failed pivot: %v", err)
		}
		return realChroot(path)
	}
	mounted = true

	// This is the new path for where the old root (prior to the pivot) has been moved to
	// This dir contains the rootfs of the caller, which we need to remove so it is not visible during extraction
	pivotDir = filepath.Join("/", filepath.Base(pivotDir))

	if err := unix.Chdir("/"); err != nil {
		return fmt.Errorf("Error changing to new root: %v", err)
	}

	// Make the pivotDir (where the old root lives) private so it can be unmounted without propagating to the host
	if err := unix.Mount("", pivotDir, "", unix.MS_PRIVATE|unix.MS_REC, ""); err != nil {
		return fmt.Errorf("Error making old root private after pivot: %v", err)
	}

	// Now unmount the old root so it's no longer visible from the new root
	if err := unix.Unmount(pivotDir, unix.MNT_DETACH); err != nil {
		return fmt.Errorf("Error while unmounting old root after pivot: %v", err)
	}
	mounted = false

	return nil
}

func realChroot(path string) error {
	if err := unix.Chroot(path); err != nil {
		return fmt.Errorf("Error after fallback to chroot: %v", err)
	}
	if err := unix.Chdir("/"); err != nil {
		return fmt.Errorf("Error changing to new root after chroot: %v", err)
	}
	return nil
}
