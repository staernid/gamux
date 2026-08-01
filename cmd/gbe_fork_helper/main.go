package main

import (
	"log"
	"log/slog"
	"os"

	"gbe_fork_helper/config"
	"gbe_fork_helper/gbe"
	"gbe_fork_helper/github"
	"gbe_fork_helper/tui"

	"github.com/urfave/cli/v2"
)

// Version of the gbe_fork_helper application
const Version = "v0.1.4"

func main() {
	// Initialize structured logging
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	app := &cli.App{
		Name:    "gbe_fork_helper",
		Usage:   "Streamline the management of your gbe_fork installation",
		Version: Version,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "config",
				Usage: "Load configuration from `FILE`",
			},
		},
		Before: func(c *cli.Context) error {
			return config.InitConfig(c.String("config"))
		},
		Commands: []*cli.Command{
			{
				Name:      "apply",
				Usage:     "Apply GBE to Steam API files and configure DLCs",
				ArgsUsage: "<platform> <appid>",
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:  "dry-run",
						Usage: "Do not move or write files, just show what would be done",
					},
				},
				Action: func(c *cli.Context) error {
					if c.Args().Len() < 2 {
						return cli.Exit("Usage: apply <platform> <appid>", 1)
					}
					platform := c.Args().Get(0)
					appID := c.Args().Get(1)
					dryRun := c.Bool("dry-run")
					return gbe.ApplyGBE(c.Context, platform, appID, dryRun)
				},
			},
			{
				Name:  "update",
				Usage: "Update the GBE fork repository",
				Action: func(c *cli.Context) error {
					return github.UpdateGBE(c.Context)
				},
			},
			{
				Name:  "tui",
				Usage: "Start the terminal user interface",
				Action: func(c *cli.Context) error {
					return tui.Run()
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
