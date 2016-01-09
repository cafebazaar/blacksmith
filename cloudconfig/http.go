package cloudconfig

import (
	"net"
	"strings"
	//	"net/url"
	//	"fmt"
	"net/http"
	"path"

	"github.com/cafebazaar/blacksmith/logging"
)

type CloudConfig struct {
	cloudRepo    *Repo
	ignitionRepo *Repo
}

func NewCloudConfig(cloudRepo *Repo, ignitionRepo *Repo) *CloudConfig {
	return &CloudConfig{
		cloudRepo:    cloudRepo,
		ignitionRepo: ignitionRepo,
	}
}

func (c *CloudConfig) Mux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/", c.handler)
	return mux
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

func (c *CloudConfig) handler(w http.ResponseWriter, r *http.Request) {
	req := strings.Split(r.URL.Path, "/")[1:]

	queryMap, _ := extractQueries(r.URL.RawQuery)

	if len(req) != 2 {
		logging.Log("CLOUDCONFIG", "Received request - request not found")
		http.NotFound(w, r)
		return
	}
	var selectedRepo *Repo
	switch req[0] {
	case "cloud":
		selectedRepo = c.cloudRepo
	case "ignition":
		selectedRepo = c.ignitionRepo
	default:
		http.NotFound(w, r)
		return
	}

	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		http.Error(w, "internal server error - parsing host and port", 500)
		logging.Log("CLOUDCONFIG", "Error - %s with mac %s - %s", req[0], req[1], err.Error())
		return
	}

	configCtx := &ConfigContext{
		MacAddr: req[1],
		IP:      ip,
	}
	config, err := selectedRepo.GenerateConfig(configCtx)
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

func ServeCloudConfig(listenAddr net.TCPAddr, workspacePath string, datasources map[string]DataSource) error {
	logging.Log("CLOUDCONFIG", "Listening on %s", listenAddr.String())
	cloudRepo, err := FromPath(datasources, path.Join(workspacePath, "config/cloudconfig"))
	if err != nil {
		return err
	}
	ignitionRepo, err := FromPath(datasources, path.Join(workspacePath, "config/ignition"))
	if err != nil {
		return err
	}
	cloudConfig := NewCloudConfig(cloudRepo, ignitionRepo)
	return http.ListenAndServe(listenAddr.String(), cloudConfig.Mux())
}
