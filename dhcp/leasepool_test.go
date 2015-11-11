package dhcp

import (
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/coreos/etcd/Godeps/_workspace/src/golang.org/x/net/context"
	etcd "github.com/coreos/etcd/client"
	"github.com/stretchr/testify/mock"
)

type myTestyDataSource struct {
	mock.Mock
}

func (d *myTestyDataSource) Get(ctx context.Context, key string, opts *etcd.GetOptions) (*etcd.Response, error) {
	args := d.Called(ctx, key, opts)
	return args.Get(0).(*etcd.Response), args.Error(1)
}
func (d *myTestyDataSource) Set(ctx context.Context, key, value string, opts *etcd.SetOptions) (*etcd.Response, error) {
	args := d.Called(ctx, key, opts)
	return nil, args.Error(1)
}
func (d *myTestyDataSource) Delete(ctx context.Context, key string, opts *etcd.DeleteOptions) (*etcd.Response, error) {
	args := d.Called(ctx, key, opts)
	return nil, args.Error(1)
}
func (d *myTestyDataSource) Create(ctx context.Context, key, value string) (*etcd.Response, error) {
	args := d.Called(ctx, key, value)
	return args.Get(0).(*etcd.Response), args.Error(1)
}
func (d *myTestyDataSource) CreateInOrder(ctx context.Context, dir, value string, opts *etcd.CreateInOrderOptions) (*etcd.Response, error) {
	args := d.Called(ctx, dir, value, opts)
	return args.Get(0).(*etcd.Response), args.Error(1)
}
func (d *myTestyDataSource) Update(ctx context.Context, key, value string) (*etcd.Response, error) {
	args := d.Called(ctx, key, value)
	return args.Get(0).(*etcd.Response), args.Error(1)
}
func (d *myTestyDataSource) Watcher(key string, opts *etcd.WatcherOptions) etcd.Watcher {
	args := d.Called(key, opts)
	return args.Get(0).(etcd.Watcher)
}

func TestLeasesMocked(t *testing.T) {
	testObj := &myTestyDataSource{}
	testObj.On("Delete", mock.Anything, "aghajoon-test/leases",
		&etcd.DeleteOptions{Dir: true, Recursive: true}).Return(nil, nil)

	r := etcd.Response{
		Action: "get",
		Node: &etcd.Node{
			Key:   "aghajoon-test/leases",
			Nodes: nil,
		},
	}
	testObj.On("Get", mock.Anything, "aghajoon-test/leases",
		&etcd.GetOptions{Recursive: true}).Return(&r, nil)
	testObj.On("Set", mock.Anything, "aghajoon-test/leases",
		&etcd.SetOptions{Dir: true}, "").Return(nil, nil)
	testObj.On("Set", mock.Anything,
		"aghajoon-test/leases/10.0.0.1",
		(*etcd.SetOptions)(nil)).Return(nil, nil)

	pool, err := NewLeasePool(testObj, "aghajoon-test", net.IP{10, 0, 0, 1}, 1, 100*time.Second)
	pool.Reset()
	if err != nil {
		t.Error(err)
	}

	nic := fmt.Sprintf("nic-%d", 0)
	ip, err := pool.Assign(nic)
	if err != nil {
		t.Error(err)
	}
	if ip.String() != "10.0.0.1" {
		t.Fatalf("Unexpected ip: %s", ip)
	}
}

func TestLeases(t *testing.T) {
	etcdClient, err := etcd.New(etcd.Config{
		Endpoints:               []string{"http://127.0.0.1:2379"},
		HeaderTimeoutPerRequest: time.Second})
	if err != nil {
		t.Skipf("skipping test because real etcd isn't available (%s).", err)
		return
	}
	kapi := etcd.NewKeysAPI(etcdClient)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, err = kapi.Delete(ctx, "aghajoon-test", &etcd.DeleteOptions{Dir: true, Recursive: true})
	if err != nil {
		etcdError, found := err.(etcd.Error)
		if !(found || etcdError.Code == etcd.ErrorCodeKeyNotFound) {
			t.Skipf("skipping test because real etcd isn't available (%s).", err)
			return
		}
	}

	pool, err := NewLeasePool(kapi, "aghajoon-test", net.IP{10, 0, 0, 1}, 254, 100*time.Second)
	pool.Reset()
	if err != nil {
		t.Error(err)
	}

	ips := make(map[string]bool, 254)
	for i := 0; i < 254; i++ {
		nic := fmt.Sprintf("nic-%d", i)
		ip, err := pool.Assign(nic)
		if err != nil {
			t.Error(err)
		}
		if _, found := ips[ip.String()]; found {
			// duplication
			t.Fail()
		}
		ips[ip.String()] = true
	}

	// check that every ip is allocated
	for i := 1; i < 255; i++ {
		ip := net.IP{10, 0, 0, byte(i)}
		if _, found := ips[ip.String()]; !found {
			t.Fail()
		}
	}
}

func TestSameLeaseForNic(t *testing.T) {
	etcdClient, err := etcd.New(etcd.Config{
		Endpoints:               []string{"http://127.0.0.1:2379"},
		HeaderTimeoutPerRequest: time.Second})
	if err != nil {
		t.Skipf("skipping test because real etcd isn't available (%s).", err)
		return
	}
	kapi := etcd.NewKeysAPI(etcdClient)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, err = kapi.Delete(ctx, "aghajoon-test", &etcd.DeleteOptions{Dir: true, Recursive: true})
	if err != nil {
		etcdError, found := err.(etcd.Error)
		if !(found || etcdError.Code == etcd.ErrorCodeKeyNotFound) {
			t.Skipf("skipping test because real etcd isn't available (%s).", err)
			return
		}
	}

	pool, err := NewLeasePool(kapi, "aghajoon-test", net.IP{10, 0, 0, 1}, 254, 100*time.Second)
	pool.Reset()
	if err != nil {
		t.Error(err)
	}

	ips := make(map[string]string, 254)
	for i := 0; i < 254; i++ {
		nic := fmt.Sprintf("nic-%d", i)
		ip, err := pool.Assign(nic)
		if err != nil {
			t.Error(err)
		}
		if _, found := ips[nic]; found {
			// duplication
			t.Fail()
		}
		ips[nic] = ip.String()
	}

	// check that every nic will allocated to same ip again
	for i := 0; i < 254; i++ {
		nic := fmt.Sprintf("nic-%d", i)
		ip, _ := pool.Assign(nic)
		if ips[nic] != ip.String() {
			t.Fail()
		}
	}
}
