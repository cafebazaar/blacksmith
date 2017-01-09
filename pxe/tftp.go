package pxe

import (
	"io"
	"net"

	log "github.com/Sirupsen/logrus"
	"go.universe.tf/netboot/tftp"
)

// ServeTFTP delivers lpxelinux.0 to machines through the old tftp protocol
func ServeTFTP(listenAddr net.UDPAddr) error {
	pxelinuxDir := FS(false)

	handler := func(string, net.Addr) (io.ReadCloser, int64, error) {
		pxelinux, err := pxelinuxDir.Open("/pxelinux/lpxelinux.0")
		if err != nil {
			return nil, 0, err
		}
		stat, err := pxelinux.Stat()
		if err != nil {
			return nil, 0, err
		}
		return pxelinux, stat.Size(), nil
	}

	tftpServer := &tftp.Server{
		Handler: handler,
		InfoLog: func(msg string) {
			log.WithFields(log.Fields{
				"where":  "pxe.ServeTFTP",
				"action": "tftp",
			}).Infof(msg)
		},
		TransferLog: func(clientAddr net.Addr, path string, err error) {
			if err != nil {
				log.WithError(err).WithFields(log.Fields{
					"where":   "pxe.ServeTFTP",
					"action":  "tftp-transfer",
					"subject": path,
				}).Warnf("error while transferring to %s", clientAddr.String())
			} else {
				log.WithFields(log.Fields{
					"where":   "pxe.ServeTFTP",
					"action":  "tftp-transfer",
					"subject": path,
				}).Debugf("transferred to %s", clientAddr.String())
			}
		},
	}

	return tftpServer.ListenAndServe(listenAddr.String())
}
