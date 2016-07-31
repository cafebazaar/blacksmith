package web // import "github.com/cafebazaar/blacksmith/web"

import (
	"net"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"

	"github.com/cafebazaar/blacksmith/datasource"
	"github.com/cafebazaar/blacksmith/logging"
)

type webServer struct {
	ds datasource.DataSource
}

// Handler uses a multiplexing router to route http requests
func (ws *webServer) Handler() http.Handler {
	mux := mux.NewRouter()

	mux.PathPrefix("/t/cc/").HandlerFunc(ws.Cloudconfig).Methods("GET")
	mux.PathPrefix("/t/ig/").HandlerFunc(ws.Ignition).Methods("GET")
	mux.PathPrefix("/t/bp/").HandlerFunc(ws.Bootparams).Methods("GET")

	mux.HandleFunc("/api/version", ws.Version)

	mux.HandleFunc("/api/nodes", ws.NodesList)

	mux.PathPrefix("/api/node/").HandlerFunc(ws.NodeSetIPMI).Methods("PUT")

	// Machine variables; used in templates
	// TODO: refactor the names (/api/flag/ -> /api/node/)
	mux.PathPrefix("/api/node/").HandlerFunc(ws.NodeFlags).Methods("GET")
	mux.PathPrefix("/api/flag/").HandlerFunc(ws.SetFlag).Methods("PUT")
	mux.PathPrefix("/api/flag/").HandlerFunc(ws.DelFlag).Methods("DELETE")

	// Cluster variables; used in templates
	mux.PathPrefix("/api/variables").HandlerFunc(ws.ClusterVariablesList).Methods("GET")
	mux.PathPrefix("/api/variables").HandlerFunc(ws.SetVariable).Methods("PUT")
	mux.PathPrefix("/api/variables").HandlerFunc(ws.DelVariable).Methods("DELETE")

	// Configuration variables; used inside blacksmith
	mux.PathPrefix("/api/configuration").HandlerFunc(ws.ConfigurationList).Methods("GET")
	mux.PathPrefix("/api/configuration").HandlerFunc(ws.SetConfiguration).Methods("PUT")
	mux.PathPrefix("/api/configuration").HandlerFunc(ws.DelConfiguration).Methods("DELETE")

	mux.HandleFunc("/upload/", ws.Upload)
	mux.HandleFunc("/files", ws.Files).Methods("GET")
	mux.HandleFunc("/files", ws.DeleteFile).Methods("DELETE")
	mux.PathPrefix("/files/").Handler(http.StripPrefix("/files/",
		http.FileServer(http.Dir(filepath.Join(ws.ds.WorkspacePath(), "files")))))

	mux.PathPrefix("/ui/").Handler(http.FileServer(FS(false)))

	return mux
}

//ServeWeb serves api of Blacksmith and a ui connected to that api
func ServeWeb(ds datasource.DataSource, listenAddr net.TCPAddr) error {
	r := &webServer{ds: ds}
	loggedRouter := handlers.LoggingHandler(os.Stdout, r.Handler())
	s := &http.Server{
		Addr:    listenAddr.String(),
		Handler: loggedRouter,
	}

	logging.Log("WEB", "Listening on %s", listenAddr.String())

	return s.ListenAndServe()
}
