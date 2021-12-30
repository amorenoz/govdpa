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

// Helper function for testing. It returns the information of a management device
// in netlink message (as would have been returned by netlink itself)
func mgmtDevToNlMessage(t *testing.T, devs ...MgmtDev) [][]byte {
	nlOps := defaultNetlinkOps{}
	attrs := make([][]*nl.RtAttr, len(devs))
	for i, dev := range devs {
		attr := []*nl.RtAttr{}
		name, err := nlOps.NewAttribute(VdpaAttrMgmtDevDevName, dev.DevName())
		assert.Nil(t, err)
		attr = append(attr, name)

		if dev.BusName() != "" {
			bus, err := nlOps.NewAttribute(VdpaAttrMgmtDevBusName, dev.BusName())
			assert.Nil(t, err)
			attr = append(attr, bus)
		}
		attrs[i] = attr
	}
	return newMockNetLinkResponse(VdpaCmdMgmtDevNew, attrs)
}

func TestMgmtDevList(t *testing.T) {
	tests := []struct {
		name     string
		err      bool
		mgmtDevs []MgmtDev
	}{
		{
			name: "Single mgmt device",
			err:  false,
			mgmtDevs: []MgmtDev{
				&mgmtDev{
					devName: "vdpasim_net",
				},
			},
		},
		{
			name: "Multiple SR-IOV and SF mgmt devices",
			err:  false,
			mgmtDevs: []MgmtDev{
				&mgmtDev{
					devName: "vdpasim_net",
				},
				&mgmtDev{
					devName: "0000:65:00.2",
					busName: "pci",
				},
				&mgmtDev{
					devName: "0000:65:00.3",
					busName: "pci",
				},
				&mgmtDev{
					devName: "mlx5_core.sf.2",
					busName: "auxiliary",
				},
				&mgmtDev{
					devName: "mlx5_core.sf.3",
					busName: "auxiliary",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s_%s", "TestMgtDevList", tt.name), func(t *testing.T) {
			netLinkMock := &mocks.NetlinkOps{}
			SetNetlinkOps(netLinkMock)
			netLinkMock.On("RunVdpaNetlinkCmd",
				VdpaCmdMgmtDevGet,
				mock.MatchedBy(func(flags int) bool {
					return (flags|unix.NLM_F_DUMP != 0)
				}),
				mock.AnythingOfType("[]*nl.RtAttr")).
				Return(mgmtDevToNlMessage(t, tt.mgmtDevs...), nil)

			devs, err := ListVdpaMgmtDevices()
			if tt.err {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
				assert.Equal(t, tt.mgmtDevs, devs)
			}
		})
	}
}

func TestMgmtDevGet(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		response MgmtDev
		busName  string
		devName  string
	}{
		{
			name: "vdpasim_net",
			response: &mgmtDev{
				devName: "vdpasim_net",
			},
			devName: "vdpasim_net",
		},
		{
			name: "pci address",
			response: &mgmtDev{
				devName: "0000:65:00.3",
				busName: "pci",
			},
			devName: "0000:65:00.3",
			busName: "pci",
		},
		{
			name: "sf",
			response: &mgmtDev{
				devName: "mlx5_core.sf.3",
				busName: "auxiliary",
			},
			devName: "mlx5_core.sf.3",
			busName: "auxiliary",
		},
		{
			name:    "wrong device",
			err:     syscall.ENODEV,
			devName: "0000:65:00.99",
			busName: "pci",
		},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s_%s", "TestMgtDevDev", tt.name), func(t *testing.T) {
			netLinkMock := &mocks.NetlinkOps{}
			SetNetlinkOps(netLinkMock)
			netLinkMock.On("NewAttribute",
				VdpaAttrMgmtDevDevName,
				tt.devName,
				mock.MatchedBy(func(data interface{}) bool {
					_, ok := data.(string)
					return ok
				})).
				Return(&nl.RtAttr{}, nil)
			if tt.busName != "" {
				netLinkMock.On("NewAttribute",
					VdpaAttrMgmtDevBusName,
					tt.busName,
					mock.MatchedBy(func(data interface{}) bool {
						_, ok := data.(string)
						return ok
					})).
					Return(&nl.RtAttr{}, nil)
			}

			if tt.err != nil {
				netLinkMock.On("RunVdpaNetlinkCmd",
					VdpaCmdMgmtDevGet,
					0,
					mock.AnythingOfType("[]*nl.RtAttr")).
					Return(nil, tt.err)
			} else {
				netLinkMock.On("RunVdpaNetlinkCmd",
					VdpaCmdMgmtDevGet,
					0,
					mock.AnythingOfType("[]*nl.RtAttr")).
					Return(mgmtDevToNlMessage(t, tt.response), nil)
			}

			dev, err := GetVdpaMgmtDevices(tt.busName, tt.devName)
			if tt.err != nil {
				assert.Equal(t, tt.err, err)
			} else {
				assert.Nil(t, err)
				assert.Equal(t, tt.devName, dev.DevName())
				assert.Equal(t, tt.busName, dev.BusName())
			}
		})
	}
}

func TestSplitMgmtDev(t *testing.T) {
	tests := []struct {
		name    string
		mgmtDev string
		busName string
		devName string
	}{{
		name:    "null bus",
		mgmtDev: "foo",
		devName: "foo",
	},
		{
			name:    "full device and bus ",
			mgmtDev: "foo/bar",
			busName: "foo",
			devName: "bar",
		},
		{
			name:    "too many bars",
			mgmtDev: "foo/bar/baz",
			devName: "",
		},
		{
			name:    "empty",
			mgmtDev: "",
			devName: "",
		},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s_%s", "TestSplitMgmtDev", tt.name), func(t *testing.T) {
			bus, name := SplitMgmtDevName(tt.mgmtDev)
			assert.Equal(t, tt.busName, bus)
			assert.Equal(t, tt.devName, name)
		})
	}
}
