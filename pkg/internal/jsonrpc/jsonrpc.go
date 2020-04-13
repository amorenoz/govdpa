// This is a jsonrpc implementation heavily basen on net/rpc/jsonrpc
// but without its incompatibility issues

/*
Current limitations of this implementation:
- It returns errors encoded as a string with the following format
("[Code = %d ] %s %v", code, message, data)
*/

package client

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/rpc"
	"sync"
)

type jsonRPCCodec struct {
	dec  *json.Decoder // for reading JSON values
	enc  *json.Encoder // for writing JSON values
	conn io.Closer
	// temporary work space
	req  jsonRPCRequest
	resp jsonRPCResponse
	// JSON-RPC responses include the request id but not the request method.
	// Package rpc expects both.
	// We save the request method in pending when sending a request
	// and then look it up by request ID when filling out the rpc Response.
	mutex   sync.Mutex        // protects pending
	pending map[uint64]string // map request id to method name
}

// NewJSONRPCCodec returns a new rpc.ClientCodec using JSON-RPC on conn.
func NewJSONRPCCodec(conn io.ReadWriteCloser) rpc.ClientCodec {
	return &jsonRPCCodec{
		dec:     json.NewDecoder(conn),
		enc:     json.NewEncoder(conn),
		conn:    conn,
		pending: make(map[uint64]string),
	}
}

type jsonRPCRequest struct {
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
	Version string      `json:"jsonrpc"`
	ID      uint64      `json:"id"`
}

type jsonRPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type jsonRPCResponse struct {
	Result  *json.RawMessage `json:"result,omitempty"`
	Error   *jsonRPCError    `json:"error,omitempty"`
	Version string           `json:"jsonrpc"`
	ID      uint64           `json:"id"`
}

func (r *jsonRPCResponse) reset() {
	r.ID = 0
	r.Result = nil
	r.Error = nil
}

func (c *jsonRPCCodec) WriteRequest(req *rpc.Request, args interface{}) error {
	c.mutex.Lock()
	c.pending[req.Seq] = req.ServiceMethod
	c.mutex.Unlock()
	c.req.Method = req.ServiceMethod
	c.req.Params = args
	c.req.ID = req.Seq
	c.req.Version = "2.0"
	return c.enc.Encode(&c.req)
}

func (c *jsonRPCCodec) ReadResponseHeader(resp *rpc.Response) error {
	c.resp.reset()
	if err := c.dec.Decode(&c.resp); err != nil {
		return err
	}
	c.mutex.Lock()
	resp.ServiceMethod = c.pending[c.resp.ID]
	delete(c.pending, c.resp.ID)
	c.mutex.Unlock()

	resp.Error = ""
	resp.Seq = c.resp.ID
	if c.resp.Error != nil {
		emsg := c.resp.Error.Message
		if emsg == "" {
			emsg = "unspecified error"
		}
		resp.Error = fmt.Sprintf("[Code = %d] %s: %v",
			c.resp.Error.Code, emsg, c.resp.Error.Data)
	}
	return nil
}

func (c *jsonRPCCodec) ReadResponseBody(x interface{}) error {
	if x == nil {
		return nil
	}
	if c.resp.Result == nil {
		return nil
	}
	return json.Unmarshal(*c.resp.Result, x)
}

func (c *jsonRPCCodec) Close() error {
	return c.conn.Close()
}

// Dial connects to a JSON-RPC server at the specified network address.
func Dial(network, address string) (*rpc.Client, error) {
	conn, err := net.Dial(network, address)
	if err != nil {
		return nil, err
	}
	return rpc.NewClientWithCodec(NewJSONRPCCodec(conn)), err
}
