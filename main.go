package main

import (
	"fmt"
	"github.com/urfave/cli/v2"
	"os"
	"tunnel-transporter/client"
	"tunnel-transporter/config"
)

func main() {
	configFileFlag := &cli.StringFlag{
		Name:    "file",
		Aliases: []string{"f"},
		Usage:   "Load yaml format configuration file from `FILE`",
		EnvVars: []string{"TT_CONFIG_FILE"},
		Value:   "./tunnel-config.yaml",
	}

	app := &cli.App{
		Name:    "tunnel-transporter",
		Usage:   "exposing proxy connections from public connections to local connections",
		Version: "1.0",
		Commands: []*cli.Command{
			{
				Name:        "server",
				Description: "server mode, deploy and run in public network",
				Category:    "mode",
				Flags:       []cli.Flag{configFileFlag},
				Action: func(context *cli.Context) error {
					if err := config.ParseConfig(context.String("file")); err != nil {
						return err
					}
					client.StartServer()
					return nil
				},
			},
			{
				Name:        "agent",
				Description: "agent mode, deploy and run in local network",
				Category:    "mode",
				Flags:       []cli.Flag{configFileFlag},
				Action: func(context *cli.Context) error {
					if err := config.ParseConfig(context.String("file")); err != nil {
						return err
					}
					client.StartAgent()
					return nil
				},
			},
		},
		CommandNotFound: func(context *cli.Context, s string) {
			fmt.Printf("command '%s' not found\n", s)
			os.Exit(-1)
		},
		Authors: []*cli.Author{{
			Name:  "zhouzhixuan",
			Email: "531149907@qq.com",
		}},
	}

	err := app.Run(os.Args)
	if err != nil {
		fmt.Printf("error running tunnel-transporter, reason: %s", err)
		return
	}
}
