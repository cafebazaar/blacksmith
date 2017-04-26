package web // import "github.com/cafebazaar/blacksmith/web"

import (
	"io/ioutil"
	"net"
	"os"

	log "github.com/Sirupsen/logrus"

	"net/http"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/loads"
	"github.com/go-openapi/runtime"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"

	"github.com/cafebazaar/blacksmith/datasource"
	"github.com/cafebazaar/blacksmith/swagger/restapi"
	"github.com/cafebazaar/blacksmith/swagger/restapi/operations"
)

type webServer struct {
	ds *datasource.EtcdDatasource
}

// Handler uses a multiplexing router to route http requests
func (ws *webServer) Handler() http.Handler {
	mux := mux.NewRouter()
	mux.PathPrefix("/t/cc/").HandlerFunc(ws.Cloudconfig).Methods("GET")
	mux.PathPrefix("/t/ig/").HandlerFunc(ws.Ignition).Methods("GET")
	mux.PathPrefix("/t/bp/").HandlerFunc(ws.Bootparams).Methods("GET")
	return mux
}

//ServeWeb serves api of Blacksmith and a ui connected to that api
func ServeWeb(ds *datasource.EtcdDatasource, listenAddr net.TCPAddr) error {

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

func ServeSwaggerAPI(ds *datasource.EtcdDatasource, listenAddr net.TCPAddr,
	tlsCert string, tlsKey string, tlsCa string) error {
	swaggerSpec, err := loads.Analyzed(restapi.SwaggerJSON, "")
	if err != nil {
		log.Fatalln(err)
	}

	ws := &webServer{ds: ds}
	api := operations.NewSalesmanAPI(swaggerSpec)

	api.ServeError = errors.ServeError
	api.JSONConsumer = runtime.JSONConsumer()
	api.JSONProducer = runtime.JSONProducer()

	api.DeleteVariablesClusterKeyHandler = operations.DeleteVariablesClusterKeyHandlerFunc(ws.swaggerDeleteVariablesClusterKeyHandler)
	api.DeleteVariablesNodesMacKeyHandler = operations.DeleteVariablesNodesMacKeyHandlerFunc(ws.swaggerDeleteVariablesNodesMacKeyHandler)
	api.GetNodesHandler = operations.GetNodesHandlerFunc(ws.swaggerGetNodesHander)
	api.GetCloudconfigCcMacHandler = operations.GetCloudconfigCcMacHandlerFunc(ws.swaggerGetCloudconfigCcMacHander)
	api.GetCloudconfigIgMacHandler = operations.GetCloudconfigIgMacHandlerFunc(ws.swaggerGetCloudconfigIgMacHander)
	api.GetCloudconfigBpMacHandler = operations.GetCloudconfigBpMacHandlerFunc(ws.swaggerGetCloudconfigBpMacHander)
	api.GetVariablesClusterHandler = operations.GetVariablesClusterHandlerFunc(ws.swaggerGetVariablesClusterHandler)
	api.GetVariablesClusterKeyHandler = operations.GetVariablesClusterKeyHandlerFunc(ws.swaggerGetVariablesClusterKeyHandler)
	api.GetVariablesNodesMacHandler = operations.GetVariablesNodesMacHandlerFunc(ws.swaggerGetVariablesNodesMacHandler)
	api.GetVariablesNodesMacKeyHandler = operations.GetVariablesNodesMacKeyHandlerFunc(ws.swaggerGetVariablesNodesMacKeyHandler)
	api.GetWorkspaceMacHandler = operations.GetWorkspaceMacHandlerFunc(ws.swaggerGetWorkspaceMacHandler)
	api.PostWorkspaceUpdateMacHandler = operations.PostWorkspaceUpdateMacHandlerFunc(ws.swaggerPostWorkspaceUpdateMacHandler)
	api.PostWorkspaceInstallMacHandler = operations.PostWorkspaceInstallMacHandlerFunc(ws.swaggerPostWorkspaceInstallMacHandler)
	api.PostWorkspacesHandler = operations.PostWorkspacesHandlerFunc(ws.swaggerPostWorkspacesHandler)
	api.PostRebootMacHandler = operations.PostRebootMacHandlerFunc(ws.swaggerPostRebootMacHandler)
	api.PostVariablesClusterKeyHandler = operations.PostVariablesClusterKeyHandlerFunc(ws.swaggerPostVariablesClusterKeyHandler)
	api.PostVariablesNodesMacKeyHandler = operations.PostVariablesNodesMacKeyHandlerFunc(ws.swaggerPostVariablesNodesMacKeyHandler)
	api.PostHeartbeatMacHeartbeatHandler = operations.PostHeartbeatMacHeartbeatHandlerFunc(ws.swaggerPostHeartbeatMacHeartbeatHandler)

	api.ServerShutdown = func() {}
	api.Logger = log.Printf

	server := restapi.NewServer(api)
	defer server.Shutdown()
	server.TLSHost = listenAddr.IP.String()
	server.TLSPort = listenAddr.Port
	tlsCertFile := tmpFile("cert", tlsCert).Name()
	tlsKeyFile := tmpFile("cert-key", tlsKey).Name()
	tlsCaFile := tmpFile("ca", tlsCa).Name()

	server.TLSCertificate = tlsCertFile
	server.TLSCertificateKey = tlsKeyFile
	server.TLSCACertificate = tlsCaFile
	return server.Serve()
}

func tmpFile(name, content string) *os.File {
	tmpfile, err := ioutil.TempFile("", name)
	if err != nil {
		log.Fatal(err)
	}
	if _, err := tmpfile.Write([]byte(content)); err != nil {
		log.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		log.Fatal(err)
	}
	return tmpfile
}
