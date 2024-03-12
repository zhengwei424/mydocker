package cmd

import (
	"encoding/json"
	"fmt"
	log "github.com/sirupsen/logrus"
	"mydocker/container"
	"mydocker/vars"
	"os"
	"path"
	"text/tabwriter"
)

func ListContainers() {
	containers, err := os.ReadDir(vars.ContainersRootPath)
	if err != nil {
		log.Errorf("Read %s error: %v", vars.ContainersRootPath, err)
		return
	}

	var containersInfo []*container.ContainerInfo
	for _, c := range containers {
		tmpContainerInfo, err := getContainerInfo(c.Name())
		if err != nil {
			log.Errorf("Get container info error %v", err)
			continue
		}
		containersInfo = append(containersInfo, tmpContainerInfo)
	}
	w := tabwriter.NewWriter(os.Stdout, 12, 1, 3, ' ', 0)
	fmt.Fprintf(w, "ID\tNAME\tPID\tSTATUS\tCOMMAND\tCREATED\n")

	for _, item := range containersInfo {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			item.Id,
			item.Name,
			item.Pid,
			item.Status,
			item.Command,
			item.CreatedTime,
		)
	}
	if err := w.Flush(); err != nil {
		log.Errorf("Flush error: %v", err)
		return
	}

}

func getContainerInfo(containerName string) (*container.ContainerInfo, error) {
	containerInfo := new(container.ContainerInfo)

	configFile := path.Join(fmt.Sprintf(vars.DefaultInfoLocation, containerName), vars.ConfigName)
	content, err := os.ReadFile(configFile)
	if err != nil {
		log.Errorf("Read file %s error: %v", configFile, err)
		return nil, err
	}

	if err := json.Unmarshal(content, containerInfo); err != nil {
		log.Errorf("Json unmarshal error: %v", err)
		return nil, err
	}

	return containerInfo, nil
}
