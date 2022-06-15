package kvdpa

import (
	"fmt"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/vishvananda/netlink/nl"
	"golang.org/x/sys/unix"

	"github.com/k8snetworkplumbingwg/govdpa/pkg/kvdpa/mocks"
)

// Helper function for testing. It returns the information of a vdpadevice
// in netlink message (as would have been returned by netlink itself)
func vdpaDevToNlMessage(t *testing.T, devs ...VdpaDevice) [][]byte {
	nlOps := defaultNetlinkOps{}
	attrs := make([][]*nl.RtAttr, len(devs))
	for i, dev := range devs {
		attr := []*nl.RtAttr{}
		name, err := nlOps.NewAttribute(VdpaAttrDevName, dev.Name())
		assert.Nil(t, err)
		attr = append(attr, name)

		if mgmtDev := dev.MgmtDev(); mgmtDev != nil {
			name, err := nlOps.NewAttribute(VdpaAttrMgmtDevDevName, mgmtDev.DevName())
			assert.Nil(t, err)
			attr = append(attr, name)

			if busName := mgmtDev.BusName(); busName != "" {
				bus, err := nlOps.NewAttribute(VdpaAttrMgmtDevBusName, busName)
				assert.Nil(t, err)
				attr = append(attr, bus)
			}
		}
		attrs[i] = attr
	}
	return newMockNetLinkResponse(VdpaCmdDevNew, attrs)
}

func TestVdpaDevList(t *testing.T) {
	tests := []struct {
		name     string
		err      bool
		response []VdpaDevice
	}{
		{
			name:     "No devices",
			err:      false,
			response: []VdpaDevice{},
		},
		{
			name: "Single device",
			err:  false,
			response: []VdpaDevice{
				&vdpaDev{
					name: "vdpa0",
					mgmtDev: &mgmtDev{
						busName: "pci",
						devName: "0000:01:01",
					},
				},
			},
		},
		{
			name: "Multiple SR-IOV and SF devices",
			err:  false,
			response: []VdpaDevice{
				&vdpaDev{
					name: "vdpa0",
					mgmtDev: &mgmtDev{
						busName: "pci",
						devName: "0000:01:01",
					},
				},
				&vdpaDev{
					name: "vdpa1",
					mgmtDev: &mgmtDev{
						busName: "pci",
						devName: "0000:01:02",
					},
				},
				&vdpaDev{
					name: "vdpa2",
					mgmtDev: &mgmtDev{
						devName: "vdpasim_net",
					},
				},
				&vdpaDev{
					name: "foo",
					mgmtDev: &mgmtDev{
						busName: "foo",
						devName: "bar",
					},
				},
				&vdpaDev{
					name: "bar",
					mgmtDev: &mgmtDev{
						busName: "auxiliary",
						devName: "driver_sf_1",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s_%s", "TestVdpaDevList", tt.name), func(t *testing.T) {
			netLinkMock := &mocks.NetlinkOps{}
			SetNetlinkOps(netLinkMock)
			netLinkMock.On("RunVdpaNetlinkCmd",
				VdpaCmdDevGet,
				mock.MatchedBy(func(flags int) bool {
					return (flags|unix.NLM_F_DUMP != 0)
				}),
				mock.AnythingOfType("[]*nl.RtAttr")).
				Return(vdpaDevToNlMessage(t, tt.response...), nil)

			devs, err := ListVdpaDevices()
			if tt.err {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
				assert.Equal(t, tt.response, devs)
			}
		})
	}
}

func TestVdpaDevGet(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		response VdpaDevice
		devName  string
	}{
		{
			name: "Single device vdpa0",
			response: &vdpaDev{
				name: "vdpa0",
				mgmtDev: &mgmtDev{
					busName: "pci",
					devName: "0000:01:01",
				},
			},
			devName: "vdpa0",
		},
		{
			name: "Single device other name",
			response: &vdpaDev{
				name: "foo_bar_baz",
				mgmtDev: &mgmtDev{
					busName: "foo",
					devName: "bar",
				},
			},
			devName: "foo_bar_baz",
		},
		{
			name:    "wrong device",
			err:     syscall.ENODEV,
			devName: "wrongdev",
		},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s_%s", "TestDevGet", tt.name), func(t *testing.T) {
			netLinkMock := &mocks.NetlinkOps{}
			SetNetlinkOps(netLinkMock)
			netLinkMock.On("NewAttribute",
				VdpaAttrDevName,
				tt.devName,
				mock.MatchedBy(func(data interface{}) bool {
					_, ok := data.(string)
					return ok
				})).
				Return(&nl.RtAttr{}, nil)

			if tt.err != nil {
				netLinkMock.On("RunVdpaNetlinkCmd",
					VdpaCmdDevGet,
					0,
					mock.AnythingOfType("[]*nl.RtAttr")).
					Return(nil, tt.err)
			} else {
				netLinkMock.On("RunVdpaNetlinkCmd",
					VdpaCmdDevGet,
					0,
					mock.AnythingOfType("[]*nl.RtAttr")).
					Return(vdpaDevToNlMessage(t, tt.response), nil)
			}

			dev, err := GetVdpaDevice(tt.devName)
			if tt.err != nil {
				assert.Equal(t, tt.err, err)
			} else {
				assert.Nil(t, err)
				assert.Equal(t, tt.devName, dev.Name())
			}
		})
	}
}

