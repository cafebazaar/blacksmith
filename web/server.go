package web // import "github.com/cafebazaar/blacksmith/web"

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/elazarl/go-bindata-assetfs"

	"github.com/cafebazaar/blacksmith/datasource"
	"github.com/cafebazaar/blacksmith/dhcp"
	"github.com/gorilla/mux"
)

//ServeWeb serves api of Blacksmith and a ui connected to that api
func ServeWeb(rest RestServer, listenAddr net.TCPAddr) error {
	s := &http.Server{
		Addr:           listenAddr.String(),
		Handler:        rest.Handler(),
		ReadTimeout:    100 * time.Second,
		WriteTimeout:   100 * time.Second,
		MaxHeaderBytes: 1 << 30,
	}

	return s.ListenAndServe()
}
