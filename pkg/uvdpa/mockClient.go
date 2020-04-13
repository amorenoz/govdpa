/*
 * Copyright (C) 2020  - All Rights Reserved
 * Author: Adrián Moreno <amorenoz@redhat.com>
 *
 * TBD: Licencing
 */

package uvdpa

import (
	"errors"
)

const (
	version = "0.1"
)

// UserClientimplements UserDaemonStub and connects to the uvdpad:
// https://gitlab.com/mcoquelin/userspace-vdpa/
type mockClient struct {
	list   []VhostIface
	nextID int
}

func (c *mockClient) Init() error {
	return nil
}

func (c *mockClient) Close() error {
	return nil
}

func (c *mockClient) Version() (string, error) {
	return version, nil
}

func (c *mockClient) ListIfaces() ([]VhostIface, error) {
	return c.list, nil
}

func (c *mockClient) Create(v VhostIface) error {
	i, _ := c.find(v.Device)
	if i >= 0 {
		return errors.New("Device already exists")
	}
	v.ID = c.nextID
	c.nextID++
	c.list = append(c.list, v)
	return nil
}

func (c *mockClient) Destroy(dev string) error {
	i, _ := c.find(dev)
	if i < 0 {
		return errors.New("Device does not exist")
	}
	c.list = append(c.list[:i], c.list[i+1:]...)
	return nil
}

// Internal functions (not part of interface)
func (c *mockClient) find(dev string) (int, VhostIface) {
	for i, iface := range c.list {
		if iface.Device == dev {
			return i, iface
		}
	}
	var empty VhostIface
	return -1, empty
}

func newMockClient() UserDaemonStub {
	return &mockClient{
		list:   make([]VhostIface, 0),
		nextID: 0,
	}
}
