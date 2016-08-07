package datasource

import (
	"log"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/cafebazaar/blacksmith/logging"
	etcd "github.com/coreos/etcd/client"
)

func getEtcdCliet() (etcd.Client, error) {
	etcdFlag := os.Getenv("ETCD_ENDPOINT")

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

	serverIP := net.IPv4(127, 0, 0, 1)

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
	ds := getDatasource(t)

	version, err := ds.GetClusterVariable("coreos-version")

	if err != nil {
		t.Errorf("error when try get coreos version, Error: %s", err)
	}

	if version != "1068.2.0" {
		t.Errorf("invalid coreos version")
	}

}

func TestMachine(t *testing.T) {
	ds := getDatasource(t)
	if !ds.WhileMaster() {
		t.Error("failed to register as the master instance")
	}

	mac1, _ := net.ParseMAC("FF:FF:FF:FF:FF:FF")
	mac2, _ := net.ParseMAC("FF:FF:FF:FF:FF:FE")

	machine1, err := ds.MachineInterface(mac1).Machine(true, nil)
	if err != nil {
		t.Errorf("error in creating first machine, Error: %s", err)
	}

	machine2, err := ds.MachineInterface(mac2).Machine(true, nil)
	if err != nil {
		t.Errorf("error in creating second machine, Error: %s", err)
	}

	if machine1.IP == nil {
		t.Errorf("unexpected nil value for machine1.IP")
	}

	if machine2.IP == nil {
		t.Errorf("unexpected nil value for machine2.IP")
	}

	if machine1.IP.Equal(machine2.IP) {
		t.Errorf("two machines got same IP address! IP:%s", machine1.IP.String())
	}

	machine3, err := ds.MachineInterface(mac2).Machine(true, nil)
	if err != nil {
		t.Errorf("error in creating third machine, Error: %s", err)
	}

	if !machine2.IP.Equal(machine3.IP) {
		t.Errorf("same MAC address got two IPs: %s, %s", machine2.IP.String(), machine3.IP.String())
	}

	if err := ds.Shutdown(); err != nil {
		t.Errorf("failed to shutdown: %s", err)
	}
}

func TestMain(m *testing.M) {
	go func() {
		logging.RecordLogs(log.New(os.Stderr, "", log.LstdFlags), false)
	}()

	testRes := m.Run()
	os.Exit(testRes)
}
