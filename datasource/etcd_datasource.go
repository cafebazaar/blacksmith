package datasource

import (
	"crypto/md5"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	yaml "gopkg.in/yaml.v2"

	log "github.com/Sirupsen/logrus"
	etcd "github.com/coreos/etcd/client"
	git "github.com/libgit2/git2go"
	"github.com/pkg/errors"
	"golang.org/x/net/context"
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
	workspaceRepo   string
	fileServer      string
	webServer       string
	dhcpAssignLock  *sync.Mutex
	instanceEtcdKey string // HA
	selfInfo        InstanceInfo
}

// WorkspacePath returns the path to the workspace
func (ds *EtcdDataSource) WorkspacePath() string {
	return ds.workspacePath
}

// FileServer returns the path to the workspace
func (ds *EtcdDataSource) FileServer() string {
	return ds.fileServer
}

func (ds *EtcdDataSource) WebServer() string {
	return ds.webServer
}

func (ds *EtcdDataSource) SetWebServer(addr string) {
	ds.webServer = addr
}

// WorkspaceRepo returns the workspace repository URL
func (ds *EtcdDataSource) WorkspaceRepo() string {
	return ds.fileServer
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

// create expects absolute key path
func (ds *EtcdDataSource) create(keyPath string, value string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, err := ds.keysAPI.Create(ctx, keyPath, value)
	return err
}

// get expects absolute key path
func (ds *EtcdDataSource) getArray(keyPath string) (interface{}, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	ops := &etcd.GetOptions{Recursive: true}
	response, err := ds.keysAPI.Get(ctx, keyPath, ops)
	if err != nil {
		return "", err
	}
	return response.Node.Nodes, nil
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

func (ds *EtcdDataSource) watchOnce(keyPath string) (*etcd.Response, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3600*time.Second)
	watcher := ds.keysAPI.Watcher(keyPath, nil)
	defer cancel()
	resp, err := watcher.Next(ctx)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// GetClusterVariable returns a cluster variables with the given name
func (ds *EtcdDataSource) GetClusterVariable(key string) (string, error) {
	return ds.get(ds.prefixifyForClusterVariables(key))
}

// GetClusterArrayVariable returns a cluster variables with the given name
func (ds *EtcdDataSource) GetArrayVariable(key string) (interface{}, error) {
	return ds.getArray(path.Join(ds.ClusterName(), key))
}

// GetClusterArrayVariable returns a cluster variables with the given name
func (ds *EtcdDataSource) GetVariable(key string) (string, error) {
	return ds.get(key)
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

// GetWorkspaceHash returns worspace hash
func (ds *EtcdDataSource) GetWorkspaceHash() (string, error) {
	workspaceHash, err := ds.get(path.Join(ds.ClusterName(), "workspace-hash"))
	return workspaceHash, err
}

func (ds *EtcdDataSource) UpdateSignal() error {
	err := ds.set(path.Join(ds.ClusterName(), "workspace-update"), hashGenerator())
	return err
}

func hashGenerator() string {

	h := md5.New()
	io.WriteString(h, time.Now().String())
	io.WriteString(h, "00f468d3bde3")

	hashStr := fmt.Sprintf("%x", h.Sum(nil))

	return hashStr
}

// UpdateWorkspace updates workspace
func (ds *EtcdDataSource) UpdateWorkspace() error {

	var erro error
	erro = nil
	cloneOptions := &git.CloneOptions{}
	// use FetchOptions instead of directly RemoteCallbacks
	// https://github.com/libgit2/git2go/commit/36e0a256fe79f87447bb730fda53e5cbc90eb47c
	cloneOptions.FetchOptions = &git.FetchOptions{
		RemoteCallbacks: git.RemoteCallbacks{
			CredentialsCallback: func(url string, username string, allowedTypes git.CredType) (git.ErrorCode, *git.Cred) {
				ret, cred := git.NewCredSshKey("git", path.Join(ds.workspacePath, "id_rsa.pub"),
					path.Join(ds.workspacePath, "id_rsa"), "")
				return git.ErrorCode(ret), &cred
			},
			CertificateCheckCallback: func(cert *git.Certificate, valid bool, hostname string) git.ErrorCode {
				return 0
			},
		},
	}

	if ds.selfInfo.DebugMode == "true" {
		cloneOptions.CheckoutBranch = "dev"
	}

	os.RemoveAll(path.Join(ds.workspacePath, "repo"))
	cloned, err := git.Clone(ds.workspaceRepo, path.Join(ds.workspacePath, "repo"), cloneOptions)
	if err != nil {
		return err
	}

	head, err := cloned.Head()
	if err != nil {
		return err
	}
	localCommit, err := cloned.LookupCommit(head.Target())
	if err != nil {
		return err
	}

	err = ds.set(path.Join(ds.ClusterName(), "blacksmith-instances", colonlessMacToMac(ds.selfInfo.Nic.String()), "workspace-commit-hash"), localCommit.Id().String())
	if err != nil {
		return err
	}

	err = ds.create(path.Join(ds.ClusterName(), "workspace-lock"), hashGenerator())
	alreadyExists := false
	if cErr, ok := err.(etcd.Error); ok {
		if cErr.Code == etcd.ErrorCodeNodeExist {
			alreadyExists = true
		}
	}
	if alreadyExists {
		log.Info("Already locked!")
		ds.watchOnce(path.Join(ds.ClusterName(), "workspace-lock"))
		log.Info("Unlocked!")
	} else {
		log.Info("Locked!")
		defer func() {
			err = ds.delete(path.Join(ds.ClusterName(), "workspace-lock"))
			if err != nil {
				erro = err
			}
		}()
	}

	c := make(chan bool)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	ops := &etcd.GetOptions{Recursive: true}
	response, err := ds.keysAPI.Get(ctx, path.Join(ds.ClusterName(), "blacksmith-instances"), ops)
	if err != nil {
		return err
	}

	for _, node := range response.Node.Nodes {
		go func(c chan bool, node *etcd.Node) {
			defer func() { c <- true }()
			for {

				workspaceCommit, err := ds.get(path.Join(node.Key, "workspace-commit-hash"))

				if err != nil {
					log.Error(err.Error())
					continue
				}

				oid, _ := git.NewOid(workspaceCommit)
				log.Info("Local commit: ", localCommit.Id())
				log.Info("Node commit: ", oid)

				log.Info("commit behind HEAD: ", localCommit.Id().Cmp(oid))
				log.Info("Checking ", path.Join(node.Key, "workspace-commit-hash"))

				if localCommit.Id().Cmp(oid) == -1 {
					resp, err := ds.watchOnce(path.Join(node.Key, "workspace-commit-hash"))
					if err != nil {
						log.Error(err.Error())
						continue
					}

					if resp != nil {
						break
					}
				} else {
					break
				}
			}

		}(c, node)
	}

	for range response.Node.Nodes {
		<-c
	}

	err = ds.set(path.Join(ds.ClusterName(), "workspace-hash"), hashGenerator())
	if err != nil {
		return err
	}

	return erro
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

// EtcdEndpoints returns a string suitable for etcdctl
// This is the etcd the Blacksmith instance is using as its datastore

func (ds *EtcdDataSource) EtcdEndpoints() (string, error) {
	membersAPI := etcd.NewMembersAPI(ds.client)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	members, err := membersAPI.List(ctx)

	if err != nil {
		return "", fmt.Errorf("Error while checking etcd members: %s", err)
	}

	var peers []string
	for _, member := range members {
		for _, peer := range member.ClientURLs {
			peers = append(peers, fmt.Sprintf("%s", peer))
		}
	}

	return strings.Join(peers, ","), err
}

func (ds *EtcdDataSource) iterateOverYaml(iVals interface{}, pathStr string) error {
	switch t := iVals.(type) {
	default:
	case string:
		currentValue, _ := ds.get(pathStr)
		if len(currentValue) == 0 {
			err := ds.set(pathStr, t)

			if err != nil {
				return fmt.Errorf("error while setting initial value (%s: %s): %s",
					t, t, err)
			}

		}
		break
	case map[interface{}]string:
		for key, value := range t {
			currentValue, _ := ds.get(path.Join(pathStr, key.(string)))
			if len(currentValue) == 0 {
				err := ds.set(path.Join(pathStr, key.(string)), value)

				if err != nil {
					return fmt.Errorf("error while setting initial value (%s: %s): %s",
						t, t, err)
				}

			}

		}
		break
	case map[interface{}]interface{}:
		for key, value := range t {
			ds.iterateOverYaml(value, path.Join(pathStr, key.(string)))
		}
		break
	case []interface{}:

		for key, value := range t {
			ds.iterateOverYaml(value, path.Join(pathStr, string(key)))
		}

		break
	}

	return nil
}

// NewEtcdDataSource gives blacksmith the ability to use an etcd endpoint as
// a MasterDataSource
func NewEtcdDataSource(kapi etcd.KeysAPI, client etcd.Client, leaseStart net.IP,
	leaseRange int, clusterName, workspacePath string, workspaceRepo string,
	fileServer string, defaultNameServers []string,
	selfInfo InstanceInfo) (DataSource, error) {

	ds := &EtcdDataSource{
		keysAPI:         kapi,
		client:          client,
		clusterName:     clusterName,
		leaseStart:      leaseStart,
		leaseRange:      leaseRange,
		workspacePath:   workspacePath,
		workspaceRepo:   workspaceRepo,
		fileServer:      fileServer,
		dhcpAssignLock:  &sync.Mutex{},
		instanceEtcdKey: invalidEtcdKey,
		selfInfo:        selfInfo,
	}

	cloneOptions := &git.CloneOptions{}
	// use FetchOptions instead of directly RemoteCallbacks
	// https://github.com/libgit2/git2go/commit/36e0a256fe79f87447bb730fda53e5cbc90eb47c
	cloneOptions.FetchOptions = &git.FetchOptions{
		RemoteCallbacks: git.RemoteCallbacks{
			CredentialsCallback: func(url string, username string, allowedTypes git.CredType) (git.ErrorCode, *git.Cred) {
				ret, cred := git.NewCredSshKey("git", path.Join(ds.workspacePath, "id_rsa.pub"),
					path.Join(ds.workspacePath, "id_rsa"), "")
				return git.ErrorCode(ret), &cred
			},
			CertificateCheckCallback: func(cert *git.Certificate, valid bool, hostname string) git.ErrorCode {
				return 0
			},
		},
	}

	if ds.selfInfo.DebugMode == "true" {
		cloneOptions.CheckoutBranch = "dev"
	}

	os.RemoveAll(path.Join(workspacePath, "repo"))
	cloned, err := git.Clone(workspaceRepo, path.Join(workspacePath, "repo"), cloneOptions)
	if err != nil {
		return nil, errors.Wrapf(err,
			"error while cloning %s to %s with options %+v", workspaceRepo, path.Join(workspacePath, "repo"), cloneOptions,
		)
	}

	head, err := cloned.Head()
	if err != nil {
		return nil, errors.Wrapf(err, "error while getting repository head for %s", workspaceRepo)
	}
	localCommit, err := cloned.LookupCommit(head.Target())
	if err != nil {
		return nil, errors.Wrapf(err, "error while looking up commit %s for %s", localCommit, workspaceRepo)
	}

	key := path.Join(ds.ClusterName(), "blacksmith-instances", colonlessMacToMac(ds.selfInfo.Nic.String()), "workspace-commit-hash")
	if err := ds.set(key, localCommit.Id().String()); err != nil {
		return nil, errors.Wrapf(err, "error while setting ds key %q to %q", "workspace-commit-hash", localCommit.Id().String())
	}

	data, err := ioutil.ReadFile(filepath.Join(workspacePath, "initial.yaml"))
	if err != nil {
		return nil, fmt.Errorf("error while trying to read initial data: %s", err)
	}

	iVals := make(map[interface{}]interface{})
	err = yaml.Unmarshal(data, &iVals)
	if err != nil {
		return nil, fmt.Errorf("error while reading initial data: %s", err)
	}

	ds.iterateOverYaml(iVals, ds.ClusterName())

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

	err = ds.set(path.Join(ds.ClusterName(), "blacksmith-instances", colonlessMacToMac(selfInfo.Nic.String()), "ip"), selfInfo.IP.String())
	if err != nil {
		return nil, err
	}
	return ds, nil
}
