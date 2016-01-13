package datasource

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	etcd "github.com/coreos/etcd/client"
	"gopkg.in/yaml.v2"
	// "github.com/elazarl/go-bindata-assetfs"
	"github.com/gorilla/mux"
	"golang.org/x/net/context"
)

//EtcdDataSource implements MasterDataSource interface using etcd as it's
//datasource
//Implements MasterDataSource interface
type EtcdDataSource struct {
	keysAPI              etcd.KeysAPI
	client               etcd.Client
	leaseStart           net.IP
	leaseRange           net.IP
	etcdDir              string
	workspacePath        string
	initialCoreOSVersion string
}

//WorkspacePath is self explanatory
//part of the GeneralDataSource interface implementation
func (ds *EtcdDataSource) WorkspacePath() string {
	return ds.workspacePath
}

//Machines returns an array of the recognized machines in etcd datasource
//part of GeneralDataSource interface implementation
func (ds *EtcdDataSource) Machines() ([]Machine, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	response, err := ds.keysAPI.Get(ctx, ds.parseKey(""), &etcd.GetOptions{Recursive: false})
	if err != nil {
		return nil, err
	}
	ret := make([]Machine, 0)
	for _, ent := range response.Node.Nodes {
		pathToMachineDir := ent.Key
		macStr := pathToMachineDir[strings.LastIndex(pathToMachineDir, "/")+1:]
		macAddr, err := net.ParseMAC(macStr)
		if err != nil {
			return nil, err
		}
		machine, _, err := ds.GetOrCreateMachine(macAddr)
		if err != nil {
			return nil, err
		}
		ret = append(ret, machine)
	}
	return ret, nil
}

//GetOrCreateMachine returns a Machine interface which is the accessor/getter/setter
//for a node in the etcd datasource. If an entry associated with the passed
//mac address does not exist, it is created and the handle (Machine) will be
//returned. bool returns value is set to true if the Machine already exists so
//it can be used like:
//machine , exists , err := GetOrCreateMachine(mac)
//part of GeneralDataSource interface implementation
func (ds *EtcdDataSource) GetOrCreateMachine(mac net.HardwareAddr) (Machine, bool, error) {
	//TODO
	return nil, false, nil
}

//CoreOSVersion gets the current value from etcd and returns it if the image folder exists
//if not, the inital CoreOS version will be returned, with the raised error
//part of GeneralDataSource interface implementation
func (ds *EtcdDataSource) CoreOSVersion() (string, error) {
	coreOSVersion, err := ds.Get("coreos-version")
	if err != nil {
		return ds.initialCoreOSVersion, err
	}

	imagesPath := filepath.Join(ds.WorkspacePath(), "images", coreOSVersion)
	files, err := ioutil.ReadDir(imagesPath)
	if err != nil {
		return ds.initialCoreOSVersion, fmt.Errorf("Error while reading coreos subdirecory: %s (path=%s)", err, imagesPath)
	} else if len(files) == 0 {
		return ds.initialCoreOSVersion, errors.New("The images subdirecory of workspace should contains at least one version of CoreOS")
	}

	return coreOSVersion, nil
}

//MacCloudConfig generates a cloud-config file based on the Mac address that is passed in
//Will generate a commented warning at the end of the cloud-config if the node's ip in the http
//request mismatches the one in etcd
//Part of CloudConfigDataSource interace implementation

func (ds *EtcdDataSource) parseKey(key string) string {
	// key = strings.Replace(key, ".", "/", -1)
	// key = strings.Replace(key, "__/", ds.etcdDir+"/", -1)
	key = "blacksmith/" + key
	return key
}

//Get parses the etcd key and returns it's value
//part of GeneralDataSource interface implementation
func (ds *EtcdDataSource) Get(key string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	response, err := ds.keysAPI.Get(ctx, ds.parseKey(key), nil)
	if err != nil {
		return "", err
	}
	return response.Node.Value, nil
}

//Set sets and etcd key to a value
//part of GeneralDataSource interface implementation
func (ds *EtcdDataSource) Set(key string, value string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, err := ds.keysAPI.Set(ctx, ds.parseKey(key), value, nil)
	return err
}

//GetAndDelete gets the value of an etcd key and returns it, and deletes the record
//afterwards
//part of GeneralDataSource interface implementation
func (ds *EtcdDataSource) GetAndDelete(key string) (string, error) {
	value, err := ds.Get(key)
	if err != nil {
		return "", err
	}
	if err = ds.Delete(key); err != nil {
		return "", err
	}
	return value, nil
}

//Delete erases the key from etcd
//part of GeneralDataSource interface implementation
func (ds *EtcdDataSource) Delete(key string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, err := ds.keysAPI.Delete(ctx, ds.parseKey(key), nil)
	return err
}

type initialValues struct {
	CoreOSVersion string `yaml:"coreos-version"`
}

//Handler uses a multiplexing router to route http requests
//part of the RestServer interface implementation
func (ds *EtcdDataSource) Handler() http.Handler {
	mux := mux.NewRouter()
	mux.HandleFunc("/api/nodes", ds.NodesList)
	mux.HandleFunc("/api/etcd-endpoints", ds.etcdEndpoints)

	mux.HandleFunc("/upload/", ds.Upload)
	mux.HandleFunc("/files", ds.Files).Methods("GET")
	mux.HandleFunc("/files", ds.DeleteFile).Methods("DELETE")
	mux.PathPrefix("/files/").Handler(http.StripPrefix("/files/",
		http.FileServer(http.Dir(filepath.Join(ds.WorkspacePath(), "files")))))
	// mux.PathPrefix("/ui/").Handler(http.StripPrefix("/ui/",
	// http.FileServer(&assetfs.AssetFS{Asset: Asset, AssetDir: AssetDir, Prefix: "/web/ui"})))

	return mux
}

