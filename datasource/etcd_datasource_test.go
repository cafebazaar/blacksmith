package datasource

import (
	"strings"
	"testing"
)

func TestCoreOSVersion(t *testing.T) {
	ds, err := ForTest()
	if err != nil {
		t.Error("error in getting a DataSource instance for our test:", err)
		return
	}

	version, err := ds.GetClusterVariable("coreos-version")
	if err != nil {
		t.Error("error while getting coreos version:", err)
	}

	if version != "1068.2.0" {
		t.Error("invalid coreos version")
	}
}

func TestEtcdMembers(t *testing.T) {
	ds, err := ForTest()
	if err != nil {
		t.Error("error in getting a DataSource instance for our test:", err)
		return
	}

	got, err := ds.EtcdMembers()
	if err != nil {
		t.Error("error while EtcdMembers:", err)
	}

	// It's not easy to know the exact value in all the test environment
	if !(strings.Contains(got, "etcd0=") && strings.HasSuffix(got, "80")) {
		t.Error("expecting EtcdMembers result to conatins etcd0= and ends with 80, got:", got)
	}
}
