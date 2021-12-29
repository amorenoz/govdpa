package kvdpa

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Exported constants
const (
	VhostVdpaDriver  = "vhost_vdpa"
	VirtioVdpaDriver = "virtio_vdpa"
)

// Private constants
const (
	vdpaBusDevDir   = "/sys/bus/vdpa/devices"
	pciBusDevDir    = "/sys/bus/pci/devices"
	vdpaVhostDevDir = "/dev"
	rootDevDir      = "/sys/devices"
)

// VdpaDevice contains information about a Vdpa Device
type VdpaDevice interface {
	Driver() string
	Name() string
	VirtioNet() VirtioNet
	VhostVdpa() VhostVdpa
	ParentDevicePath() (string, error)
}

// vdpaDev implements VdpaDevice interface
type vdpaDev struct {
	name      string
	driver    string
	virtioNet VirtioNet
	vhostVdpa VhostVdpa
}

// Driver resturns de device's driver name
func (vd *vdpaDev) Driver() string {
	return vd.driver
}

// Driver resturns de device's name
func (vd *vdpaDev) Name() string {
	return vd.name
}

// VhostVdpa returns the VhostVdpa device information associated
// or nil if the device is not bound to the vhost_vdpa driver
func (vd *vdpaDev) VhostVdpa() VhostVdpa {
	return vd.vhostVdpa
}

// Virtionet returns the VirtioNet device information associated
// or nil if the device is not bound to the virtio_vdpa driver
func (vd *vdpaDev) VirtioNet() VirtioNet {
	return vd.virtioNet
}

// getBusInfo populates the vdpa bus information
// the vdpa device must have at least the name prepopulated
func (vd *vdpaDev) getBusInfo() error {
	driverLink, err := os.Readlink(filepath.Join(vdpaBusDevDir, vd.name, "driver"))
	if err != nil {
		// No error if driver is not present. The device is simply not bound to any.
		return nil
	}

	vd.driver = filepath.Base(driverLink)

	switch vd.driver {
	case VhostVdpaDriver:
		vd.vhostVdpa, err = vd.getVhostVdpaDev()
		if err != nil {
			return err
		}
	case VirtioVdpaDriver:
		vd.virtioNet, err = vd.getVirtioVdpaDev()
		if err != nil {
			return err
		}
	}

	return nil
}

/* Finds the vhost vdpa device of a vdpa device and returns it's path */
func (vd *vdpaDev) getVhostVdpaDev() (VhostVdpa, error) {
	// vhost vdpa devices live in the vdpa device's path
	path := filepath.Join(vdpaBusDevDir, vd.name)
	return GetVhostVdpaDevInPath(path)
}

/* ParentDevice returns the path of the parent device (e.g: PCI) of the device */
func (vd *vdpaDev) ParentDevicePath() (string, error) {
	vdpaDevicePath := filepath.Join(vdpaBusDevDir, vd.name)

	/* For pci devices we have:
	/sys/bud/vdpa/devices/vdpaX ->
	    ../../../devices/pci0000:00/.../0000:05:00:1/vdpaX

	Resolving the symlinks should give us the parent PCI device.
	*/
	devicePath, err := filepath.EvalSymlinks(vdpaDevicePath)
	if err != nil {
		return "", err
	}

	/* If the parent device is the root device /sys/devices, there is
	no parent (e.g: vdpasim).
	*/
	parent := filepath.Dir(devicePath)
	if parent == rootDevDir {
		return devicePath, nil
	}

	return parent, nil
}

/* Finds the virtio vdpa device of a vdpa device and returns its path
Currently, PCI-based devices have the following sysfs structure:
/sys/bus/vdpa/devices/
    vdpa1 -> ../../../devices/pci0000:00/0000:00:03.2/0000:05:00.2/vdpa1

In order to find the virtio device we look for virtio* devices inside the parent device:
	sys/devices/pci0000:00/0000:00:03.2/0000:05:00.2/virtio{N}

We also check the virtio device exists in the virtio bus:
/sys/bus/virtio/devices
    virtio{N} -> ../../../devices/pci0000:00/0000:00:03.2/0000:05:00.2/virtio{N}
*/
func (vd *vdpaDev) getVirtioVdpaDev() (VirtioNet, error) {
	parentPath, err := vd.ParentDevicePath()
	if err != nil {
		return nil, err
	}
	return GetVirtioNetInPath(parentPath)
}

/*GetVdpaDeviceByName returns the vdpa device information by a vdpa device name */
func GetVdpaDeviceByName(name string) (VdpaDevice, error) {
	if _, err := os.Readlink(filepath.Join(vdpaBusDevDir, name)); err != nil {
		return nil, err
	}

	vdpaDev := &vdpaDev{
		name: name,
	}

	if err := vdpaDev.getBusInfo(); err != nil {
		return nil, err
	}
	return vdpaDev, nil
}

/*GetVdpaDeviceByPci returns the vdpa device information corresponding to a PCI device*/
/* Based on the following directory hiearchy:
/sys/bus/pci/devices/{PCIDev}/
    /vdpa{N}/

/sys/bus/vdpa/devices/vdpa{N} -> ../../../devices/pci.../{PCIDev}/vdpa{N}
*/
func GetVdpaDeviceByPci(pciAddr string) (VdpaDevice, error) {
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
	if err != nil {
		return nil, err
	}
	for _, file := range fileInfos {
		if strings.Contains(file.Name(), "vdpa") {
			vdpaDev, err := GetVdpaDeviceByName(file.Name())
			if err != nil {
				return nil, err
			}
			parent, err := vdpaDev.ParentDevicePath()
			if err != nil {
				return nil, err
			}
			if parent != path {
				return nil, fmt.Errorf("vdpa device %s parent (%s) does not match containing dir (%s)",
					file.Name(), parent, path)
			}
			return vdpaDev, nil
		}
	}
	return nil, fmt.Errorf("PCI address %s does not contain a vdpa device", pciAddr)
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
			vdpaDevList = append(vdpaDevList, vdpaDev)
		}
	}

	if len(errors) > 0 {
		return vdpaDevList, fmt.Errorf(strings.Join(errors, ";"))
	}
	return vdpaDevList, nil
}
