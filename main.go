package main

import (
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"os"
)

const usage = `mydocker is a simple container runtime implementation.
               The purpose of this project is to learn how docker works and how to write a docker by ourselves.
               Enjoy it, just for fun.`

func main() {
	app := cli.NewApp()
	app.Name = "mydocker"
	app.Usage = usage

	app.Commands = []cli.Command{
		initCommand,
		runCommand,
		commitCommand,
		listCommand,
		logCommand,
		execCommand,
		stopCommand,
		removeCommand,
		networkCommand,
	}

	// cmd.Context 用于检索args，解析命令行的options
	app.Before = func(context *cli.Context) error {
		// 设置日志格式为json
		log.SetFormatter(&log.JSONFormatter{})
		log.SetOutput(os.Stdout)
		return nil
	}
	//os.Args = []string{"/tmp/docker/mydocker", "run", "-ti", "-name", "test", "busybox", "/bin/sh"}
	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}

}
