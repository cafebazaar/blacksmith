package datasource

//
// import (
// 	"errors"
// 	"strings"
// 	"time"
//
// 	"github.com/cafebazaar/blacksmith/cloudconfig"
// 	"github.com/coreos/etcd/Godeps/_workspace/src/golang.org/x/net/context"
// 	etcd "github.com/coreos/etcd/client"
// )
//
// var BadKeyError error = errors.New("bad key given")
//
// type Flags struct {
// 	kapi etcd.KeysAPI
// 	dir  string
// }
//
// func NewFlags(kapi etcd.KeysAPI, dir string) (*Flags, error) {
// 	return &Flags{
// 		kapi: kapi,
// 		dir:  dir,
// 	}, nil
// }
//
// func (f *Flags) parseKey(confCtx *cloudconfig.ConfigContext, key string) (string, error) {
// 	keys := strings.Split(key, ".")
// 	if len(keys) != 2 {
// 		return "", BadKeyError
// 	}
// 	switch keys[0] {
// 	case "me":
// 		if confCtx == nil {
// 			return "", BadKeyError
// 		}
// 		return f.dir + "/" + confCtx.MacAddr + "/" + strings.Join(keys[1:], "/"), nil
// 	case "global":
// 		return f.dir + "/" + "global" + "/" + strings.Join(keys[1:], "/"), nil
// 	}
// 	return "", BadKeyError
// }
//
// func (f *Flags) GetValue(confCtx *cloudconfig.ConfigContext, key string) (string, error) {
// 	key, err := f.parseKey(confCtx, key)
// 	if err != nil {
// 		return "", err
// 	}
//
// 	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
// 	defer cancel()
//
// 	response, err := f.kapi.Get(ctx, key, nil)
// 	if err != nil {
// 		// return empty string with no error if the key is not found
// 		etcdError, ok := err.(etcd.Error)
// 		if ok && etcdError.Code == etcd.ErrorCodeKeyNotFound {
// 			return "", nil
// 		}
// 		return "", err
// 	}
// 	return response.Node.Value, nil
// }
//
// func (f *Flags) Set(confCtx *cloudconfig.ConfigContext, key string, value string) error {
// 	key, err := f.parseKey(confCtx, key)
// 	if err != nil {
// 		return err
// 	}
// 	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
// 	defer cancel()
// 	_, err = f.kapi.Set(ctx, key, value, nil)
// 	return err
// }
//
// func (f *Flags) GetAndDelete(confCtx *cloudconfig.ConfigContext, key string) (string, error) {
// 	value, err := f.GetValue(confCtx, key)
// 	if err != nil {
// 		return "", err
// 	}
// 	if err = f.Delete(confCtx, key); err != nil {
// 		return "", err
// 	}
// 	return value, nil
// }
//
// func (f *Flags) Delete(confCtx *cloudconfig.ConfigContext, key string) error {
// 	key, err := f.parseKey(confCtx, key)
// 	if err != nil {
// 		return err
// 	}
// 	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
// 	defer cancel()
// 	_, err = f.kapi.Delete(ctx, key, nil)
// 	if err != nil {
// 		// return empty string with no error if the key is not found
// 		etcdError, ok := err.(etcd.Error)
// 		if ok && etcdError.Code == etcd.ErrorCodeKeyNotFound {
// 			return nil
// 		}
// 		return err
// 	}
// 	return nil
// }
