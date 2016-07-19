package web

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"path"
	"time"

	"github.com/cafebazaar/blacksmith/datasource"
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
	FirstAssigned time.Time `json:"firstAssigned"`
	LastAssigned  time.Time `json:"lastAssigned"`
}

func nodeToDetails(node datasource.Machine) (*nodeDetails, error) {
	name := node.Name()
	mac := node.Mac()
	ip, err := node.IP()
	if err != nil {
		return nil, errors.New("IP")
	}
	first, err := node.FirstSeen()
	if err != nil {
		return nil, errors.New("FIRST")
	}
	last, err := node.LastSeen()
	if err != nil {
		return nil, errors.New("LAST")
	}
	return &nodeDetails{name, mac.String(), ip, first, last}, nil
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

    err := ws.ds.DeleteClusterVariable(name);
    
 	if err != nil {
		http.Error(w, `{"error": "Error while delleting value"}`, http.StatusInternalServerError)
		return
	}

	io.WriteString(w, `"OK"`)
}
