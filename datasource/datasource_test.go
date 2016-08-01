package datasource

import (
	"net"
	"strings"
	"testing"
	"time"

	"github.com/cafebazaar/blacksmith/utils"
	etcd "github.com/coreos/etcd/client"
)

func getEtcdCliet() (etcd.Client, error) {
	etcdFlag := "http://127.0.0.1:2379"

	etcdClient, err := etcd.New(etcd.Config{
		Endpoints:               strings.Split(etcdFlag, ","),
		HeaderTimeoutPerRequest: 5 * time.Second,
	})

	return etcdClient, err
}

func getDatasource(t *testing.T) DataSource {
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

	serverIP, err := utils.InterfaceIP(dhcpIF)
	if err != nil {
		t.Errorf("\nError while trying to get the ip from the interface (%v)\n", dhcpIF)
	}

	etcdClient, err := getEtcdCliet()

	if err != nil {
		t.Error("etcd instance not found")
	}

	kapi := etcd.NewKeysAPI(etcdClient)

	selfInfo := InstanceInfo{
		IP:               serverIP,
		Nic:              dhcpIF.HardwareAddr,
		WebPort:          8000,
		Version:          "unknown",
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

func TestCoreOSVersion(t *testing.T) {
	etcdDataSource := getDatasource(t)

	version, err := etcdDataSource.CoreOSVersion()

	if err != nil {
		t.Errorf("error when try get coreos version, Error: %s", err)
	}

	if version != "1068.2.0" {
		t.Errorf("invalid coreos version")
	}

}
