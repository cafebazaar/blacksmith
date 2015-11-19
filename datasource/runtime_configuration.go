package datasource // import "github.com/cafebazaar/aghajoon/datasource"

import (
	"errors"
	"fmt"
	"io/ioutil"
	"path"
	"path/filepath"
	"time"

	"github.com/coreos/etcd/Godeps/_workspace/src/golang.org/x/net/context"
	etcd "github.com/coreos/etcd/client"
	"gopkg.in/yaml.v2"
)

// RuntimeConfiguration is the the connector between aghajoon components and the state
type RuntimeConfiguration struct {
	DataSource           *etcd.KeysAPI
	EtcdDir              string
	WorkspacePath        string
	initialCoreOSVersion string
}

type initialValues struct {
	CoreOSVersion string `yaml:"coreos-version"`
}

func NewRuntimeConfiguration(dataSource *etcd.KeysAPI, etcdDir string, workspacePath string) (*RuntimeConfiguration, error) {
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
			_, err = (*dataSource).Set(ctx, path.Join(etcdDir, "/coreos-version"), iVals.CoreOSVersion, nil)
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
	ctxGet, cancelGet := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancelGet()
	resp, err := (*rc.DataSource).Get(ctxGet, path.Join(rc.EtcdDir, "/coreos-version"), nil)
	if err != nil {
		return rc.initialCoreOSVersion, err
	}
	coreOSVersion := resp.Node.Value

	imagesPath := filepath.Join(rc.WorkspacePath, "images", coreOSVersion)
	files, err := ioutil.ReadDir(imagesPath)
	if err != nil {
		return rc.initialCoreOSVersion, fmt.Errorf("Error while reading coreos subdirecory: %s (path=%s)", err, imagesPath)
	} else if len(files) == 0 {
		return rc.initialCoreOSVersion, errors.New("The images subdirecory of workspace should contains at least one version of CoreOS")
	}

	return coreOSVersion, nil
}
