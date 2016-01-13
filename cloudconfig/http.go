package cloudconfig

//
// import (
// 	"net"
// 	"strings"
// 	//	"net/url"
// 	//	"fmt"
// 	"net/http"
// 	"path"
// 	"sync"
// 	"text/template"
//
// 	"github.com/cafebazaar/blacksmith/datasource"
// 	"github.com/cafebazaar/blacksmith/logging"
// )
//
// //cloudConfigDataSourceWrapper embedds a CloudConfigDataSource which is an
// //interface and provides a means of conceptually using the interface as the
// //method receiver
// type cloudConfigDataSource struct {
// 	datasource.KeyValueGeneralDataSource
// 	executeLock         *sync.Mutex
// 	cloudConfigTemplate *template.Template
// }
//
// func (datasource *cloudConfigDataSource) handler(w http.ResponseWriter, r *http.Request) {
// 	req := strings.Split(r.URL.Path, "/")[1:]
//
// 	queryMap, _ := extractQueries(r.URL.RawQuery)
//
// 	if len(req) != 2 {
// 		logging.Log("CLOUDCONFIG", "Received request - request not found")
// 		http.NotFound(w, r)
// 		return
// 	}
//
// 	if req[0] != "cloud" {
// 		//No ignition support for now
// 		http.NotFound(w, r)
// 		return
// 	}
//
// 	logging.Log("REFACT CLOUDCONFIG", "cloud request ! ! !")
//
// 	clientIP, _, err := net.SplitHostPort(r.RemoteAddr)
// 	if err != nil {
// 		http.Error(w, "internal server error - parsing host and port", 500)
// 		logging.Log("CLOUDCONFIG", "Error - %s with mac %s - %s", req[0], req[1], err.Error())
// 		return
// 	}
//
// 	clientMacAddress := req[1]
// 	datasource.executeLock.Lock()
// 	defer datasource.executeLock.Unlock()
// 	config, err := datasource.macCloudConfig(clientMacAddress)
// 	if err != nil {
// 		http.Error(w, "internal server error - error in generating config", 500)
// 		logging.Log("CLOUDCONFIG", "Error when generating config - %s with mac %s - %s", req[0], req[1], err.Error())
// 		return
// 	}
// 	w.Header().Set("Content-Type", "application/x-yaml")
//
// 	//always validate the cloudconfig. Don't if explicitly stated.
// 	if value, exists := queryMap["validate"]; !exists || value != "false" {
// 		config += validateCloudConfig(config)
// 	}
//
// 	w.Write([]byte(config))
// 	logging.Log("CLOUDCONFIG", "Received request - %s with mac %s", req[0], req[1])
// }
//
// func extractQueries(rawQueryString string) (map[string]string, error) {
// 	// queries for which the value is not specified will be stored as : "queryKey" -> "true"
// 	queries := strings.Split(rawQueryString, "&") // Ampersand separated queries
// 	retMap := make(map[string]string)
// 	for _, q := range queries {
// 		equalSignIndex := strings.Index(q, "=")
// 		var key, value string
// 		if equalSignIndex == -1 { // no value, setting to true
// 			key = q
// 			value = "true"
// 		} else { // key=value
// 			key = q[:equalSignIndex]
// 			value = q[equalSignIndex+1:]
// 		}
// 		retMap[key] = value
// 	}
// 	return retMap, nil
// }
//
// func serveUtilityMultiplexer(datasource datasource.KeyValueGeneralDataSource) *http.ServeMux {
// 	mux := http.NewServeMux()
//
// 	mux.HandleFunc("/", datasource.handler)
// 	return mux
// }
//
// //ServeCloudConfig is run cuncurrently alongside other blacksmith services
// //Provides cloudconfig to machines at boot time
// func ServeCloudConfig(listenAddr net.TCPAddr, workspacePath string, datasource datasource.KeyValueGeneralDataSource) error {
// 	logging.Log("CLOUDCONFIG", "Listening on %s", listenAddr.String())
//
// 	cctemplates, err := FromPath(datasource, path.Join(datasource.WorkspacePath(), "config/cloudconfig"))
// 	ccdataSource := cloudConfigDataSource{datasource, &sync.Mutex, cctemplates}
//
// 	return http.ListenAndServe(listenAddr.String(), serveUtilityMultiplexer(datasource))
// }
