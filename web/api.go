package web

import (
	"archive/tar"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"

	"github.com/cafebazaar/blacksmith/datasource"
	"github.com/gorilla/mux"
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

func (ws *webServer) WorkspaceUploadHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	hash := vars["hash"]

	workspaceParentPath := path.Dir(ws.ds.WorkspacePath())
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

	extractPath := path.Join(workspaceParentPath, hash)
	err = untar(tarPath, extractPath)
	if err != nil {
		http.Error(w, `{"error": "Error untaring the workspace"}`, http.StatusInternalServerError)
		return
	}

	currentWorkspacePath := path.Join(workspaceParentPath, "current/")
	destiniedCurrentWorkspacePath := path.Join(extractPath, "workspace/")
	err = os.Symlink(destiniedCurrentWorkspacePath, currentWorkspacePath)
	if err != nil {
		http.Error(w, `{"error": "Error untaring the workspace"}`, http.StatusInternalServerError)
		return
	}

	imagesWorkspaceDir := path.Join(currentWorkspacePath, "files/")
	imagesWorkspaceTarPath := path.Join(imagesWorkspaceDir, "workspace.tar")
	os.Rename(tarPath, imagesWorkspaceTarPath)
	if err != nil {
		http.Error(w, `{"error": "Error while moving uploaded tar itself into workspace"}`, http.StatusInternalServerError)
		return
	}

	ws.ds.(*datasource.EtcdDataSource).FillEtcdFromWorkspace()

	io.WriteString(w, `"OK"`)
}

// https://gist.github.com/svett/dc27b7fb04c2549e3ada#file-untarball-go
func untar(tarball, target string) error {
	reader, err := os.Open(tarball)
	if err != nil {
		return err
	}
	defer reader.Close()
	tarReader := tar.NewReader(reader)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}

		path := filepath.Join(target, header.Name)
		info := header.FileInfo()
		if info.IsDir() {
			if err = os.MkdirAll(path, info.Mode()); err != nil {
				return err
			}
			continue
		}

		file, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, info.Mode())
		if err != nil {
			return err
		}
		defer file.Close()
		_, err = io.Copy(file, tarReader)
		if err != nil {
			return err
		}
	}
	return nil
}
