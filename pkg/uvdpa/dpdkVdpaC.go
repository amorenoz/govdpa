package uvdpa

import (
	jsonrpc "github.com/amorenoz/user-vdpa-cli/pkg/internal/jsonrpc"
	"net/rpc"
	"sync"
)

const (
	daemonSocketFile = "/var/run/uvdpad.sock"
)

var (
	instance userClient
	once     sync.Once
)

// Structs of responses as defined in API
type listResponse struct {
	count      int
	interfaces []VhostIface
}

type createArgs struct {
	Device string `json:"device-id"`
	Socket string `json:"socket-path"`
	Mode   string `json:"socket-mode"`
}

type statusResponse struct {
	Status string `json:"status"`
}

// UserClientimplements UserDaemonStub and connects to the uvdpad:
// https://gitlab.com/mcoquelin/userspace-vdpa/
type userClient struct {
	client *rpc.Client
}

func (c *userClient) Init() error {
	var err error
	c.client, err = jsonrpc.Dial("unix", daemonSocketFile)
	if err != nil {
		return err
	}
	return nil
}

func (c *userClient) Close() error {
	return c.client.Close()
}

func (c *userClient) Version() (string, error) {
	var version string
	err := c.client.Call("version", nil, &version)
	if err != nil {
		return "", err
	}
	return version, nil
}

func (c *userClient) ListIfaces() ([]VhostIface, error) {
	var res listResponse
	err := c.client.Call("list-interfaces", nil, &res)
	return res.interfaces, err
}

func (c *userClient) Create(v VhostIface) error {
	var res statusResponse
	arg := createArgs{
		Device: v.Device,
		Socket: v.Socket,
		Mode:   v.Mode,
	}

	err := c.client.Call("create-interface", &arg, &res)
	return err
}

func (c *userClient) Destroy(dev string) error {
	var res statusResponse
	arg := dev
	err := c.client.Call("destroy-interfaces", &arg, &res)
	return err
}

func newDpdkClient() UserDaemonStub {
	once.Do(func() {
		instance = userClient{}
	})
	return &instance
}
