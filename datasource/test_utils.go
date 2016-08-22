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
func ForTest() (DataSource, error) {
	var err error

	listenIFFlag := "lo"
	workspacePathFlag := "/tmp/blacksmith/workspaces/test-workspace"
	leaseStart := net.ParseIP("192.168.100.1")
	leaseRange := 10
	dnsIPStrings := []string{
		"8.8.8.8",
	}

	// For tests to be safe for parallel execution
	forTestLock.Lock()
	clusterNameFlag := fmt.Sprintf("blacksmith-%04d", forTestIndex)
	forTestIndex++
	forTestLock.Unlock()

	var dhcpIF *net.Interface
	dhcpIF, err = net.InterfaceByName(listenIFFlag)
	if err != nil {
		return nil,
			fmt.Errorf("error while trying to get the interface (%s): %s", listenIFFlag, err)
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
		workspacePathFlag,
		dnsIPStrings,
		selfInfo,
	)

	if err != nil {
		return nil, fmt.Errorf("couldn't create runtime configuration: %s", err)
	}

	return etcdDataSource, nil
}
