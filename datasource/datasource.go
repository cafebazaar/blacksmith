package datasource

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

func parseKey(sources map[string]DataSource, confCtx *ConfigContext, key string) (DataSource, string, error) {
	keys := strings.Split(key, ".")
	if len(keys) == 0 {
		return nil, "", ErrDataSourceNotFound
	}
	datasource, ok := sources[keys[0]]
	if !ok {
		return nil, "", ErrDataSourceNotFound
	}

	if strings.Contains(key, "$") {
		// substitute config context values in key
		confCtxMap := confCtx.Map()
		for i := range keys {
			if strings.HasPrefix(keys[i], "$") {
				v, ok := confCtxMap[strings.TrimPrefix(keys[i], "$")]
				if !ok {
					return nil, "", ErrKeyNotFound
				}
				keys[i] = fmt.Sprintf("%s", v)
			}
		}
	}
	return datasource, strings.Join(keys[1:], "."), nil
}

func GetValue(sources map[string]DataSource, confCtx *ConfigContext, key string) (string, error) {
	// handling config context values
	confMap := confCtx.Map()
	val, ok := confMap[key]
	if ok {
		return fmt.Sprintf("%s", val), nil
	}

	datasource, key, err := parseKey(sources, confCtx, key)
	if err != nil {
		return "", err
	}
	return datasource.GetValue(confCtx, key)
}

func Set(sources map[string]DataSource, confCtx *ConfigContext, key string, value string) (string, error) {
	datasource, key, err := parseKey(sources, confCtx, key)
	if err != nil {
		return "", err
	}
	return "", datasource.Set(confCtx, key, value)
}

func GetAndDelete(sources map[string]DataSource, confCtx *ConfigContext, key string) (string, error) {
	datasource, key, err := parseKey(sources, confCtx, key)
	if err != nil {
		return "", err
	}
	return datasource.GetAndDelete(confCtx, key)
}

func Delete(sources map[string]DataSource, confCtx *ConfigContext, key string) (string, error) {
	datasource, key, err := parseKey(sources, confCtx, key)
	if err != nil {
		return "", err
	}
	return "", datasource.Delete(confCtx, key)
}
