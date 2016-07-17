package datasource

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	etcd "github.com/coreos/etcd/client"
	"github.com/krolaw/dhcp4"
	"golang.org/x/net/context"
	"gopkg.in/yaml.v2"

	"github.com/cafebazaar/blacksmith/logging"
)

const (
	coreosVersionKey = "coreos-version"
)

func (ii *InstanceInfo) String() string {
	marshaled, err := json.Marshal(ii)
	if err != nil {
		logging.Log(debugTag, "Failed to marshal instanceInfo: %s", err)
		return ""
	}
	return string(marshaled)
}

// func InstanceInfoFromString(iiStr string) (*InstanceInfo, error) {
// 	var ii *InstanceInfo
// 	if err := json.Unmarshal([]byte(iiStr), ii); err != nil {
// 		return nil, err
// 	}
// 	return ii, nil
// }

// EtcdDataSource implements MasterDataSource interface using etcd as it's
// datasource
// Implements MasterDataSource interface
type EtcdDataSource struct {
	keysAPI              etcd.KeysAPI
	client               etcd.Client
	leaseStart           net.IP
	leaseRange           int
	clusterName          string
	workspacePath        string
	initialCoreOSVersion string
	dhcpAssignLock       *sync.Mutex
	dhcpDataLock         *sync.Mutex
	instanceEtcdKey      string // HA
	selfInfo             InstanceInfo
}

// WorkspacePath returns the path to the workspace
// part of the GeneralDataSource interface implementation
func (ds *EtcdDataSource) WorkspacePath() string {
	return ds.workspacePath
}

// SelfInfo return InstanceInfo of this instance of blacksmith
func (ds *EtcdDataSource) SelfInfo() InstanceInfo {
	return ds.selfInfo
}

// Machines returns an array of the recognized machines in etcd datasource
// part of GeneralDataSource interface implementation
func (ds *EtcdDataSource) Machines() ([]Machine, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	response, err := ds.keysAPI.Get(ctx, ds.prefixify("/machines"), &etcd.GetOptions{Recursive: false})
	if err != nil {
		return nil, err
	}
	var ret []Machine
	for _, ent := range response.Node.Nodes {
		pathToMachineDir := ent.Key
		machineName := pathToMachineDir[strings.LastIndex(pathToMachineDir, "/")+1:]
		macStr := macFromName(machineName)
		macAddr, err := net.ParseMAC(macStr)
		if err != nil {
			return nil, err
		}
		machine, exist := ds.GetMachine(macAddr)
		if !exist {
			return nil, errors.New("Inconsistent datasource")
		}
		ret = append(ret, machine)
	}
	return ret, nil
}

// GetMachine returns a Machine interface which is the accessor/getter/setter
// for a node in the etcd datasource. If an entry associated with the passed
// mac address does not exist the second return value will be set to false
// part of GeneralDataSource interface implementation
func (ds *EtcdDataSource) GetMachine(mac net.HardwareAddr) (Machine, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	machineName := nameFromMac(mac.String())
	response, err := ds.keysAPI.Get(ctx, ds.prefixify(path.Join("machines/"+machineName)), nil)
	if err != nil {
		return nil, false
	}
	if response.Node.Key[strings.LastIndex(response.Node.Key, "/")+1:] == machineName {
		return &EtcdMachine{mac, ds, ds.keysAPI}, true
	}
	return nil, false
}

