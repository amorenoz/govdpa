package main

import (
	"fmt"
	"os"
	"text/template"

	vdpa "github.com/k8snetworkplumbingwg/govdpa/pkg/kvdpa"
	cli "github.com/urfave/cli/v2"
)

const deviceTemplate = ` - Name: {{ .Name }}
   Vdpa device: {{ vdpaDevice . }}
`

func vdpaDeviceWrapper(dev vdpa.VduseDevice) string {
	vdpaDev, err := dev.VdpaDevice()
	if err != nil {
		return fmt.Sprintf("error %v", err)
	}
	return vdpaDev.Name()
}

func listAction(c *cli.Context) error {
	var devs []vdpa.VduseDevice
	var err error
	devs, err = vdpa.ListVduseDevices()
	if err != nil {
		return err
	}

	tmpl := template.Must(template.New("device").Funcs(template.FuncMap{
		"vdpaDevice": vdpaDeviceWrapper,
	}).Parse(deviceTemplate))
	for _, dev := range devs {
		if err := tmpl.Execute(os.Stdout, dev); err != nil {
			panic(err)
		}
	}
	return nil
}

func deleteAction(c *cli.Context) error {
	if c.Args().Len() != 1 {
		err := cli.ShowAppHelp(c)
		return err
	}

	devName := c.Args().Get(0)

	return vdpa.DestroyVduseDevice(devName)
}

func main() {
	app := &cli.App{
		Name:  "vduse-cli",
		Usage: "Interact with Kernel vDPA devices",
		Commands: []*cli.Command{
			{Name: "list",
				Usage:  "List vduse devices",
				Action: listAction,
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "mgmtdev",
						Usage: "Name of the management device: [busName/]devName",
					},
				},
			},
			{Name: "del",
				Usage:     "Delete a vduse device",
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
