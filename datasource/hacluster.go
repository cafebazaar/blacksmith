package datasource

import (
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
	resp, err := ds.keysAPI.CreateInOrder(ctx, ds.prefixify(instancesEtcdDir), ds.serverIP.String(), &masterOrderOption)
	if err != nil {
		return err
	}

	ds.instancesEtcdDir = resp.Node.Key
	return nil
}

func (ds *EtcdDataSource) etcdHeartbeat() error {
	ctx, cancel := context.WithTimeout(context.Background(), etcdTimeout)
	defer cancel()
	masterSetOption := etcd.SetOptions{
		PrevExist: etcd.PrevExist,
		TTL:       MasterTtlTime,
	}
	_, err := ds.keysAPI.Set(ctx, ds.instancesEtcdDir, ds.serverIP.String(), &masterSetOption)
	return err
}

// IsMaster checks for being master, and makes a heartbeat
func (ds *EtcdDataSource) IsMaster() bool {
	var err error
	if ds.instancesEtcdDir == invalidEtcdKey {
		err = ds.registerOnEtcd()
		if err != nil {
			logging.Log(debugTag, "error while registerOnEtcd: %s", err)
			return false
		}
	} else {
		err = ds.etcdHeartbeat()
		if err != nil {
			ds.instancesEtcdDir = invalidEtcdKey
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
	if resp.Node.Nodes[0].Key == ds.instancesEtcdDir {
		return true
	}
	return false
}

// RemoveInstance removes the instance key from the list of instances, used to
// gracefully shutdown the instance
func (ds *EtcdDataSource) RemoveInstance() error {
	ctx, cancel := context.WithTimeout(context.Background(), etcdTimeout)
	defer cancel()
	_, err := ds.keysAPI.Delete(ctx, ds.instancesEtcdDir, nil)
	return err
}


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
