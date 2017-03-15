package web

import (
	"fmt"
	"net"
	"net/http"
	"path"

	yaml "gopkg.in/yaml.v2"

	"os"

	"github.com/cafebazaar/blacksmith/merger"
	"github.com/cafebazaar/blacksmith/templating"
	"github.com/coreos/coreos-cloudinit/config"
)

const (
	templatesDebugTag = "WEB-T"
)

func (ws *webServer) generateTemplateForMachine(templateName string, w http.ResponseWriter, r *http.Request) string {
	_, macStr := path.Split(r.URL.Path)

	mac, err := net.ParseMAC(macStr)
	if err != nil {
		http.Error(w, fmt.Sprintf(`Error while parsing the mac: %q`, err), 500)
		return ""
	}

	machineInterface := ws.ds.GetEtcdMachine(mac)
	_, err = machineInterface.Machine(false, nil)
	if err != nil {
		http.Error(w, "Machine not found", 404)
		return ""
	}

	var ccuser string
	_, err = os.Stat(path.Join(ws.ds.WorkspacePath(), "repo", "config", templateName, "main"))
	if !os.IsNotExist(err) {
		ccuser, err = templating.ExecuteTemplateFolder(
			path.Join(ws.ds.WorkspacePath(), "repo", "config", templateName), "main", ws.ds, machineInterface)
		if err != nil {
			http.Error(w, fmt.Sprintf(`Error while executing the template: %q`, err), 500)
			return ""
		}
	}

	tmpl, err := templating.FSString(false, "/files/"+templateName)
	if err != nil {
		http.Error(w, "Ebedded template not found: "+err.Error(), 500)
		return ""
	}
	ccbase, err := templating.ExecuteTemplateFile(tmpl, ws.ds, machineInterface)
	if err != nil {
		http.Error(w, "Ebedded template can't be rendered: "+err.Error(), 500)
		return ""
	}

	baseCC := config.CloudConfig{}
	if err := yaml.Unmarshal([]byte(ccbase), &baseCC); err != nil {
		http.Error(w, err.Error(), 500)
		return ""
	}
	userCC := config.CloudConfig{}
	if err := yaml.Unmarshal([]byte(ccuser), &userCC); err != nil {
		http.Error(w, err.Error(), 500)
		return ""
	}
	merged, err := merger.Merge(baseCC, userCC)
	if err := yaml.Unmarshal([]byte(ccuser), &userCC); err != nil {
		http.Error(w, err.Error(), 500)
		return ""
	}

	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(merged.String()))
	return merged.String()
}

// Cloudconfig generates and writes cloudconfig for the machine specified by the
// mac in the request url path
func (ws *webServer) Cloudconfig(w http.ResponseWriter, r *http.Request) {
	config := ws.generateTemplateForMachine("cloudconfig", w, r)

	if config != "" && r.FormValue("validate") != "" {
		w.Write([]byte(templating.ValidateCloudConfig(config)))
	}
}

// Ignition generates and writes ignition for the machine specified by the
// mac in the request url path
func (ws *webServer) Ignition(w http.ResponseWriter, r *http.Request) {
	ws.generateTemplateForMachine("ignition", w, r)
}

// Bootparams generates and writes bootparams for the machine specified by the
// mac in the request url path. (Just for validation purpose)
func (ws *webServer) Bootparams(w http.ResponseWriter, r *http.Request) {
	ws.generateTemplateForMachine("bootparams", w, r)
}
