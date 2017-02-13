package datasource

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"path"
	"strconv"
	"strings"
	"time"

	etcd "github.com/coreos/etcd/client"
	"github.com/krolaw/dhcp4"
)

// EtcdMachine provides the interface for querying/altering
// Machine entries in the datasource
type EtcdMachine struct {
	mac    net.HardwareAddr
	etcdDS *EtcdDatasource
}

// Mac returns the hardware address of the associated machine
func (m *EtcdMachine) Mac() net.HardwareAddr {
	return m.mac
}

// Hostname returns the mac address formatted as a string suitable for hostname
func (m *EtcdMachine) Hostname() string {
	return strings.Replace(m.mac.String(), ":", "", -1)
}

// Machine creates a record for the associated mac if needed
// and asked for, and returns a Machine with the stored values.
// If createIfNeeded is true, and there is no machine associated to
// this mac, the machine will be created, stored, and returned.
// In this case, if createWithIP is empty, the IP will be assigned
// automatically, otherwise the given will be used. An error will be
// raised if createWithIP is currently assigned to another mac. Also
// the Type will be automatically set to MTNormal if createWithIP is
// nil, otherwise to MTStatic.
// If createIfNeeded is false, the createWithIP is expected to be nil.
// Note: if the machine exists, createWithIP is ignored. It's possible
// for the returned Machine to have an IP different from createWithIP.
func (m *EtcdMachine) Machine(createIfNeeded bool,
	createWithIP net.IP) (Machine, error) {
	var machine Machine

	if !createIfNeeded && (createWithIP != nil) {
		return machine, errors.New(
			"if createIfNeeded is false, the createWithIP is expected to be nil")
	}

	resp, err := m.etcdDS.get(m.prefixifyForMachine("_machine"))
	if err != nil {
		if !(etcd.IsKeyNotFound(err) && createIfNeeded) {
			return machine, fmt.Errorf("error while retrieving _machine: %s", err)
		}

		machine := Machine{
			IP:        createWithIP, // to be assigned automatically
			FirstSeen: time.Now().Unix(),
		}
		err := m.store(&machine)
		if err != nil {
			return machine, fmt.Errorf("error while storing _machine: %s", err)
		}
		return machine, nil
	}
	json.Unmarshal([]byte(resp), &machine)
	return machine, nil
}

func (m *EtcdMachine) store(machine *Machine) error {
	if machine.Type == 0 {
		if machine.IP == nil {
			machine.Type = MTNormal
		} else {
			machine.Type = MTStatic
		}
	}

	m.etcdDS.dhcpAssignLock.Lock()
	defer m.etcdDS.dhcpAssignLock.Unlock()

	machineInterfaces, err := m.etcdDS.GetEtcdMachines()
	if err != nil {
		return fmt.Errorf("error while getting the machine interfaces: %s", err)
	}
	ipToMac := make(map[string]net.HardwareAddr)
	for _, mi := range machineInterfaces {
		machine, err := mi.Machine(false, nil)
		if err != nil {
			return fmt.Errorf("error while getting the machine for (%s): %s",
				mi.Mac().String(), err)
		}
		ipToMac[machine.IP.String()] = mi.Mac()
	}

	if machine.IP == nil {
		// To avoid concurrency problems
		// We expect rhis part to be triggered only through DHCP, so we expect
		// IsMaster() to returns true
		if err := m.etcdDS.IsMaster(); err != nil {
			return fmt.Errorf(
				"only the master instance is allowed to store machine info: %s",
				err)
		}

		counter := len(ipToMac) % m.etcdDS.leaseRange
		firstCandidateIP := dhcp4.IPAdd(m.etcdDS.leaseStart, counter) // kickstarted
		candidateIP := net.IPv4(
			firstCandidateIP[0], firstCandidateIP[1],
			firstCandidateIP[2], firstCandidateIP[3]) // copy

		for _, isAssigned := ipToMac[candidateIP.String()]; isAssigned; {
			candidateIP = dhcp4.IPAdd(candidateIP, 1)
			counter++
			if counter == m.etcdDS.leaseRange {
				candidateIP = m.etcdDS.leaseStart
				counter = 0
			}
			if firstCandidateIP.Equal(candidateIP) {
				break
			}
		}

		if _, isAssigned := ipToMac[candidateIP.String()]; isAssigned {
			return fmt.Errorf("no unassigned IP was found")
		}

		machine.IP = candidateIP
	} else {
		if m, isAssigned := ipToMac[machine.IP.String()]; isAssigned {
			return fmt.Errorf(
				"the requested IP(%s) is already assigned to another machine(%s)",
				machine.IP.String(), m.String())
		}
	}

	jsonedStats, err := json.Marshal(*machine)
	if err != nil {
		return fmt.Errorf("error while marshaling the machine: %s", err)
	}
	err = m.etcdDS.set(m.prefixifyForMachine("_machine"), string(jsonedStats))
	if err != nil {
		return fmt.Errorf("error while setting the marshaled machine: %s", err)
	}

	return nil
}

