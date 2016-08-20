package datasource

import (
	"fmt"
	"io/ioutil"
	"net"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
	etcd "github.com/coreos/etcd/client"
	"golang.org/x/net/context"
	"gopkg.in/yaml.v2"
)

const (
	etcdMachinesDirName      = "machines"
	etcdCluserVarsDirName    = "cluster-variables"
	etcdConfigurationDirName = "configuration"
	etcdFilesDirName         = "files"
)

// EtcdDataSource implements MasterDataSource interface using etcd as it's
// datasource
// Implements MasterDataSource interface
type EtcdDataSource struct {
	keysAPI         etcd.KeysAPI
	client          etcd.Client
	leaseStart      net.IP
	leaseRange      int
	clusterName     string
	workspacePath   string
	dhcpAssignLock  *sync.Mutex
	instanceEtcdKey string // HA
	selfInfo        InstanceInfo
}

// WorkspacePath returns the path to the workspace
func (ds *EtcdDataSource) WorkspacePath() string {
	return ds.workspacePath
}

// MachineInterfaces returns all the machines in the cluster, as a slice of
// MachineInterfaces
func (ds *EtcdDataSource) MachineInterfaces() ([]MachineInterface, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var ret []MachineInterface

	response, err := ds.keysAPI.Get(ctx, path.Join(ds.clusterName, etcdMachinesDirName), &etcd.GetOptions{Recursive: false})
	if err != nil {
		if etcd.IsKeyNotFound(err) {
			return ret, nil
		}
		return nil, err
	}
	for _, ent := range response.Node.Nodes {
		pathToMachineDir := ent.Key
		machineName := pathToMachineDir[strings.LastIndex(pathToMachineDir, "/")+1:]
		macAddr, err := macFromName(machineName)
		if err != nil {
			return nil, fmt.Errorf("error while converting name to mac: %s", err)
		}
		ret = append(ret, ds.MachineInterface(macAddr))
	}
	return ret, nil
}

// MachineInterface returns the MachineInterface associated with the given mac
func (ds *EtcdDataSource) MachineInterface(mac net.HardwareAddr) MachineInterface {
	return &etcdMachineInterface{
		mac:     mac,
		etcdDS:  ds,
		keysAPI: ds.keysAPI,
	}
}

// Add prefix for cluster variable keys
func (ds *EtcdDataSource) prefixifyForClusterVariables(key string) string {
	return path.Join(ds.ClusterName(), etcdCluserVarsDirName, key)
}

// get expects absolute key path
func (ds *EtcdDataSource) get(keyPath string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	response, err := ds.keysAPI.Get(ctx, keyPath, nil)
	if err != nil {
		return "", err
	}
	return response.Node.Value, nil
}

// set expects absolute key path
func (ds *EtcdDataSource) set(keyPath string, value string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, err := ds.keysAPI.Set(ctx, keyPath, value, nil)
	return err
}

// delete expects absolute key path
func (ds *EtcdDataSource) delete(keyPath string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, err := ds.keysAPI.Delete(ctx, keyPath, nil)
	return err
}

// GetClusterVariable returns a cluster variables with the given name
func (ds *EtcdDataSource) GetClusterVariable(key string) (string, error) {
	return ds.get(ds.prefixifyForClusterVariables(key))
}

