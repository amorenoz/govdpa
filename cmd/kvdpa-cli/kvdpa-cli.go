package main

import (
	"fmt"
	"os"
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
		bus, name := vdpa.SplitMgmtDevName(c.String("mgmtdev"))
		if name == "" {
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
		},
	}
	err := app.Run(os.Args)
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}
