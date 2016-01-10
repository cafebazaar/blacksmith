package datasource // import "github.com/cafebazaar/blacksmith/datasource"

import (
	"net"
	"time"
)

type Machine interface {
	Nic() net.HardwareAddr
	IP() net.IP
	FirstSeen() time.Time
	LastSeen() time.Time

	GetFlag(key string) (string, error)
	SetFlag(key string, value string) error
	GetAndDeleteFlag(key string) (string, error)
	DeleteFlag(key string) error
}

type DataSource interface {
	CoreOSVersion() (string, error)
	GetOrCreateMachine(net.HardwareAddr) (Machine, error)
	WorkspacePath() string
	Machines() []Machine
}
