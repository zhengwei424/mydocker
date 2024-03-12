package main

import (
	"log"
	"os"
	"os/exec"
	"syscall"
)

// 隔离System V IPC和POSIX message queues
// ipcxx相关命令 ipcs ipcmk ipcrm
func main() {
	cmd := exec.Command("bash")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUTS | syscall.CLONE_NEWIPC,
	}
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		log.Fatal(err)
	}
}
