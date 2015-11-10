package dhcp

import (
	"fmt"
	"net"
	"testing"
	"time"
)



func TestLeases(t *testing.T) {
	pool, err := NewLeasePool("http://127.0.0.1:2379", "aghajoon-test", net.IP{10, 0, 0, 1}, 254, 100*time.Second)
	pool.Reset()
	if err != nil {
		t.Error(err)
	}

	ips := make(map[string]bool, 254)
	for i := 0; i < 254; i++ {
		nic := fmt.Sprintf("nic-%d", i)
		ip, err := pool.Assign(nic)
		if err != nil {
			t.Error(err)
		}
		if _, found := ips[ip.String()]; found {
			// duplication
			t.Fail()
		}
		ips[ip.String()] = true
	}
	
	
	// check that every ip is allocated
	for i := 1; i < 255; i++ {
		ip := net.IP{10, 0, 0, byte(i)}
		if _, found := ips[ip.String()]; !found {
			t.Fail()
		}
	}
}

func TestSameLeaseForNic(t *testing.T) {
	pool, err := NewLeasePool("http://127.0.0.1:2379", "aghajoon-test", net.IP{10, 0, 0, 1}, 254, 100*time.Second)
	pool.Reset()
	if err != nil {
		t.Error(err)
	}

	ips := make(map[string]string, 254)
	for i := 0; i < 254; i++ {
		nic := fmt.Sprintf("nic-%d", i)
		ip, err := pool.Assign(nic)
		if err != nil {
			t.Error(err)
		}
		if _, found := ips[nic]; found {
			// duplication
			t.Fail()
		}
		ips[nic] = ip.String()
	}
	
	
	// check that every nic will allocated to same ip again
	for i := 0; i < 254; i++ {
		nic := fmt.Sprintf("nic-%d", i)
		ip, _ := pool.Assign(nic)
		if ips[nic] != ip.String() {
			t.Fail()
		}
	}
}
