package pxe

import (
	"io"
	"net"

	log "github.com/Sirupsen/logrus"
	"go.universe.tf/netboot/tftp"
)

func ServeTFTP(listenAddr net.UDPAddr) error {
	pxelinuxDir := FS(false)
	tftp.Logf = func(msg string, args ...interface{}) {
		log.WithFields(log.Fields{
			"where":  "pxe.ServeTFTP",
			"action": "tftp-said",
		}).Infof(msg, args...)
	}
	tftp.Debug = func(msg string, args ...interface{}) {
		log.WithFields(log.Fields{
			"where":  "pxe.ServeTFTP",
			"action": "tftp-said",
		}).Debugf(msg, args...)
	}

	handler := func(string, net.Addr) (io.ReadCloser, error) {
		pxelinux, err := pxelinuxDir.Open("/pxelinux/lpxelinux.0")
		if err != nil {
			return nil, err
		}
		return pxelinux, nil
	}

	return tftp.ListenAndServe("udp4", listenAddr.String(), handler)
}
