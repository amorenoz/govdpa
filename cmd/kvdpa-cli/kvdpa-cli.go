package main

import (
	"fmt"
	"os"
	"strings"
	"text/template"

	vdpa "github.com/k8snetworkplumbingwg/govdpa/pkg/kvdpa"
	cli "github.com/urfave/cli/v2"
)

const deviceTemplate = ` - Name: {{ .Name }}
   Management Device: {{ .MgmtDev.Name }}
   Driver: {{ .Driver }}
{{- if eq .Driver "virtio_vdpa" }}
   Virtio Net Device:
      Name: {{ .VirtioNet.Name }}
      NetDev: {{ .VirtioNet.NetDev }}
{{ else if eq .Driver "vhost_vdpa" }}
   Vhost Vdpa Device:
      Name: {{ .VhostVdpa.Name }}
      Path: {{ .VhostVdpa.Path }}
{{ end }}`

func listAction(c *cli.Context) error {
	var devs []vdpa.VdpaDevice
	var err error
	if c.String("mgmtdev") != "" {
		var bus, name string
		nameParts := strings.Split(c.String("mgmtdev"), "/")
		if len(nameParts) == 1 {
			name = nameParts[0]
		} else if len(nameParts) == 2 {
			bus = nameParts[0]
			name = nameParts[1]
		} else {
			return fmt.Errorf("Invalid management device name %s", c.String("mgmtdev"))
		}
		devs, err = vdpa.GetVdpaDevicesByMgmtDev(bus, name)
		if err != nil {
			return err
		}
	} else {
		devs, err = vdpa.ListVdpaDevices()
		if err != nil {
			fmt.Println(err)
		}
	}
	tmpl := template.Must(template.New("device").Parse(deviceTemplate))

	for _, dev := range devs {
		if err := tmpl.Execute(os.Stdout, dev); err != nil {
			panic(err)
		}
	}
	return nil
}

func getAction(c *cli.Context) error {
	tmpl := template.Must(template.New("device").Parse(deviceTemplate))
	for i := 0; i < c.Args().Len(); i++ {
		name := c.Args().Get(i)
		dev, err := vdpa.GetVdpaDevice(name)
		if err != nil {
			return err
		}
		if err := tmpl.Execute(os.Stdout, dev); err != nil {
			panic(err)
		}
	}
	return nil
}

func addAction(c *cli.Context) error {
	if c.Args().Len() != 2 {
		err := cli.ShowAppHelp(c)
		return err
	}

	mgmtDevName := c.Args().Get(0)
	devName := c.Args().Get(1)

	return vdpa.AddVdpaDevice(mgmtDevName, devName)
}

func deleteAction(c *cli.Context) error {
	if c.Args().Len() != 1 {
		err := cli.ShowAppHelp(c)
		return err
	}

	devName := c.Args().Get(0)

	return vdpa.DeleteVdpaDevice(devName)
}

func main() {
	app := &cli.App{
		Name:  "kvdpa-cli",
		Usage: "Interact with Kernel vDPA devices",
		Commands: []*cli.Command{
			{Name: "list",
				Usage:  "List vdpa devices",
				Action: listAction,
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "mgmtdev",
						Usage: "Name of the management device: [busName/]devName",
					},
				},
			},
			{Name: "get",
				Usage:     "Get a specific vdpa device",
				Action:    getAction,
				ArgsUsage: "[name]",
			},
			{Name: "add",
				Usage:     "Add a new vdpa device",
				Action:    addAction,
				ArgsUsage: "[mgmtdev] [dev]",
			},
			{Name: "del",
				Usage:     "Delete a vdpa device",
				Action:    deleteAction,
				ArgsUsage: "[dev]",
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}
