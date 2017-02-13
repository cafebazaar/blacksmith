package datasource

import (
	"strings"
	"testing"
)

func TestCoreOSVersion(t *testing.T) {
	ds, err := ForTest(nil)
	if err != nil {
		t.Error("error in getting a EtcdDatasource instance for our test:", err)
		return
	}

	version, err := ds.GetClusterVariable("coreos-version")
	if err != nil {
		t.Error("error while getting coreos version:", err)
		return
	}

	if want := "1192.2.0"; version != want {
		t.Errorf("invalid coreos version; want %q, got %q", want, version)
	}
}

func TestEtcdMembers(t *testing.T) {
	ds, err := ForTest(nil)
	if err != nil {
		t.Error("error in getting a EtcdDatasource instance for our test:", err)
		return
	}

	got, err := ds.EtcdMembers()
	if err != nil {
		t.Error("error while EtcdMembers:", err)
	}

	// It's not easy to know the exact value in all the test environment
	if !(strings.Contains(got, "etcd0=") && strings.HasSuffix(got, "80")) {
		t.Error("expecting EtcdMembers result to contains etcd0= and ends with 80, got:", got)
	}
}