//Upload does what it is supposed to do!
//part of UIRestServer interface implementation
func (ds *EtcdDataSource) Upload(w http.ResponseWriter, r *http.Request) {
	const MaxFileSize = 1 << 30
	// This feels like a bad hack...
	if r.ContentLength > MaxFileSize {
		http.Error(w, "Request too large", 400)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, MaxFileSize)

	err := r.ParseMultipartForm(1024)
	if err != nil {
		http.Error(w, "File too large", 400)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		panic(err)
	}

	dst, err := os.Create(filepath.Join(ds.WorkspacePath(), "files", header.Filename))
	defer dst.Close()
	if err != nil {
		http.Error(w, err.Error(), 500)
	}

	written, err := io.Copy(dst, io.LimitReader(file, MaxFileSize))
	if err != nil {
		http.Error(w, err.Error(), 500)
	}

	if written == MaxFileSize {
		http.Error(w, "File too large", 400)
		return
	}
}

//DeleteFile allows the deletion of a file through http Request
//part of the UIRestServer interface implementation
func (ds *EtcdDataSource) DeleteFile(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("name")

	if name != "" {
		err := os.Remove(filepath.Join(ds.WorkspacePath(), "files", name))

		if err != nil {
			http.Error(w, err.Error(), 404)

			return
		}
	} else {
		http.Error(w, "No file name specified.", 400)
	}

}

//NodesList creates a list of the currently known nodes based on the etcd
//entries
//part of UIRestServer interface implementation
func (ds *EtcdDataSource) NodesList(w http.ResponseWriter, r *http.Request) {
	//TODO

	// leases, err := a.pool.Leases()
	// if err != nil {
	// 	http.Error(w, "Error in fetching lease data", 500)
	// }
	// nodesJSON, err := json.Marshal(leases)
	// if err != nil {
	// 	io.WriteString(w, fmt.Sprintf("{'error': %s}", err))
	// 	return
	// }
	// io.WriteString(w, string(nodesJSON))
}

type uploadedFile struct {
	Name                 string    `json:"name"`
	Size                 int64     `json:"size"`
	LastModificationDate time.Time `json:"lastModifiedDate"`
}

//Files allows utilization of the uploaded/shared files through http requests
//part of UIRestServer interface implementation
func (ds *EtcdDataSource) Files(w http.ResponseWriter, r *http.Request) {
	files, err := ioutil.ReadDir(filepath.Join(ds.WorkspacePath(), "files"))
	if err != nil {
		http.Error(w, err.Error(), 500)
	}

	var filesList []uploadedFile
	for _, f := range files {
		if f.Name()[0] == '.' {
			continue
		}
		var uploadedFile uploadedFile
		uploadedFile.Size = f.Size()
		uploadedFile.LastModificationDate = f.ModTime()
		uploadedFile.Name = f.Name()
		filesList = append(filesList, uploadedFile)
	}

	jsoned, _ := json.Marshal(filesList)
	io.WriteString(w, string(jsoned))
}

func (ds *EtcdDataSource) etcdEndpoints(w http.ResponseWriter, r *http.Request) {

}

//LeaseStart returns the first IP address that the DHCP server can offer to a
//DHCP client
//part of DHCPDataSource interface implementation
func (ds *EtcdDataSource) LeaseStart() net.IP {
	//TODO
	return nil
}

//LeaseRange returns the IP range from which IP addresses are assignable to
//clients by the DHCP server
//part of DHCPDataSource interface implementation
func (ds *EtcdDataSource) LeaseRange() net.IP {
	//TODO
	return nil
}

//NewEtcdDataSource gives blacksmith the ability to use an etcd endpoint as
//a MasterDataSource
func NewEtcdDataSource(kapi etcd.KeysAPI, client etcd.Client, leaseStart,
	leaseRange net.IP, etcdDir, workspacePath string) (MasterDataSource, error) {

	data, err := ioutil.ReadFile(filepath.Join(workspacePath, "initial.yaml"))
	if err != nil {
		return nil, fmt.Errorf("Error while trying to read initial data: %s", err)
	}

	var iVals initialValues
	err = yaml.Unmarshal(data, &iVals)
	if err != nil {
		return nil, fmt.Errorf("Error while reading initial data: %s", err)
	}
	if iVals.CoreOSVersion == "" {
		return nil, errors.New("A valid initial CoreOS version is required in initial data")
	}

	fmt.Printf("Initial Values: CoreOSVersion=%s\n", iVals.CoreOSVersion)

	instance := &EtcdDataSource{
		keysAPI:              kapi,
		client:               client,
		etcdDir:              etcdDir,
		workspacePath:        workspacePath,
		initialCoreOSVersion: iVals.CoreOSVersion,
	}

	_, err = instance.CoreOSVersion()
	if err != nil {
		etcdError, found := err.(etcd.Error)
		if found && etcdError.Code == etcd.ErrorCodeKeyNotFound {
			// Initializing
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			_, err = instance.keysAPI.Set(ctx, path.Join(etcdDir, "/coreos-version"), iVals.CoreOSVersion, nil)
			if err != nil {
				return nil, fmt.Errorf("Error while initializing etcd tree: %s", err)
			}
			fmt.Printf("Initialized etcd tree (%s)", etcdDir)
		} else {
			return nil, fmt.Errorf("Error while checking GetCoreOSVersion: %s", err)
		}
	}

	return instance, nil
}
