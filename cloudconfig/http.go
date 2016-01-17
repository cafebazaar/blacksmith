package cloudconfig

import (
	"net"
	"strings"
	//	"net/url"
	//	"fmt"
	"bytes"
	"net/http"
	"path"
	"sync"
	"text/template"

	"github.com/cafebazaar/blacksmith/datasource"
	"github.com/cafebazaar/blacksmith/logging"
)

//cloudConfigDataSource embedds a CloudConfigDataSource which is an
//interface and provides a means of conceptually using the interface as the
//method receiver

type cloudConfigDataSource struct {
	datasource.GeneralDataSource
	executeLock    *sync.Mutex
	templates      *template.Template
	currentMachine datasource.Machine
}

type bootParamsDataSource struct {
	datasource.GeneralDataSource
	executeLock    *sync.Mutex
	templates      *template.Template
	currentMachine datasource.Machine
}

func (datasource *cloudConfigDataSource) handler(w http.ResponseWriter, r *http.Request) {
	req := strings.Split(r.URL.Path, "/")[1:]

	queryMap, _ := extractQueries(r.URL.RawQuery)

	if len(req) != 2 {
		logging.Log("CLOUDCONFIG", "Received request - request not found")
		http.NotFound(w, r)
		return
	}

	if req[0] != "cloud" {
		//No ignition support for now
		http.NotFound(w, r)
		return
	}

	logging.Log("REFACT CLOUDCONFIG", "cloud request ! ! !")

	// clientIP, _, err := net.SplitHostPort(r.RemoteAddr)
	// if err != nil {
	// 	http.Error(w, "internal server error - parsing host and port", 500)
	// 	logging.Log("CLOUDCONFIG", "Error - %s with mac %s - %s", req[0], req[1], err.Error())
	// 	return
	// }

	clientMacAddressString := req[1]
	if strings.Index(clientMacAddressString, ":") == -1 {
		var tmpmac bytes.Buffer
		for i := 0; i < 12; i++ { // mac address length
			tmpmac.WriteString(clientMacAddressString[i : i+1])
			if i%2 == 1 {
				tmpmac.WriteString(":")
			}
		}
		clientMacAddressString = tmpmac.String()[:len(tmpmac.String())-1]
	}
	// logging.Log("#CLOUD", clientMacAddressString)
	clientMac, err := net.ParseMAC(clientMacAddressString)
	if err != nil {
		return
	}
	machine, exist := datasource.GeneralDataSource.GetMachine(clientMac)
	if !exist {
		return
	}
	theIp, _ := machine.IP()
	logging.Log("#CLOUD", theIp.String())
	datasource.currentMachine = machine
	datasource.executeLock.Lock()
	defer datasource.executeLock.Unlock()
	config, err := datasource.macCloudConfig(clientMacAddressString)
	if err != nil {
		http.Error(w, "internal server error - error in generating config", 500)
		logging.Log("CLOUDCONFIG", "Error when generating config - %s with mac %s - %s", req[0], req[1], err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/x-yaml")

	//always validate the cloudconfig. Don't if explicitly stated.
	if value, exists := queryMap["validate"]; !exists || value != "false" {
		config += validateCloudConfig(config)
	}

	w.Write([]byte(config))
	logging.Log("CLOUDCONFIG", "Received request - %s with mac %s", req[0], req[1])
}

func extractQueries(rawQueryString string) (map[string]string, error) {
	// queries for which the value is not specified will be stored as : "queryKey" -> "true"
	queries := strings.Split(rawQueryString, "&") // Ampersand separated queries
	retMap := make(map[string]string)
	for _, q := range queries {
		equalSignIndex := strings.Index(q, "=")
		var key, value string
		if equalSignIndex == -1 { // no value, setting to true
			key = q
			value = "true"
		} else { // key=value
			key = q[:equalSignIndex]
			value = q[equalSignIndex+1:]
		}
		retMap[key] = value
	}
	return retMap, nil
}

func serveUtilityMultiplexer(datasource cloudConfigDataSource) *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("/", datasource.handler)
	return mux
}

//ServeCloudConfig is run cuncurrently alongside other blacksmith services
//Provides cloudconfig to machines at boot time
func ServeCloudConfig(listenAddr net.TCPAddr, workspacePath string, datasource datasource.GeneralDataSource) error {
	logging.Log("CLOUDCONFIG", "Listening on %s", listenAddr.String())

	cctemplates, err := FromPath(datasource, path.Join(datasource.WorkspacePath(), "config/cloudconfig"))
	if err != nil {
		return err
	}
	ccdataSource := cloudConfigDataSource{datasource, &sync.Mutex{}, cctemplates, nil}

	return http.ListenAndServe(listenAddr.String(), serveUtilityMultiplexer(ccdataSource))
}
