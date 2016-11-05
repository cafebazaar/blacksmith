package datasource

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
)

const (
	// SpecialKeyCoreosVersion is a special key for the coreos version of the machines
	SpecialKeyCoreosVersion = "coreos-version"
	// SpecialKeyNetworkConfiguration is a special key for the network of the cluster
	SpecialKeyNetworkConfiguration = "net-conf"
)

// NetworkConfiguration is used to configure clients through dhcp
type NetworkConfiguration struct {
	Netmask              net.IP                     `json:"netmask"`
	Router               net.IP                     `json:"router"`
	ClasslessRouteOption []ClasslessRouteOptionPart `json:"classlessRouteOption"`
}

// ClasslessRouteOptionPart is the static route which consists of a destination
// descriptor and the IP address of the router that should be used to reach
// that destination.
// https://tools.ietf.org/html/rfc3442
type ClasslessRouteOptionPart struct {
	Router      net.IP `json:"router"`
	Size        byte   `json:"size"`
	Destination net.IP `json:"destination"`
}

// ToBytes formats the static route according to rfc3442
func (c *ClasslessRouteOptionPart) ToBytes() []byte {
	var ret []byte
	ret = append(ret, c.Size)

	dst := c.Destination.To4()

	if c.Size > 0 {
		ret = append(ret, dst[0])
	}
	if c.Size > 8 {
		ret = append(ret, dst[1])
	}
	if c.Size > 16 {
		ret = append(ret, dst[2])
	}
	if c.Size > 24 {
		ret = append(ret, dst[3])
	}

	ret = append(ret, (c.Router.To4())...)
	return ret
}

var (
	emptyNotAllowed = map[string]bool{
		SpecialKeyCoreosVersion:        true,
		SpecialKeyNetworkConfiguration: true,
	}
)

// UnmarshalNetworkConfiguration returns a pointer to a newly constructed
// NetworkConfiguration from the given string
func UnmarshalNetworkConfiguration(netConfStr string) (*NetworkConfiguration, error) {
	var netConf NetworkConfiguration
	if err := json.Unmarshal([]byte(netConfStr), &netConf); err != nil {
		return nil, err
	}
	// TODO: more validation on netmask and ...
	return &netConf, nil
}

func validateVariable(key, value string, forMachine bool) error {
	if key == "" {
		return errors.New("empty value for key is not permitted")
	}
	if key[0] == '_' {
		return errors.New("hidden key is not permitted")
	}
	if value == "" && emptyNotAllowed[key] {
		return fmt.Errorf("empty value for %q is not permitted", key)
	}
	if forMachine && key[0] == '#' {
		return errors.New("hashtagged keys are only permitted for cluster-wide variables")
	}
	switch key {
	case SpecialKeyCoreosVersion:
		// TODO: more validation
	case SpecialKeyNetworkConfiguration:
		_, err := UnmarshalNetworkConfiguration(value)
		return err
	}
	return nil
}
