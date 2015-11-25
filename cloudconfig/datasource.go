package cloudconfig

import (
	"errors"
	"fmt"
	"github.com/coreos/etcd/Godeps/_workspace/src/golang.org/x/net/context"
	etcd "github.com/coreos/etcd/client"
	"reflect"
	"strings"
	"time"
)

var (
	ErrDuplicatedKey      = errors.New("key is duplicated")
	ErrDataSourceNotFound = errors.New("data source not found")
	ErrKeyNotFound        = errors.New("key not found")
)

type ConfigContext struct {
	IP       string
	MacAddr  string
	EtcdPath string
}

func (c *ConfigContext) Map() map[string]interface{} {
	// configContext to map
	confCtxMap := make(map[string]interface{})
	val := reflect.Indirect(reflect.ValueOf(c))
	typ := val.Type()
	for i := 0; ; i++ {
		field := typ.Field(i)
		if field.Name == "" {
			break
		}
		confCtxMap[field.Name] = val.Field(i).Interface()
	}
	return confCtxMap
}

type DataSource interface {
	Value(key string) (interface{}, error)
}

type EtcdDataSource struct {
	etcdKapi       etcd.KeysAPI
	etcdDefaultDir string
}

func NewEtcdDataSource(etcdKapi etcd.KeysAPI, defaultDir string) (*EtcdDataSource, error) {
	return &EtcdDataSource{
		etcdKapi:       etcdKapi,
		etcdDefaultDir: defaultDir,
	}, nil
}

func (e *EtcdDataSource) Value(key string) (interface{}, error) {
	key = strings.Replace(key, ".", "/", -1)
	key = strings.Replace(key, "__/", e.etcdDefaultDir+"/", -1)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	response, err := e.etcdKapi.Get(ctx, key, nil)
	if err != nil {
		return nil, err
	}
	return response.Node.Value, nil
}

func Value(sources map[string]DataSource, confCtx *ConfigContext, key string) (interface{}, error) {
	keys := strings.Split(key, ".")
	if len(keys) == 0 {
		return nil, ErrDataSourceNotFound
	}
	datasource, ok := sources[keys[0]]
	if !ok {
		return nil, ErrDataSourceNotFound
	}

	if strings.Contains(key, "$") {
		// substitute config context values in key
		confCtxMap := confCtx.Map()
		for i := range keys {
			if strings.HasPrefix(keys[i], "$") {
				v, ok := confCtxMap[strings.TrimPrefix(keys[i], "$")]
				if !ok {
					return nil, ErrKeyNotFound
				}
				keys[i] = fmt.Sprintf("%s", v)
			}
		}
	}
	val, err := datasource.Value(strings.Join(keys[1:], "."))
	return val, err
}
