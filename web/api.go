package web

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"path"

	"github.com/cafebazaar/blacksmith/datasource"
        "io/ioutil"
        "github.com/cafebazaar/blacksmith/logging"
)

// Version returns json encoded version details
func (ws *webServer) Version(w http.ResponseWriter, r *http.Request) {
	versionJSON, err := json.Marshal(ws.ds.SelfInfo())
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error": %q}`, err), 500)
		return
	}
	io.WriteString(w, string(versionJSON))
}

type nodeDetails struct {
	Name          string    `json:"name"`
	Nic           string    `json:"nic"`
	IP            net.IP    `json:"ip"`
	IPMInode      string    `json:"IPMInode"`
	FirstAssigned int64     `json:"firstAssigned"`
	LastAssigned  int64     `json:"lastAssigned"`
}

func nodeToDetails(node datasource.Machine) (*nodeDetails, error) {
	name := node.Name()
	mac := node.Mac()
	stats, err := node.GetStats()
	if err != nil {
		return nil, errors.New("stats")
	}
	last, err := node.LastSeen()
	if err != nil {
		return nil, errors.New("LAST")
	}
	return &nodeDetails{name, mac.String(), stats.IP, stats.IPMInode, stats.FirstSeen, last}, nil
}

// NodesList creates a list of the currently known nodes based on the etcd
// entries
func (ws *webServer) NodesList(w http.ResponseWriter, r *http.Request) {
	machines, err := ws.ds.Machines()
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error": %q}`, err), http.StatusInternalServerError)
		return
	}
	if len(machines) == 0 {
		io.WriteString(w, "[]")
		return
	}
	nodes := make([]*nodeDetails, 0, len(machines))
	for _, node := range machines {
		l, err := nodeToDetails(node)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error": %q}`, err), http.StatusInternalServerError)
			return
		}
		if l != nil {
			nodes = append(nodes, l)
		}
	}

	nodesJSON, err := json.Marshal(nodes)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error": %q}`, err), http.StatusInternalServerError)
		return
	}
	io.WriteString(w, string(nodesJSON))
}

// ClusterVariables returns all the cluster general variables
func (ws *webServer) ClusterVariablesList(w http.ResponseWriter, r *http.Request) {
	flags, err := ws.ds.ListClusterVariables()
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error": %q}`, err), http.StatusInternalServerError)
		return
	}

	flagsJSON, err := json.Marshal(flags)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error": %q}`, err), http.StatusInternalServerError)
		return
	}
	io.WriteString(w, string(flagsJSON))
}

// NodeFlags returns all the flags set for the node
func (ws *webServer) NodeFlags(w http.ResponseWriter, r *http.Request) {
	_, macStr := path.Split(r.URL.Path)

	mac, err := net.ParseMAC(macStr)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error": %q}`, err), http.StatusInternalServerError)
		return
	}

	machine, exists := ws.ds.GetMachine(mac)
	if !exists {
		http.Error(w, fmt.Sprintf(`{"error": "Machine not found"}`), http.StatusNotFound)
		return
	}

	flags, err := machine.ListFlags()
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error": %q}`, err), http.StatusInternalServerError)
		return
	}

	flagsJSON, err := json.Marshal(flags)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error": %q}`, err), http.StatusInternalServerError)
		return
	}
	io.WriteString(w, string(flagsJSON))
}

func (ws *webServer) NodeSetIPMI(w http.ResponseWriter, r *http.Request) {
        defer r.Body.Close()
        body, _ := ioutil.ReadAll(io.LimitReader(r.Body, 1048576))

        var data map[string]string
        json.Unmarshal(body, &data)
        nodeMac, err := net.ParseMAC(data["node"])
        if err != nil {
                return http.Error(w, `{"error": "Machine not found"}`, http.StatusInternalServerError)
        }
        IPMInodeMac, err := net.ParseMAC(data["IPMInode"])
        if err != nil {
                return http.Error(w, `{"error": "Machine not found"}`, http.StatusInternalServerError)
        }
        machine, _ := ws.ds.GetMachine(nodeMac)
        machine.SetIPMI(IPMInodeMac)
}