func TestVdpaDevGetByMgmt(t *testing.T) {

	listResult := []VdpaDevice{
		&vdpaDev{
			name: "vdpa0",
			mgmtDev: &mgmtDev{
				busName: "pci",
				devName: "0000:01:01",
			},
		},
		&vdpaDev{
			name: "vdpa1",
			mgmtDev: &mgmtDev{
				busName: "pci",
				devName: "0000:01:02",
			},
		},
		&vdpaDev{
			name: "vdpa2",
			mgmtDev: &mgmtDev{
				devName: "vdpasim_net",
			},
		},
		&vdpaDev{
			name: "foo",
			mgmtDev: &mgmtDev{
				busName: "foo",
				devName: "bar",
			},
		},
		&vdpaDev{
			name: "bar",
			mgmtDev: &mgmtDev{
				busName: "auxiliary",
				devName: "driver_sf_1",
			},
		},
		&vdpaDev{
			name: "baz",
			mgmtDev: &mgmtDev{
				busName: "auxiliary",
				devName: "driver_sf_1",
			},
		},
	}

	tests := []struct {
		name        string
		err         error
		response    []VdpaDevice
		mgmtDevName string
		mgmtBusName string
	}{
		{
			name: "Empty bus",
			response: []VdpaDevice{
				&vdpaDev{
					name: "vdpa2",
					mgmtDev: &mgmtDev{
						devName: "vdpasim_net",
					},
				},
			},
			mgmtDevName: "vdpasim_net",
		},
		{
			name: "PCI Address",
			response: []VdpaDevice{
				&vdpaDev{
					name: "vdpa1",
					mgmtDev: &mgmtDev{
						busName: "pci",
						devName: "0000:01:02",
					},
				},
			},
			mgmtDevName: "0000:01:02",
			mgmtBusName: "pci",
		},
		{
			name: "Multiple SF",
			response: []VdpaDevice{
				&vdpaDev{
					name: "bar",
					mgmtDev: &mgmtDev{
						busName: "auxiliary",
						devName: "driver_sf_1",
					},
				},
				&vdpaDev{
					name: "baz",
					mgmtDev: &mgmtDev{
						busName: "auxiliary",
						devName: "driver_sf_1",
					},
				},
			},
			mgmtDevName: "driver_sf_1",
			mgmtBusName: "auxiliary",
		},
		{
			name:        "Wrong",
			err:         unix.ENODEV,
			mgmtDevName: "noDev",
			mgmtBusName: "wrongBus",
		},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s_%s", "TestDevGetByMgmt", tt.name), func(t *testing.T) {
			netLinkMock := &mocks.NetlinkOps{}
			SetNetlinkOps(netLinkMock)
			netLinkMock.On("RunVdpaNetlinkCmd",
				VdpaCmdDevGet,
				mock.MatchedBy(func(flags int) bool {
					return (flags|unix.NLM_F_DUMP != 0)
				}),
				mock.AnythingOfType("[]*nl.RtAttr")).
				Return(vdpaDevToNlMessage(t, listResult...), nil)

			devs, err := GetVdpaDevicesByMgmtDev(tt.mgmtBusName, tt.mgmtDevName)
			if tt.err != nil {
				assert.Equal(t, tt.err, err)
			} else {
				assert.Nil(t, err)
				assert.Equal(t, tt.response, devs)
			}
		})
	}
}
