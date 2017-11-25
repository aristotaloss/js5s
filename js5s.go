package main

import (
	"github.com/Velocity-/gorune"
	"github.com/urfave/cli"
	"os"
	"fmt"
)

func main() {
	c := cli.NewApp()

	c.Description = "RuneScape file server application (Js5 protocol)"
	c.Author = "Bart Pelle, OS-Scape"
	c.Name = "js5s"

	c.Commands = []cli.Command{
		{
			Name:        "run",
			ShortName:   "r",
			Description: "Starts the js5 server",
			Action:      startServer,
			Flags: []cli.Flag{
				cli.StringFlag{
					Name: "filestore,fs",
				},
				cli.IntFlag{
					Name: "revision,rev",
				},
			},
		},
	}

	c.Run(os.Args)
}

func startServer(c *cli.Context) {
	// Validate parameters passed..
	if !c.IsSet("filestore") {
		panic("'filestore' argument is required")
	} else if !c.IsSet("revision") {
		panic("'revision' argument is required")
	}

	// Load the file system and scan for all indices in the folder
	fs, err := gorune.Load(c.String("filestore"), true)
	if err != nil {
		panic(err)
	}

	revision := c.Int("revision")
	if revision <= 0 {
		panic(fmt.Errorf("invalid revision %d, must be a positive number", revision))
	}

	StartServer(fs, revision)
}
