/*
 * Copyright (C) 2020  - All Rights Reserved
 * Author: Adrián Moreno <amorenoz@redhat.com>
 *
 * TBD: Licencing
 */

package uvdpa

import (
	jsonrpc "github.com/amorenoz/govdpa/pkg/internal/jsonrpc"
	"net/rpc"
	"sync"
)

const (
	daemonSocketFile = "/var/run/uvdpa/uvdpad.sock"
)

var (
	instance userClient
	once     sync.Once
)

// UserClientimplements UserDaemonStub and connects to the uvdpad:
// https://gitlab.com/mcoquelin/userspace-vdpa/
type userClient struct {
	client *rpc.Client
}

func (c *userClient) Init() error {
	var err error
	c.client, err = jsonrpc.Dial("unix", daemonSocketFile)
	if err != nil {
		return err
	}
	return nil
}

func (c *userClient) Close() error {
	return c.client.Close()
}

func (c *userClient) Version() (string, error) {
	var version string
	err := c.client.Call("version", nil, &version)
	if err != nil {
		return "", err
	}
	return version, nil
}

func (c *userClient) ListIfaces() ([]VhostIface, error) {
	var res []VhostIface
	err := c.client.Call("list-interfaces", nil, &res)
	return res, err
}

func (c *userClient) Create(v VhostIface) error {
	err := c.client.Call("create-interface", &v, nil)
	return err
}

func (c *userClient) Destroy(dev string) error {
	arg := VhostIface{
		Device: dev,
	}
	err := c.client.Call("destroy-interface", &arg, nil)
	return err
}

func newDpdkClient() UserDaemonStub {
	once.Do(func() {
		instance = userClient{}
	})
	return &instance
}
