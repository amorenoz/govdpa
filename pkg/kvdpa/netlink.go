package kvdpa

import (
	"fmt"

	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netlink/nl"
	"golang.org/x/sys/unix"
)

/* Vdpa Netlink Name */
const (
	VdpaGenlName = "vdpa"
)

/* VDPA Netlink Commands */
const (
	VdpaCmdUnspec uint8 = iota
	VdpaCmdMgmtDevNew
	VdpaCmdMgmtDevGet /* can dump */
	VdpaCmdDevNew
	VdpaCmdDevDel
	VdpaCmdDevGet       /* can dump */
	VdpaCmdDevConfigGet /* can dump */
)

/* VDPA Netlink Attributes */
const (
	VdpaAttrUnspec = iota

	/* bus name (optional) + dev name together make the parent device handle */
	VdpaAttrMgmtDevBusName          /* string */
	VdpaAttrMgmtDevDevName          /* string */
	VdpaAttrMgmtDevSupportedClasses /* u64 */

	VdpaAttrDevName      /* string */
	VdpaAttrDevID        /* u32 */
	VdpaAttrDevVendorID  /* u32 */
	VdpaAttrDevMaxVqs    /* u32 */
	VdpaAttrDevMaxVqSize /* u16 */
	VdpaAttrDevMinVqSize /* u16 */

	VdpaAttrDevNetCfgMacAddr /* binary */
	VdpaAttrDevNetStatus     /* u8 */
	VdpaAttrDevNetCfgMaxVqp  /* u16 */
	VdpaAttrGetNetCfgMTU     /* u16 */

	/* new attributes must be added above here */
	VdpaAttrMax
)

var (
	commonNetlinkFlags = unix.NLM_F_REQUEST | unix.NLM_F_ACK
)

// NetlinkOps defines the Netlink Operations
type NetlinkOps interface {
	RunVdpaNetlinkCmd(command uint8, flags int, data []*nl.RtAttr) ([][]byte, error)
	NewAttribute(attrType int, data interface{}) (*nl.RtAttr, error)
}

type defaultNetlinkOps struct {
}

var netlinkOps NetlinkOps = &defaultNetlinkOps{}

// SetNetlinkOps method would be used by unit tests
func SetNetlinkOps(mockInst NetlinkOps) {
	netlinkOps = mockInst
}

// GetNetlinkOps will be invoked by functions in other packages that would need access to the sriovnet library methods.
func GetNetlinkOps() NetlinkOps {
	return netlinkOps
}

// RunVdpaNerlinkCmd runs a vdpa netlink command and returns the response
func (defaultNetlinkOps) RunVdpaNetlinkCmd(command uint8, flags int, data []*nl.RtAttr) ([][]byte, error) {
	f, err := netlink.GenlFamilyGet(VdpaGenlName)
	if err != nil {
		return nil, err
	}

	msg := &nl.Genlmsg{
		Command: command,
		Version: nl.GENL_CTRL_VERSION,
	}
	req := nl.NewNetlinkRequest(int(f.ID), commonNetlinkFlags|flags)

	req.AddData(msg)
	for _, d := range data {
		req.AddData(d)
	}

	msgs, err := req.Execute(unix.NETLINK_GENERIC, 0)
	if err != nil {
		return nil, err
	}
	return msgs, nil
}

// NewAttribute returns a new netlink attribute based on the provided data
func (defaultNetlinkOps) NewAttribute(attrType int, data interface{}) (*nl.RtAttr, error) {
	switch attrType {
	case VdpaAttrMgmtDevBusName, VdpaAttrMgmtDevDevName, VdpaAttrDevName:
		strData, ok := data.(string)
		if !ok {
			return nil, fmt.Errorf("Attribute type %d requires string data", attrType)
		}
		bytes := make([]byte, len(strData)+1)
		copy(bytes, strData)
		return nl.NewRtAttr(attrType, bytes), nil
		/* TODO
		case:
		    VdpaAttrMgmtDevBusName          string
		    VdpaAttrMgmtDevDevName           string
		    VdpaAttrMgmtDevSupportedClasses  u64

		    VdpaAttrDevName       string
		    VdpaAttrDevID         u32
		    VdpaAttrDevVendorID   u32
		    VdpaAttrDevMaxVqs     u32
		    VdpaAttrDevMaxVqSize  u16
		    VdpaAttrDevMinVqSize  u16

		    VdpaAttrDevNetCfgMacAddr  binary
		    VdpaAttrDevNetStatus      u8
		    VdpaAttrDevNetCfgMaxVqp   u16
		    VdpaAttrGetNetCfgMTU      u16
		*/
	default:
		return nil, fmt.Errorf("Invalid attribute type %d", attrType)
	}

}
