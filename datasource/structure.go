package datasource // import "github.com/cafebazaar/blacksmith/datasource"

import "net"

// MachineType distinguishes normal servers from static ones, and from the BMC inside those machines
type MachineType int16

const (
	// MTNormal is for the ethernet of a server machine attached to our private
	// network and its ip is provided by our DHCP
	MTNormal MachineType = 1
	// MTStatic is for the ethernet of a server machine attached to our private
	// network, but the IP address is forces. Currentl just the Blacksmith
	// instances are created in this manner.
	MTStatic MachineType = 2
	// MTBMC is for the baseboard management controller embedded on the
	// motherboard of the server machines
	MTBMC MachineType = 3
)

// Machine details
type Machine struct {
	IP        net.IP      `json:"ip"`
	FirstSeen int64       `json:"first_seen"`
	Type      MachineType `json:"type"`
}

// MachineInterface provides the interface for querying/altering
// Machine entries in the datasource
type MachineInterface interface {
	// Mac returns the hardware address of the associated machine
	Mac() net.HardwareAddr

	// Hostname returns the mac address formatted as a string suitable for hostname
	Hostname() string

	// Machine creates a record for the associated mac if needed
	// and asked for, and returns a Machine with the stored values.
	// If createIfNeeded is true, and there is no machine associated to
	// this mac, the machine will be created, stored, and returned.
	// In this case, if createWithIP is empty, the IP will be assigned
	// automatically, otherwise the given will be used. An error will be
	// raised if createWithIP is currently assigned to another mac. Also
	// the Type will be automatically set to MTNormal if createWithIP is
	// nil, otherwise to MTStatic.
	// If createIfNeeded is false, the createWithIP is expected to be nil.
	// Note: if the machine exists, createWithIP is ignored. It's possible
	// for the returned Machine to have an IP different from createWithIP.
	Machine(createIfNeeded bool, createWithIP net.IP) (Machine, error)

	// LastSeen returns the last time the machine has been seen
	LastSeen() (int64, error)

	// DeleteMachine deletes a machine from the store entirely
	DeleteMachine() error

	// CheckIn updates the _last_seen field of the machine
	CheckIn()

	// ListVariables returns the list of all the flgas of a machine from Etcd
	ListVariables() (map[string]string, error)

	// GetVariable Gets a machine's variable, or the global if it was not
	// set for the machine
	GetVariable(key string) (string, error)

	// SetVariable sets the value of the specified key
	SetVariable(key string, value string) error

	// DeleteVariable erases the entry specified by key
	DeleteVariable(key string) error
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

// File describes a file located inside our workspace
type File struct {
	ID                   string `json:"id,omitempty"`
	Name                 string `json:"name"`
	FromInstance         string `json:"fromInstance"`
	Location             string `json:"location"`
	UploadedAt           int64  `json:"uploadedAt"` // unix timestamp
	Size                 int64  `json:"size"`
	LastModificationDate int64  `json:"lastModifiedDate"`
}

// DataSource provides the interface for querying general information
type DataSource interface {
	// SelfInfo return InstanceInfo of this instance of blacksmith
	SelfInfo() InstanceInfo

	// Instances returns the InstanceInfo of all the present instances of
	// blacksmith in our cluster
	Instances() ([]InstanceInfo, error)

	// IsMaster checks for being master
	IsMaster() error

	// WhileMaster makes a heartbeat and returns IsMaster()
	WhileMaster() error

	// Shutdown removes the instance key from the list of instances, used to
	// gracefully shutdown the instance
	Shutdown() error

	// ClusterName returns the name of the ClusterName
	ClusterName() string

	// WorkspacePath returns the path to the workspace which is used after the
	// machines are booted up
	WorkspacePath() string

	// FilesPath returns the path to the files which is used for download
	// needed files
	FileServer() string

	// MachineInterfaces returns all the machines in the cluster, as a slice of
	// MachineInterfaces
	MachineInterfaces() ([]MachineInterface, error)

	// MachineInterface returns the MachineInterface associated with the given
	// mac
	MachineInterface(mac net.HardwareAddr) MachineInterface

	// ListClusterVariables returns the list of all the cluster variables
	ListClusterVariables() (map[string]string, error)

	// GetClusterVariable returns a cluster variables with the given name
	GetClusterVariable(key string) (string, error)

	// GetClusterArrayVariable returns a cluster variables with the given name
	GetArrayVariable(key string) (interface{}, error)

	// SetClusterVariable sets a cluster variable
	SetClusterVariable(key string, value string) error

	// DeleteClusterVariable delete a cluster variable from etcd.
	DeleteClusterVariable(key string) error

	// UpdateWorkspace Update workspace
	UpdateWorkspace() error

	// WorkspaceHash returns workspace hash
	GetWorkspaceHash() (string, error)

	UpdateSignal() error

	WebServer() string
	SetWebServer(string)

	// GetVariable get etcd variable
	GetVariable(string) (string, error)

	// EtcdMembers returns a string suitable for `-initial-cluster`
	// This is the etcd the Blacksmith instance is using as its datastore
	// Smelly function to be here! but it's a lot helpful.
	EtcdMembers() (string, error)

	// EtcdMembers returns a string suitable for etcdctl
	// This is the etcd the Blacksmith instance is using as its datastore
	// Smelly function to be here too! but it's a lot helpful.
	EtcdEndpoints() (string, error)
}
