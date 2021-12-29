package kvdpa

import (
	"fmt"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/vishvananda/netlink/nl"

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
				},
			},
		},
		{
			name: "Multiple SR-IOV and SF devices",
			err:  false,
			response: []VdpaDevice{
				&vdpaDev{
					name: "vdpa0",
				},
				&vdpaDev{
					name: "vdpa1",
				},
				&vdpaDev{
					name: "vdpa2",
				},
				&vdpaDev{
					name: "foo",
				},
				&vdpaDev{
					name: "bar",
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
					return (flags|syscall.NLM_F_DUMP != 0)
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
			},
			devName: "vdpa0",
		},
		{
			name: "Single device other name",
			response: &vdpaDev{
				name: "foo_bar_baz",
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
		t.Run(fmt.Sprintf("%s_%s", "TestMgtDevDev", tt.name), func(t *testing.T) {
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

			dev, err := GetVdpaDeviceByName(tt.devName)
			if tt.err != nil {
				assert.Equal(t, tt.err, err)
			} else {
				assert.Nil(t, err)
				assert.Equal(t, tt.devName, dev.Name())
			}
		})
	}
}
