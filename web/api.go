package web

import (
	"net"

	"github.com/cafebazaar/blacksmith/datasource"
	"github.com/pkg/errors"
)

type machineDetails struct {
	Name          string                 `json:"name"`
	Nic           string                 `json:"nic"`
	IP            net.IP                 `json:"ip"`
	Type          datasource.MachineType `json:"type"`
	FirstAssigned int64                  `json:"firstAssigned"`
	LastAssigned  int64                  `json:"lastAssigned"`
}

func machineToDetails(machineInterface *datasource.EtcdMachine) (*machineDetails, error) {
	name := machineInterface.Hostname()
	mac := machineInterface.Mac()

	machine, err := machineInterface.Machine(false, nil)
	if err != nil {
		return nil, errors.Wrap(err, "error in retrieving machine details")
	}
	last, _ := machineInterface.LastSeen()

	return &machineDetails{
		name, mac.String(),
		machine.IP, machine.Type,
		machine.FirstSeen, last}, nil
}
