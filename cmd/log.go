package cmd

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"io"
	"mydocker/vars"
	"os"
	"path"
)

// 读取指定容器日志文件并全量输出到终端
func LogContainer(containerName string) {
	dirUrl := fmt.Sprintf(vars.DefaultInfoLocation, containerName)
	logFileLocation := path.Join(dirUrl, vars.ContainerLogFile)
	file, err := os.Open(logFileLocation)
	defer file.Close()
	if err != nil {
		log.Errorf("Log container open file %s error %v", logFileLocation, err)
		return
	}
	content, err := io.ReadAll(file)
	if err != nil {
		log.Errorf("Log container read file %s error %v", logFileLocation, err)
	}
	fmt.Fprintln(os.Stdout, string(content))
}