// createMachine Creates a machine, returns the handle, and writes directories and flags to etcd
// Second return value determines whether or not Machine creation has been
// successful
// part of GeneralDataSource interface implementation
func (ds *EtcdDataSource) createMachine(mac net.HardwareAddr, ip net.IP) (Machine, bool) {
	machines, err := ds.Machines()

	if err != nil {
		return nil, false
	}
	for _, node := range machines {
		if node.Mac().String() == mac.String() {
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
	machine := &EtcdMachine{mac, ds, ds.keysAPI}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	ds.keysAPI.Set(ctx, ds.prefixify("machines/"+machine.Name()), "", &etcd.SetOptions{Dir: true})
	ctx1, cancel1 := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel1()
	ds.keysAPI.Set(ctx1, ds.prefixify("machines/"+machine.Name()+"/_IP"), ip.String(), &etcd.SetOptions{})

	ctx2, cancel2 := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel2()
	ds.keysAPI.Set(ctx2, ds.prefixify("machines/"+machine.Name()+"/_mac"), machine.Mac().String(), &etcd.SetOptions{})

	ctx3, cancel3 := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel3()
	ds.keysAPI.Set(ctx3, ds.prefixify("machines/"+machine.Name()+"/_first_seen"),
		strconv.FormatInt(time.Now().UnixNano(), 10), &etcd.SetOptions{})

	ctx4, cancel4 := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel4()
	ds.keysAPI.Set(ctx4, "skydns/"+ds.clusterName+"/"+machine.Name(), fmt.Sprintf(`{"host":"%s"}`, ip.String()), nil)

	machine.CheckIn()
	machine.SetFlag("state", "unknown")
	return machine, true
}

// CoreOSVersion gets the current value from etcd and returns it if the image folder exists
// if not, the inital CoreOS version will be returned, with the raised error
// part of GeneralDataSource interface implementation
func (ds *EtcdDataSource) CoreOSVersion() (string, error) {
	coreOSVersion, err := ds.Get(coreosVersionKey)
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

func (ds *EtcdDataSource) prefixify(key string) string {
	return path.Join(ds.clusterName, key)
}

// Get parses the etcd key and returns it's value
// part of GeneralDataSource interface implementation
func (ds *EtcdDataSource) Get(key string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	response, err := ds.keysAPI.Get(ctx, ds.prefixify(key), nil)
	if err != nil {
		return "", err
	}
	return response.Node.Value, nil
}

// Set sets and etcd key to a value
// part of GeneralDataSource interface implementation
func (ds *EtcdDataSource) Set(key string, value string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, err := ds.keysAPI.Set(ctx, ds.prefixify(key), value, nil)
	return err
}

// GetAndDelete gets the value of an etcd key and returns it, and deletes the record
// afterwards
// part of GeneralDataSource interface implementation
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

// Delete erases the key from etcd
// part of GeneralDataSource interface implementation
func (ds *EtcdDataSource) Delete(key string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, err := ds.keysAPI.Delete(ctx, ds.prefixify(key), nil)
	return err
}

// ClusterName returns the name of the cluster
func (ds *EtcdDataSource) ClusterName() string {
	return ds.clusterName
}

type initialValues struct {
	CoreOSVersion string `yaml:"coreos-version"`
}

// LeaseStart returns the first IP address that the DHCP server can offer to a
// DHCP client
// part of DHCPDataSource interface implementation
func (ds *EtcdDataSource) LeaseStart() net.IP {
	return ds.leaseStart
}

// LeaseRange returns the IP range from which IP addresses are assignable to
// clients by the DHCP server
// part of DHCPDataSource interface implementation
func (ds *EtcdDataSource) LeaseRange() int {
	return ds.leaseRange
}

// DNSAddresses returns the ip addresses of the present skydns servers in the
// network, marshalled as specified in rfc2132 (option 6)
// part of DHCPDataSource ineterface implementation
func (ds *EtcdDataSource) DNSAddresses() ([]byte, error) {
	ret := make([]byte, 0)

	// These values are set by hacluster.registerOnEtcd
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	response, err := ds.keysAPI.Get(ctx, ds.prefixify(instancesEtcdDir), &etcd.GetOptions{Recursive: false})
	if err != nil {
		return ret, err
	}
	for _, ent := range response.Node.Nodes {
		ipString := ent.Value

		ret = append(ret, (net.ParseIP(ipString).To4())...)
	}
	return ret, nil
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
	ds.keysAPI.Set(ctx, ds.prefixify("machines/"+m.Name()+"/_IP"),
		ip.String(), &etcd.SetOptions{})

	ctx1, cancel1 := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel1()
	ds.keysAPI.Set(ctx1, ds.prefixify("machines/"+m.Name()+"/_last_seen"),
		strconv.FormatInt(time.Now().UnixNano(), 10), &etcd.SetOptions{})

	ctx2, cancel2 := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel2()
	ds.keysAPI.Set(ctx2, ds.prefixify("machines/"+m.Name()+"/_mac"),
		m.Mac().String(), &etcd.SetOptions{})

}

// Assign assigns an ip to the node with the specified nic
// Will use etcd machines records as LeasePool
// part of DHCPDataSource interface implementation
func (ds *EtcdDataSource) Assign(nic string) (net.IP, error) {
	ds.lockDHCPAssign()
	defer ds.unlockdhcpAssign()

	// TODO: first try to retrieve the machine, if exists (for performance)

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

	//find an unused ip
	for i := 0; i < ds.LeaseRange(); i++ {
		ip := dhcp4.IPAdd(ds.LeaseStart(), i)
		if _, exists := assignedIPs[ip.String()]; !exists {
			macAddress, _ := net.ParseMAC(nic)
			ds.createMachine(macAddress, ip)
			return ip, nil
		}
	}

	//use an expired ip
	//not implemented
	logging.Log(debugTag, "DHCP pool is full")

	return nil, nil
}

// Request answers a dhcp request
// Uses etcd as backend
// part of DHCPDataSource interface implementation
func (ds *EtcdDataSource) Request(nic string, currentIP net.IP) (net.IP, error) {
	ds.lockDHCPAssign()
	defer ds.unlockdhcpAssign()

	machines, _ := ds.Machines()

	macExists, ipExists := false, false

	for _, node := range machines {
		thisNodeIP, _ := node.IP()
		ipMatch := thisNodeIP.String() == currentIP.String()
		macMatch := nic == node.Mac().String()

		if ipMatch && macMatch {
			ds.store(node, thisNodeIP)
			return currentIP, nil
		}

		ipExists = ipExists || ipMatch
		macExists = macExists || macMatch

	}
	if ipExists || macExists {
		return nil, errors.New("Missmatch in lease pool")
	}
	macAddress, _ := net.ParseMAC(nic)
	ds.createMachine(macAddress, currentIP)
	return currentIP, nil
}

// NewEtcdDataSource gives blacksmith the ability to use an etcd endpoint as
// a MasterDataSource
func NewEtcdDataSource(kapi etcd.KeysAPI, client etcd.Client, leaseStart net.IP,
	leaseRange int, clusterName, workspacePath string, defaultNameServers []string,
	selfInfo InstanceInfo) (DataSource, error) {

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
		clusterName:          clusterName,
		leaseStart:           leaseStart,
		leaseRange:           leaseRange,
		workspacePath:        workspacePath,
		initialCoreOSVersion: iVals.CoreOSVersion,
		dhcpAssignLock:       &sync.Mutex{},
		dhcpDataLock:         &sync.Mutex{},
		instanceEtcdKey:      invalidEtcdKey,
		selfInfo:             selfInfo,
	}

	_, err = instance.CoreOSVersion()
	if err != nil {
		etcdError, converted := err.(etcd.Error)
		if converted && etcdError.Code == etcd.ErrorCodeKeyNotFound {
			// Initializing
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			_, err = instance.keysAPI.Set(ctx, instance.prefixify(coreosVersionKey), iVals.CoreOSVersion, nil)
			if err != nil {
				return nil, fmt.Errorf("Error while initializing etcd tree: %s", err)
			}
			fmt.Printf("Initialized etcd tree (%s)", clusterName)
		} else {
			return nil, fmt.Errorf("Error while checking GetCoreOSVersion: %s", err)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	instance.keysAPI.Set(ctx, instance.prefixify("machines"), "", &etcd.SetOptions{Dir: true})

	ctx2, cancel2 := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel2()
	instance.keysAPI.Set(ctx2, "skydns", "", &etcd.SetOptions{Dir: true})

	ctx3, cancel3 := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel3()
	instance.keysAPI.Set(ctx3, "skydns/"+instance.clusterName, "", &etcd.SetOptions{Dir: true})
	quoteEnclosedNameservers := make([]string, 0)
	for _, v := range defaultNameServers {
		quoteEnclosedNameservers = append(quoteEnclosedNameservers, fmt.Sprintf(`"%s:53"`, v))
	}
	commaSeparatedQouteEnclosedNameservers := strings.Join(quoteEnclosedNameservers, ",")

	skydnsconfig := fmt.Sprintf(`{"dns_addr":"0.0.0.0:53","nameservers":[%s],"domain":"%s."}`, commaSeparatedQouteEnclosedNameservers, clusterName)
	ctx4, cancel4 := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel4()
	instance.keysAPI.Set(ctx4, "skydns/config", skydnsconfig, nil)

	return instance, nil
}
