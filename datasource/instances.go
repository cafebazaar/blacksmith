package datasource

import (
	"encoding/json"
	"time"

	"github.com/cafebazaar/blacksmith/logging"
	etcd "github.com/coreos/etcd/client"
	"golang.org/x/net/context"
)

const (
	ActiveMasterUpdateTime  = 10 * time.Second
	StandbyMasterUpdateTime = 15 * time.Second
	MasterTtlTime           = ActiveMasterUpdateTime * 3

	debugTag         = "HA"
	invalidEtcdKey   = "INVALID"
	instancesEtcdDir = "instances"
	etcdTimeout      = 5 * time.Second
)

func (ds *EtcdDataSource) registerOnEtcd() error {
	ctx, cancel := context.WithTimeout(context.Background(), etcdTimeout)
	defer cancel()
	masterOrderOption := etcd.CreateInOrderOptions{
		TTL: MasterTtlTime,
	}
	resp, err := ds.keysAPI.CreateInOrder(ctx, ds.prefixify(instancesEtcdDir),
		ds.selfInfo.String(), &masterOrderOption)
	if err != nil {
		return err
	}

	ds.instanceEtcdKey = resp.Node.Key
	return nil
}

func (ds *EtcdDataSource) etcdHeartbeat() error {
	ctx, cancel := context.WithTimeout(context.Background(), etcdTimeout)
	defer cancel()
	masterSetOption := etcd.SetOptions{
		PrevExist: etcd.PrevExist,
		TTL:       MasterTtlTime,
	}
	_, err := ds.keysAPI.Set(ctx, ds.instanceEtcdKey, ds.selfInfo.String(), &masterSetOption)
	return err
}

// IsMaster checks for being master, and makes a heartbeat
func (ds *EtcdDataSource) IsMaster() bool {
	var err error
	if ds.instanceEtcdKey == invalidEtcdKey {
		err = ds.registerOnEtcd()
		if err != nil {
			logging.Log(debugTag, "error while registerOnEtcd: %s", err)
			return false
		}
	} else {
		err = ds.etcdHeartbeat()
		if err != nil {
			ds.instanceEtcdKey = invalidEtcdKey
			logging.Log(debugTag, "error while updateOnEtcd: %s", err)
			return false
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), etcdTimeout)
	defer cancel()
	masterGetOptions := etcd.GetOptions{
		Recursive: true,
		Quorum:    true,
		Sort:      true,
	}
	resp, err := ds.keysAPI.Get(ctx, ds.prefixify(instancesEtcdDir), &masterGetOptions)
	if err != nil {
		logging.Log(debugTag, "error while getting the dir list from etcd: %s", err)
		return false
	}
	if len(resp.Node.Nodes) < 1 {
		logging.Log(debugTag, "empty list while getting the dir list from etcd")
		return false
	}
	if resp.Node.Nodes[0].Key == ds.instanceEtcdKey {
		return true
	}
	return false
}

// RemoveInstance removes the instance key from the list of instances, used to
// gracefully shutdown the instance
func (ds *EtcdDataSource) RemoveInstance() error {
	ctx, cancel := context.WithTimeout(context.Background(), etcdTimeout)
	defer cancel()
	_, err := ds.keysAPI.Delete(ctx, ds.instanceEtcdKey, nil)
	return err
}

// DNSAddressesForDHCP returns the ip addresses of the present skydns servers
// in the network, marshalled as specified in rfc2132 (option 6)
func (ds *EtcdDataSource) DNSAddressesForDHCP() ([]byte, error) {
	var ret []byte

	// These values are set by hacluster.registerOnEtcd
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	response, err := ds.keysAPI.Get(ctx, ds.prefixify(instancesEtcdDir), &etcd.GetOptions{Recursive: false})
	if err != nil {
		return nil, err
	}

	for _, ent := range response.Node.Nodes {
		instanceInfoStr := ent.Value

		var instanceInfo InstanceInfo
		if err := json.Unmarshal([]byte(instanceInfoStr), &instanceInfo); err != nil {
			logging.Log(debugTag, "failed to unmarshal instance info: %s / instanceInfoStr=%q",
				err, instanceInfoStr)
			continue
		}

		ret = append(ret, (instanceInfo.IP.To4())...)
	}

	return ret, nil
}

// SelfInfo return InstanceInfo of this instance of blacksmith
func (ds *EtcdDataSource) SelfInfo() InstanceInfo {
	return ds.selfInfo
}

func (ii *InstanceInfo) String() string {
	marshaled, err := json.Marshal(ii)
	if err != nil {
		logging.Log(debugTag, "Failed to marshal instanceInfo: %s", err)
		return ""
	}
	return string(marshaled)

// Get all alive instances
func (ds *EtcdDataSource) GetAllInstances() ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), etcdTimeout)
	defer cancel()

	masterGetOptions := etcd.GetOptions{
		Recursive: true,
		Quorum:    true,
		Sort:      true,
	}
	resp, err := ds.keysAPI.Get(ctx, ds.prefixify(instancesEtcdDir), &masterGetOptions)
	if err != nil {
		logging.Log(debugTag, "error while getting the dir list from etcd: %s", err)
		return nil, err
	}

	var result = make([]string, len(resp.Node.Nodes))
	for i, node := range resp.Node.Nodes {
		result[i] = node.Value
	}
	return result, nil
}

// Get all other alive instances
func (ds *EtcdDataSource) GetAllOtherInstances() ([]string, error) {
	resp, err := ds.GetAllInstances()
	if err != nil {
		return nil, err
	}

	var result []string
	for _, node := range resp {
		if node != ds.serverIP.String() {
			result = append(result, node)
		}
	}
	return result, nil
}
