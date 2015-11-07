package pxe

import (
	"aghajoon/logging"
	"bytes"
	"errors"
	"fmt"
	"golang.org/x/net/ipv4"
	"net"
)

var dhcpMagic = []byte{99, 130, 83, 99}

type DHCPPacket struct {
	TID  []byte
	MAC  net.HardwareAddr
	GUID []byte

	ServerIP net.IP
}

type PXEPacket struct {
	DHCPPacket
	ClientIP net.IP
	// The boot type requested by the client. We need to mirror this
	// in the PXE reply.
	BootType []byte

	HTTPServer string
}

func dhcpOption(b []byte) (typ byte, val []byte, next []byte) {
	if len(b) < 2 || b[0] == 255 {
		return 255, nil, nil
	}
	typ, l := b[0], int(b[1])
	if len(b) < l+2 {
		return 255, nil, nil
	}
	return typ, b[2 : 2+l], b[2+l:]
}

func ServePXE(listenAddr net.UDPAddr, serverIP net.IP, httpAddr net.TCPAddr) error {
	conn, err := net.ListenPacket("udp4", listenAddr.String())
	if err != nil {
		return err
	}
	defer conn.Close()
	l := ipv4.NewPacketConn(conn)
	if err = l.SetControlMessage(ipv4.FlagInterface, true); err != nil {
		return err
	}

	logging.Log("PXE", "Listening on %s", listenAddr.String())
	buf := make([]byte, 1024)
	for {
		n, msg, addr, err := l.ReadFrom(buf)
		if err != nil {
			logging.Log("PXE", "Error reading from socket: %s", err)
			continue
		}

		req, err := ParsePXE(buf[:n])
		if err != nil {
			logging.Debug("PXE", "ParsePXE: %s", err)
			continue
		}

		req.ServerIP = serverIP
		req.HTTPServer = fmt.Sprintf("http://%s/", httpAddr.String())

		logging.Log("PXE", "Chainloading %s (%s) to pxelinux (via %s)", req.MAC, req.ClientIP, req.ServerIP)

		if _, err := l.WriteTo(ReplyPXE(req), &ipv4.ControlMessage{
			IfIndex: msg.IfIndex,
		}, addr); err != nil {
			logging.Log("PXE", "Responding to %s: %s", req.MAC, err)
			continue
		}
	}
}

func ReplyPXE(p *PXEPacket) []byte {
	var b bytes.Buffer

	// Fixed length BOOTP response
	var bootp [236]byte
	bootp[0] = 2     // BOOTP reply
	bootp[1] = 1     // PHY = ethernet
	bootp[2] = 6     // Hardware address length
	bootp[10] = 0x80 // Please speak broadcast
	copy(bootp[4:], p.TID)
	copy(bootp[16:], p.ClientIP)
	copy(bootp[20:], p.ServerIP)
	copy(bootp[28:], p.MAC)
	// Boot file name. Our TFTP server unconditionally serves up
	// pxelinux no matter the name, so we just put something that
	// looks nice in packet dumps.
	copy(bootp[108:], "boot")
	b.Write(bootp[:])

	// DHCP magic
	b.Write(dhcpMagic)
	// Type = DHCPACK
	b.Write([]byte{53, 1, 5})
	// Server ID
	b.Write([]byte{54, 4})
	b.Write(p.ServerIP)
	// Vendor class
	b.Write([]byte{60, 9})
	b.WriteString("PXEClient")
	// Client UUID
	b.Write([]byte{97, 17, 0})
	b.Write(p.GUID)
	// Mirror the menu selection back at the client
	b.Write([]byte{43, 7, 71, 4})
	b.Write(p.BootType)
	b.WriteByte(255)
	// Pxelinux path prefix, which makes pxelinux use HTTP for
	// everything.
	b.Write([]byte{210, byte(len(p.HTTPServer))})
	b.WriteString(p.HTTPServer)
	// If boot fails, make pxelinux reboot after 5 seconds to try
	// again.
	b.Write([]byte{211, 4, 0, 0, 0, 5})

	// End DHCP options
	b.WriteByte(255)

	return b.Bytes()
}

func ParsePXE(b []byte) (req *PXEPacket, err error) {
	if len(b) < 240 {
		return nil, errors.New("packet too short")
	}

	ret := &PXEPacket{
		DHCPPacket: DHCPPacket{
			TID: b[4:8],
			MAC: net.HardwareAddr(b[28:34]),
		},
		ClientIP: net.IP(b[12:16]),
	}

	// We do lighter packet verification here, because the PXE port
	// should not have random unrelated traffic on it, and if there
	// is, the clients deserve everything they get.
	if !bytes.Equal(b[236:240], dhcpMagic) {
		return nil, fmt.Errorf("packet from %s (%s) is not a DHCP request", ret.MAC, ret.ClientIP)
	}

	typ, val, opts := dhcpOption(b[240:])
	for typ != 255 {
		switch typ {
		case 43:
			pxeTyp, pxeVal, val := dhcpOption(val)
			for pxeTyp != 255 {
				if pxeTyp == 71 {
					ret.BootType = pxeVal
					break
				}
				pxeTyp, pxeVal, val = dhcpOption(val)
			}
		case 97:
			if len(val) != 17 || val[0] != 0 {
				return nil, fmt.Errorf("packet from %s (%s) has malformed option 97", ret.MAC, ret.ClientIP)
			}
			ret.GUID = val[1:]
		}
		typ, val, opts = dhcpOption(opts)
	}

	if ret.GUID == nil {
		return nil, fmt.Errorf("%s (%s) is not a PXE client", ret.MAC, ret.ClientIP)
	}
	if ret.BootType == nil {
		return nil, fmt.Errorf("%s (%s) hasn't selected a menu option", ret.MAC, ret.ClientIP)
	}

	// Valid PXE request!
	return ret, nil
}
