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
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cafebazaar/blacksmith/logging"
	etcd "github.com/coreos/etcd/client"
	"github.com/krolaw/dhcp4"
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
	leaseRange           int
	etcdDir              string
	workspacePath        string
	initialCoreOSVersion string
	dhcpAssignLock       *sync.Mutex
	dhcpDataLock         *sync.Mutex
}

//WorkspacePath is self explanatory
//part of the GeneralDataSource interface implementation
func (ds *EtcdDataSource) WorkspacePath() string {
	return ds.workspacePath
}

//Machines returns an array of the recognized machines in etcd datasource
//part of GeneralDataSource interface implementation
func (ds *EtcdDataSource) Machines() ([]Machine, error) {
	// logging.Log("#MACHINES", "called")
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	response, err := ds.keysAPI.Get(ctx, ds.parseKey("/machines"), &etcd.GetOptions{Recursive: false})
	if err != nil {
		return nil, err
	}
	// is := "yes"
	// if !response.Node.Dir {
	// 	is = "no"
	// }
	// logging.Log("ISDIR", is)
	// logging.Log("NUNODES", fmt.Sprint(len(response.Node.Nodes)))
	ret := make([]Machine, 0)
	for _, ent := range response.Node.Nodes {
		// logging.Log("#MACHINES ENTRY", ent.Key)
		pathToMachineDir := ent.Key
		macStr := pathToMachineDir[strings.LastIndex(pathToMachineDir, "/")+1:]
		macAddr, err := net.ParseMAC(macStr)
		if err != nil {
			return nil, err
		}
		machine, exist := ds.GetMachine(macAddr)
		if !exist {
			return nil, errors.New("Inconsistent datasource")
		}
		// logging.Log("Adding to machines", machine.Mac().String())
		ret = append(ret, machine)
	}
	// logging.Log("#MACHINES", "return good")
	return ret, nil
}

//GetMachine returns a Machine interface which is the accessor/getter/setter
//for a node in the etcd datasource. If an entry associated with the passed
//mac address does not exist the second return value will be set to false
//part of GeneralDataSource interface implementation
func (ds *EtcdDataSource) GetMachine(mac net.HardwareAddr) (Machine, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	response, err := ds.keysAPI.Get(ctx, ds.parseKey(path.Join("machines/"+mac.String())), nil)
	if err != nil {
		return nil, false
	}
	if response.Node.Key[strings.LastIndex(response.Node.Key, "/")+1:] == mac.String() {
		return &EtcdMachine{mac, ds}, true
	}
	return nil, false
}

