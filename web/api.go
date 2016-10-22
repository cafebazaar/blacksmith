package web

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path"
	"sync"

	"github.com/cafebazaar/blacksmith/datasource"
	"github.com/cafebazaar/blacksmith/utils"
	"github.com/gorilla/mux"

	log "github.com/Sirupsen/logrus"
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

type machineDetails struct {
	Name          string                 `json:"name"`
	Nic           string                 `json:"nic"`
	IP            net.IP                 `json:"ip"`
	Type          datasource.MachineType `json:"type"`
	FirstAssigned int64                  `json:"firstAssigned"`
	LastAssigned  int64                  `json:"lastAssigned"`
}

func machineToDetails(machineInterface datasource.MachineInterface) (*machineDetails, error) {

	name := machineInterface.Hostname()
	mac := machineInterface.Mac()

	machine, err := machineInterface.Machine(false, nil)
	if err != nil {
		return nil, errors.New("error in retrieving machine details")
	}
	last, _ := machineInterface.LastSeen()

	return &machineDetails{
		name, mac.String(),
		machine.IP, machine.Type,
		machine.FirstSeen, last}, nil
}

// MachinesList creates a list of the currently known machines based on the etcd
// entries
func (ws *webServer) MachinesList(w http.ResponseWriter, r *http.Request) {
	machines, err := ws.ds.MachineInterfaces()
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error": %q}`, err), http.StatusInternalServerError)
		return
	}
	if len(machines) == 0 {
		io.WriteString(w, "[]")
		return
	}
	machinesArray := make([]*machineDetails, 0, len(machines))
	for _, machine := range machines {
		l, err := machineToDetails(machine)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error": %q}`, err), http.StatusInternalServerError)
			return
		}
		if l != nil {
			machinesArray = append(machinesArray, l)
		}
	}

	machinesJSON, err := json.Marshal(machinesArray)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error": %q}`, err), http.StatusInternalServerError)
		return
	}
	io.WriteString(w, string(machinesJSON))
}

// MachineDelete deletes associated information of a machine entirely
func (ws *webServer) MachineDelete(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	macString := vars["mac"]

	mac, err := net.ParseMAC(macString)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error": %q}`, err), http.StatusInternalServerError)
		return
	}

	machineInterface := ws.ds.MachineInterface(mac)
	err = machineInterface.DeleteMachine()
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error": %q}`, err), http.StatusInternalServerError)
		return
	}

	io.WriteString(w, `"OK"`)
}

// MachineVariable returns all the flags set for the machine
func (ws *webServer) MachineVariables(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	macString := vars["mac"]

	mac, err := net.ParseMAC(macString)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error": %q}`, err), http.StatusInternalServerError)
		return
	}

	machineInterface := ws.ds.MachineInterface(mac)

	flags, err := machineInterface.ListVariables()
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

func (ws *webServer) SetMachineVariable(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	macStr := vars["mac"]
	name := vars["name"]
	value := r.FormValue("value")

	var machineInterface datasource.MachineInterface
	if macStr != "" {
		mac, err := net.ParseMAC(macStr)
		if err != nil {
			http.Error(w, `{"error": "Error while parsing the mac"}`, http.StatusInternalServerError)
			return
		}

		machineInterface = ws.ds.MachineInterface(mac)
	}

	var err error

	err = machineInterface.SetVariable(name, value)

	if err != nil {
		http.Error(w, `{"error": "Error while setting value"}`, http.StatusInternalServerError)
		return
	}

	io.WriteString(w, `"OK"`)
}

func (ws *webServer) DelMachineVariable(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	macStr := vars["mac"]
	name := vars["name"]

	var machineInterface datasource.MachineInterface
	if macStr != "" {
		mac, err := net.ParseMAC(macStr)
		if err != nil {
			http.Error(w, `{"error": "Error while parsing the mac"}`, http.StatusInternalServerError)
			return
		}

		machineInterface = ws.ds.MachineInterface(mac)
	}

	var err error
	machineInterface.DeleteVariable(name)
	if err != nil {
		http.Error(w, `{"error": "Error while delleting value"}`, http.StatusInternalServerError)
		return
	}

	io.WriteString(w, `"OK"`)
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

func (ws *webServer) SetClusterVariables(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]
	value := r.FormValue("value")

	var err error
	err = ws.ds.SetClusterVariable(name, value)

	if err != nil {
		http.Error(w, `{"error": "Error while setting value"}`, http.StatusInternalServerError)
		return
	}

	io.WriteString(w, `"OK"`)
}

func (ws *webServer) DelClusterVariables(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]

	err := ws.ds.DeleteClusterVariable(name)

	if err != nil {
		http.Error(w, `{"error": "Error while delleting value"}`, http.StatusInternalServerError)
		return
	}

	io.WriteString(w, `"OK"`)
}

var workspaceUploadLock = &sync.Mutex{}

func (ws *webServer) WorkspaceUploadHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	hash := vars["hash"]

	workspaceUploadLock.Lock()

	workspacePath := ws.ds.WorkspacePath()

	workspaceParentPath := path.Dir(workspacePath)
	tarPath := path.Join(workspaceParentPath, "workspace.tar")
	file, err := os.Create(tarPath)
	if err != nil {
		http.Error(w, `{"error": "Error while creating new file for storing the upload"}`, http.StatusInternalServerError)
		return
	}
	_, err = io.Copy(file, r.Body)
	if err != nil {
		http.Error(w, `{"error": "Error while storing the workspace"}`, http.StatusInternalServerError)
		return
	}

	actualHash, _ := utils.HashFileMD5(tarPath)
	if actualHash != hash {
		http.Error(w, `{"error": "Actual hash of uploaded file doesn't match with the request"}`, http.StatusInternalServerError)
		return
	}

	extractPath := path.Join(workspaceParentPath, hash)
	err = utils.Untar(tarPath, extractPath)
	if err != nil {
		http.Error(w, `{"error": "Error untaring the uploaded workspace"}`, http.StatusInternalServerError)
		return
	}

	os.Remove(workspacePath)
	if err != nil {
		log.Info("Failed to unlink \"current\" symlink, but it is a no issue as it could be the first initialization")
	}

	destinedCurrentWorkspacePath := path.Join(extractPath, "workspace/")
	err = os.Symlink(destinedCurrentWorkspacePath, workspacePath)
	if err != nil {
		http.Error(w, `{"error": "Error symlinking current to the untared workspace"}`, http.StatusInternalServerError)
		return
	}

	imagesWorkspaceDir := path.Join(workspacePath, "files/")
	imagesWorkspaceTarPath := path.Join(imagesWorkspaceDir, "workspace.tar")
	os.Rename(tarPath, imagesWorkspaceTarPath)
	if err != nil {
		http.Error(w, `{"error": "Error while moving uploaded tar itself into workspace"}`, http.StatusInternalServerError)
		return
	}

	ws.ds.(*datasource.EtcdDataSource).FillEtcdFromWorkspace()

	err = ws.ds.SetClusterVariable(datasource.ActiveWorkspaceHashKey, hash)
	if err != nil {
		http.Error(w, `{"error": "Unable to set current workspace hash"}`, http.StatusInternalServerError)
		return
	}

	io.WriteString(w, `"OK"`)

	workspaceUploadLock.Unlock()
}
