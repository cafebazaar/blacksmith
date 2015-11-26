package cloudconfig

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
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
	GetValue(key string) (string, error)
}

func GetValue(sources map[string]DataSource, confCtx *ConfigContext, key string) (string, error) {
	keys := strings.Split(key, ".")
	if len(keys) == 0 {
		return "", ErrDataSourceNotFound
	}
	datasource, ok := sources[keys[0]]
	if !ok {
		return "", ErrDataSourceNotFound
	}

	if strings.Contains(key, "$") {
		// substitute config context values in key
		confCtxMap := confCtx.Map()
		for i := range keys {
			if strings.HasPrefix(keys[i], "$") {
				v, ok := confCtxMap[strings.TrimPrefix(keys[i], "$")]
				if !ok {
					return "", ErrKeyNotFound
				}
				keys[i] = fmt.Sprintf("%s", v)
			}
		}
	}
	val, err := datasource.GetValue(strings.Join(keys[1:], "."))
	return val, err
}
