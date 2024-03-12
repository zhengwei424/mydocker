package main

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"mydocker/cgroups/subsystems"
	mycli "mydocker/cmd"
	"mydocker/container"
	"mydocker/network"
)

// Flags的作用类似于运行命令时使用--来指定参数
var runCommand = cli.Command{
	Name:  "run",
	Usage: `Create a container with namespace and cgroups limit mydocker run -ti [command]`,
	Flags: []cli.Flag{
		// 交互模式
		cli.BoolFlag{
			Name:  "ti",
			Usage: "enable tty",
		},
		// 后台运行
		cli.BoolFlag{
			Name:  "d",
			Usage: "detach container",
		},
		// 设置挂载
		cli.StringFlag{
			Name:  "v",
			Usage: "volume",
		},
		// 设置内存
		cli.StringFlag{
			Name:  "m",
			Usage: "memory limit",
		},
		cli.StringFlag{
			Name:  "cpushare",
			Usage: "cpushare limit",
		},
		cli.StringFlag{
			Name:  "cpuset",
			Usage: "cpuset limit",
		},
		// 设置容器名
		cli.StringFlag{
			Name:  "name",
			Usage: "container name",
		},
		// 设置环境变量
		cli.StringSliceFlag{
			Name:  "e",
			Usage: "set environment",
		},
		// 设置网络
		cli.StringFlag{
			Name:  "net",
			Usage: "container network",
		},
		// 设置端口映射
		cli.StringSliceFlag{
			Name:  "p",
			Usage: "port mapping",
		},
	},
	/*
		这里是run命令执行的真正函数。
		1. 判断参数是否包含command
		2. 获取用户指定的command
		3. 调用Run function去准备启动容器
	*/

	// cmd.Context 用于检索args，解析命令行的options
	Action: func(context *cli.Context) error {
		// 非flag会被归到args！！！！！
		if len(context.Args()) < 1 {
			return fmt.Errorf("Missing container command")
		}
		var commandArray []string
		for _, arg := range context.Args() {
			commandArray = append(commandArray, arg)
		}

		resConf := &subsystems.ResourceConfig{
			MemoryLimit: context.String("m"), // 能够获取参数对应的值
			CpuSet:      context.String("cpuset"),
			CpuShare:    context.String("cpushare"),
		}

		createTty := context.Bool("ti")
		volume := context.String("v")
		detach := context.Bool("d")
		containerName := context.String("name")
		env := context.StringSlice("e")
		networkName := context.String("net")
		portMapping := context.StringSlice("p")

		if createTty && detach {
			return fmt.Errorf("ti and d parameter can not both provided")
		}

		log.Infof("createTty %v", createTty)
		mycli.Run(createTty, commandArray, resConf, volume, containerName, env, networkName, portMapping)
		return nil
	},
}

var initCommand = cli.Command{
	Name:  "init",
	Usage: "Init container process run user's process in container. Do not call it outside",
	/*
		1. 获取传递过来的command参数
		2. 执行容器初始化操作
	*/

	// cmd.Context 用于检索args，解析命令行的options
	Action: func(context *cli.Context) error {
		log.Infof("initing...")
		err := container.RunContainerInitProcess(context.Args().Get(0))
		return err
	},
}

var commitCommand = cli.Command{
	Name:  "commit",
	Usage: "commit <containerName> <imageName>",
	Action: func(context *cli.Context) error {
		if len(context.Args()) < 1 {
			return fmt.Errorf("missing container name")
		}
		containerName := context.Args().Get(0)
		imageName := context.Args().Get(1)
		mycli.CommitContainer(containerName, imageName)
		return nil
	},
}

var listCommand = cli.Command{
	Name:  "ps",
	Usage: "list all the containers",
	Action: func(context *cli.Context) error {
		mycli.ListContainers()
		return nil
	},
}

var logCommand = cli.Command{
	Name:  "logs",
	Usage: "print logs of a container",
	Action: func(context *cli.Context) error {
		if len(context.Args()) < 1 {
			return fmt.Errorf("Please input your container name")
		}
		containerName := context.Args().Get(0)
		mycli.LogContainer(containerName)
		return nil
	},
}

var execCommand = cli.Command{
	Name:  "exec",
	Usage: "exec a command into container",
	Action: func(context *cli.Context) error {

		if len(context.Args()) < 2 {
			return fmt.Errorf("Missing container name or command")
		}

		containerName := context.Args().Get(0)

		var commandArray []string

		for _, arg := range context.Args().Tail() {
			commandArray = append(commandArray, arg)
		}

		mycli.ExecContainer(containerName, commandArray)
		return nil
	},
}

var stopCommand = cli.Command{
	Name:  "stop",
	Usage: "stop a container",
	Action: func(context *cli.Context) error {
		if len(context.Args()) < 1 {
			return fmt.Errorf("Missing container name")
		}
		containerName := context.Args().Get(0)
		mycli.StopContainer(containerName)
		return nil
	},
}

var removeCommand = cli.Command{
	Name:  "rm",
	Usage: "remove container",
	Action: func(context *cli.Context) error {
		if len(context.Args()) < 1 {
			return fmt.Errorf("Missing container name")
		}
		containerName := context.Args().Get(0)
		mycli.RemoveContainer(containerName)
		return nil
	},
}

var networkCommand = cli.Command{
	Name:  "network",
	Usage: "container network commands",
	Subcommands: []cli.Command{
		{
			Name:  "create",
			Usage: "create a container network",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "driver",
					Usage: "network driver",
				},
				cli.StringFlag{
					Name:  "subnet",
					Usage: "subnet cidr",
				},
			},
			Action: func(context *cli.Context) error {
				if len(context.Args()) < 1 {
					return fmt.Errorf("Missing network name")
				}

				network.Init()
				err := network.CreateNetwork(context.String("driver"), context.String("subnet"), context.Args()[0])
				if err != nil {
					return fmt.Errorf("create network error: %v", err)
				}
				return nil
			},
		},
		{
			Name:  "list",
			Usage: "list container network",
			Action: func(context *cli.Context) error {
				network.Init()
				network.ListNetwork()
				return nil
			},
		},
		{
			Name:  "remove",
			Usage: "remove container network",
			Action: func(context *cli.Context) error {
				if len(context.Args()) < 1 {
					return fmt.Errorf("Missing network name")
				}
				network.Init()
				err := network.DeleteNetwork(context.Args()[0])
				if err != nil {
					return fmt.Errorf("remove network error: %v", err)
				}
				return nil
			},
		},
	},
}
