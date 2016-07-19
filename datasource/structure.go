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

// InstanceInfo describes an active instance of blacksmith running on some machine
type InstanceInfo struct {
	IP               net.IP 		`json:"ip"`
	Nic              net.HardwareAddr	`json:"nic"`
	WebPort          int    		`json:"webPort"`
	Version          string 		`json:"version"`
	Commit           string 		`json:"commit"`
	BuildTime        string 		`json:"buildTime"`
	ServiceStartTime int64  		`json:"serviceStartTime"`
}

// DataSource provides the interface for querying general information
type DataSource interface {
	// SelfInfo return InstanceInfo of this instance of blacksmith
	SelfInfo() InstanceInfo

	// CoreOSVerison returns the coreOs version that blacksmith supplies
	CoreOSVersion() (string, error)

	// GetMachine returns The Machine object with the specified Hardware
	// address. Returns a flag to specify whether or not the entry exists
	GetMachine(net.HardwareAddr) (Machine, bool)

	// WorkspacePath returns the path to the workspace which is used after the
	// machines are booted up
	WorkspacePath() string

	// Machines returns a slice of Machines whose entries are present in the
	// datasource storage
	Machines() ([]Machine, error)

	// Get returns value associated with cluster variable key
	GetClusterVariable(key string) (string, error)

	// Set sets cluster variable equal to value.
	SetClusterVariable(key, value string) error
    
    // ListClusterVariables list all cluster variables stored in etcd
    ListClusterVariables()(map[string]string, error)
    
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

	// IsMaster checks for being master, and makes a heartbeat
	IsMaster() bool

	// RemoveInstance removes the instance key from the list of instances, used to
	// gracefully shutdown the instance
	RemoveInstance() error

	EtcdMembers() (string, error)
}
