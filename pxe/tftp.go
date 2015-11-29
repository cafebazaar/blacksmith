package pxe

import (
	"net"

	"github.com/cafebazaar/aghajoon/logging"
	"github.com/danderson/pixiecore/tftp"
)

func ServeTFTP(listenAddr net.UDPAddr) error {
	pxelinux, err := Asset("pxelinux/lpxelinux.0")
	if err != nil {
		return err
	}
	tftp.Log = func(msg string, args ...interface{}) { logging.Log("TFTP", msg, args...) }
	tftp.Debug = func(msg string, args ...interface{}) { logging.Debug("TFTP", msg, args...) }
	return tftp.ListenAndServe("udp4", listenAddr.String(), tftp.Blob(pxelinux))
}
