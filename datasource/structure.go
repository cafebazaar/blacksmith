package datasource // import "github.com/cafebazaar/blacksmith/datasource"

import (
	"net"
	"time"
)

// Machine provides the interface for querying/altering Machine entries
// in the datasource
type Machine interface {
	// Nic returns the hardware address of the machine
	Mac() net.HardwareAddr

	// IP reutrns the IP address associated with the machine
	IP() (net.IP, error)

	// Name returns the hostname of the machine
	Name() string

	// Domain returns the domain name of the machine
	Domain() string

	// FirstSeen returns the time upon which the machine has
	// been seen
	FirstSeen() (time.Time, error)

	// LastSeen returns the last time the machine has been seen
	LastSeen() (time.Time, error)

	// ListFlags returns the list of all the flgas of a machine from Etcd
	ListFlags() (map[string]string, error)

	// GetFlag returns the value of the supplied key
	GetFlag(key string) (string, error)

	// SetFlag sets the value of the specified key
	SetFlag(key string, value string) error

	// GetAndDeleteFlag gets the value associated with the key
	// and erases it afterwards
	GetAndDeleteFlag(key string) (string, error)

	// DeleteFlag erases the entry specified by key
	DeleteFlag(key string) error
}

// GeneralDataSource provides the interface for querying general information
type DataSource interface {
	Version() BlacksmithVersion

	// CoreOSVerison returns the coreOs version that blacksmith supplies
	CoreOSVersion() (string, error)

	// GetMachine returns The Machine object with the specified Hardware
	// address. Returns a flag to specify whether or not the entry exists
	GetMachine(net.HardwareAddr) (Machine, bool)

	// CreateMachine creates a machine with the specified hardware address and IP
	// the second return value will be set to true in case of successful machine
	// creation and to false in case of duplicate hardware address or IP
	CreateMachine(net.HardwareAddr, net.IP) (Machine, bool)

	// WorkspacePath returns the path to the workspace which is used after the
	// machines are booted up
	WorkspacePath() string

	// Machines returns a slice of Machines whose entries are present in the
	// datasource storage
	Machines() ([]Machine, error)

	// Get returns value associated with key
	Get(key string) (string, error)

	// Set sets key equal to value.
	Set(key, value string) error

	// Delete erases a key from the datasource
	Delete(key string) error

	// Gets a key, returns it's value and deletes it
	GetAndDelete(key string) (string, error)

	// ClusterName returns the name of the ClusterName
	ClusterName() string

	// LeaseStart specifies dhcp pool starting ip
	LeaseStart() net.IP
	// LeaseRange specifies number of IPs the dhcp server can assign
	LeaseRange() int

	// Assign finds an IP for the specified nic
	Assign(nic string) (net.IP, error)

	// Request is how to client requests to use the Ip address
	Request(nic string, currentIP net.IP) (net.IP, error)

	// DNSAddresses returns addresses of the dns servers present in the network which
	// can answer "what is the ip address of nodeX ?"
	// a byte slice is returned to be used as option 6 (rfc2132) in a dhcp Request
	// reply packet
	DNSAddresses() ([]byte, error)

	IsMaster() bool
	RemoveInstance() error
}
