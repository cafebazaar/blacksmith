package datasource // import "github.com/cafebazaar/blacksmith/datasource"

import (
	"net"
	"time"
	"github.com/coreos/etcd/client"
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

	// DeleteFlag erases the entry specified by key
	DeleteFlag(key string) error
}

// InstanceInfo describes an active instance of blacksmith running on some machine
type InstanceInfo struct {
	IP               net.IP           `json:"ip"`
	Nic              net.HardwareAddr `json:"nic"`
	WebPort          int              `json:"webPort"`
	Version          string           `json:"version"`
	Commit           string           `json:"commit"`
	BuildTime        string           `json:"buildTime"`
	ServiceStartTime int64            `json:"serviceStartTime"`
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

	// ListClusterVariables returns the list of all the cluster variables
	ListClusterVariables() (map[string]string, error)

	// GetClusterVariable returns a cluster variables with the given name
	GetClusterVariable(key string) (string, error)

	// SetClusterVariable sets a cluster variable
	SetClusterVariable(key string, value string) error

	// DeleteClusterVariable deletes a cluster variable
	DeleteClusterVariable(key string) error

	// ListClusterVariables list all cluster variables stored in etcd
	ListClusterVariables() (map[string]string, error)

	// Get returns value associated with key
	Get(key string) (string, error)
	
	// Get children nodes of a node with key
	GetNodes(key string) (client.Nodes, error)

	// GetConfiguration returns a configuration variables with the given name
	GetConfiguration(key string) (string, error)

	// SetConfiguration sets a configuration variable
	SetConfiguration(key, value string) error

	// DeleteConfiguration deletes a configuration variable
	DeleteConfiguration(key string) error

	// ClusterName returns the name of the ClusterName
	ClusterName() string

	// Assign finds an IP for the specified nic
	Assign(nic string) (net.IP, error)

	// Request is how to client requests to use the Ip address
	Request(nic string, currentIP net.IP) (net.IP, error)

	// DNSAddressesForDHCP returns the ip addresses of the present skydns servers
	// in the network, marshalled as specified in rfc2132 (option 6)
	DNSAddressesForDHCP() ([]byte, error)

	// IsMaster checks for being master, and makes a heartbeat
	IsMaster() bool

	// RemoveInstance removes the instance key from the list of instances, used to
	// gracefully shutdown the instance
	RemoveInstance() error

	EtcdMembers() (string, error)

	// Get all instances
	GetAllInstances() ([]string, error)
	GetAllOtherInstances() ([]string, error)

	// Create a new file node in Etcd
	NewFile(name string, file *os.File)
	WatchFileChanges()
}

type File struct  {
	Id			string		`json:"id,omitempty"`
	Name			string		`json:"name"`
	FromInstance		string		`json:"fromInstance"`
	Location		string		`json:"location"`
	UploadedAt		int64		`json:"uploadedAt"` // unix timestamp
	Size			int64           `json:"size"`
	LastModificationDate 	int64	 	`json:"lastModifiedDate"`
}
