package datasource

import (
	"net"
	"testing"
)

func TestAssign(t *testing.T) {
	ds, err := ForTest(nil)
	if err != nil {
		t.Error("error in getting a DataSource instance for our test:", err)
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

	mac1, _ := net.ParseMAC("FF:FF:FF:FF:FF:FF")
	mac2, _ := net.ParseMAC("FF:FF:FF:FF:FF:FE")

	machine1, err := ds.MachineInterface(mac1).Machine(true, nil)
	if err != nil {
		t.Error("error in creating first machine:", err)
		return
	}

	machine2, err := ds.MachineInterface(mac2).Machine(true, nil)
	if err != nil {
		t.Error("error in creating second machine:", err)
		return
	}

	if machine1.IP == nil {
		t.Error("unexpected nil value for machine1.IP")
		return
	}

	if machine1.Type != MTNormal {
		t.Error("unexpected type for machine1:", machine1.Type)
		return
	}

	if machine2.IP == nil {
		t.Error("unexpected nil value for machine2.IP")
		return
	}

	if machine1.IP.Equal(machine2.IP) {
		t.Error("two machines got same IP address:", machine1.IP.String())
		return
	}

	machine3, err := ds.MachineInterface(mac2).Machine(true, nil)
	if err != nil {
		t.Error("error in creating third machine:", err)
		return
	}

	if !machine2.IP.Equal(machine3.IP) {
		t.Error("same MAC address got two different IPs:", machine2.IP.String(), machine3.IP.String())
		return
	}

	anIP := net.IPv4(127, 0, 0, 10)
	machine4, err := ds.MachineInterface(mac2).Machine(true, anIP)
	if err != nil {
		t.Error("error in creating fourth machine:", err)
		return
	}

	if !machine2.IP.Equal(machine4.IP) {
		t.Error("same MAC address got two different IPs, createWithIP wasn't ignored?", machine2.IP.String(), machine3.IP.String())
		return
	}

	_, err = ds.MachineInterface(mac2).Machine(false, anIP)
	if err == nil {
		t.Error("expecting 'if createIfNeeded is false, the createWithIP is expected to be nil' error")
		return
	}

	mac3, _ := net.ParseMAC("FF:FF:FF:FF:FF:FD")
	_, err = ds.MachineInterface(mac3).Machine(true, machine1.IP)
	if err == nil {
		t.Error("expecting 'the requested IP is already assigned' error")
		return
	}
}

func TestLeaseRange(t *testing.T) {
	testLeaseRange := 10
	ds, err := ForTest(&ForTestParams{leaseRange: &testLeaseRange})
	if err != nil {
		t.Error("error in getting a DataSource instance for our test:", err)
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

	for i := 1; i <= testLeaseRange; i++ {
		mac := net.HardwareAddr{1, 1, 1, 1, 1, byte(i)}
		_, err := ds.MachineInterface(mac).Machine(true, nil)
		// TODO: check if the given IP is valid
		if err != nil {
			t.Error("error in creating machine:", err)
			return
		}
	}

	mac := net.HardwareAddr{1, 1, 1, 1, 1, 11}
	_, err = ds.MachineInterface(mac).Machine(true, nil)
	if err == nil {
		t.Error("expecting 'no unassigned IP was found' error")
		return
	}
}
