package kvdpa

/*
#include <stdlib.h>
#include <string.h>
#include <linux/vduse.h>
#include <linux/virtio_net.h>
*/
import "C"

import (
	"fmt"
	"os"
	"path/filepath"
	"unsafe"

	"golang.org/x/sys/unix"
)

// Private constants
const (
	vduseDevDir    = "/dev/vduse/"
	VduseNameMax   = 256
	VduseCreateDev = 0x41508102
	VduseDeleteDev = 0x41008103
	VduseVqSetup   = 0x40208114
)

/*
VduseDevConfig holds the configuration needed to create a VDUSE device.
See "struct vduse_dev_config" (include/uapi/linux/vduse.h).
*/
type VduseDevConfig struct {
	Name     string
	VendorID uint32
	DeviceID uint32
	Features uint64
	VQNum    uint32
	VQAlign  uint32
	Config   VirtioConf
}

// VirtioConf is an interface to be used by VduseDevConfig to provide configuration to the underlying virtio device.
type VirtioConf interface {
	// Copy the underlying configuration to a buffer.
	// The user is responsible for providing a buffer with enough space.
	// CopyLength() can be used to determine the minimum size of the buffer.
	CopyToBuf(dst unsafe.Pointer) error
	// Return the length of underlying configuration structure
	ConfigLength() uint32
}

// VirtioNetConf represents the virtio-net configuration
type VirtioNetConf struct {
	Mac                          [6]uint8
	Status                       uint16
	MaxVirtqueuePairs            uint16
	MTU                          uint16
	Speed                        uint32
	Duplex                       uint8
	RSSMaxKeySize                uint8
	RSSMaxIndirectionTableLength uint16
	SupportedHashTypes           uint32
}

// Golang type of "struct virtio_net_config".
type virtioNetConfC C.struct_virtio_net_config

// ConfigLength returnss the length the VirtioNetConf C struct.
func (v *VirtioNetConf) ConfigLength() uint32 {
	return C.sizeof_struct_virtio_net_config
}

// CopyToBuf copies the VirtioNetConf to the provided buffer.
func (v *VirtioNetConf) CopyToBuf(dst unsafe.Pointer) error {
	vc := new(virtioNetConfC)
	for i := 0; i < 6; i++ {
		vc.mac[i] = C.__u8(v.Mac[i])
	}
	vc.status = C.__u16(v.Status)
	vc.max_virtqueue_pairs = C.__u16(v.MaxVirtqueuePairs)
	vc.mtu = C.__u16(v.MTU)
	vc.speed = C.__u32(v.Speed)
	vc.duplex = C.__u8(v.Duplex)
	vc.rss_max_key_size = C.__u8(v.RSSMaxKeySize)
	vc.rss_max_indirection_table_length = C.__u16(v.RSSMaxIndirectionTableLength)
	vc.supported_hash_types = C.__u32(v.SupportedHashTypes)

	src := unsafe.Pointer(vc)
	l := C.size_t(v.ConfigLength())
	C.memcpy(dst, src, l)
	return nil
}

type vduseDevConfigC C.struct_vduse_dev_config
type vduseVqConfigC C.struct_vduse_vq_config

/*AddVduseDevice adds a new VDUSE device with the given configuration*/
func AddVduseDevice(config VduseDevConfig) error {
	if _, err := os.Stat(filepath.Join(vduseDevDir, config.Name)); !os.IsNotExist(err) {
		return fmt.Errorf("device already exists")
	}
	controlFile, err := unix.Open(filepath.Join(vduseDevDir, "control"), unix.O_RDWR, 0)
	if err != nil {
		return fmt.Errorf("failed to open vduse control device: %w", err)
	}
	defer unix.Close(controlFile)

	configLen := config.Config.ConfigLength()
	totalSize := C.sizeof_struct_vduse_dev_config + C.size_t(configLen)
	buf := C.malloc(totalSize)
	if buf == nil {
		return fmt.Errorf("Failed to allocate memory")
	}
	defer C.free(buf)

	C.memset(buf, 0, totalSize)
	cDev := (*vduseDevConfigC)(buf)

	for i := 0; i < len(config.Name) && i < VduseNameMax; i++ {
		cDev.name[i] = C.char(config.Name[i])
	}
	cDev.device_id = C.__u32(config.DeviceID)
	cDev.vendor_id = C.__u32(config.VendorID)
	cDev.features = C.__u64(config.Features)
	cDev.vq_num = C.__u32(config.VQNum)
	if config.VQAlign != 0 {
		cDev.vq_align = C.__u32(config.VQAlign)
	} else {
		cDev.vq_align = C.__u32(os.Getpagesize())
	}
	cDev.config_size = C.__u32(configLen)

	if configLen > 0 {
		config.Config.CopyToBuf(unsafe.Add(buf, C.sizeof_struct_vduse_dev_config))
	}
	//fmt.Printf("VDUSE_CREATE_DEV %+v\n", cDev)
	_, _, errno := unix.Syscall(
		unix.SYS_IOCTL,
		uintptr(controlFile),
		uintptr(VduseCreateDev),
		uintptr(buf),
	)
	if errno != 0 {
		return fmt.Errorf("ioctl VDUSE_CREATE_DEV failed: %s", errno.Error())
	}

	devFd, err := unix.Open(filepath.Join(vduseDevDir, config.Name), unix.O_RDWR, 0)
	if errno != 0 {
		DestroyVduseDevice(config.Name)
		return fmt.Errorf("%s: cannot open vduse device: %s", config.Name, errno.Error())
	}
	defer unix.Close(devFd)

	_, _, errno = unix.Syscall(
		unix.SYS_FCNTL,
		uintptr(devFd),
		unix.F_SETFL, unix.O_NONBLOCK)
	if errno != 0 {
		DestroyVduseDevice(config.Name)
		return fmt.Errorf("%s: cannot set vduse device non-blocking: %s", config.Name, errno.Error())
	}

	for i := 0; i < int(config.VQNum); i++ {
		vqConfig := new(vduseVqConfigC)
		vqConfig.index = C.__u32(i)
		vqConfig.max_size = 1024

		//fmt.Printf("VDUSE_VQ_SETUP %+v\n", vqConfig)
		_, _, errno := unix.Syscall(
			unix.SYS_IOCTL,
			uintptr(devFd),
			uintptr(VduseVqSetup),
			uintptr(unsafe.Pointer(vqConfig)),
		)
		if errno != 0 {
			DestroyVduseDevice(config.Name)
			return fmt.Errorf("%s: ioctl VDUSE_VQ_SETUP %d failed: %s", config.Name, i, errno.Error())
		}
	}

	return nil
}

// DestroyVduseDevice destroys a VDUSE device with a given name
func DestroyVduseDevice(name string) error {
	controlFile, err := unix.Open(filepath.Join(vduseDevDir, "control"), unix.O_RDWR, 0)
	if err != nil {
		return fmt.Errorf("failed to open vduse control device: %w", err)
	}
	defer unix.Close(controlFile)

	nameCstr := C.CString(name)
	defer C.free(unsafe.Pointer(nameCstr))

	_, _, errno := unix.Syscall(
		unix.SYS_IOCTL,
		uintptr(controlFile),
		uintptr(VduseDeleteDev),
		uintptr(unsafe.Pointer(nameCstr)),
	)
	if errno != 0 {
		// The 'errno' is returned as the error in Go.
		return fmt.Errorf("ioctl VDUSE_DESTROY_DEV failed: %s", errno.Error())
	}
	return nil
}
