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
