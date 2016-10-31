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
	mux.HandleFunc("/api/machines/{mac}", ws.MachineDelete).Methods("DELETE")

	// mux.PathPrefix("/api/machine/").HandlerFunc(ws.NodeSetIPMI).Methods("PUT")

	// Machine variables; used in templates
	mux.PathPrefix("/api/machines/{mac}/variables").HandlerFunc(ws.MachineVariables).Methods("GET")
	mux.PathPrefix("/api/machines/{mac}/variables/{name}").HandlerFunc(ws.SetMachineVariable).Methods("PUT")
	mux.PathPrefix("/api/machines/{mac}/variables/{name}").HandlerFunc(ws.DelMachineVariable).Methods("DELETE")

	// Cluster variables; used in templates
	mux.PathPrefix("/api/variables").HandlerFunc(ws.ClusterVariablesList).Methods("GET")
	mux.PathPrefix("/api/variables/{name}").HandlerFunc(ws.SetClusterVariables).Methods("PUT")
	mux.PathPrefix("/api/variables/{name}").HandlerFunc(ws.DelClusterVariables).Methods("DELETE")

	// TODO: returning other files functionalities
	mux.PathPrefix("/files/images/").Handler(http.StripPrefix("/files/images",
		http.FileServer(http.Dir(filepath.Join(ws.ds.WorkspacePath(), "images")))))
	mux.PathPrefix("/files/").Handler(http.StripPrefix("/files/",
		http.FileServer(http.Dir(filepath.Join(ws.ds.WorkspacePath(), "files")))))

	mux.Path("/ui").Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/ui/", 302)
	}))

	mux.PathPrefix("/ui/").Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		index, _ := FSByte(false, "/static/index.html")
		w.Write(index)
	}))

	mux.PathPrefix("/static/").Handler(http.FileServer(FS(false)))

	mux.PathPrefix("/uploadworkspace/{hash}").HandlerFunc(ws.WorkspaceUploadHandler).Methods("POST")

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
