package datasource

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	etcd "github.com/coreos/etcd/client"
	"github.com/krolaw/dhcp4"
	"golang.org/x/net/context"
	"gopkg.in/yaml.v2"

	"github.com/cafebazaar/blacksmith/logging"
	"encoding/json"
)

const (
	coreosVersionKey = "coreos-version"

	etcdMachinesDirName      = "machines"
	etcdCluserVarsDirName    = "cluster-variables"
	etcdConfigurationDirName = "configuration"
)

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
// Second return value will be an error if creating machine wasn't successful otherwise it will be nil
// part of GeneralDataSource interface implementation
func (ds *EtcdDataSource) createMachine(mac net.HardwareAddr, ip net.IP) (Machine, error) {
	machines, err := ds.Machines()

	if err != nil {
		return nil, err
	}
	for _, node := range machines {
		if node.Mac().String() == mac.String() {
			return nil, err
		}
		stats, err:= node.GetStats()
		if err != nil {
			return nil, err
		}
		if stats.IP.String() == ip.String() {
			return nil, err
		}
	}
	machine := &EtcdMachine{mac, ds, ds.keysAPI}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	ds.keysAPI.Set(ctx, ds.prefixify("machines/"+machine.Name()), "", &etcd.SetOptions{Dir: true})

	ds.store(machine, ip)

	ctx4, cancel4 := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel4()
	ds.keysAPI.Set(ctx4, "skydns/"+ds.clusterName+"/"+machine.Name(), fmt.Sprintf(`{"host":"%s"}`, ip.String()), nil)

	machine.CheckIn()
	machine.SetFlag("state", "unknown")
	return machine, nil
}

// CoreOSVersion gets the current value from etcd and returns it if the image folder exists
// if not, the inital CoreOS version will be returned, with the raised error
// part of GeneralDataSource interface implementation
func (ds *EtcdDataSource) CoreOSVersion() (string, error) {
	coreOSVersion, err := ds.get(coreosVersionKey)
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

// Add prefix for cluster variable keys
func (ds *EtcdDataSource) prefixifyForClusterVariables(key string) string {
	return path.Join(ds.clusterName, etcdCluserVarsDirName, key)
}

func (ds *EtcdDataSource) prefixifyForConfiguration(key string) string {
	return path.Join(ds.clusterName, etcdConfigurationDirName, key)
}

func (ds *EtcdDataSource) get(keyPath string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	response, err := ds.keysAPI.Get(ctx, keyPath, nil)
	if err != nil {
		return "", err
	}
	return response.Node.Value, nil
}

func (ds *EtcdDataSource) set(keyPath string, value string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, err := ds.keysAPI.Set(ctx, keyPath, value, nil)
	return err
}

// GetClusterVariable returns a cluster variables with the given name
func (ds *EtcdDataSource) GetClusterVariable(key string) (string, error) {
	return ds.get(ds.prefixifyForClusterVariables(key))
}

// GetConfiguration returns a configuration variables with the given name
func (ds *EtcdDataSource) GetConfiguration(key string) (string, error) {
	return ds.get(ds.prefixifyForClusterVariables(key))
}

func (ds *EtcdDataSource) delete(keyPath string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, err := ds.keysAPI.Delete(ctx, keyPath, nil)
	return err
}

// DeleteClusterVariable deletes a cluster variable
func (ds *EtcdDataSource) DeleteClusterVariable(key string) error {
	return ds.delete(ds.prefixifyForClusterVariables(key))
}

// DeleteConfiguration deletes a configuration variable
func (ds *EtcdDataSource) DeleteConfiguration(key string) error {
	return ds.delete(ds.prefixifyForConfiguration(key))

}

// Get parses the etcd key and returns it's value
// part of GeneralDataSource interface implementation
func (ds *EtcdDataSource) Get(key string) (string, error) {
	return ds.GetAbsolute(ds.prefixify(key))
}

func (ds *EtcdDataSource) GetAbsolute(absoluteKey string)(string, error){
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	response, err := ds.keysAPI.Get(ctx, absoluteKey, nil)
	if err != nil {
		return "", err
	}
	return response.Node.Value, nil
}

func (ds *EtcdDataSource) listNonDirKeyValues(dir string) (map[string]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3 * time.Second)
	defer cancel()

	response, err := ds.keysAPI.Get(ctx, dir, nil)
	if err != nil {
		return nil, err
	}

	flags := make(map[string]string)
	for _, n := range response.Node.Nodes {
		if n.Dir {
			continue
		}
		_, k := path.Split(n.Key)
		flags[k] = n.Value
	}

	return flags, nil
}

