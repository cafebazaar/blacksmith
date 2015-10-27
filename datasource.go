package main

import (
	"time"

	"github.com/coreos/etcd/client"
	"golang.org/x/net/context"
)

type dataSource struct {
	aPI client.KeysAPI
}

func (d *dataSource) GetCoreOSVersion() (string, error) {
	resp, err := d.aPI.Get(context.Background(), "/coreos-version", nil)
	if err != nil {
		return "", err
	}
	return resp.Node.Value, nil
}

// DataSource ask etcd for data
func DataSource(endpoints []string) (*dataSource, error) {
	cfg := client.Config{
		Endpoints:               endpoints,
		Transport:               client.DefaultTransport,
		HeaderTimeoutPerRequest: time.Second,
	}
	c, err := client.New(cfg)
	if err != nil {
		return nil, err
	}
	return &dataSource{client.NewKeysAPIWithPrefix(c, "aghajoon")}, nil
}
