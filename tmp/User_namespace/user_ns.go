package main

import (
	"log"
	"os"
	"os/exec"
	"syscall"
)

// 隔离uid 和 gid
func main() {
	cmd := exec.Command("bash")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUTS | syscall.CLONE_NEWIPC | syscall.CLONE_NEWPID | syscall.CLONE_NEWNS | syscall.CLONE_NEWUSER,
	}

	// 添加以下代码会报错 fork/exec /bin/bash: operation not permitted
	//cmd.SysProcAttr.Credential = &syscall.Credential{
	//	Uid:    uint32(0),
	//	Gid:    uint32(0),
	//	Groups: []uint32{0},
	//}

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		log.Fatal(err)
	}

	os.Exit(-1)
}
