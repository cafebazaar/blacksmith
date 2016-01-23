package cloudconfig

import (
	"bytes"
	"net"
	"strings"
	//	"net/url"
	//	"fmt"

	"net/http"
	"path"
	"sync"
	"text/template"

	"github.com/cafebazaar/blacksmith/datasource"
	"github.com/cafebazaar/blacksmith/logging"
)

const (
	debugTag = "CLOUDCONFIG"
)

// cloudConfigDataSource embedds a CloudConfigDataSource which is an
// interface and provides a means of conceptually using the interface as the
// method receiver
type cloudConfigDataSource struct {
	datasource.GeneralDataSource
	executeLock       *sync.Mutex
	templates         *template.Template
	ignitionTemplates *template.Template
	currentMachine    datasource.Machine
}

type bootParamsDataSource struct {
	datasource.GeneralDataSource
	executeLock    *sync.Mutex
	templates      *template.Template
	currentMachine datasource.Machine
}

func (datasource *cloudConfigDataSource) handler(w http.ResponseWriter, r *http.Request) {
	logging.LogHTTPRequest(debugTag, r)

	req := strings.Split(r.URL.Path, "/")[1:]

	queryMap, _ := extractQueries(r.URL.RawQuery)

	if len(req) != 2 {
		logging.Log(debugTag, "Received request - request not found")
		http.NotFound(w, r)
		return
	}

	if req[0] != "cloud" && req[0] != "ignition" {
		http.NotFound(w, r)
		return
	}

	clientMacAddressString := colonLessMacToMac(req[1])

	clientMac, err := net.ParseMAC(clientMacAddressString)
	if err != nil {
		return
	}
	machine, exist := datasource.GeneralDataSource.GetMachine(clientMac)
	if !exist {
		return
	}
	datasource.currentMachine = machine
	datasource.executeLock.Lock()
	defer datasource.executeLock.Unlock()
	var config string
	if req[0] == "cloud" {
		config, err = datasource.macCloudConfig(clientMacAddressString)
	} else {
		config, err = datasource.ignition()
	}
	if err != nil {
		http.Error(w, "internal server error - error in generating config", 500)
		logging.Log(debugTag, "Error when generating config - %s with mac %s - %s", req[0], req[1], err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/x-yaml")

	//always validate the cloudconfig. Don't if explicitly stated.
	if value, exists := queryMap["validate"]; req[0] == "cloud" && (!exists || value != "false") {
		config += validateCloudConfig(config)
	}

	w.Write([]byte(config))
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

// ServeCloudConfig is run cuncurrently alongside other blacksmith services
// Provides cloudconfig to machines at boot time
func ServeCloudConfig(listenAddr net.TCPAddr, workspacePath string, datasource datasource.GeneralDataSource) error {
	logging.Log(debugTag, "Listening on %s", listenAddr.String())

	cctemplates, err := FromPath(datasource, path.Join(datasource.WorkspacePath(), "config/cloudconfig"))
	if err != nil {
		return err
	}
	igtemplates, err := FromPath(datasource, path.Join(datasource.WorkspacePath(), "config/ignition"))
	if err != nil {
		return err
	}

	ccdataSource := cloudConfigDataSource{datasource, &sync.Mutex{}, cctemplates, igtemplates, nil}

	return http.ListenAndServe(listenAddr.String(), serveUtilityMultiplexer(ccdataSource))
}

func colonLessMacToMac(colonLess string) string {
	coloned := colonLess
	if strings.Index(colonLess, ":") == -1 {
		var tmpmac bytes.Buffer
		for i := 0; i < 12; i++ { // colon-less mac address length
			tmpmac.WriteString(colonLess[i : i+1])
			if i%2 == 1 {
				tmpmac.WriteString(":")
			}
		}
		coloned = tmpmac.String()[:len(tmpmac.String())-1]
	}
	return coloned
}
