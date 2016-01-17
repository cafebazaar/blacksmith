package hacluster // import "github.com/cafebazaar/blacksmith/hacluster"

import (
	"fmt"
	"time"

	"github.com/cafebazaar/blacksmith/logging"
	etcd "github.com/coreos/etcd/client"
	"golang.org/x/net/context"
)

const (
	ActiveMasterUpdateTime  = 10 * time.Second
	StandbyMasterUpdateTime = 15 * time.Second
	MasterTtlTime           = ActiveMasterUpdateTime * 3

	debugTag       = "HA"
	invalidEtcdKey = "INVALID"
	masterKey      = "instances"
	etcdTimeout    = 5 * time.Second
)

type MasterHaSpec struct {
	etcdDir string
	etcdKey string
	kapi    etcd.KeysAPI
}

func NewMasterHaSpec(kapi etcd.KeysAPI, etcdDir string) *MasterHaSpec {
	return &MasterHaSpec{
		etcdKey: invalidEtcdKey,
		etcdDir: fmt.Sprintf("%s/%s", etcdDir, masterKey),
		kapi:    kapi,
	}
}

func (b *MasterHaSpec) registerOnEtcd() error {
	ctx, cancel := context.WithTimeout(context.Background(), etcdTimeout)
	defer cancel()
	masterOrderOption := etcd.CreateInOrderOptions{
		TTL: MasterTtlTime,
	}
	resp, err := b.kapi.CreateInOrder(ctx, b.etcdDir, "", &masterOrderOption)
	if err != nil {
		return err
	}

	b.etcdKey = resp.Node.Key
	return nil
}

func (b *MasterHaSpec) updateOnEtcd() error {
	ctx, cancel := context.WithTimeout(context.Background(), etcdTimeout)
	defer cancel()
	masterSetOption := etcd.SetOptions{
		PrevExist: etcd.PrevExist,
		TTL:       MasterTtlTime,
	}
	_, err := b.kapi.Set(ctx, b.etcdKey, "", &masterSetOption)
	return err
}

func (b *MasterHaSpec) IsMaster() bool {
	var err error
	if b.etcdKey == invalidEtcdKey {
		err = b.registerOnEtcd()
		if err != nil {
			logging.Log(debugTag, "error while registerOnEtcd: %s", err)
			return false
		}
	} else {
		err = b.updateOnEtcd()
		if err != nil {
			b.etcdKey = invalidEtcdKey
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
	resp, err := b.kapi.Get(ctx, b.etcdDir, &masterGetOptions)
	if err != nil {
		logging.Log(debugTag, "error while getting the dir list from etcd: %s", err)
		return false
	}
	if len(resp.Node.Nodes) < 1 {
		logging.Log(debugTag, "empty list while getting the dir list from etcd")
		return false
	}
	if resp.Node.Nodes[0].Key == b.etcdKey {
		return true
	}
	return false
}
