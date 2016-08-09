package web // import "github.com/cafebazaar/blacksmith/web"

import (
	"net"
	"net/http"
	"path/filepath"

	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"

	"github.com/cafebazaar/blacksmith/datasource"
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

	mux.HandleFunc("/api/machines", ws.MachinesList)

	// mux.PathPrefix("/api/machine/").HandlerFunc(ws.NodeSetIPMI).Methods("PUT")

	// Machine variables; used in templates
	mux.PathPrefix("/api/machines/{mac}/variables").HandlerFunc(ws.MachineVariables).Methods("GET")
	mux.PathPrefix("/api/machines/{mac}/variables/{name}/{value}").HandlerFunc(ws.SetMachineVariable).Methods("PUT")
	mux.PathPrefix("/api/machines/{mac}/variables/{name}").HandlerFunc(ws.DelMachineVariable).Methods("DELETE")

	// Cluster variables; used in templates
	mux.PathPrefix("/api/variables").HandlerFunc(ws.ClusterVariablesList).Methods("GET")
	mux.PathPrefix("/api/variables").HandlerFunc(ws.SetVariable).Methods("PUT")
	mux.PathPrefix("/api/variables").HandlerFunc(ws.DelVariable).Methods("DELETE")

	// TODO: returning other files functionalities
	mux.PathPrefix("/files/").Handler(http.StripPrefix("/files/",
		http.FileServer(http.Dir(filepath.Join(ws.ds.WorkspacePath(), "files")))))

	mux.PathPrefix("/ui/").Handler(http.FileServer(FS(false)))

	return mux
}

func logHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.ServeHTTP(w, r)
	})
}

//ServeWeb serves api of Blacksmith and a ui connected to that api
func ServeWeb(ds datasource.DataSource, listenAddr net.TCPAddr) error {
	r := &webServer{ds: ds}

	logWriter := log.StandardLogger().Writer()
	defer logWriter.Close()

	loggedRouter := handlers.LoggingHandler(logWriter, r.Handler())
	s := &http.Server{
		Addr:    listenAddr.String(),
		Handler: loggedRouter,
	}

	log.WithFields(log.Fields{
		"where":  "web.ServeWeb",
		"action": "announce",
	}).Infof("Listening on %s", listenAddr.String())

	return s.ListenAndServe()
}
