package web

import (
	"net"
	"net/http"
	"path"

	yaml "gopkg.in/yaml.v2"

	"os"

	"github.com/cafebazaar/blacksmith/merger"
	"github.com/cafebazaar/blacksmith/templating"
	"github.com/pkg/errors"
)

const (
	templatesDebugTag = "WEB-T"
)

func (ws *webServer) generateTemplateForMachine(templateName string, macStr string) (string, error) {
	mac, err := net.ParseMAC(macStr)
	if err != nil {
		return "", errors.Wrap(err, "Error while parsing the mac")
	}

	machineInterface := ws.ds.GetEtcdMachine(mac)
	_, err = machineInterface.Machine(false, nil)
	if err != nil {
		return "", errors.New("Machine not found")
	}

	var ccuser string
	_, err = os.Stat(path.Join(ws.ds.WorkspacePath(), "repo", "config", templateName, "main"))
	if !os.IsNotExist(err) {
		ccuser, err = templating.ExecuteTemplateFolder(
			path.Join(ws.ds.WorkspacePath(), "repo", "config", templateName), "main", ws.ds, machineInterface)
		if err != nil {
			return "", errors.Wrap(err, "Error while executing the template")
		}
	}

	tmpl, err := templating.FSString(false, "/files/"+templateName)
	if err != nil {
		return "", errors.Wrap(err, "Ebedded template not found")
	}

	ccbase, err := templating.ExecuteTemplateFile(tmpl, ws.ds, machineInterface)
	if err != nil {
		return "", errors.Wrap(err, "Ebedded template can't be rendered")
	}

	var content string
	if templateName == "cloudconfig" {
		baseCC := merger.CloudConfig{}
		if err := yaml.Unmarshal([]byte(ccbase), &baseCC); err != nil {
			return "", err
		}
		userCC := merger.CloudConfig{}
		if err := yaml.Unmarshal([]byte(ccuser), &userCC); err != nil {
			return "", err
		}

		merged, err := merger.Merge(baseCC, userCC)
		if err != nil {
			return "", err
		}
		content = merged.String()
	} else {
		content = ccbase + "\n" + ccuser
	}

	return content, nil
}

// Cloudconfig generates and writes cloudconfig for the machine specified by the
// mac in the request url path
func (ws *webServer) Cloudconfig(w http.ResponseWriter, r *http.Request) {
	_, macStr := path.Split(r.URL.Path)
	config, err := ws.generateTemplateForMachine("cloudconfig", macStr)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(config))

	if config != "" && r.FormValue("validate") != "" {
		w.Write([]byte(templating.ValidateCloudConfig(config)))
	}
}

// Ignition generates and writes ignition for the machine specified by the
// mac in the request url path
func (ws *webServer) Ignition(w http.ResponseWriter, r *http.Request) {
	_, macStr := path.Split(r.URL.Path)
	config, err := ws.generateTemplateForMachine("ignition", macStr)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(config))
}

// Bootparams generates and writes bootparams for the machine specified by the
// mac in the request url path. (Just for validation purpose)
func (ws *webServer) Bootparams(w http.ResponseWriter, r *http.Request) {
	_, macStr := path.Split(r.URL.Path)
	config, err := ws.generateTemplateForMachine("bootparams", macStr)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(config))
}
