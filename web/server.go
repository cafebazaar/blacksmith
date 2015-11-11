package web

import (
	"github.com/cafebazaar/aghajoon/dhcp"
	"io"
	"net"
	"net/http"
	"encoding/json"
)

type RestServer struct {
	pool *dhcp.LeasePool
	uiPath *string
}

func NewRest(leasePool *dhcp.LeasePool, uiPath *string) *RestServer {
	return &RestServer{
		pool: leasePool,
		uiPath: uiPath,
	}
}

func (a *RestServer) Mux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/nodes", a.nodesList)
	
	if *a.uiPath != "" {
		ui := http.FileServer(http.Dir(*a.uiPath))	
		mux.Handle("/ui/", http.StripPrefix("/ui/", ui))
	}
	
	return mux
}

func (a *RestServer) nodesList(w http.ResponseWriter, r *http.Request) {
	leases, err := a.pool.Leases()
	if err != nil {
		panic(err)
	}
	json, _ := json.Marshal(leases)
	io.WriteString(w, string(json))
}

func ServeWeb(rest *RestServer, listenAddr net.TCPAddr) error {
	mux := rest.Mux()
	return http.ListenAndServe(listenAddr.String(), mux)
}