// ListClusterVariables returns the list of all the cluster variables from etcd
func (ds *EtcdDataSource) ListClusterVariables() (map[string]string, error) {
	return ds.listNonDirKeyValues(path.Join(ds.clusterName, etcdCluserVarsDirName))
}

// Get an etcd key and returns it's chlidren nodes
func (ds *EtcdDataSource) GetNodes(key string) (etcd.Nodes, error) {
	ctx, cancel := context.WithTimeout(context.Background(), etcdTimeout)
	defer cancel()

	options := etcd.GetOptions{
		Recursive: true,
		Quorum:    true,
		Sort:      true,
	}
	resp, err := ds.keysAPI.Get(ctx, key, &options)
	if err != nil {
		logging.Debug(debugTag, "couldn't get files from etcd due to: %s", err)
		return nil, err
	}
	return resp.Node.Nodes, nil

}

// Set sets and etcd key to a value
// part of GeneralDataSource interface implementation
func (ds *EtcdDataSource) Set(key string, value string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, err := ds.keysAPI.Set(ctx, ds.prefixify(key), value, nil)
	return err
}

// ListConfigurations returns the list of all the configuration variables from etcd
func (ds *EtcdDataSource) ListConfigurations() (map[string]string, error) {
	return ds.listNonDirKeyValues(path.Join(ds.clusterName, etcdConfigurationDirName))
}

// Delete erases the key from etcd and return the node
// part of GeneralDataSource interface implementation
func (ds *EtcdDataSource) Delete(key string) (*etcd.Node, error) {
	return ds.DeleteAbsolute(ds.prefixify(key))
}

// Delete a node on etcd with a key and return the node
func (ds *EtcdDataSource) DeleteAbsolute(absoluteKey string) (*etcd.Node, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	resp, err := ds.keysAPI.Delete(ctx, absoluteKey, nil)
	return resp.PrevNode, err
}

// SetClusterVariable sets a cluster variable inside etcd
func (ds *EtcdDataSource) SetClusterVariable(key string, value string) error {
	return ds.set(ds.prefixifyForClusterVariables(key), value)
}

// SetConfiguration sets a configuration variable inside etcd
func (ds *EtcdDataSource) SetConfiguration(key string, value string) error {
	return ds.set(ds.prefixifyForConfiguration(key), value)
}

// ClusterName returns the name of the cluster
func (ds *EtcdDataSource) ClusterName() string {
	return ds.clusterName
}

type initialValues struct {
	CoreOSVersion string `yaml:"coreos-version"`
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

	stats := EtcdMachineStats{
		IP:ip,
		Mac:m.Mac().String(),
		FirstSeen:time.Now().Unix(),
	}
	jsonedStats, _ := json.Marshal(stats)
	ds.keysAPI.Set(ctx, ds.prefixify("machines/"+m.Name()+"/_stats"), string(jsonedStats), &etcd.SetOptions{})
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
			stats, _ := node.GetStats()
			ds.store(node, stats.IP)
			return stats.IP, nil
		}
		stats, _ := node.GetStats()
		assignedIPs[stats.IP.String()] = true
	}

	//find an unused ip
	for i := 0; i < ds.leaseRange; i++ {
		ip := dhcp4.IPAdd(ds.leaseStart, i)
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
		stats, _ := node.GetStats()
		ipMatch := stats.IP.String() == currentIP.String()
		macMatch := nic == node.Mac().String()

		if ipMatch && macMatch {
			ds.store(node, stats.IP)
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

//EtcdMembers get etcd members
func (ds *EtcdDataSource) EtcdMembers() (string, error) {
	membersAPI := etcd.NewMembersAPI(ds.client)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	members, err := membersAPI.List(ctx)

	if err != nil {
		return "", fmt.Errorf("Error while checking etcd members: %s", err)
	}

	var buffer bytes.Buffer

	for _, member := range members {
		lastIndex := len(member.PeerURLs) - 1

		for i, peer := range member.PeerURLs {
			buffer.WriteString(member.Name)
			buffer.WriteString("=")
			buffer.WriteString(peer)

			if i != lastIndex {
				buffer.WriteString(",")
			}
		}
	}

	return buffer.String(), err
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

	_, err = instance.createMachine(selfInfo.Nic, selfInfo.IP)
	if err != nil {
		logging.Debug(debugTag, "couldn't create machine instance inside etcd for itself due to: %s", err)
	}

	return instance, nil
}
