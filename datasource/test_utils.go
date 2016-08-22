package datasource

import (
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/context"

	etcd "github.com/coreos/etcd/client"
)

// ForTestParams is the way to create a customized DataSource to be
// used in a test. Fields with value=nil will be ignored.
type ForTestParams struct {
	leaseStart    *net.IP
	leaseRange    *int
	workspacePath *string
	listenIF      *string
	dnsIPStrings  *[]string
}

const (
	forTestDefaultLeaseStart    = "127.0.0.2"
	forTestDefaultLeaseRange    = 10
	forTestDefaultWorkspacePath = "/tmp/blacksmith/workspaces/test-workspace"
	forTestDefaultListenIF      = "lo"
	forTestDNSIPStrings         = "8.8.8.8"
)

var (
	forTestLock  = &sync.Mutex{}
	forTestIndex = 1
)

func etcdClietForTest() (etcd.Client, error) {
	etcdFlag := os.Getenv("ETCD_ENDPOINT")

	etcdClient, err := etcd.New(etcd.Config{
		Endpoints:               strings.Split(etcdFlag, ","),
		HeaderTimeoutPerRequest: 5 * time.Second,
	})

	return etcdClient, err
}

// ForTest constructs a DataSource to be used in tests
func ForTest(params *ForTestParams) (DataSource, error) {
	var err error

	leaseStart := net.ParseIP(forTestDefaultLeaseStart)
	leaseRange := forTestDefaultLeaseRange
	workspacePath := forTestDefaultWorkspacePath
	listenIF := "lo"
	dnsIPStrings := strings.Split(forTestDNSIPStrings, ",")

	if params != nil {
		if params.leaseStart != nil {
			leaseStart = *params.leaseStart
		}
		if params.leaseRange != nil {
			leaseRange = *params.leaseRange
		}
		if params.workspacePath != nil {
			workspacePath = *params.workspacePath
		}
		if params.listenIF != nil {
			listenIF = *params.listenIF
		}
		if params.dnsIPStrings != nil {
			dnsIPStrings = *params.dnsIPStrings
		}
	}

	// For tests to be safe for parallel execution
	forTestLock.Lock()
	clusterNameFlag := fmt.Sprintf("blacksmith-%04d", forTestIndex)
	forTestIndex++
	forTestLock.Unlock()

	var dhcpIF *net.Interface
	dhcpIF, err = net.InterfaceByName(listenIF)
	if err != nil {
		return nil,
			fmt.Errorf("error while trying to get the interface (%s): %s", listenIF, err)
	}

	serverIP := net.IPv4(127, 0, 0, 1)

	etcdClient, err := etcdClietForTest()

	if err != nil {
		return nil, fmt.Errorf("etcd instance not found: %s", err)
	}

	kapi := etcd.NewKeysAPI(etcdClient)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, err = kapi.Delete(ctx, clusterNameFlag,
		&etcd.DeleteOptions{Dir: true, Recursive: true})
	if err != nil && !etcd.IsKeyNotFound(err) {
		return nil, fmt.Errorf("error while purging previous data from etcd: %s", err)
	}

	selfInfo := InstanceInfo{
		IP:               serverIP,
		Nic:              dhcpIF.HardwareAddr,
		WebPort:          8000,
		Version:          "test",
		Commit:           "unknown",
		BuildTime:        "unknown",
		ServiceStartTime: time.Now().UTC().Unix(),
	}

	etcdDataSource, err := NewEtcdDataSource(
		kapi,
		etcdClient,
		leaseStart,
		leaseRange,
		clusterNameFlag,
		workspacePath,
		dnsIPStrings,
		selfInfo,
	)

	if err != nil {
		return nil, fmt.Errorf("couldn't create runtime configuration: %s", err)
	}

	return etcdDataSource, nil
}
