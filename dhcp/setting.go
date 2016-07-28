package dhcp

import "net"

const NetConfigurationKey = "net-conf"

type classlessRouteOptionPart struct {
	Router      net.IP `json:"router"`
	Size        byte   `json:"size"`
	Destination net.IP `json:"destination"`
}

func (c *classlessRouteOptionPart) toBytes() []byte {
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

type networkConfiguration struct {
	Netmask              net.IP                     `json:"netmask"`
	Router               net.IP                     `json:"router"`
	ClasslessRouteOption []classlessRouteOptionPart `json:"classlessRouteOption"`
}
