package datasource

import (
	"bytes"
	"crypto/md5"
	"encoding/base64"
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

	"srcd.works/go-git.v4/plumbing/transport"
	gitssh "srcd.works/go-git.v4/plumbing/transport/ssh"

	git "srcd.works/go-git.v4"
	"srcd.works/go-git.v4/plumbing"
	"srcd.works/go-git.v4/plumbing/object"

	yaml "gopkg.in/yaml.v2"

	log "github.com/Sirupsen/logrus"
	etcd "github.com/coreos/etcd/client"
	"github.com/pkg/errors"
	"golang.org/x/crypto/ssh"
	"golang.org/x/net/context"
)

const (
	etcdMachinesDirName           = "machines"
	etcdCluserVarsDirName         = "cluster-variables"
	etcdBlacksmithConfVarsDirName = "blacksmith-variables"
	etcdConfigurationDirName      = "configuration"
	etcdFilesDirName              = "files"
	etcdUpdateMeValue             = "update-me"
)

// EtcdDatasource provides the interface for querying general information
type EtcdDatasource struct {
	keysAPI         etcd.KeysAPI
	client          etcd.Client
	leaseStart      net.IP
	leaseRange      int
	clusterName     string
	workspacePath   string
	workspaceRepo   string
	workspaceBranch string
	initialConfig   string
	fileServer      string
	webServer       string
	dhcpAssignLock  *sync.Mutex
	instanceEtcdKey string
	selfInfo        InstanceInfo
}

// WorkspacePath returns the path to the workspace
func (ds *EtcdDatasource) WorkspacePath() string {
	return ds.workspacePath
}

// FileServer returns the path to the workspace
func (ds *EtcdDatasource) FileServer() string {
	return ds.fileServer
}

func (ds *EtcdDatasource) WebServer() string {
	return ds.webServer
}

func (ds *EtcdDatasource) SetWebServer(addr string) {
	ds.webServer = addr
}

// WorkspaceRepo returns the workspace repository URL
func (ds *EtcdDatasource) WorkspaceRepo() string {
	return ds.fileServer
}

// GetEtcdMachines returns all the machines in the cluster
func (ds *EtcdDatasource) GetEtcdMachines() ([]*EtcdMachine, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var ret []*EtcdMachine

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
			return nil, fmt.Errorf("failed to convert name %q to mac: %s", machineName, err)
		}
		ret = append(ret, ds.GetEtcdMachine(macAddr))
	}
	return ret, nil
}

// GetEtcdMachine returns the EtcdMachine associated with the given mac
func (ds *EtcdDatasource) GetEtcdMachine(mac net.HardwareAddr) *EtcdMachine {
	return &EtcdMachine{
		mac:    mac,
		etcdDS: ds,
	}
}

// Add prefix for cluster variable keys
func (ds *EtcdDatasource) prefixifyForClusterVariables(key string) string {
	return path.Join(ds.ClusterName(), etcdCluserVarsDirName, key)
}

func (ds *EtcdDatasource) prefixifyForBlacksmithVariables(key string) string {
	return path.Join(ds.ClusterName(), etcdBlacksmithConfVarsDirName, key)
}

// get expects absolute key path
func (ds *EtcdDatasource) get(keyPath string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	response, err := ds.keysAPI.Get(ctx, keyPath, nil)
	if err != nil {
		return "", err
	}
	return response.Node.Value, nil
}

// create expects absolute key path
func (ds *EtcdDatasource) create(keyPath string, value string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, err := ds.keysAPI.Create(ctx, keyPath, value)
	return err
}

// get expects absolute key path
func (ds *EtcdDatasource) getArray(keyPath string) (etcd.Nodes, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	ops := &etcd.GetOptions{Recursive: true}
	response, err := ds.keysAPI.Get(ctx, keyPath, ops)
	if err != nil {
		return nil, err
	}
	return response.Node.Nodes, nil
}

func (ds *EtcdDatasource) setArray(keyPath string, values []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	for i, v := range values {
		_, err := ds.keysAPI.Set(ctx, path.Join(keyPath, fmt.Sprintf("item-%d", i)), v, nil)
		if err != nil {
			return err
		}
	}
	return nil
}

// set expects absolute key path
func (ds *EtcdDatasource) set(keyPath string, value string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, err := ds.keysAPI.Set(ctx, keyPath, value, nil)
	return err
}

// delete expects absolute key path
func (ds *EtcdDatasource) delete(keyPath string, opts *etcd.DeleteOptions) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, err := ds.keysAPI.Delete(ctx, keyPath, opts)
	return err
}

func (ds *EtcdDatasource) watchOnce(keyPath string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	watcher := ds.keysAPI.Watcher(keyPath, nil)
	defer cancel()
	resp, err := watcher.Next(ctx)
	if err != nil {
		return "", err
	}
	return resp.Node.Value, nil
}

