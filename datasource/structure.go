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

// InstanceInfo describes an active instance of blacksmith running on some machine
type InstanceInfo struct {
	IP               net.IP           `json:"ip"`
	Nic              net.HardwareAddr `json:"nic"`
	WebPort          int              `json:"webPort"`
	Version          string           `json:"version"`
	Commit           string           `json:"commit"`
	BuildTime        string           `json:"buildTime"`
	DebugMode        string           `json:"debugMode"`
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
	MachineInterfaces() ([]EtcdMachineInterface, error)

	// EtcdMachineInterface returns the EtcdMachineInterface associated with the given
	// mac
	GetMachineInterface(mac net.HardwareAddr) EtcdMachineInterface

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
