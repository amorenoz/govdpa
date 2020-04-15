/*
 * Copyright (C) 2020  - All Rights Reserved
 * Author: Adrián Moreno <amorenoz@redhat.com>
 *
 * TBD: Licencing
 */

package uvdpa

// VhostIface represents a vhost-user interface
type VhostIface struct {
	ID     int    `json:"vdpa_id,omitempty"`
	Device string `json:"device-id"`
	Socket string `json:"socket-path,omitempty"`
	Mode   string `json:"socket-mode,omitempty"`
	Driver string `json:"driver,omitempty"`
}

// UserDaemonStub is the Interface with the vDPA User Framework
type UserDaemonStub interface {
	// Initialize the Client
	Init() error
	// Close the Client
	Close() error
	// Version retrives the Framework's version
	Version() (string, error)
	// ListIfaces retrieves a list of active vhost interfaces
	ListIfaces() ([]VhostIface, error)
	// Create a vhost interface
	Create(VhostIface) error
	// Destroy a vhost interface identified by the device_id
	Destroy(string) error
}

// NewVdpaClient returns a VdpaClient instance
func NewVdpaClient(mock bool) UserDaemonStub {
	if mock {
		return newMockClient()
	}
	return newDpdkClient()
}
