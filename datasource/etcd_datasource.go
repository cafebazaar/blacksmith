package datasource

import(
	etcd "github.com/coreos/etcd/client"

)

//EtcdDataSource implements MasterDataSource interface using etcd as it's
//datasource
type EtcdDataSource struct {

}

func NewEtcdDataSource(
