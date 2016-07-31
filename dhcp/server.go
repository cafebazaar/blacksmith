package dhcp // import "github.com/cafebazaar/blacksmith/dhcp"

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"strings"
	"time"

	"github.com/cafebazaar/blacksmith/datasource"
	"github.com/cafebazaar/blacksmith/logging"
	"github.com/krolaw/dhcp4"
)

const (
	minLeaseHours = 24
	maxLeaseHours = 48

	debugTag = "DHCP"
)

func randLeaseDuration() time.Duration {
	n := (minLeaseHours + rand.Intn(maxLeaseHours-minLeaseHours))
	return time.Duration(n) * time.Hour
}

// StartDHCP ListenAndServe for dhcp on port 67, binds on interface=ifName if it's
// not empty
func StartDHCP(ifName string, serverIP net.IP, datasource datasource.DataSource) error {
	handler := &Handler{
		ifName:      ifName,
		serverIP:    serverIP,
		datasource:  datasource,
		bootMessage: fmt.Sprintf("Blacksmith (%s)", datasource.SelfInfo().Version),
	}

	logging.Log("DHCP", "Listening on %s:67 (interface: %s)", serverIP.String(), ifName)
	var err error
	if ifName != "" {
		err = dhcp4.ListenAndServeIf(ifName, handler)
	} else {
		err = dhcp4.ListenAndServe(handler)
	}

	rand.Seed(time.Now().UTC().UnixNano())

	return err
}

// Handler is passed to dhcp4 package to handle DHCP packets
type Handler struct {
	ifName      string
	serverIP    net.IP
	datasource  datasource.DataSource
	dhcpOptions dhcp4.Options
	bootMessage string
}

func (h *Handler) fillPXE() []byte {
	// PXE vendor options
	var pxe bytes.Buffer
	var l byte
	// Discovery Control - disable broadcast and multicast boot server discovery
	pxe.Write([]byte{6, 1, 3})
	// PXE boot server
	pxe.Write([]byte{8, 7, 0x80, 0x00, 1})
	pxe.Write(h.serverIP.To4())
	// PXE boot menu - one entry, pointing to the above PXE boot server
	l = byte(3 + len(h.bootMessage))
	pxe.Write([]byte{9, l, 0x80, 0x00, 9})
	pxe.WriteString(h.bootMessage)
	// PXE menu prompt+timeout
	l = byte(1 + len(h.bootMessage))
	pxe.Write([]byte{10, l, 0x2})
	pxe.WriteString(h.bootMessage)
	// End vendor options
	pxe.WriteByte(255)
	return pxe.Bytes()
}

