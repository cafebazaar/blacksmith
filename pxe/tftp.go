package pxe

import (
	"io"
	"net"

	"github.com/cafebazaar/blacksmith/logging"
	"github.com/danderson/pixiecore/tftp"
)

func ServeTFTP(listenAddr net.UDPAddr) error {
	pxelinuxDir := FS(false)
	pxelinux, err := pxelinuxDir.Open("/pxelinux/lpxelinux.0")
	if err != nil {
		return err
	}
	tftp.Log = func(msg string, args ...interface{}) { logging.Log("TFTP", msg, args...) }
	tftp.Debug = func(msg string, args ...interface{}) { logging.Debug("TFTP", msg, args...) }

	handler := func(string, net.Addr) (io.ReadCloser, error) {
		return pxelinux, nil
	}

	return tftp.ListenAndServe("udp4", listenAddr.String(), handler)
}
