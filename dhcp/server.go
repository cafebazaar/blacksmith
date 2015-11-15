package dhcp // import "github.com/cafebazaar/aghajoon/dhcp"

import (
	"bytes"
	"net"
	"time"

	"github.com/cafebazaar/aghajoon/logging"
	"github.com/krolaw/dhcp4"
)

type DHCPSetting struct {
	IFName        string
	LeaseDuration time.Duration // TTL of this lease range
	ServerIP      net.IP
	RouterAddr    net.IP
	SubnetMask    net.IP
	DNSAddr       net.IP
}

func ServeDHCP(settings *DHCPSetting, leasePool *LeasePool) error {
	handler, err := newDHCPHandler(settings, leasePool)
	if err != nil {
		logging.Debug("DHCP", "Error in connecting etcd - %s", err.Error())
		return err
	}
	logging.Log("DHCP", "Listening on :67 - with server IP %s", settings.ServerIP.String())
	if settings.IFName != "" {
		err = dhcp4.ListenAndServeIf(settings.IFName, handler)
	} else {
		err = dhcp4.ListenAndServe(handler)
	}
	if err != nil {
		logging.Debug("DHCP", "Error in server - %s", err.Error())
	}
	return err
}

// DHCP handler that passed to dhcp4 package

type DHCPHandler struct {
	settings    *DHCPSetting
	leasePool   *LeasePool
	dhcpOptions dhcp4.Options
}

func newDHCPHandler(settings *DHCPSetting, leasePool *LeasePool) (*DHCPHandler, error) {
	h := &DHCPHandler{
		settings: settings,
	}
	h.dhcpOptions = dhcp4.Options{
		dhcp4.OptionSubnetMask:       []byte{255, 255, 255, 0},
		dhcp4.OptionRouter:           []byte(settings.RouterAddr),
		dhcp4.OptionDomainNameServer: []byte(settings.DNSAddr),
	}
	var err error
	h.leasePool = leasePool
	if err != nil {
		return nil, err
	}
	return h, nil
}

func (h *DHCPHandler) ServeDHCP(p dhcp4.Packet, msgType dhcp4.MessageType, options dhcp4.Options) (d dhcp4.Packet) {
	switch msgType {
	case dhcp4.Discover:
		guidVal, is_pxe := options[97]

		ip, err := h.leasePool.Assign(p.CHAddr().String())
		if err != nil {
			logging.Debug("DHCP", "err in lease pool - %s", err.Error())
			return nil // pool is full
		}
		replyOptions := h.dhcpOptions.SelectOrderOrAll(options[dhcp4.OptionParameterRequestList])

		// this is a pxe request
		if is_pxe {
			logging.Log("DHCP", "dhcp discover with PXE - CHADDR %s - IP %s", p.CHAddr().String(), ip.String())
			guid := guidVal[1:]
			replyOptions = append(replyOptions, dhcp4.Option{Code: 60, Value: []byte("PXEClient")})
			replyOptions = append(replyOptions, dhcp4.Option{Code: 97, Value: guid})

			// PXE vendor options
			var pxe bytes.Buffer
			// Discovery Control - disable broadcast and multicast boot server discovery
			pxe.Write([]byte{6, 1, 3})
			// PXE boot server
			pxe.Write([]byte{8, 7, 0x80, 0x00, 1})
			pxe.Write(h.settings.ServerIP)
			// PXE boot menu - one entry, pointing to the above PXE boot server
			pxe.Write([]byte{9, 12, 0x80, 0x00, 9})
			pxe.WriteString("aghjo-0.1")
			// PXE menu prompt+timeout
			pxe.Write([]byte{10, 10, 0})
			pxe.WriteString("aghjo-0.1")
			// End vendor options
			pxe.WriteByte(255)
			replyOptions = append(replyOptions, dhcp4.Option{Code: 43, Value: pxe.Bytes()})
		} else {
			logging.Log("DHCP", "dhcp discover - CHADDR %s - IP %s", p.CHAddr().String(), ip.String())
		}

		packet := dhcp4.ReplyPacket(p, dhcp4.Offer, h.settings.ServerIP, ip, h.settings.LeaseDuration, replyOptions)
		return packet
	case dhcp4.Request:
		if server, ok := options[dhcp4.OptionServerIdentifier]; ok && !net.IP(server).Equal(h.settings.ServerIP) {
			return nil // this message is not ours
		}
		requestedIP := net.IP(options[dhcp4.OptionRequestedIPAddress])
		if requestedIP == nil {
			requestedIP = net.IP(p.CIAddr())
		}
		if len(requestedIP) == 4 && !requestedIP.Equal(net.IPv4zero) {
			_, err := h.leasePool.Request(p.CHAddr().String(), requestedIP)
			if err != nil {
				goto nomatch
			}
			logging.Debug("DHCP", "dhcp request - CHADDR %s - Requested IP %s - ACCEPTED", p.CHAddr().String(), requestedIP.String())
			return dhcp4.ReplyPacket(p, dhcp4.ACK, h.settings.ServerIP, net.IP(options[dhcp4.OptionRequestedIPAddress]), h.settings.LeaseDuration,
				h.dhcpOptions.SelectOrderOrAll(options[dhcp4.OptionParameterRequestList]))
		}
	nomatch:
		logging.Debug("DHCP", "dhcp request - CHADDR %s - Requested IP %s - NO MATCH", p.CHAddr().String(), requestedIP.String())
		return dhcp4.ReplyPacket(p, dhcp4.NAK, h.settings.ServerIP, nil, 0, nil)
	case dhcp4.Release, dhcp4.Decline:
		return nil
	}
	return nil
}
