package kvdpa

import (
	"os"
)

// Private constants
const (
	vduseDevDir = "/dev/vduse"
)

// VduseDevice contains information about a VDUSE Device
type VduseDevice interface {
	Name() string
}

// vduseDev implements VduseDevice interface
type vduseDev struct {
	name string
}

// Name returns the Vduse device's name
func (vu *vduseDev) Name() string {
	return vu.name
}

/*ListVduseDevices returns a list of all available VDUSE devices */
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
