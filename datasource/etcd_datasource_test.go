package datasource

import "testing"

func TestCoreOSVersion(t *testing.T) {
	ds := ForTest(t)

	version, err := ds.GetClusterVariable("coreos-version")
	if err != nil {
		t.Error("error while getting coreos version:", err)
	}

	if version != "1068.2.0" {
		t.Error("invalid coreos version")
	}
}

func TestEtcdMembers(t *testing.T) {
	ds := ForTest(t)

	got, err := ds.EtcdMembers()
	if err != nil {
		t.Error("error while EtcdMembers:", err)
	}

	expected := "etcd0=http://127.0.0.1:20380"
	if got != expected {
		t.Error("expecting %q, got %q", expected, got)
	}
}