func (ws *webServer) SetFlag(w http.ResponseWriter, r *http.Request) {
	_, name := path.Split(r.URL.Path)
	value := r.FormValue("value")

	macStr := r.FormValue("mac")
	var machine datasource.Machine
	var exist bool
	if macStr != "" {
		mac, err := net.ParseMAC(macStr)
		if err != nil {
			http.Error(w, `{"error": "Error while parsing the mac"}`, http.StatusInternalServerError)
			return
		}

		machine, exist = ws.ds.GetMachine(mac)
		if !exist {
			http.Error(w, `{"error": "Machine not found"}`, http.StatusInternalServerError)
			return
		}
	}

	var err error
	if machine != nil {
		err = machine.SetFlag(name, value)
	} else {
		// TODO deafult flags
		http.Error(w, `{"error": "Default flags not supported yet"}`, http.StatusInternalServerError)
	}

	if err != nil {
		http.Error(w, `{"error": "Error while setting value"}`, http.StatusInternalServerError)
		return
	}

	io.WriteString(w, `"OK"`)
}

func (ws *webServer) DelFlag(w http.ResponseWriter, r *http.Request) {
	_, name := path.Split(r.URL.Path)

	macStr := r.FormValue("mac")
	var machine datasource.Machine
	var exist bool
	if macStr != "" {
		mac, err := net.ParseMAC(macStr)
		if err != nil {
			http.Error(w, `{"error": "Error while parsing the mac"}`, http.StatusInternalServerError)
			return
		}

		machine, exist = ws.ds.GetMachine(mac)
		if !exist {
			http.Error(w, `{"error": "Machine not found"}`, http.StatusInternalServerError)
			return
		}
	}

	var err error
	if machine != nil {
		err = machine.DeleteFlag(name)
	} else {
		// TODO deafult flags
		http.Error(w, `{"error": "Default flags not supported yet"}`, http.StatusInternalServerError)
	}

	if err != nil {
		http.Error(w, `{"error": "Error while delleting value"}`, http.StatusInternalServerError)
		return
	}

	io.WriteString(w, `"OK"`)
}

func (ws *webServer) SetVariable(w http.ResponseWriter, r *http.Request) {
	_, name := path.Split(r.URL.Path)
	value := r.FormValue("value")

	var err error
	err = ws.ds.SetClusterVariable(name, value)

	if err != nil {
		http.Error(w, `{"error": "Error while setting value"}`, http.StatusInternalServerError)
		return
	}

	io.WriteString(w, `"OK"`)
}

func (ws *webServer) DelVariable(w http.ResponseWriter, r *http.Request) {
	_, name := path.Split(r.URL.Path)

	err := ws.ds.DeleteClusterVariable(name)

	if err != nil {
		http.Error(w, `{"error": "Error while delleting value"}`, http.StatusInternalServerError)
		return
	}

	io.WriteString(w, `"OK"`)
}

func (ws *webServer) SetConfiguration(w http.ResponseWriter, r *http.Request) {
	_, name := path.Split(r.URL.Path)
	value := r.FormValue("value")

	var err error
	err = ws.ds.SetConfiguration(name, value)

	if err != nil {
		http.Error(w, `{"error": "Error while setting value"}`, http.StatusInternalServerError)
		return
	}

	io.WriteString(w, `"OK"`)
}

func (ws *webServer) DelConfiguration(w http.ResponseWriter, r *http.Request) {
	_, name := path.Split(r.URL.Path)

	err := ws.ds.DeleteConfiguration(name)

	if err != nil {
		http.Error(w, `{"error": "Error while delleting value"}`, http.StatusInternalServerError)
		return
	}

	io.WriteString(w, `"OK"`)
}

func (ws *webServer) ConfigurationList(w http.ResponseWriter, r *http.Request) {
	variables, err := ws.ds.ListConfigurations()
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error": %q}`, err), http.StatusInternalServerError)
		return
	}

	variablesJSON, err := json.Marshal(variables)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error": %q}`, err), http.StatusInternalServerError)
		return
	}
	io.WriteString(w, string(variablesJSON))
}