func (ds *EtcdDataSource) listNonDirKeyValues(dir string) (map[string]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
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

// ListConfigurations returns the list of all the configuration variables from etcd
func (ds *EtcdDataSource) ListConfigurations() (map[string]string, error) {
	return ds.listNonDirKeyValues(path.Join(ds.clusterName, etcdConfigurationDirName))
}

// SetClusterVariable sets a cluster variable inside etcd
func (ds *EtcdDataSource) SetClusterVariable(key string, value string) error {
	err := validateVariable(key, value)
	if err != nil {
		return err
	}
	return ds.set(ds.prefixifyForClusterVariables(key), value)
}

// DeleteClusterVariable deletes a cluster variable
func (ds *EtcdDataSource) DeleteClusterVariable(key string) error {
	return ds.delete(ds.prefixifyForClusterVariables(key))
}

// ClusterName returns the name of the cluster
func (ds *EtcdDataSource) ClusterName() string {
	return ds.clusterName
}

// EtcdMembers returns a string suitable for `-initial-cluster`
// This is the etcd the Blacksmith instance is using as its datastore
func (ds *EtcdDataSource) EtcdMembers() (string, error) {
	membersAPI := etcd.NewMembersAPI(ds.client)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	members, err := membersAPI.List(ctx)

	if err != nil {
		return "", fmt.Errorf("Error while checking etcd members: %s", err)
	}

	var peers []string
	for _, member := range members {
		for _, peer := range member.PeerURLs {
			peers = append(peers, fmt.Sprintf("%s=%s", member.Name, peer))
		}
	}

	return strings.Join(peers, ","), err
}

// NewEtcdDataSource gives blacksmith the ability to use an etcd endpoint as
// a MasterDataSource
func NewEtcdDataSource(kapi etcd.KeysAPI, client etcd.Client, leaseStart net.IP,
	leaseRange int, clusterName, workspacePath string, defaultNameServers []string,
	selfInfo InstanceInfo) (DataSource, error) {

	data, err := ioutil.ReadFile(filepath.Join(workspacePath, "initial.yaml"))
	if err != nil {
		return nil, fmt.Errorf("error while trying to read initial data: %s", err)
	}

	iVals := make(map[string]string)
	err = yaml.Unmarshal(data, &iVals)
	if err != nil {
		return nil, fmt.Errorf("error while reading initial data: %s", err)
	}

	ds := &EtcdDataSource{
		keysAPI:         kapi,
		client:          client,
		clusterName:     clusterName,
		leaseStart:      leaseStart,
		leaseRange:      leaseRange,
		workspacePath:   workspacePath,
		dhcpAssignLock:  &sync.Mutex{},
		instanceEtcdKey: invalidEtcdKey,
		selfInfo:        selfInfo,
	}

	for key, value := range iVals {
		currentValue, _ := ds.GetClusterVariable(key)
		if len(currentValue) == 0 {
			err := ds.SetClusterVariable(key, value)

			if err != nil {
				return nil,
					fmt.Errorf("error while setting initial value (%q: %q): %s",
						key, value, err)
			}

			currentValue = value
		}
		log.WithFields(log.Fields{
			"where":   "datasource.NewEtcdDataSource",
			"action":  "debug",
			"object":  "ClusterVariable",
			"subject": key,
		}).Debugf("%s=%q", key, value)
	}

	// TODO: Integrate DNS service into Blacksmith
	ctx2, cancel2 := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel2()
	ds.keysAPI.Set(ctx2, "skydns", "", &etcd.SetOptions{Dir: true})

	ctx3, cancel3 := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel3()
	ds.keysAPI.Set(ctx3, "skydns/"+ds.clusterName, "", &etcd.SetOptions{Dir: true})
	var quoteEnclosedNameservers []string
	for _, v := range defaultNameServers {
		quoteEnclosedNameservers = append(quoteEnclosedNameservers, fmt.Sprintf(`"%s:53"`, v))
	}
	commaSeparatedQouteEnclosedNameservers := strings.Join(quoteEnclosedNameservers, ",")

	skydnsconfig := fmt.Sprintf(`{"dns_addr":"0.0.0.0:53","nameservers":[%s],"domain":"%s."}`, commaSeparatedQouteEnclosedNameservers, clusterName)
	ctx4, cancel4 := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel4()
	ds.keysAPI.Set(ctx4, "skydns/config", skydnsconfig, nil)

	_, err = ds.MachineInterface(selfInfo.Nic).Machine(true, selfInfo.IP)
	if err != nil {
		return nil, fmt.Errorf("error while creating the machine representation of self: %s", err)
	}

	return ds, nil
}
