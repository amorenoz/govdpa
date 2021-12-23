package main

import (
	"fmt"
	"os"

	vdpa "github.com/k8snetworkplumbingwg/govdpa/pkg/kvdpa"
	cli "github.com/urfave/cli/v2"
)

func listAction(c *cli.Context) error {
	devs, err := vdpa.GetVdpaDeviceList()
	if err != nil {
		fmt.Println(err)
	}

	for _, dev := range devs {
		fmt.Printf("%+v\n", dev)
	}
	return nil
}

func getAction(c *cli.Context) error {
	for i := 0; i < c.Args().Len(); i++ {
		pci := c.Args().Get(i)
		dev, err := vdpa.GetVdpaDeviceByPci(pci)
		if err != nil {
			return err
		}
		fmt.Printf("%s : %+v\n", pci, dev)
	}
	return nil
}

func main() {
	app := &cli.App{
		Name:  "kvdpa-cli",
		Usage: "Interact with Kernel vDPA devices",
		Commands: []*cli.Command{
			{Name: "list",
				Usage:  "List all vdpa devices",
				Action: listAction,
			},
			{Name: "get",
				Usage:     "Get a specific vdpa device",
				Action:    getAction,
				ArgsUsage: "[pci addresses]",
			},
		},
	}
	err := app.Run(os.Args)
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}
