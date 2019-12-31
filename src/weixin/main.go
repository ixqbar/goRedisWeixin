package main

import (
	"fmt"
	"github.com/urfave/cli"
	"os"
	"weixin/common"
	"weixin/core"
)

func main()  {
	app := cli.NewApp()
	app.Name = "chat"
	app.Version = common.VERSION
	app.Description = "weixin token & ticket api server"
	app.Usage = ""

	app.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:  "config,c",
			Usage: "load configuration from `FILE`",
		},
	}

	app.Action = func(c *cli.Context) error {
		configFile := c.String("config")
		if len(configFile) == 0 {
			cli.ShowAppHelp(c)
			return nil
		}

		common.Logger.Printf("run with config file %s", configFile)

		if _, err := common.ParseConfig(configFile); err != nil {
			return fmt.Errorf("not found config file %v", configFile)
		}

		return core.Run()
	}

	err := app.Run(os.Args)
	if err != nil {
		common.Logger.Print(err)
	}
}
