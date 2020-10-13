package kvdpa

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	vdpaBusDevDir = "/sys/bus/vdpa/devices"
	pciBusDevDir  = "/sys/bus/pci/devices"

	vdpaDriverVhost = "vhost_vdpa"
	vdpaVhostDevDir = "/dev"

	vdpaDriverVirtio = "virtio_vdpa"
)

/*VdpaDevice contains information about a Vdpa Device*/
type VdpaDevice struct {
	name   string
	driver string
	path   string // Path of the vhost or virtio device
}

/*GetVdpaDeviceList returns a list of all available vdpa devices */
func GetVdpaDeviceList() ([]VdpaDevice, error) {
	vdpaDevList := make([]VdpaDevice, 0)
	fd, err := os.Open(vdpaBusDevDir)
	if err != nil {
		return nil, err
	}
	defer fd.Close()

	fileInfos, err := fd.Readdir(-1)
	if err != nil {
		return nil, err
	}
	var errors []string
	for _, file := range fileInfos {
		if vdpaDev, err := GetVdpaDeviceByName(file.Name()); err != nil {
			errors = append(errors, err.Error())
		} else {
			vdpaDevList = append(vdpaDevList, *vdpaDev)
		}
	}

	if len(errors) > 0 {
		return vdpaDevList, fmt.Errorf(strings.Join(errors, ";"))
	}
	return vdpaDevList, nil
}

/*GetVdpaDeviceByName returns the vdpa device information by a vdpa device name */
func GetVdpaDeviceByName(name string) (*VdpaDevice, error) {
	var err error
	var path string

	driverLink, err := os.Readlink(filepath.Join(vdpaBusDevDir, name, "driver"))
	if err != nil {
		return nil, err
	}

	driver := filepath.Base(driverLink)
	switch driver {
	case vdpaDriverVhost:
		path, err = getVhostVdpaDev(name)
		if err != nil {
			return nil, err
		}
	case vdpaDriverVirtio:
		return nil, fmt.Errorf("Not implemented ", driver)
	default:
		return nil, fmt.Errorf("Unknown vdpa bus driver %s", driver)
	}

	return &VdpaDevice{
		name:   name,
		driver: driver,
		path:   path,
	}, nil
}

/* Finds the vhost vdpa device of a vdpa device and returns it's path */
func getVhostVdpaDev(name string) (string, error) {
	file := filepath.Join(vdpaBusDevDir, name)
	fd, err := os.Open(file)
	if err != nil {
		return "", err
	}
	defer fd.Close()

	fileInfos, err := fd.Readdir(-1)
	for _, file := range fileInfos {
		if strings.Contains(file.Name(), "vhost-vdpa") &&
			file.IsDir() {
			devicePath := filepath.Join(vdpaVhostDevDir, file.Name())
			info, err := os.Stat(devicePath)
			if err != nil {
				return "", err
			}
			if info.Mode()&os.ModeDevice == 0 {
				return "", fmt.Errorf("vhost device %s is not a valid device", devicePath)
			}
			return devicePath, nil
		}
	}
	return "", fmt.Errorf("vhost device not found for vdpa device %s", name)
}

/*GetVdpaDeviceByPci returns the vdpa device information corresponding to a PCI device*/
/* Based on the following directory hiearchy:
/sys/bus/pci/devices/{PCIDev}/
    /vdpa{N}/

/sys/bus/vdpa/devices/vdpa{N} -> ../../../devices/pci.../{PCIDev}/vdpa{N}
*/
func GetVdpaDeviceByPci(pciAddr string) (*VdpaDevice, error) {
	path, err := filepath.EvalSymlinks(filepath.Join(pciBusDevDir, pciAddr))
	if err != nil {
		return nil, err
	}
	fd, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer fd.Close()

	fileInfos, err := fd.Readdir(-1)
	for _, file := range fileInfos {
		if strings.Contains(file.Name(), "vdpa") {
			parent, err := filepath.EvalSymlinks(filepath.Join(vdpaBusDevDir, file.Name()))
			if err != nil {
				return nil, err
			}
			if filepath.Dir(parent) != path {
				return nil, fmt.Errorf("vdpa device %s parent (%s) does not match containing dir (%s)",
					file.Name(), parent, path)
			}
			return GetVdpaDeviceByName(file.Name())
		}
	}
	return nil, fmt.Errorf("PCI address %s does not contain a vdpa device", pciAddr)
}
