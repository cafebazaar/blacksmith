package datasource

import (
	"encoding/json"
	"fmt"
	"path"
	"time"

	log "github.com/Sirupsen/logrus"
	etcd "github.com/coreos/etcd/client"
	"golang.org/x/net/context"
)

const (
	ActiveMasterUpdateTime  = 10 * time.Second
	StandbyMasterUpdateTime = 15 * time.Second

	masterTTLTime = ActiveMasterUpdateTime * 3

	invalidEtcdKey   = "INVALID"
	instancesEtcdDir = "instances"
	etcdTimeout      = 10 * time.Second
)

func (ds *EtcdDatasource) registerOnEtcd() error {
	ctx, cancel := context.WithTimeout(context.Background(), etcdTimeout)
	defer cancel()
	masterOrderOption := etcd.CreateInOrderOptions{
		TTL: masterTTLTime,
	}
	resp, err := ds.keysAPI.CreateInOrder(ctx, path.Join(ds.ClusterName(), instancesEtcdDir),
		ds.selfInfo.String(), &masterOrderOption)
	if err != nil {
		return err
	}

	ds.instanceEtcdKey = resp.Node.Key
	return nil
}

func (ds *EtcdDatasource) etcdHeartbeat() error {
	ctx, cancel := context.WithTimeout(context.Background(), etcdTimeout)
	defer cancel()
	masterSetOption := etcd.SetOptions{
		PrevExist: etcd.PrevExist,
		TTL:       masterTTLTime,
	}
	_, err := ds.keysAPI.Set(ctx, ds.instanceEtcdKey, ds.selfInfo.String(), &masterSetOption)
	return err
}

// IsMaster checks for being master
func (ds *EtcdDatasource) IsMaster() error {
	ctx, cancel := context.WithTimeout(context.Background(), etcdTimeout)
	defer cancel()
	masterGetOptions := etcd.GetOptions{
		Recursive: true,
		Quorum:    true,
		Sort:      true,
	}
	resp, err := ds.keysAPI.Get(ctx, path.Join(ds.ClusterName(), instancesEtcdDir), &masterGetOptions)
	if err != nil {
		return fmt.Errorf("error while getting the dir list from etcd: %s", err)
	}
	if len(resp.Node.Nodes) < 1 {
		return fmt.Errorf("empty list while getting the dir list from etcd")
	}
	if resp.Node.Nodes[0].Key == ds.instanceEtcdKey {
		return nil
	}
	return fmt.Errorf("this is not the master instance")
}

// WhileMaster makes a heartbeat and returns IsMaster()
func (ds *EtcdDatasource) WhileMaster() error {
	var err error
	if ds.instanceEtcdKey == invalidEtcdKey {
		err = ds.registerOnEtcd()
		if err != nil {
			return fmt.Errorf("error while registerOnEtcd: %s", err)
		}
	} else {
		err = ds.etcdHeartbeat()
		if err != nil {
			ds.instanceEtcdKey = invalidEtcdKey
			return fmt.Errorf("error while updateOnEtcd: %s", err)
		}
	}

	return ds.IsMaster()
}

// Shutdown removes the instance key from the list of instances, used to
// gracefully shutdown the instance
func (ds *EtcdDatasource) Shutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), etcdTimeout)
	defer cancel()
	_, err := ds.keysAPI.Delete(ctx, ds.instanceEtcdKey, nil)
	return err
}

// Instances returns the InstanceInfo of all the present instances of
// blacksmith in our cluster
func (ds *EtcdDatasource) Instances() ([]InstanceInfo, error) {
	var instances []InstanceInfo

	// These values are set by hacluster.registerOnEtcd
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	response, err := ds.keysAPI.Get(ctx, path.Join(ds.ClusterName(), instancesEtcdDir), &etcd.GetOptions{Recursive: false})
	if err != nil {
		return nil, err
	}

	for _, ent := range response.Node.Nodes {
		instanceInfoStr := ent.Value

		var instanceInfo InstanceInfo
		if err := json.Unmarshal([]byte(instanceInfoStr), &instanceInfo); err != nil {
			return nil, fmt.Errorf("failed to unmarshal instance info: %s / instanceInfoStr=%q",
				err, instanceInfoStr)
		}

		instances = append(instances, instanceInfo)
	}

	return instances, nil
}

// SelfInfo return InstanceInfo of this instance of blacksmith
func (ds *EtcdDatasource) SelfInfo() InstanceInfo {
	return ds.selfInfo
}

func (ii *InstanceInfo) String() string {
	marshaled, err := json.Marshal(ii)
	if err != nil {
		log.WithField("where", "InstanceInfo.String").WithError(err).Warnf("failed to marshal instanceInfo")
		return ""
	}
	return string(marshaled)
}