// CheckIn updates the _last_seen field of the machine
func (m *EtcdMachine) CheckIn() {
	m.etcdDS.set(m.prefixifyForMachine("_last_seen"), strconv.FormatInt(time.Now().Unix(), 10))
}

// LastSeen returns the last time the machine has been seen, 0 for never
func (m *EtcdMachine) LastSeen() (int64, error) {
	unixString, err := m.etcdDS.get(m.prefixifyForMachine("_last_seen"))
	if err != nil {
		return 0, err
	}
	unixInt64, _ := strconv.ParseInt(unixString, 10, 64)
	return unixInt64, nil
}

// DeleteMachine deletes associated etcd folder of a machine entirely
func (m *EtcdMachine) DeleteMachine() error {
	return m.etcdDS.delete(path.Join(m.etcdDS.ClusterName(), etcdMachinesDirName, m.Hostname()),
		&etcd.DeleteOptions{Dir: true, Recursive: true})
}

// ListFlags returns the list of all the flgas of a machine from Etcd
// etcd and machine prefix will be added to the path
func (m *EtcdMachine) ListVariables() (map[string]string, error) {
	nodes, err := m.etcdDS.GetArrayVariable(path.Join("machines", m.Hostname()))
	if err != nil {
		return nil, err
	}

	flags := make(map[string]string)
	for i := range nodes {
		_, k := path.Split(nodes[i].Key)
		flags[k] = nodes[i].Value
	}

	return flags, nil
}

// GetVariable Gets a machine's variable, or the global if it was not
// set for the machine
func (m *EtcdMachine) GetVariable(key string) (string, error) {
	value, err := m.etcdDS.get(m.prefixifyForMachine(key))

	if err != nil {
		if !etcd.IsKeyNotFound(err) {
			return "", fmt.Errorf(
				"error while getting variable key=%s for machine=%s: %s",
				key, m.mac, err)
		}

		// Key was not found for the machine
		value, err := m.etcdDS.GetClusterVariable(key)
		if err != nil {
			if !etcd.IsKeyNotFound(err) {
				return "", fmt.Errorf(
					"error while getting variable key=%s for machine=%s (global check): %s",
					key, m.mac, err)

			}
			return "", nil // Not set, not for machine, nor globally
		}
		return value, nil
	}

	return value, nil
}

// SetVariable sets the value of the specified key
func (m *EtcdMachine) SetVariable(key, value string) error {
	// err := validateVariable(key, value)
	// if err != nil {
	// 	return err
	// }
	return m.etcdDS.set(m.prefixifyForMachine(key), value)
}

// DeleteVariable erases the entry specified by key
func (m *EtcdMachine) DeleteVariable(key string) error {
	return m.etcdDS.delete(m.prefixifyForMachine(key), nil)
}

func (m *EtcdMachine) prefixifyForMachine(key string) string {
	return path.Join(m.etcdDS.ClusterName(), etcdMachinesDirName, m.Hostname(),
		key)
}

func macFromName(name string) (net.HardwareAddr, error) {
	name = strings.Split(name, ".")[0]
	return net.ParseMAC(colonlessMacToMac(name))
}

func colonlessMacToMac(colonless string) string {
	if strings.Index(colonless, ":") != -1 {
		return colonless
	}

	var buf bytes.Buffer
	for i, c := range colonless {
		if i != 0 && i%2 == 0 {
			buf.WriteString(":")
		}
		buf.WriteString(string(c))
	}
	return buf.String()
}