// GetClusterVariable returns a cluster variables with the given name
func (ds *EtcdDatasource) GetClusterVariable(key string) (string, error) {
	return ds.get(ds.prefixifyForClusterVariables(key))
}

func (ds *EtcdDatasource) SetArrayVariable(key string, values []string) error {
	return ds.setArray(path.Join(ds.ClusterName(), key), values)
}

// GetClusterArrayVariable returns a cluster variables with the given name
func (ds *EtcdDatasource) GetArrayVariable(key string) (etcd.Nodes, error) {
	return ds.getArray(path.Join(ds.ClusterName(), key))
}

// GetClusterArrayVariable returns a cluster variables with the given name
func (ds *EtcdDatasource) GetVariable(key string) (string, error) {
	return ds.get(key)
}

func (ds *EtcdDatasource) listNonDirKeyValues(dir string) (map[string]string, error) {
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
func (ds *EtcdDatasource) ListClusterVariables() (map[string]string, error) {
	return ds.listNonDirKeyValues(path.Join(ds.clusterName, etcdCluserVarsDirName))
}

// ListConfigurations returns the list of all the configuration variables from etcd
func (ds *EtcdDatasource) ListConfigurations() (map[string]string, error) {
	return ds.listNonDirKeyValues(path.Join(ds.clusterName, etcdConfigurationDirName))
}

// SetBlacksmithVariable sets a blacksmith config variable inside etcd
func (ds *EtcdDatasource) SetBlacksmithVariable(key string, value string) error {
	err := validateVariable(key, value)
	if err != nil {
		return err
	}
	return ds.set(ds.prefixifyForBlacksmithVariables(key), value)
}

func (ds *EtcdDatasource) GetBlacksmithVariable(key string) (string, error) {
	return ds.get(ds.prefixifyForBlacksmithVariables(key))
}

// SetClusterVariable sets a cluster variable inside etcd
func (ds *EtcdDatasource) SetClusterVariable(key string, value string) error {
	err := validateVariable(key, value)
	if err != nil {
		return err
	}
	return ds.set(ds.prefixifyForClusterVariables(key), value)
}

// DeleteClusterVariable deletes a cluster variable
func (ds *EtcdDatasource) DeleteClusterVariable(key string) error {
	return ds.delete(ds.prefixifyForClusterVariables(key), nil)
}

// GetWorkspaceHash returns worspace hash
func (ds *EtcdDatasource) GetWorkspaceHash() (string, error) {
	// TODO: this is meaningless if machines are updated one by one
	workspaceHash, err := ds.get(path.Join(ds.ClusterName(), "workspace-hash"))
	return workspaceHash, err
}

func hashGenerator() string {

	h := md5.New()
	io.WriteString(h, time.Now().String())
	io.WriteString(h, "00f468d3bde3")

	hashStr := fmt.Sprintf("%x", h.Sum(nil))

	return hashStr
}

func (ds *EtcdDatasource) getPrivateKey() []byte {
	priKey, err := ds.GetBlacksmithVariable("private-key")
	if err != nil {
		return []byte{}
	}
	if priKey == "" {
		return []byte{}
	}
	priKeyBytes, err := base64.StdEncoding.DecodeString(priKey)
	if err != nil {
		return []byte{}
	}
	return priKeyBytes
}

func (ds *EtcdDatasource) UpdateMyWorkspaceLoop() error {
	k := path.Join(
		ds.ClusterName(),
		"blacksmith-instances",
		colonlessMacToMac(ds.selfInfo.Nic.String()),
		"workspace-commit-hash",
	)

	watcher := ds.keysAPI.Watcher(k, nil)
	log.WithFields(log.Fields{
		"key": k,
	}).Info("UPDATE: watching for update signal")
	for {
		resp, err := watcher.Next(context.Background())
		if err != nil {
			return err
		}
		log.WithFields(log.Fields{
			"key":   k,
			"value": resp.Node.Value,
		}).Info("UPDATE: new value")
		if resp.Node.Value == etcdUpdateMeValue {
			ds.UpdateMyWorkspace()
		}
	}
}

// UpdateMyWorkspace clones the workspace repo of the current instance
func (ds *EtcdDatasource) UpdateMyWorkspace() error {
	branch := fmt.Sprintf("refs/heads/%s", ds.workspaceBranch)
	dir := path.Join(ds.workspacePath, "repo")
	repo, err := clone(dir, ds.workspaceRepo, branch, ds.getPrivateKey())
	if err != nil {
		return errors.Wrapf(err,
			"cloning %s branch %s to %s failed",
			ds.workspaceRepo, branch, dir,
		)
	}
	head, err := repo.Head()
	if err != nil {
		return errors.Wrap(err, "failed to get repo head")
	}

	k := path.Join(
		ds.ClusterName(),
		"blacksmith-instances",
		colonlessMacToMac(ds.selfInfo.Nic.String()),
		"workspace-commit-hash",
	)
	return ds.set(k, head.Hash().String())
}

// UpdateWorkspaces requests all instances to update
// their workspace. It blocks until all instances have
// completed their cloning.
func (ds *EtcdDatasource) UpdateWorkspaces() error {
	log.Info("UPDATE: UpdateWorkspaces called!")
	ctx, _ := context.WithTimeout(context.Background(), 3*time.Second)
	response, err := ds.keysAPI.Get(
		ctx,
		path.Join(ds.ClusterName(), "blacksmith-instances"),
		&etcd.GetOptions{Recursive: true},
	)
	if err != nil {
		return err
	}

	wg := sync.WaitGroup{}
	for _, node := range response.Node.Nodes {
		wg.Add(1)
		go func(node *etcd.Node) {
			defer wg.Done()
			k := path.Join(node.Key, "workspace-commit-hash")
			log.Infof("UPDATE: node: %s", k)
			if err := ds.set(k, etcdUpdateMeValue); err != nil {
				log.Error(err)
				return
			}
			// Wait until node's blacksmith changes the value from
			// etcdUpdateMeValue to a commit hash.
			if _, err := ds.watchOnce(k); err != nil {
				log.Error(err)
			}
		}(node)
	}
	wg.Wait()

	newWorkspaceCommitHash := ""
	for _, node := range response.Node.Nodes {
		k := path.Join(node.Key, "workspace-commit-hash")
		val, err := ds.get(k)
		if err != nil {
			return err
		}
		if newWorkspaceCommitHash == "" {
			newWorkspaceCommitHash = val
		} else if newWorkspaceCommitHash != val {
			return fmt.Errorf("workspace commit hashes on at least one instances does not match, %s != %s", newWorkspaceCommitHash, val)
		}
		if err := ds.delete(k, nil); err != nil {
			return err
		}
	}

	return ds.set(path.Join(ds.ClusterName(), "workspace"), newWorkspaceCommitHash)
}

// ClusterName returns the name of the cluster
func (ds *EtcdDatasource) ClusterName() string {
	return ds.clusterName
}

// EtcdMembers returns a string suitable for `-initial-cluster`
// This is the etcd the Blacksmith instance is using as its datastore
func (ds *EtcdDatasource) EtcdMembers() (string, error) {
	membersAPI := etcd.NewMembersAPI(ds.client)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	members, err := membersAPI.List(ctx)

	if err != nil {
		return "", err
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
func (ds *EtcdDatasource) EtcdEndpoints() (string, error) {
	membersAPI := etcd.NewMembersAPI(ds.client)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	members, err := membersAPI.List(ctx)
	if err != nil {
		return "", err
	}

	var clients []string
	for _, member := range members {
		for _, peer := range member.ClientURLs {
			clients = append(clients, fmt.Sprintf("%s", peer))
		}
	}

	return strings.Join(clients, ","), err
}

func (ds *EtcdDatasource) copyToDatasource(iVals interface{}, pathStr string) error {
	switch val := iVals.(type) {
	default:
	case string:
		currentValue, _ := ds.get(pathStr)
		if currentValue == "" {
			err := ds.set(pathStr, val)
			if err != nil {
				return errors.Wrapf(err, "Failed to set initial value %s=%s", pathStr, val)
			}
		}
	case map[interface{}]string:
		for key, value := range val {
			currentValue, _ := ds.get(path.Join(pathStr, key.(string)))
			if currentValue == "" {
				err := ds.set(path.Join(pathStr, key.(string)), value)
				if err != nil {
					return errors.Wrapf(err, "Failed to set initial value %s=%s)", pathStr, val)
				}
			}
		}
	case map[interface{}]interface{}:
		for key, value := range val {
			ds.copyToDatasource(value, path.Join(pathStr, key.(string)))
		}
	case []interface{}:
		for idx, value := range val {
			ds.copyToDatasource(value, path.Join(pathStr, string(idx)))
		}
	}

	return nil
}

// NewEtcdDataSource gives blacksmith the ability to use an etcd endpoint as
// a MasterDataSource
func NewEtcdDataSource(
	kapi etcd.KeysAPI,
	client etcd.Client,
	leaseStart net.IP,
	leaseRange int,
	clusterName,
	workspacePath string,
	workspaceRepo string,
	workspaceBranch string,
	privateKey string,
	initialConfig string,
	fileServer string,
	defaultNameServers []string,
	selfInfo InstanceInfo) (*EtcdDatasource, error) {

	ds := &EtcdDatasource{
		keysAPI:         kapi,
		client:          client,
		clusterName:     clusterName,
		leaseStart:      leaseStart,
		leaseRange:      leaseRange,
		workspacePath:   workspacePath,
		workspaceRepo:   workspaceRepo,
		workspaceBranch: workspaceBranch,
		initialConfig:   initialConfig,
		fileServer:      fileServer,
		dhcpAssignLock:  &sync.Mutex{},
		instanceEtcdKey: invalidEtcdKey,
		selfInfo:        selfInfo,
	}

	branch := fmt.Sprintf("refs/heads/%s", ds.workspaceBranch)
	dir := path.Join(ds.workspacePath, "repo")
	ds.SetBlacksmithVariable("private-key", privateKey)
	repo, err := clone(dir, ds.workspaceRepo, branch, ds.getPrivateKey())
	if err != nil {
		return nil, errors.Wrapf(err,
			"cloning %s branch %s to %s failed",
			ds.workspaceRepo, branch,
			dir,
		)
	}
	head, err := repo.Head()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get repo head for %s", ds.workspaceRepo)
	}

	key := path.Join(ds.ClusterName(), "blacksmith-instances", colonlessMacToMac(ds.selfInfo.Nic.String()), "workspace-commit-hash")
	if err := ds.set(key, head.Hash().String()); err != nil {
		return nil, errors.Wrapf(err, "failed to set ds key %q to %q", "workspace-commit-hash", head.Hash().String())
	}

	data, err := ioutil.ReadFile(ds.initialConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to read initial data: %s", err)
	}

	iVals := make(map[interface{}]interface{})
	err = yaml.Unmarshal(data, &iVals)
	if err != nil {
		return nil, fmt.Errorf("failed to read initial data: %s", err)
	}

	ds.copyToDatasource(iVals, ds.ClusterName())

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

	_, err = ds.GetEtcdMachine(selfInfo.Nic).Machine(true, selfInfo.IP)
	if err != nil {
		return nil, fmt.Errorf("failed to create the machine representation of self: %s", err)
	}

	err = ds.set(path.Join(ds.ClusterName(), "blacksmith-instances", colonlessMacToMac(selfInfo.Nic.String()), "ip"), selfInfo.IP.String())
	if err != nil {
		return nil, err
	}
	return ds, nil
}

func clone(path string, url string, ref string, privateKey []byte) (*git.Repository, error) {
	log.WithFields(log.Fields{
		"url":  url,
		"path": path,
		"ref":  ref,
	}).Info("cloning")

	opts := git.CloneOptions{
		URL:           url,
		ReferenceName: plumbing.ReferenceName(ref),
		SingleBranch:  true,
		Depth:         1,
		Progress:      os.Stdout,
	}

	if len(privateKey) != 0 {
		log.Println("Using private-key")
		signer, err := ssh.ParsePrivateKey(privateKey)
		if err != nil {
			return nil, errors.Wrap(err, "failed to parse private-key")
		}
		opts.Auth = &gitssh.PublicKeys{
			User:   "git",
			Signer: signer,
		}
	}

	os.RemoveAll(path)
	r, err := git.PlainClone(path, false, &opts)
	if err == transport.ErrInvalidAuthMethod && len(privateKey) != 0 {
		// Try again without the privateKey
		log.WithFields(log.Fields{"err": err}).Info("retrying clone without key")
		opts.Auth = nil
		os.RemoveAll(path)
		r, err = git.PlainClone(path, false, &opts)
	}

	if err != nil {
		return nil, err
	}

	return r, err
}

func checkoutRepo(r *git.Repository, path string) error {
	ref, err := r.Head()
	if err != nil {
		return err
	}

	commit, err := r.Commit(ref.Hash())
	if err != nil {
		return err
	}

	tree, err := commit.Tree()
	if err != nil {
		return err
	}

	tree.Files().ForEach(func(f *object.File) error {
		contents, err := fileContents(f)
		if err != nil {
			return err
		}

		name := filepath.Join(path, f.Name)
		err = os.MkdirAll(filepath.Dir(name), 0755)
		if err != nil {
			return err
		}

		err = ioutil.WriteFile(name, []byte(contents), f.Mode)
		if err != nil {
			return err
		}
		return nil
	})

	return nil
}

func fileContents(f *object.File) (content []byte, err error) {
	reader, err := f.Reader()
	if err != nil {
		return []byte(""), err
	}
	defer func(c io.Closer, err *error) {
		if cerr := reader.Close(); cerr != nil && *err == nil {
			*err = cerr
		}
	}(reader, &err)

	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(reader); err != nil {
		return []byte(""), err
	}

	return buf.Bytes(), nil
}
