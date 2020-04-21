/*
 * Copyright (C) 2020  - All Rights Reserved
 * Author: Adrián Moreno <amorenoz@redhat.com>
 *
 * TBD: Licencing
 */

package main

import (
	"bufio"
	"fmt"
	vdpa "github.com/amorenoz/govdpa/pkg/uvdpa"
	"os"
	"strings"
)

var client vdpa.UserDaemonStub

func version() {
	ver, err := client.Version()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Command Error\n")
		fmt.Fprintln(os.Stderr, err)
		return
	}
	fmt.Printf("Version: %s\n", ver)
}

func list() {
	ifaceList, err := client.ListIfaces()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Command Error\n")
		fmt.Fprintln(os.Stderr, err)
		return
	}
	if len(ifaceList) == 0 {
		fmt.Printf("Empty\n")
		return
	}
	for _, iface := range ifaceList {
		fmt.Printf("device: %s \t socket: %s \t mode: %s \t driver: %s\n",
			iface.Device, iface.Socket, iface.Mode, iface.Driver)
	}
}

func create(device string, socket string, mode string) {
	var iface vdpa.VhostIface
	iface.Device = device
	iface.Socket = socket
	iface.Mode = mode

	err := client.Create(iface)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Command Error\n")
		fmt.Fprintln(os.Stderr, err)
		return
	}
	fmt.Printf("Success\n")
}

func destroy(device string) {
	err := client.Destroy(device)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Command Error\n")
		fmt.Fprintln(os.Stderr, err)
		return
	}
	fmt.Printf("Success\n")
}

func runCmdStr(cmd string) {
	cmd = strings.TrimSuffix(cmd, "\n")
	cmdArr := strings.Fields(cmd)
	runCmd(cmdArr)
}

func runCmd(cmdArr []string) {
	var err error
	if len(cmdArr) == 0 {
		return
	}

	switch cmdArr[0] {
	case "exit":
		err = client.Close()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Close() Error\n")
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		os.Exit(0)
	case "help":
		fmt.Println("Commands")
		fmt.Println("help                               Print this help")
		fmt.Println("exit                               Quit program")
		fmt.Println("version                            Print daemon version")
		fmt.Println("list                               List configured interfaces")
		fmt.Println("create [DEVICE] [SOCKET] [MODE]    Create interface")
		fmt.Println("	    DEVICE: A device PCI address, e.g: 00:0f:01")
		fmt.Println("	    SOCKET: A socketfile path, e.g: /tmp/vdpa1.sock")
		fmt.Println("	    MODE: [client|server]")
		fmt.Println("destroy [DEVICE]                   Destroy a device")
		fmt.Println("	    DEVICE: A device PCI address, e.g: 00:0f:01")

	case "version":
		version()
	case "list":
		list()
	case "create":
		if len(cmdArr) != 4 {
			fmt.Fprintf(os.Stderr, "Invalid arguments\n")
		} else {
			create(cmdArr[1], cmdArr[2], cmdArr[3])
		}
	case "destroy":
		if len(cmdArr) != 2 {
			fmt.Fprintf(os.Stderr, "Invalid arguments\n")
		} else {
			destroy(cmdArr[1])
		}
	default:
		fmt.Println("Unknown command")

	}
}
func main() {
	var err error
	client = vdpa.NewVdpaClient(false)

	err = client.Init()
	if err != nil {
		fmt.Printf("Init() failed\n")
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("VdpaClient Initialized\n")

	if len(os.Args) > 1 {
		runCmd(os.Args[1:])
		os.Exit(0)
	}

	fmt.Printf("Staring vdpa cli (type \"help\" to list the available commands)\n")
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("[vdpacli] $ ")

		cmd, err := reader.ReadString('\n')
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			continue
		}
		runCmdStr(cmd)
	}
}
