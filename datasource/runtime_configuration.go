package datasource // import "github.com/cafebazaar/aghajoon/datasource"

import (
	"errors"
	"fmt"
	"io/ioutil"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/cafebazaar/aghajoon/cloudconfig"
	"github.com/coreos/etcd/Godeps/_workspace/src/golang.org/x/net/context"
	etcd "github.com/coreos/etcd/client"
	"gopkg.in/yaml.v2"
)

// RuntimeConfiguration is the the connector between aghajoon components and the state
type RuntimeConfiguration struct {
	DataSource           etcd.KeysAPI
	EtcdClient           etcd.Client
	EtcdDir              string
	WorkspacePath        string
	initialCoreOSVersion string
}

type initialValues struct {
	CoreOSVersion string `yaml:"coreos-version"`
}

// NewRuntimeConfiguration initialize etcd tree, do some validations, and returns a ready RuntimeConfiguration
func NewRuntimeConfiguration(dataSource etcd.KeysAPI, etcdClient etcd.Client, etcdDir string, workspacePath string) (*RuntimeConfiguration, error) {
	data, err := ioutil.ReadFile(filepath.Join(workspacePath, "initial.yaml"))
	if err != nil {
		return nil, fmt.Errorf("Error while trying to read initial data: %s", err)
	}

	var iVals initialValues
	err = yaml.Unmarshal(data, &iVals)
	if err != nil {
		return nil, fmt.Errorf("Error while reading initial data: %s", err)
	}
	if iVals.CoreOSVersion == "" {
		return nil, errors.New("A valid initial CoreOS version is required in initial data")
	}

	fmt.Printf("Initial Values: CoreOSVersion=%s\n", iVals.CoreOSVersion)

	instance := &RuntimeConfiguration{
		DataSource:           dataSource,
		EtcdClient:           etcdClient,
		EtcdDir:              etcdDir,
		WorkspacePath:        workspacePath,
		initialCoreOSVersion: iVals.CoreOSVersion,
	}

	_, err = instance.GetCoreOSVersion()
	if err != nil {
		etcdError, found := err.(etcd.Error)
		if found && etcdError.Code == etcd.ErrorCodeKeyNotFound {
			// Initializing
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			_, err = dataSource.Set(ctx, path.Join(etcdDir, "/coreos-version"), iVals.CoreOSVersion, nil)
			if err != nil {
				return nil, fmt.Errorf("Error while initializing etcd tree: %s", err)
			}
			fmt.Printf("Initialized etcd tree (%s)", etcdDir)
		} else {
			return nil, fmt.Errorf("Error while checking GetCoreOSVersion: %s", err)
		}
	}

	return instance, nil
}

// GetCoreOSVersion gets the current value from etcd and returns it if the image folder exists
// if not, the inital CoreOS version will be returned, with the raised error
func (rc *RuntimeConfiguration) GetCoreOSVersion() (string, error) {
	coreOSVersion, err := rc.GetValue(nil, "coreos-version")
	if err != nil {
		return rc.initialCoreOSVersion, err
	}

	imagesPath := filepath.Join(rc.WorkspacePath, "images", coreOSVersion)
	files, err := ioutil.ReadDir(imagesPath)
	if err != nil {
		return rc.initialCoreOSVersion, fmt.Errorf("Error while reading coreos subdirecory: %s (path=%s)", err, imagesPath)
	} else if len(files) == 0 {
		return rc.initialCoreOSVersion, errors.New("The images subdirecory of workspace should contains at least one version of CoreOS")
	}

	return coreOSVersion, nil
}

func (rc *RuntimeConfiguration) parseKey(key string) string {
	key = strings.Replace(key, ".", "/", -1)
	key = strings.Replace(key, "__/", rc.EtcdDir+"/", -1)
	return key
}

// GetValue normalizes the key parameter, retrive the value from etcd, and returns it
func (rc *RuntimeConfiguration) GetValue(confCtx *cloudconfig.ConfigContext, key string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	response, err := rc.DataSource.Get(ctx, rc.parseKey(key), nil)
	if err != nil {
		return "", err
	}
	return response.Node.Value, nil
}

func (rc *RuntimeConfiguration) Set(confCtx *cloudconfig.ConfigContext, key string, value string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, err := rc.DataSource.Set(ctx, rc.parseKey(key), value, nil)
	return err
}

func (rc *RuntimeConfiguration) GetAndDelete(confCtx *cloudconfig.ConfigContext, key string) (string, error) {
	value, err := rc.GetValue(confCtx, key)
	if err != nil {
		return "", err
	}
	if err = rc.Delete(confCtx, key); err != nil {
		return "", err
	}
	return value, nil
}

func (rc *RuntimeConfiguration) Delete(confCtx *cloudconfig.ConfigContext, key string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, err := rc.DataSource.Delete(ctx, rc.parseKey(key), nil)
	return err
}
