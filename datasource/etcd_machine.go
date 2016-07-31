package datasource

import (
	"bytes"
	"errors"
	"net"
	"path"
	"strings"

	etcd "github.com/coreos/etcd/client"
	"golang.org/x/net/context"
	"github.com/cafebazaar/blacksmith/logging"
	"encoding/json"
	"time"
	"strconv"
)

// EtcdMachine implements datasource.Machine interface using etcd as it's
// datasource
type EtcdMachine struct {
	mac     net.HardwareAddr
	etcd    *EtcdDataSource
	keysAPI etcd.KeysAPI
}

// stats like IP, first seen and IPMI node being stored inside machine directory
type EtcdMachineStats struct {
	IP		net.IP        `json:"ip"`
	Mac		string        `json:"mac"`
	FirstSeen	int64         `json:"first_seen"`
	IPMInode	string        `json:"IPMInode,omitempty"`
}

// Mac Returns this machine's hardware address
func (m *EtcdMachine) Mac() net.HardwareAddr {
	return m.mac
}

// Name returns this machine's hostname
func (m *EtcdMachine) Name() string {
	return nameFromMac(m.Mac().String())
}

// Domain returns this machine's domain which is equal to the cluster name
func (m *EtcdMachine) Domain() string {
	return m.etcd.ClusterName()
}

func timeError(err error) (int64, error) {
	return time.Now().Unix(), err
}

// CheckIn updates the _last_seen entry of this machine in etcd
func (m *EtcdMachine) CheckIn() {
	m.selfSet("_last_seen", strconv.FormatInt(time.Now().Unix(), 10))
}

// Get stats of machine like ip, mac, first seen and IPMI and returns it as EtcdMachineStats instance
func (m *EtcdMachine) GetStats() (EtcdMachineStats, error) {
	resp, err := m.selfGet("_stats")
	if err != nil {
		logging.Debug(debugTag, "couldn't retrive _stats from %s due to: %s", m.Name(), err)
		return EtcdMachineStats{}, err
	}
	etcdMachineStats := &EtcdMachineStats{}
	json.Unmarshal([]byte(resp), etcdMachineStats)
	return *etcdMachineStats, nil
}

func (m *EtcdMachine) SetStats(stats EtcdMachineStats) error {
        jsonedStats, err := json.Marshal(stats)
        if err != nil {
                logging.Debug(debugTag, "err marshal: %s", err)
        }
        err = m.etcd.Set(m.prefixify("_stats"), string(jsonedStats))
        if err != nil {
                logging.Debug(debugTag, "couldn't set stats due to: %s", err)
                return err
        }
        return nil
}
func (m *EtcdMachine) SetIPMI(mac net.HardwareAddr) {
        stats, err := m.GetStats()
        if err != nil {
                logging.Debug(debugTag, "couldn't get stats due to: %s", err)
        }
        stats.IPMInode = mac.String()
        m.SetStats(stats)
}

// LastSeen returns the last time the machine has  been ???
// part of Machine interface implementation
func (m *EtcdMachine) LastSeen() (int64, error) {
	unixString, err := m.selfGet("_last_seen")
	if err != nil {
		return timeError(err)
	}
	unixInt64, _ := strconv.ParseInt(unixString, 10, 64)
	return unixInt64, nil
}

// ListFlags returns the list of all the flgas of a machine from Etcd
// etcd and machine prefix will be added to the path
func (m *EtcdMachine) ListFlags() (map[string]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	response, err := m.keysAPI.Get(ctx, path.Join(m.etcd.ClusterName(), "machines", m.Name()), nil)
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

// GetFlag Gets a machine's flag from Etcd
// etcd and machine prefix will be added to the path
func (m *EtcdMachine) GetFlag(key string) (string, error) {
	return m.selfGet(key)
}

// SetFlag Sets a machin'es flag in Etcd
// etcd and machine prefix will be added to the PathPrefix
func (m *EtcdMachine) SetFlag(key, value string) error {
	if len(key) > 0 && key[0] == '_' {
		return errors.New("NotPermitted")
	}
	return m.selfSet(key, value)
}

// DeleteFlag deletes the record associated with key from Etcd
func (m *EtcdMachine) DeleteFlag(key string) error {
	return m.selfDelete(key)
}

func (m *EtcdMachine) prefixify(str string) string {
	return "machines/" + m.Name() + "/" + str
}

func (m *EtcdMachine) selfGet(key string) (string, error) {
	return m.etcd.get(m.prefixify(key))
}

func (m *EtcdMachine) selfSet(key, value string) error {
	return m.etcd.set(m.prefixify(key), value)
}

func (m *EtcdMachine) selfDelete(key string) error {
	_, err:= m.etcd.Delete(m.prefixify(key))
	return err
}

func nameFromMac(mac string) string {
	return strings.Replace("node"+mac, ":", "", -1)
}

func macFromName(name string) string {
	name = strings.Split(name, ".")[0]
	return colonLessMacToMac(name[len("node"):])
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
