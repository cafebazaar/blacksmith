package web // import "github.com/cafebazaar/blacksmith/web"

import (
	"net"

	log "github.com/Sirupsen/logrus"

	"net/http"

	"github.com/go-openapi/loads"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"

	"github.com/cafebazaar/blacksmith/datasource"
	"github.com/cafebazaar/blacksmith/swagger/restapi"
	"github.com/cafebazaar/blacksmith/swagger/restapi/operations"
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

	// Machine variables; used in templates
	mux.PathPrefix("/api/machines/{mac}/variables").HandlerFunc(ws.MachineVariables).Methods("GET")
	mux.PathPrefix("/api/machines/{mac}/variables/{name}").HandlerFunc(ws.SetMachineVariable).Methods("PUT")
	mux.PathPrefix("/api/machines/{mac}/variables/{name}").HandlerFunc(ws.DelMachineVariable).Methods("DELETE")

	// Cluster variables; used in templates
	mux.PathPrefix("/api/variables").HandlerFunc(ws.ClusterVariablesList).Methods("GET")
	mux.PathPrefix("/api/variables/{name}").HandlerFunc(ws.SetClusterVariables).Methods("PUT")
	mux.PathPrefix("/api/variables/{name}").HandlerFunc(ws.DelClusterVariables).Methods("DELETE")

	mux.PathPrefix("/api/update").HandlerFunc(ws.UpdateWorkspace).Methods("POST")
	mux.PathPrefix("/api/update").HandlerFunc(ws.GetWorkspaceHash).Methods("GET")

	// TODO: returning other files functionalities
	// mux.PathPrefix("/files/").Handler(http.StripPrefix("/files/",
	// http.FileServer(http.Dir(filepath.Join(ws.ds.WorkspacePath(), "files")))))

	mux.Path("/ui").Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/ui/", 302)
	}))

	mux.PathPrefix("/ui/").Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		index, _ := FSByte(false, "/static/index.html")
		w.Write(index)
	}))

	mux.PathPrefix("/static/").Handler(http.FileServer(FS(false)))

	return mux
}

func logHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.ServeHTTP(w, r)
	})
}

//ServeWeb serves api of Blacksmith and a ui connected to that api
func ServeWeb(ds datasource.DataSource, listenAddr net.TCPAddr) error {

	ws := &webServer{ds: ds}
	log.WithFields(log.Fields{
		"where":  "web.ServeWeb",
		"action": "announce",
	}).Infof("Listening on %s", listenAddr.String())

	logWriter := log.StandardLogger().Writer()
	defer logWriter.Close()

	loggedRouter := handlers.LoggingHandler(logWriter, ws.Handler())
	s := &http.Server{
		Addr:    listenAddr.String(),
		Handler: loggedRouter,
	}

	return s.ListenAndServe()
}

func ServeSwaggerAPI(ds datasource.DataSource, listenAddr net.TCPAddr) error {

	ws := &webServer{ds: ds}

	swaggerSpec, err := loads.Analyzed(restapi.SwaggerJSON, "")
	if err != nil {
		log.Fatalln(err)
	}

	api := operations.NewSalesmanAPI(swaggerSpec)

	api.DeleteVariablesClusterKeyHandler = operations.DeleteVariablesClusterKeyHandlerFunc(ws.swaggerDeleteVariablesClusterKeyHandler)
	api.DeleteVariablesNodesMacKeyHandler = operations.DeleteVariablesNodesMacKeyHandlerFunc(ws.swaggerDeleteVariablesNodesMacKeyHandler)
	api.GetNodesHandler = operations.GetNodesHandlerFunc(ws.swaggerGetNodesHander)
	api.GetVariablesClusterHandler = operations.GetVariablesClusterHandlerFunc(ws.swaggerGetVariablesClusterHandler)
	api.GetVariablesClusterKeyHandler = operations.GetVariablesClusterKeyHandlerFunc(ws.swaggerGetVariablesClusterKeyHandler)
	api.GetVariablesNodesMacHandler = operations.GetVariablesNodesMacHandlerFunc(ws.swaggerGetVariablesNodesMacHandler)
	api.GetVariablesNodesMacKeyHandler = operations.GetVariablesNodesMacKeyHandlerFunc(ws.swaggerGetVariablesNodesMacKeyHandler)
	api.GetWorkspaceHandler = operations.GetWorkspaceHandlerFunc(ws.swaggerGetWorkspaceHandler)
	api.PostVariablesClusterKeyHandler = operations.PostVariablesClusterKeyHandlerFunc(ws.swaggerPostVariablesClusterKeyHandler)
	api.PostVariablesNodesMacKeyHandler = operations.PostVariablesNodesMacKeyHandlerFunc(ws.swaggerPostVariablesNodesMacKeyHandler)
	api.PostWorkspaceHandler = operations.PostWorkspaceHandlerFunc(ws.swaggerPostWorkspaceHandler)

	server := restapi.NewServer(api)
	server.Port = listenAddr.Port
	defer server.Shutdown()
	return server.Serve()
}
