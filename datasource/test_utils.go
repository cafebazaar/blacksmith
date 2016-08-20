package datasource

import (
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"golang.org/x/net/context"

	etcd "github.com/coreos/etcd/client"
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

	// mocked data
	listenIFFlag := "lo"
	clusterNameFlag := "blacksmith"
	workspacePathFlag := "/tmp/blacksmith/workspaces/test-workspace"
	leaseStart := net.ParseIP("192.168.100.1")
	leaseRange := 10
	dnsIPStrings := []string{
		"8.8.8.8",
	}

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
	if err != nil {
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
		return nil, fmt.Errorf("Couldn't create runtime configuration: %s", err)
	}

	return etcdDataSource, nil
}
