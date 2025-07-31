package kvdpa

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	vduseDevDir = "/dev/vduse"
)

// VduseDevice contains information about a VDUSE Device
type VduseDevice interface {
	Name() string
	VdpaDevice() (VdpaDevice, error)
}

// vduseDev implements VduseDevice interface
type vduseDev struct {
	name string
}

// Name returns the Vduse device's name
func (vu *vduseDev) Name() string {
	return vu.name
}

// VdpaDevice returns the vdpa associated with a vduse device if any
func (vu *vduseDev) VdpaDevice() (VdpaDevice, error) {
	devices, err := listVdpaDevicesWithBusDevName("", "vduse")
	if err != nil {
		return nil, err
	}
	for _, dev := range devices {
		// A more reliable way would be to check if /dev/{vu.Name} has
		// the same major/minor number as {dev}.ParentDevicePath()/device.
		// But golang Stat is unable to return device information. Looking
		// at the name seems good enough.
		parent, err := dev.ParentDevicePath()
		if err != nil {
			return nil, err
		}
		if filepath.Base(parent) == vu.name {
			return dev, nil
		}
	}
	return nil, nil
}

// ListVduseDevices returns a list of all available VDUSE devices
func ListVduseDevices() ([]VduseDevice, error) {
	nodes, err := os.ReadDir(vduseDevDir)
	if err != nil {
		return nil, err
	}

	devices := make([]VduseDevice, 0, len(nodes))

	for _, node := range nodes {
		if node.Name() == "control" {
			continue
		}
		dev := &vduseDev{
			name: node.Name(),
		}
		devices = append(devices, dev)
	}
	return devices, nil
}

// GetVduseDevice returns the vduse device with a given name
func GetVduseDevice(name string) (VduseDevice, error) {
	file := filepath.Join(vduseDevDir, name)
	if _, err := os.Stat(file); os.IsNotExist(err) {
		return nil, fmt.Errorf("vduse device %s does not exist", name)
	}
	return &vduseDev{
		name: name,
	}, nil
}
