package datasource

import (
	"bytes"
	"errors"
	"net"
	"path"
	"strconv"
	"strings"
	"time"

	etcd "github.com/coreos/etcd/client"
	"golang.org/x/net/context"
)

// EtcdMachine implements datasource.Machine interface using etcd as it's
// datasource
type EtcdMachine struct {
	mac     net.HardwareAddr
	etcd    GeneralDataSource
	keysAPI etcd.KeysAPI
}

// Mac Returns this machine's hardware address
// part of Machine interface implementation
func (m *EtcdMachine) Mac() net.HardwareAddr {
	return m.mac
}

// IP Returns this machine's IP
// queries etcd
// part of Machine interface implementation
func (m *EtcdMachine) IP() (net.IP, error) {
	ipstring, err := m.selfGet("_IP")
	if err != nil {
		return nil, err
	}
	IP := net.ParseIP(ipstring)
	return IP, nil
}

// Name returns this machine's hostname
func (m *EtcdMachine) Name() string {
	return nameFromMac(m.Mac().String())
}

func unixNanoStringToTime(unixNano string) (time.Time, error) {
	unixNanoi64, err := strconv.ParseInt(unixNano, 10, 64)
	if err != nil {
		return time.Now(), err
	}
	return time.Unix(0, unixNanoi64), nil

}

func timeError(err error) (time.Time, error) {
	return time.Now(), err
}

// CheckIn updates the _last_seen entry of this machine in etcd
// part of EtcdMachine interface implementation
func (m *EtcdMachine) CheckIn() {
	m.selfSet("_last_seen", strconv.FormatInt(time.Now().UnixNano(), 10))
}

// FirstSeen returns the time upon which that the machine has been first seen
// queries etcd
// part of Machine interface implementaiton
func (m *EtcdMachine) FirstSeen() (time.Time, error) {
	unixNanoString, err := m.selfGet("_first_seen")
	if err != nil {
		return timeError(err)
	}
	return unixNanoStringToTime(unixNanoString)
}

// LastSeen returns the last time the machine has  been ???
// part of Machine interface implementation
func (m *EtcdMachine) LastSeen() (time.Time, error) {
	unixNanoString, err := m.selfGet("_last_seen")
	if err != nil {
		return timeError(err)
	}
	return unixNanoStringToTime(unixNanoString)
}

// ListFlags returns the list of all the flgas of a machine from Etcd
// etcd and machine prefix will be added to the path
// part of Machine interface implementation
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
// part of Machine interface implementation
func (m *EtcdMachine) GetFlag(key string) (string, error) {
	return m.selfGet(key)
}

// SetFlag Sets a machin'es flag in Etcd
// etcd and machine prefix will be added to the PathPrefix
// part of Machine interface implementation
func (m *EtcdMachine) SetFlag(key, value string) error {
	if len(key) > 0 && key[0] == '_' {
		return errors.New("NotPermitted")
	}
	return m.selfSet(key, value)
}

// GetAndDeleteFlag doesn't do an awful lot of magic.
// just combines GetFlag and DeleteFlag operations
// part of Machine interface implementation
func (m *EtcdMachine) GetAndDeleteFlag(key string) (string, error) {
	val, err := m.GetFlag(key)
	if err != nil {
		return "", err
	}
	err = m.DeleteFlag(key)
	return val, err
}

// DeleteFlag deletes the record associated with key from Etcd
// part of Machine interface implementation
func (m *EtcdMachine) DeleteFlag(key string) error {
	return m.selfDelete(key)
}

func (m *EtcdMachine) prefixify(str string) string {
	return "machines/" + m.Name() + "/" + str
}

func (m *EtcdMachine) selfGet(key string) (string, error) {
	return m.etcd.Get(m.prefixify(key))
}

func (m *EtcdMachine) selfSet(key, value string) error {
	return m.etcd.Set(m.prefixify(key), value)
}

func (m *EtcdMachine) selfDelete(key string) error {
	return m.etcd.Delete(m.prefixify(key))
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
