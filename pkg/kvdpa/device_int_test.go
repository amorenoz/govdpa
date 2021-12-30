package kvdpa

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/sys/unix"
)

func TestIntegrationVdpaDev(t *testing.T) {
	// Should have modprobed vdpa and vdpa_sim_net
	// Also, netlink ADMIN capability is required
	minKernelRequired(t, 5, 12)
	SetNetlinkOps(&defaultNetlinkOps{})

	// Create a vdpa device
	err := AddVdpaDevice("vdpa_test0", "vdpasim_net")
	assert.Nil(t, err)

	// Retrieve it back
	dev, err := GetVdpaDevice("vdpa_test0")
	assert.Nil(t, err)
	assert.Equal(t, "vdpa_test0", dev.Name())
	assert.Equal(t, "vdpasim_net", dev.MgmtDev().Name())

	// Delte it
	err = DeleteVdpaDevice("vdpa_test0")
	assert.Nil(t, err)

	// Retrieve it again
	_, err = GetVdpaDevice("vdpa_test0")
	assert.Equal(t, unix.ENODEV, err)

}
