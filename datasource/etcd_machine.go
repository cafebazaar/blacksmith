package datasource

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"path"
	"strconv"
	"strings"
	"time"

	etcd "github.com/coreos/etcd/client"
	"github.com/krolaw/dhcp4"
	"golang.org/x/net/context"
)

// etcdMachineInterface implements datasource.MachineInterface
// interface using etcd as it's datasource
type etcdMachineInterface struct {
	mac     net.HardwareAddr
	etcdDS  *EtcdDataSource
	keysAPI etcd.KeysAPI
}

// Mac returns the hardware address of the associated machine
func (m *etcdMachineInterface) Mac() net.HardwareAddr {
	return m.mac
}

// Hostname returns the mac address formatted as a string suitable for hostname
func (m *etcdMachineInterface) Hostname() string {
	return strings.Replace(m.mac.String(), ":", "", -1)
}

// Machine returns the associated Machine
// If the createIfNeed is true, and there is no machine associated to this
// mac, the machine will be created and returned. If the assignedIP is empty,
// the IP will be assigned automatically, otherwise the given will be used.
// In this case, an error will be raised if the given IP is currently assigned
// to another mac.
// If createIfNeed == nil, the assignedIP will be ignored.
func (m *etcdMachineInterface) Machine(createIfNeed bool, assignedIP net.IP) (Machine, error) {
	var machine Machine

	resp, err := m.selfGet("_machine")
	if err != nil {
		errorIsKeyNotFound := etcd.IsKeyNotFound(err)

		if !(errorIsKeyNotFound && createIfNeed) {
			return machine, fmt.Errorf("error while retrieving _machine: %s", err)
		}

		machine := Machine{
			IP:        assignedIP, // to be assigned automatically
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

func (m *etcdMachineInterface) store(machine *Machine) error {
	if machine.Type == 0 {
		if machine.IP == nil {
			machine.Type = MTNormal
		} else {
			machine.Type = MTStatic
		}
	}

	if machine.IP == nil {
		// To avoid concurrency problems
		if err := m.etcdDS.IsMaster(); err != nil {
			return fmt.Errorf("only the master instance is allowed to store machine info: %s", err)
		}

		m.etcdDS.dhcpAssignLock.Lock()
		defer m.etcdDS.dhcpAssignLock.Unlock()

		machineInterfaces, err := m.etcdDS.MachineInterfaces()
		if err != nil {
			return fmt.Errorf("error while getting the machine interfaces: %s", err)
		}
		ipSet := make(map[string]bool)
		for _, mi := range machineInterfaces {
			machine, err := mi.Machine(false, nil)
			if err != nil {
				return fmt.Errorf("error while getting the machine for (%s): %s", mi.Mac().String(), err)
			}
			ipSet[machine.IP.String()] = true
		}
		counter := len(ipSet)
		candidateIP := dhcp4.IPAdd(m.etcdDS.leaseStart, counter)
		for _, isAssigned := ipSet[candidateIP.String()]; isAssigned; {
			candidateIP = dhcp4.IPAdd(candidateIP, 1)
			counter++
		}

		// or >= ? :D
		if counter > m.etcdDS.leaseRange {
			return fmt.Errorf("lease range is exceeded (%d > %d)", counter, m.etcdDS.leaseRange)
		}

		machine.IP = candidateIP
	}

	jsonedStats, err := json.Marshal(*machine)
	if err != nil {
		return fmt.Errorf("error while marshaling the machine: %s", err)
	}
	err = m.selfSet("_machine", string(jsonedStats))
	if err != nil {
		return fmt.Errorf("error while setting the marshaled machine: %s", err)
	}

	return nil
}

// CheckIn updates the _last_seen field of the machine
func (m *etcdMachineInterface) CheckIn() {
	m.selfSet("_last_seen", strconv.FormatInt(time.Now().Unix(), 10))
}

// LastSeen returns the last time the machine has been seen, 0 for never
func (m *etcdMachineInterface) LastSeen() (int64, error) {
	unixString, err := m.selfGet("_last_seen")
	if err != nil {
		return 0, err
	}
	unixInt64, _ := strconv.ParseInt(unixString, 10, 64)
	return unixInt64, nil
}

// ListFlags returns the list of all the flgas of a machine from Etcd
// etcd and machine prefix will be added to the path
func (m *etcdMachineInterface) ListVariables() (map[string]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	response, err := m.keysAPI.Get(ctx, path.Join(m.etcdDS.ClusterName(), "machines", m.Hostname()), nil)
	if err != nil {
		return nil, err
	}

	flags := make(map[string]string)
	for i := range response.Node.Nodes {
		_, k := path.Split(response.Node.Nodes[i].Key)
		flags[k] = response.Node.Nodes[i].Value
	}

	return flags, nil
}

// GetVariable Gets a machine's variable, or the global if it was not
// set for the machine
func (m *etcdMachineInterface) GetVariable(key string) (string, error) {
	value, err := m.selfGet(key)

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
func (m *etcdMachineInterface) SetVariable(key, value string) error {
	err := validateVariable(key, value)
	if err != nil {
		return err
	}
	return m.selfSet(key, value)
}

// DeleteVariable erases the entry specified by key
func (m *etcdMachineInterface) DeleteVariable(key string) error {
	return m.selfDelete(key)
}

func (m *etcdMachineInterface) prefixifyForMachine(key string) string {
	return path.Join(m.etcdDS.ClusterName(), etcdMachinesDirName, m.Hostname(), key)
}

func (m *etcdMachineInterface) selfGet(key string) (string, error) {
	return m.etcdDS.get(m.prefixifyForMachine(key))
}

func (m *etcdMachineInterface) selfSet(key, value string) error {
	return m.etcdDS.set(m.prefixifyForMachine(key), value)
}

func (m *etcdMachineInterface) selfDelete(key string) error {
	err := m.etcdDS.delete(m.prefixifyForMachine(key))
	return err
}

func macFromName(name string) (net.HardwareAddr, error) {
	name = strings.Split(name, ".")[0]
	return net.ParseMAC(colonLessMacToMac(name))
}

func colonLessMacToMac(colonLess string) string {
	coloned := colonLess
	if strings.Index(colonLess, ":") == -1 {
		var tmpmac bytes.Buffer
		for i := 0; i < 12; i++ { // colon-less mac address length
			tmpmac.WriteString(colonLess[i : i+1])
			if i%2 == 1 {
				tmpmac.WriteString(":")
			}
		}
		coloned = tmpmac.String()[:len(tmpmac.String())-1]
	}
	return coloned
}