// ServeDHCP replies a dhcp request
func (h *Handler) ServeDHCP(p dhcp4.Packet, msgType dhcp4.MessageType, options dhcp4.Options) (d dhcp4.Packet) {
	dns, err := h.datasource.DNSAddressesForDHCP()
	if err != nil {
		logging.Log(debugTag, "Failed to read dns addresses")
		return nil
	}

	netConfStr, err := h.datasource.GetConfiguration(NetConfigurationKey)
	if err != nil {
		logging.Log(debugTag, "Failed to get network configuration")
		return nil
	}

	var netConf networkConfiguration
	if err := json.Unmarshal([]byte(netConfStr), &netConf); err != nil {
		logging.Log(debugTag, "failed to unmarshal network configuration: %s / network configuration=%q",
			err, netConfStr)
		return nil
	}

	dhcpOptions := dhcp4.Options{
		dhcp4.OptionSubnetMask:       netConf.Netmask.To4(),
		dhcp4.OptionDomainNameServer: dns,
	}

	if netConf.Router != nil {
		dhcpOptions[dhcp4.OptionRouter] = netConf.Router.To4()
	}
	if len(netConf.ClasslessRouteOption) != 0 {
		var res []byte
		for _, part := range netConf.ClasslessRouteOption {
			res = append(res, part.toBytes()...)
		}
		dhcpOptions[dhcp4.OptionClasslessRouteFormat] = res

	}

	macAddress := strings.Join(strings.Split(p.CHAddr().String(), ":"), "")
	switch msgType {
	case dhcp4.Discover:
		ip, err := h.datasource.Assign(p.CHAddr().String())
		if err != nil {
			logging.Debug("DHCP", "err in lease pool - %s", err.Error())
			return nil // pool is full
		}
		replyOptions := dhcpOptions.SelectOrderOrAll(options[dhcp4.OptionParameterRequestList])

		guidVal, isPxe := options[97]
		if isPxe { // this is a pxe request
			logging.Log("DHCP", "dhcp discover with PXE - CHADDR %s - IP %s", p.CHAddr().String(), ip.String())
			guid := guidVal[1:]
			replyOptions = append(replyOptions,
				dhcp4.Option{
					Code:  dhcp4.OptionVendorClassIdentifier,
					Value: []byte("PXEClient"),
				},
				dhcp4.Option{
					Code:  97, // UUID/GUID-based Client Identifier
					Value: guid,
				},
				dhcp4.Option{
					Code:  dhcp4.OptionVendorSpecificInformation,
					Value: h.fillPXE(),
				},
			)
		} else {
			logging.Log("DHCP", "dhcp discover - CHADDR %s - IP %s", p.CHAddr().String(), ip.String())
		}
		packet := dhcp4.ReplyPacket(p, dhcp4.Offer, h.serverIP, ip, randLeaseDuration(), replyOptions)
		return packet
	case dhcp4.Request:
		if server, ok := options[dhcp4.OptionServerIdentifier]; ok && !net.IP(server).Equal(h.serverIP) {
			return nil // this message is not ours
		}
		requestedIP := net.IP(options[dhcp4.OptionRequestedIPAddress])
		if requestedIP == nil {
			requestedIP = net.IP(p.CIAddr())
		}
		if len(requestedIP) != 4 || requestedIP.Equal(net.IPv4zero) {
			logging.Debug("DHCP", "dhcp request - CHADDR %s - bad request", p.CHAddr().String())
			return nil
		}
		_, err := h.datasource.Request(p.CHAddr().String(), requestedIP)
		if err != nil {
			logging.Debug("DHCP", "dhcp request - CHADDR %s - Requested IP %s - NO MATCH", p.CHAddr().String(), requestedIP.String())

			return dhcp4.ReplyPacket(p, dhcp4.NAK, h.serverIP, nil, 0, nil)
		}

		replyOptions := dhcpOptions.SelectOrderOrAll(options[dhcp4.OptionParameterRequestList])
		replyOptions = append(replyOptions,
			dhcp4.Option{
				Code:  dhcp4.OptionHostName,
				Value: []byte("node" + macAddress + "." + h.datasource.ClusterName()),
			},
		)

		guidVal, isPxe := options[97]
		if isPxe { // this is a pxe request
			logging.Log("DHCP", "dhcp request with PXE - CHADDR %s - Requested IP %s - ACCEPTED", p.CHAddr().String(), requestedIP.String())
			guid := guidVal[1:]
			replyOptions = append(replyOptions,
				dhcp4.Option{
					Code:  dhcp4.OptionVendorClassIdentifier,
					Value: []byte("PXEClient"),
				},
				dhcp4.Option{
					Code:  97, // UUID/GUID-based Client Identifier
					Value: guid,
				},
				dhcp4.Option{
					Code:  dhcp4.OptionVendorSpecificInformation,
					Value: h.fillPXE(),
				},
			)
		} else {
			logging.Log("DHCP", "dhcp request - CHADDR %s - Requested IP %s - ACCEPTED", p.CHAddr().String(), requestedIP.String())
		}
		packet := dhcp4.ReplyPacket(p, dhcp4.ACK, h.serverIP, requestedIP, randLeaseDuration(), replyOptions)
		return packet
	case dhcp4.Release, dhcp4.Decline:

		return nil
	}
	return nil
}
