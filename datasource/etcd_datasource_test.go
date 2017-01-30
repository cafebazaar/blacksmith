package datasource

import (
	"log"
	"strings"
	"testing"
)

func TestCoreOSVersion(t *testing.T) {
	ds, err := ForTest(nil)
	if err != nil {
		t.Error("error in getting a DataSource instance for our test:", err)
		return
	}

	ds.SetClusterVariable("testing", "it works!")
	log.Println(ds.GetClusterVariable("testing"))
	log.Println(ds.GetClusterVariable("workspace-repo"))
	log.Println(ds.GetClusterVariable("coreos-version"))

	version, err := ds.GetClusterVariable("coreos-version")
	if err != nil {
		t.Error("error while getting coreos version:", err)
		return
	}

	if want := "1068.2.0"; version != want {
		t.Error("invalid coreos version; want %q, got %q", want, version)
	}
}

func TestEtcdMembers(t *testing.T) {
	ds, err := ForTest(nil)
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
		t.Error("expecting EtcdMembers result to contains etcd0= and ends with 80, got:", got)
	}
}
