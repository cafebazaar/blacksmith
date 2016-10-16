package dhcp // import "github.com/cafebazaar/blacksmith/dhcp"

import (
	"bytes"
	"fmt"
	"math/rand"
	"net"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/cafebazaar/blacksmith/datasource"
	"github.com/krolaw/dhcp4"
)

const (
	minLeaseHours = 24
	maxLeaseHours = 48
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

	log.WithFields(log.Fields{
		"where":  "dhcp.StartDHCP",
		"action": "announce",
	}).Infof("Listening on %s:67 (interface: %s)", serverIP.String(), ifName)

	var err error
	if ifName != "" {
		err = dhcp4.ListenAndServeIf(ifName, handler)
	} else {
		err = dhcp4.ListenAndServe(handler)
	}

	// https://groups.google.com/forum/#!topic/coreos-user/Qbn3OdVtrZU
	if len(datasource.ClusterName()) > 50 { // 63 - 12(mac) - 1(.)
		log.WithField("where", "dhcp.StartDHCP").Warn(
			"Warning: ClusterName is too long. It may break the behaviour of the DHCP clients")
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

// dnsAddressesForDHCP returns instances. marshalled as specified in
// rfc2132 (option 6), without the length byte
func dnsAddressesForDHCP(instances *[]datasource.InstanceInfo) []byte {
	var res []byte

	for _, instanceInfo := range *instances {
		res = append(res, instanceInfo.IP.To4()...)
	}

	return res
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

	switch msgType {
	case dhcp4.Discover, dhcp4.Request:
		if server, ok := options[dhcp4.OptionServerIdentifier]; ok && !net.IP(server).Equal(h.serverIP) {
			if msgType == dhcp4.Discover {
				log.WithField("where", "dhcp.ServeDHCP").Debugf(
					"identifying dhcp server in Discover?! (%v)", p)
			}
			return nil // this message is not ours
		}

		machineInterface := h.datasource.MachineInterface(p.CHAddr())
		machine, err := machineInterface.Machine(true, nil)
		if err != nil {
			log.WithField("where", "dhcp.ServeDHCP").WithError(err).Warn(
				"failed to get machine")
			return nil
		}

		netConfStr, err := machineInterface.GetVariable(datasource.SpecialKeyNetworkConfiguration)
		if err != nil {
			log.WithField("where", "dhcp.ServeDHCP").WithError(err).Warn(
				"failed to get network configuration")
			return nil
		}

		netConf, err := datasource.UnmarshalNetworkConfiguration(netConfStr)
		if err != nil {
			log.WithField("where", "dhcp.ServeDHCP").WithError(err).Warn(
				"failed to unmarshal network-configuration=%q", netConfStr)
			return nil
		}

		instanceInfos, err := h.datasource.Instances()
		if err != nil {
			log.WithField("where", "dhcp.ServeDHCP").WithError(err).Warn(
				"failed to get instances")
			return nil
		}

		hostname := strings.Join(strings.Split(p.CHAddr().String(), ":"), "")
		hostname += "." + h.datasource.ClusterName()

		dhcpOptions := dhcp4.Options{
			dhcp4.OptionSubnetMask:       netConf.Netmask.To4(),
			dhcp4.OptionDomainNameServer: dnsAddressesForDHCP(&instanceInfos),
			dhcp4.OptionHostName:         []byte(hostname),
		}

		if netConf.Router != nil {
			dhcpOptions[dhcp4.OptionRouter] = netConf.Router.To4()
		}
		if len(netConf.ClasslessRouteOption) != 0 {
			var res []byte
			for _, part := range netConf.ClasslessRouteOption {
				res = append(res, part.ToBytes()...)
			}
			dhcpOptions[dhcp4.OptionClasslessRouteFormat] = res
		}

		responseMsgType := dhcp4.Offer
		if msgType == dhcp4.Request {
			responseMsgType = dhcp4.ACK

			requestedIP := net.IP(options[dhcp4.OptionRequestedIPAddress])
			if requestedIP == nil {
				requestedIP = net.IP(p.CIAddr())
			}
			if len(requestedIP) != 4 || requestedIP.Equal(net.IPv4zero) {
				log.WithFields(log.Fields{
					"where":   "dhcp.ServeDHCP",
					"object":  p.CHAddr().String(),
					"subject": msgType,
				}).Debugf("bad request")
				return nil
			}
			if !requestedIP.Equal(machine.IP) {
				log.WithFields(log.Fields{
					"where":   "dhcp.ServeDHCP",
					"object":  p.CHAddr().String(),
					"subject": msgType,
				}).Debugf("requestedIP(%s) != assignedIp(%s)",
					requestedIP.String(), machine.IP.String())
				return nil
			}

			machineInterface.CheckIn()
		}

		guidVal, isPxe := options[97]

		log.WithFields(log.Fields{
			"where":   "dhcp.ServeDHCP",
			"action":  "debug",
			"object":  p.CHAddr().String(),
			"subject": msgType,
		}).Infof("assignedIp=%s isPxe=%v", machine.IP.String(), isPxe)

		replyOptions := dhcpOptions.SelectOrderOrAll(options[dhcp4.OptionParameterRequestList])

		if isPxe { // this is a pxe request
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

			// TODO: Use the one on api.go
			hash, err := h.datasource.GetClusterVariable("CurrentWorkspaceHash")
			if err == nil {
				machineInterface.SetVariable("booted-workspace-hash", hash)
			}
		}
		packet := dhcp4.ReplyPacket(p, responseMsgType, h.serverIP, machine.IP,
			randLeaseDuration(), replyOptions)
		return packet

	case dhcp4.Release, dhcp4.Decline:
		return nil
	}
	return nil
}
