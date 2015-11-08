package dhcp

import (
	"encoding/binary"
	"encoding/gob"
	"errors"
	"github.com/krolaw/dhcp4"
	"net"
	"os"
	"time"
)

var (
	ErrLeasePoolIsFull = errors.New("there is no empty IP address at the moment")
	ErrRefreshNoMatch  = errors.New("there is no match between specified ip and nic")
)

type lease struct {
	Nic string
	IP  net.IP

	LastAssigned time.Time
	ExpireTime   time.Time
}

func newLease(nic string, ip net.IP, expireDuration time.Duration) lease {
	now := time.Now()
	return lease{
		Nic:          nic,
		IP:           ip,
		LastAssigned: now,
		ExpireTime:   now.Add(expireDuration),
	}
}

type LeasePool struct {
	dataFile       string
	startIP        net.IP
	rangeLen       int
	leases         map[uint32]lease
	expireDuration time.Duration
}

func NewLeasePool(dataFile string, startIP net.IP, rangeLen int, expireDuration time.Duration) *LeasePool {
	pool := &LeasePool{
		dataFile:       dataFile,
		startIP:        startIP,
		expireDuration: expireDuration,
		rangeLen:       rangeLen,
		leases:         make(map[uint32]lease, 10),
	}
	pool.reloadFromFile()
	return pool
}

func (p *LeasePool) reloadFromFile() {
	if _, err := os.Stat(p.dataFile); os.IsNotExist(err) {
		return
	}
	f, err := os.Open(p.dataFile)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	decoder := gob.NewDecoder(f)
	if err = decoder.Decode(&p.leases); err != nil {
		panic(err)
	}
	
	// TODO check for IPs stored in p.lease that rest outside of current lease range
}

func (p *LeasePool) dumpToFile() {
	f, err := os.Create(p.dataFile)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	encoder := gob.NewEncoder(f)
	if err = encoder.Encode(p.leases); err != nil {
		panic(err)
	}
}

func (p *LeasePool) Assign(nic string) (net.IP, error) {
	// try to find by mac address
	for leaseIP, lease := range p.leases {
		if lease.Nic == nic {
			p.leases[leaseIP] = newLease(nic, lease.IP, p.expireDuration)
			p.dumpToFile()
			return lease.IP, nil
		}
	}
	// find an unseen ip
	for i := 0; i < p.rangeLen; i++ {
		ip := dhcp4.IPAdd(p.startIP, i)
		_, exists := p.leases[binary.BigEndian.Uint32(ip)]
		if !exists {
			p.leases[binary.BigEndian.Uint32(ip)] = newLease(nic, ip, p.expireDuration)
			p.dumpToFile()
			return ip, nil
		}
	}
	// find an expired ip
	now := time.Now()
	for leaseIP, lease := range p.leases {
		if lease.ExpireTime.Before(now) {
			p.leases[leaseIP] = newLease(nic, lease.IP, p.expireDuration)
			p.dumpToFile()
			return lease.IP, nil
		}
	}
	return nil, ErrLeasePoolIsFull
}

func (p *LeasePool) Refresh(nic string, currentIP net.IP) (net.IP, error) {
	now := time.Now()
	lease, exists := p.leases[binary.BigEndian.Uint32(currentIP)]
	if exists && lease.Nic == nic {
		p.leases[binary.BigEndian.Uint32(lease.IP)] = newLease(nic, lease.IP, p.expireDuration)
		p.dumpToFile()
		return lease.IP, nil
	} else if exists && lease.ExpireTime.After(now) {
		return nil, ErrRefreshNoMatch
	} else if !exists {
		// try to find by mac address
		for _, lease := range p.leases {
			// there exists an ip for this mac
			if lease.Nic == nic && lease.ExpireTime.After(now) {
				return nil, ErrRefreshNoMatch
			}
		}
		// assign that ip to this nic
		p.leases[binary.BigEndian.Uint32(currentIP)] = newLease(nic, currentIP, p.expireDuration)
		p.dumpToFile()
		return currentIP, nil
	}
	return nil, ErrRefreshNoMatch
}
