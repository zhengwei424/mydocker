package cmd

import (
	log "github.com/sirupsen/logrus"
	"mydocker/vars"
	"os/exec"
	"path"
)

func CommitContainer(containerName, imageName string) {
	mntPath := path.Join(vars.MntDir, containerName)
	imageTarPath := path.Join(vars.ImagesDir, imageName+".tar")
	if _, err := exec.Command("tar", "-zcf", imageTarPath, "-C", mntPath, ".").CombinedOutput(); err != nil {
		log.Errorf("commit %s error: %v", containerName, err)
	}
}
