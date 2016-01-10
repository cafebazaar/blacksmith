package datasource // import "github.com/cafebazaar/blacksmith/datasource"

import (
	"net"
	"time"
)

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
	//LeaseUpperbound specifies the last ip in the dhcp pool
	LeaseUpperbound() net.IP
}

//MasterDataSource embedds GeneralDataSource and DHCPDataSource
type MasterDataSource interface {
	GeneralDataSource
	DHCPDataSource
}
