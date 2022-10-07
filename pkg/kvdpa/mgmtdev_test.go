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
					return (flags|syscall.NLM_F_DUMP != 0)
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

func TestDevAdd(t *testing.T) {
	tests := []struct {
		mgmtBusName string
		mgmtDevName string
		devName     string
		err         error
	}{
		{
			mgmtBusName: "",
			mgmtDevName: "vdpasim_net",
			devName:     "vdpa0",
			err:         nil,
		},
		{
			mgmtBusName: "pci",
			mgmtDevName: "0000:65:00.2",
			devName:     "vdpa0",
			err:         nil,
		},
		{
			mgmtBusName: "",
			mgmtDevName: "",
			devName:     "vdpa0",
			err:         unix.EINVAL,
		},
		{
			mgmtBusName: "",
			mgmtDevName: "vdpasim_net",
			devName:     "",
			err:         unix.EINVAL,
		},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s_%s_%s", "TestDevAdd", tt.mgmtDevName, tt.devName), func(t *testing.T) {
			netLinkMock := &mocks.NetlinkOps{}
			SetNetlinkOps(netLinkMock)
			if tt.mgmtBusName != "" {
				netLinkMock.On("NewAttribute",
					VdpaAttrMgmtDevBusName,
					tt.mgmtBusName,
					mock.MatchedBy(func(data interface{}) bool {
						_, ok := data.(string)
						return ok
					})).Return(&nl.RtAttr{}, nil)
			}
			netLinkMock.On("NewAttribute",
				VdpaAttrMgmtDevDevName,
				tt.mgmtDevName,
				mock.MatchedBy(func(data interface{}) bool {
					_, ok := data.(string)
					return ok
				})).Return(&nl.RtAttr{}, nil)
			netLinkMock.On("NewAttribute",
				VdpaAttrDevName,
				tt.devName,
				mock.MatchedBy(func(data interface{}) bool {
					_, ok := data.(string)
					return ok
				})).Return(&nl.RtAttr{}, nil)
			netLinkMock.On("RunVdpaNetlinkCmd",
				VdpaCmdDevNew,
				mock.MatchedBy(func(flags int) bool {
					return (flags|unix.NLM_F_ACK != 0 && flags|unix.NLM_F_REQUEST != 0)
				}),
				mock.AnythingOfType("[]*nl.RtAttr")).Return([][]byte{}, tt.err)

			err := AddVdpaDevice(tt.mgmtBusName+"/"+tt.mgmtDevName, tt.devName)
			assert.Equal(t, tt.err, err)
		})
	}
}

func TestDevDelete(t *testing.T) {
	tests := []struct {
		devName string
		err     error
	}{
		{
			devName: "vdpa0",
			err:     nil,
		},
		{
			devName: "",
			err:     unix.EINVAL,
		},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s_%s", "TestDevDel", tt.devName), func(t *testing.T) {
			netLinkMock := &mocks.NetlinkOps{}
			SetNetlinkOps(netLinkMock)
			netLinkMock.On("NewAttribute",
				VdpaAttrDevName,
				tt.devName,
				mock.MatchedBy(func(data interface{}) bool {
					_, ok := data.(string)
					return ok
				})).Return(&nl.RtAttr{}, nil)
			netLinkMock.On("RunVdpaNetlinkCmd",
				VdpaCmdDevDel,
				mock.MatchedBy(func(flags int) bool {
					return (flags|unix.NLM_F_ACK != 0 && flags|unix.NLM_F_REQUEST != 0)
				}),
				mock.AnythingOfType("[]*nl.RtAttr")).Return([][]byte{}, tt.err)

			err := DeleteVdpaDevice(tt.devName)
			assert.Equal(t, tt.err, err)
		})
	}
}
