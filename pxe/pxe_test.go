package pxe

import (
	"net"
	"testing"
)

func TestReplyPXE(t *testing.T) {
	pxePacket := &PXEPacket{
		DHCPPacket: DHCPPacket{
			TID: ([]byte)("1"),
			MAC: net.HardwareAddr("123456"),
		},
		ClientIP: net.IP("1234"),
	}

	response := ReplyPXE(pxePacket)
	n := len(response)
	if response[n-1] != 255 {
		t.Errorf("PXE packets end with 255, this one ends with %d", response[n-1])
	}
}