//CreateMachine Creates a machine, returns the handle, and writes directories and flags to etcd
//Second return value determines whether or not Machine creation has been
//successful
//part of GeneralDataSource interface implementation
func (ds *EtcdDataSource) CreateMachine(mac net.HardwareAddr, ip net.IP) (Machine, bool) {
	// logging.Log("#CREATE", mac.String()+" "+ip.String())
	machines, err := ds.Machines()

	if err != nil {
		return nil, false
	}
	for _, node := range machines {
		// logging.Log("#Creating", "entry : "+node.Mac().String())
		if node.Mac().String() == mac.String() {
			// logging.Log("#CREATING", "same mac")
			return nil, false
		}
		nodeip, err := node.IP()
		if err != nil {
			return nil, false
		}
		if nodeip.String() == ip.String() {
			return nil, false
		}
	}
	// logging.Log("#IP and MAC GOOD", "")
	machine := &EtcdMachine{mac, ds}
	// create it !
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	// logging.Log("#DIR CREATION", machine.Mac().String())

	ds.keysAPI.Set(ctx, ds.parseKey("machines/"+machine.Mac().String()), "", &etcd.SetOptions{Dir: true})
	ctx1, cancel1 := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel1()
	ds.keysAPI.Set(ctx1, ds.parseKey("machines/"+machine.Mac().String()+"/_IP"), ip.String(), &etcd.SetOptions{})

	ctx2, cancel2 := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel2()
	ds.keysAPI.Set(ctx2, ds.parseKey("machines/"+machine.Mac().String()+"/_name"), machine.Name(), &etcd.SetOptions{})

	ctx3, cancel3 := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel3()
	ds.keysAPI.Set(ctx3, ds.parseKey("machines/"+machine.Mac().String()+"/_first_seen"),
		strconv.FormatInt(time.Now().UnixNano(), 10), &etcd.SetOptions{})
	machine.CheckIn()
	machine.SetFlag("state", "unknown")
	return machine, true
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
	return path.Join("blacksmith/", key)
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

//
// func (ds *EtcdDataSource) Ls(key string) []string {
// 	ret := make([]string, 0)
// 	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
// 	defer cancel()
//
// 	response, err := ds.keysAPI.Get(ctx, ds.parseKey(key), &etcd.GetOptions{Recursive: false})
// 	if err != nil {
// 		return nil
// 	}
// 	for _, ent := range response.Node.Nodes {
// 		ret = append(ret, ent.Key)
// 	}
// 	return ret
// }

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
	pwd, _ := os.Getwd()
	mux.PathPrefix("/ui/").Handler(http.FileServer(http.Dir(filepath.Join(pwd, "web/"))))
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

type lease struct {
	Nic           string
	IP            net.IP
	FirstAssigned time.Time
	LastAssigned  time.Time
	ExpireTime    time.Time
}

func nodeToLease(node Machine) (*lease, error) {
	mac := node.Mac()
	ip, err := node.IP()
	if err != nil {
		return nil, errors.New("IP")
		return nil, err
	}
	first, err := node.FirstSeen()
	if err != nil {
		return nil, errors.New("FIRST")
		return nil, err
	}
	last, err := node.LastSeen()
	if err != nil {
		return nil, errors.New("LAST")
		return nil, err
	}
	exp := time.Now() // <- ??? TODO
	return &lease{mac.String(), ip, first, last, exp}, nil
}

//NodesList creates a list of the currently known nodes based on the etcd
//entries
//part of UIRestServer interface implementation
func (ds *EtcdDataSource) NodesList(w http.ResponseWriter, r *http.Request) {
	// logging.Log("#NODELIST", "CALLED")
	leases := make(map[string]lease)
	machines, err := ds.Machines()
	if err != nil || machines == nil {
		http.Error(w, "Error in fetching lease data", 500)
	}
	for _, node := range machines {
		l, err := nodeToLease(node)
		if err != nil {
			http.Error(w, "Error in fetching lease data", 500)
		}
		if l == nil {
			// logging.Log("#NODELIST", "couldn't fill lease")
			// logging.Log("#NODELIST", err.Error())
		}
		leases[node.Name()] = *l
	}

	nodesJSON, err := json.Marshal(leases)
	if err != nil {
		io.WriteString(w, fmt.Sprintf("{'error': %s}", err))
		return
	}
	io.WriteString(w, string(nodesJSON))
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
	endpointsJSON, err := json.Marshal(ds.client.Endpoints())
	if err != nil {
		io.WriteString(w, fmt.Sprintf("{'error': %s}", err))
		return
	}
	io.WriteString(w, string(endpointsJSON))
}

//LeaseStart returns the first IP address that the DHCP server can offer to a
//DHCP client
//part of DHCPDataSource interface implementation
func (ds *EtcdDataSource) LeaseStart() net.IP {
	return ds.leaseStart
}

//LeaseRange returns the IP range from which IP addresses are assignable to
//clients by the DHCP server
//part of DHCPDataSource interface implementation
func (ds *EtcdDataSource) LeaseRange() int {
	return ds.leaseRange
}

func (ds *EtcdDataSource) lockDHCPAssign() {
	ds.dhcpAssignLock.Lock()
}

func (ds *EtcdDataSource) unlockdhcpAssign() {
	ds.dhcpAssignLock.Unlock()
}

func (ds *EtcdDataSource) lockDHCPData() {
	ds.dhcpDataLock.Lock()
}

func (ds *EtcdDataSource) unlockDHCPData() {
	ds.dhcpDataLock.Unlock()
}

func (ds *EtcdDataSource) store(m Machine, ip net.IP) {
	ds.lockDHCPData()
	defer ds.unlockDHCPData()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	ds.keysAPI.Set(ctx, ds.parseKey("machines/"+m.Mac().String()+"/_IP"),
		ip.String(), &etcd.SetOptions{})

	ctx1, cancel1 := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel1()
	ds.keysAPI.Set(ctx1, ds.parseKey("machines/"+m.Mac().String()+"/_last_seen"),
		strconv.FormatInt(time.Now().UnixNano(), 10), &etcd.SetOptions{})
}

//Assign assigns an ip to the node with the specified nic
//Will use etcd machines records as LeasePool
//part of DHCPDataSource interface implementation
func (ds *EtcdDataSource) Assign(nic string) (net.IP, error) {
	ds.lockDHCPAssign()
	defer ds.unlockdhcpAssign()

	assignedIPs := make(map[string]bool)
	//find by Mac
	machines, _ := ds.Machines()
	for _, node := range machines {
		if node.Mac().String() == nic {
			ip, _ := node.IP()
			ds.store(node, ip)
			return ip, nil
		}
		nodeIP, _ := node.IP()
		assignedIPs[nodeIP.String()] = true
	}

	// logging.Log("#DHCP", "already assigned")
	// for k, _ := range assignedIPs {
	// 	logging.Log("#DHCP", k)
	// }
	// logging.Log("#Trying to find me : ", nic)
	// logging.Log("#DHCP", "mac not found")
	// logging.Log("#DHCP", fmt.Sprintf("lease range : %d", ds.LeaseRange()))
	// logging.Log("#DHCP", "start : "+ds.LeaseStart().String())
	// return nil, nil
	//find an unused ip
	for i := 0; i < ds.LeaseRange(); i++ {
		ip := dhcp4.IPAdd(ds.LeaseStart(), i)
		if _, exists := assignedIPs[ip.String()]; !exists {
			macAddress, _ := net.ParseMAC(nic)
			// logging.Log("#DHCP", "creating machine")
			// logging.Log("#DHCP", macAddress.String()+" "+ip.String())
			ds.CreateMachine(macAddress, ip)
			return ip, nil
		} else {
			// logging.Log("#DHCP", "already exists : "+ip.String() + " nic : )
		}
	}

	//use an expired ip
	//not implemented yet

	return nil, nil
}

//Request answers a dhcp request
//Uses etcd as backend
//part of DHCPDataSource interface implementation
func (ds *EtcdDataSource) Request(nic string, currentIP net.IP) (net.IP, error) {
	logging.Log("#Request : ", nic+" "+currentIP.String())
	ds.lockDHCPAssign()
	defer ds.unlockdhcpAssign()

	machines, _ := ds.Machines()

	macExists, ipExists := false, false

	// logging.Log("#REQUEST : ", "my ip : "+currentIP.String()+" my mac : "+nic)

	for _, node := range machines {
		thisNodeIP, _ := node.IP()
		ipMatch := thisNodeIP.String() == currentIP.String()
		macMatch := nic == node.Mac().String()
		// logging.Log("#REQUEST : ", "their : "+thisNodeIP.String()+" th mac : "+node.Mac().String())

		if ipMatch && macMatch {
			ds.store(node, thisNodeIP)
			return currentIP, nil
		}

		ipExists = ipExists || ipMatch
		macExists = macExists || macMatch

	}
	if ipExists || macExists {
		// logging.Log("#Request", "Missmatch in lease pool")
		return nil, errors.New("Missmatch in lease pool")
	}
	macAddress, _ := net.ParseMAC(nic)
	ds.CreateMachine(macAddress, currentIP)
	return currentIP, nil
}

//NewEtcdDataSource gives blacksmith the ability to use an etcd endpoint as
//a MasterDataSource
func NewEtcdDataSource(kapi etcd.KeysAPI, client etcd.Client, leaseStart net.IP,
	leaseRange int, etcdDir, workspacePath string) (MasterDataSource, error) {

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

	// logging.Log("#NEW", "supplied starting ip"+leaseStart.String())

	instance := &EtcdDataSource{
		keysAPI:              kapi,
		client:               client,
		etcdDir:              etcdDir,
		leaseStart:           leaseStart,
		leaseRange:           leaseRange,
		workspacePath:        workspacePath,
		initialCoreOSVersion: iVals.CoreOSVersion,
		dhcpAssignLock:       &sync.Mutex{},
		dhcpDataLock:         &sync.Mutex{},
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

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	instance.keysAPI.Set(ctx, instance.parseKey("machines"), "", &etcd.SetOptions{Dir: true})

	return instance, nil
}
