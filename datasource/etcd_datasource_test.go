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
