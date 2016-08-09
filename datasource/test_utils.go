package datasource

import (
	"net"
	"os"
	"strings"
	"testing"
	"time"

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
func ForTest(t *testing.T) DataSource {
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
		t.Errorf("\nError while trying to get the interface (%s): %s\n", listenIFFlag, err)
	}

	serverIP := net.IPv4(127, 0, 0, 1)

	etcdClient, err := etcdClietForTest()

	if err != nil {
		t.Error("etcd instance not found")
	}

	kapi := etcd.NewKeysAPI(etcdClient)

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
		t.Errorf("\nCouldn't create runtime configuration: %s\n", err)
	}

	return etcdDataSource
}
