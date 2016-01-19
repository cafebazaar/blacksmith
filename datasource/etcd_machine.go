package datasource

import (
	"errors"
	"net"
	"strconv"
	"strings"
	"time"
)

// EtcdMachine implements datasource.Machine interface using etcd as it's
// datasource
type EtcdMachine struct {
	mac  net.HardwareAddr
	etcd GeneralDataSource
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
	tempName := "node" + m.Mac().String()
	return strings.Replace(tempName, ":", "", -1)
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
	return "machines/" + m.Mac().String() + "/" + str
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
