package web // import "github.com/cafebazaar/blacksmith/web"

import (
	"net"
	"net/http"
	"time"

	"github.com/cafebazaar/blacksmith/datasource"
)

//ServeWeb serves api of Blacksmith and a ui connected to that api
func ServeWeb(rest datasource.RestServer, listenAddr net.TCPAddr) error {
	s := &http.Server{
		Addr:           listenAddr.String(),
		Handler:        rest.Handler(),
		ReadTimeout:    100 * time.Second,
		WriteTimeout:   100 * time.Second,
		MaxHeaderBytes: 1 << 30,
	}

	return s.ListenAndServe()
}
