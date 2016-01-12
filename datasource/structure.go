package datasource // import "github.com/cafebazaar/blacksmith/datasource"

import (
	"net"
	"time"
)

//Machine provides the interface for querying/altering Machine entries
//in the datasource
type Machine interface {
	//Nic returns the hardware address of the machine
	Nic() net.HardwareAddr

	//IP reutrns the IP address associated with the machine
	IP() net.IP

	//FirstSeen returns the time upon which the machine has
	//been seen
	FirstSeen() time.Time

	//LastSeen returns the last time the machine has been seen
	LastSeen() time.Time

	//GetFlag returns the value of the supplied key
	GetFlag(key string) (string, error)

	//SetFlag sets the value of the specified key
	SetFlag(key string, value string) error

	//GetAndDeleteFlag gets the value associated with the key
	//and erases it afterwards
	GetAndDeleteFlag(key string) (string, error)

	//DeleteFlag erases the entry specified by key
	DeleteFlag(key string) error
}

//GeneralDataSource provides the interface for querying general information
type GeneralDataSource interface {
	//CoreOSVerison returns the coreOs version that blacksmith supplies
	CoreOSVersion() (string, error)

	//GetOrCreateMachine returns The Machine object with the specified Hardware
	//address if it exists. Otherwise creates it and returns a handle.
	//second return value should be set to true if the Machine already exists
	GetOrCreateMachine(net.HardwareAddr) (Machine, bool, error)

	//WorkspacePath returns the path to the workspace which is used after the
	//machines are booted up
	WorkspacePath() string

	//Machines returns a slice of Machines whose entries are present in the
	//datasource storage
	Machines() []Machine
}

//DHCPDataSource is the functionality that a DHCP datasource has to provide
type DHCPDataSource interface {
	//LeaseStart specifies dhcp pool starting ip
	LeaseStart() net.IP
	//LeaseRange specifies the dhcp pool ip range
	LeaseRange() net.IP
}

//CloudConfigDatasource is the interface that any cloud-config file server
//has to implement.
type CloudConfigDataSource interface {
	//Generates the cloud-config file using the IP Address + Mac Address from
	//the IPMac which is passed to it as config context
	//IP and mac address is currently passed in through a URL, therefore the
	//function signature will be as simple as the situation (string instead of
	//net.IP and net.HardwareAddr) and no simpler
	IPMacCloudConfig(ip, mac string)
}

type KeyValueDataSource interface {
	//Get returns value associated with key
	Get(key string) (string, error)

	//Set sets key equal to value.
	Set(key, value string) error

	//Delete erases a key from the datasource
	Delete(key string) error

	//Gets a key, returns it's value and deletes it
	GetAndDelete() (string, error)
}

//RestServer defines the interface that a rest server has to implement to work
//with Blacksmith
type RestServer interface {
	//Handler returns an http handler which can be used to serve http requests
	Handler() http.Handler
}

//UIRestServer specifies the functionality that is expected from a rest server
//which will act as the backend for Blacksmith web UI
type UIRestServer interface {
	//DeleteFile provides file deleting functionality
	DeleteFile(w http.ResponseWriter, r *http.Request)

	//Files provides the means for accessing uploaded files
	Files(w http.ResponseWriter, r *http.Request)

	//NodesList returns a view of the nodes recognized by blacksmith
	//Also provides useful information about each one and allows you to modify
	//certain settings
	NodesList(w http.ResponseWriter, r *http.Request)

	//Upload provides file uploading functionality
	Upload(w http.ResponseWriter, r *http.Request)
}

//MasterDataSource embedds GeneralDataSource, DHCPDataSource,
//CloudConfigDataSource and KeyValueDataStore
type MasterDataSource interface {
	GeneralDataSource
	DHCPDataSource
	CloudConfigDataSource
	KeyValueDataStore
}
