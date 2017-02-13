package datasource

import "testing"

func TestInstances(t *testing.T) {
	ds, err := ForTest(nil)
	if err != nil {
		t.Error("error in getting a EtcdDatasource instance for our test:", err)
		return
	}

	if err := ds.WhileMaster(); err != nil {
		t.Error("failed to register as the master instance:", err)
	}
	defer func() {
		if err := ds.Shutdown(); err != nil {
			t.Error("failed to shutdown:", err)
		}
	}()

	instances, err := ds.Instances()
	if err != nil {
		t.Error("error in getting instances:", err)
		return
	}

	if len(instances) != 1 {
		t.Error("expecting instances to have exactly 1 member, instances=", instances)
		return
	}

	i0 := instances[0]
	self := ds.SelfInfo()

	if self.IP == nil {
		t.Error("self.IP shouln't be nil")
		return
	}

	if !self.IP.Equal(i0.IP) || i0.Commit != self.Commit || i0.ServiceStartTime != self.ServiceStartTime {
		t.Errorf("expecting i0 to be same as self, but i0=%v self=%v", i0, self)
		return
	}
}
