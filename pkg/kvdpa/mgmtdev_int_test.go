package kvdpa

import (
	"fmt"
	"os"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
)

/* From vishvananda/netlink */
func KernelVersion() (kernel, major int, err error) {
	uts := syscall.Utsname{}
	if err = syscall.Uname(&uts); err != nil {
		return
	}

	ba := make([]byte, 0, len(uts.Release))
	for _, b := range uts.Release {
		if b == 0 {
			break
		}
		ba = append(ba, byte(b))
	}
	var rest string
	if n, _ := fmt.Sscanf(string(ba), "%d.%d%s", &kernel, &major, &rest); n < 2 {
		err = fmt.Errorf("can't parse kernel version in %q", string(ba))
	}
	return
}
func minKernelRequired(t *testing.T, kernel, major int) {
	k, m, err := KernelVersion()
	assert.Nil(t, err)
	if k < kernel || k == kernel && m < major {
		t.Skipf("Host Kernel (%d.%d) does not meet test's minimum required version: (%d.%d)",
			k, m, kernel, major)
	}
	if os.Getenv("INTEGRATION") != "yes" {
		t.Skipf("Skipping integration test")
	}
}

func TestIntegrationMgmtDevList(t *testing.T) {
	// Should have modprobed vdpa and vdpa_sim_net
	minKernelRequired(t, 5, 12)
	devs, err := ListVdpaMgmtDevices()
	assert.Nil(t, err)

	assert.Greaterf(t, len(devs), 0, "No mgmnt devices found. Please modprobe vdpa_sim_net")

	found := false
	for _, dev := range devs {
		t.Logf("%s: %#+v\n", dev.Name(), dev)
		if dev.Name() == "vdpasim_net" {
			found = true
			break
		}
	}
	assert.Truef(t, found, "No mgmnt devices found")
}

func TestIntegrationMgmtDevGet(t *testing.T) {
	// Should have modprobed vdpa and vdpa_sim_net
	minKernelRequired(t, 5, 12)
	devs, err := ListVdpaMgmtDevices()
	assert.Nil(t, err)

	assert.Greaterf(t, len(devs), 0, "No mgmnt devices found. Please modprobe vdpa_sim_net")

	for _, dev := range devs {
		t.Logf("%s: %#+v\n", dev.Name(), dev)
		gotDev, err := GetVdpaMgmtDevices(dev.BusName(), dev.DevName())
		assert.Nil(t, err)
		assert.Equal(t, gotDev.Name(), dev.Name())
	}

	dev2, err := GetVdpaMgmtDevices("wrongbus", "wrongdev")
	assert.Equal(t, syscall.ENODEV, err)
	assert.Nil(t, dev2)
}
