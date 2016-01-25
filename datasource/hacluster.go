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
