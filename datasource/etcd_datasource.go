package datasource

import (
	etcd "github.com/coreos/etcd/client"
)

//EtcdDataSource implements MasterDataSource interface using etcd as it's
//datasource
//Implements MasterDataSource interface
type EtcdDataSource struct {
	keysAPI              etcd.KeysAPI
	client               etcd.Client
	leaseStart           net.IP
	leaseRange           net.IP
	etcdDir              string
	workspacePath        string
	initialCoreOSVersion string
}

//CoreOSVersion gets the current value from etcd and returns it if the image folder exists
//if not, the inital CoreOS version will be returned, with the raised error
//part of GeneralDataSource interface implementation
func (ds *EtcdDataSource) CoreOSVersion() (string, error) {
	coreOSVersion, err := ds.GetValue(nil, "coreos-version")
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

func (ds *EtcdDataSource) parseKey(key string) string {
	key = strings.Replace(key, ".", "/", -1)
	key = strings.Replace(key, "__/", ds.etcdDir+"/", -1)
	return key
}

//Get parses the etcd key and returns it's value
//part of KeyValueDataSource interface implementation
func (ds *EtcdDataSource) Get(key string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	response, err := ds.DataSource.Get(ctx, ds.parseKey(key), nil)
	if err != nil {
		return "", err
	}
	return response.Node.Value, nil
}

//Set sets and etcd key to a value
//part of KeyValueDataSource interface implementation
func (ds *EtcdDataSource) Set(key string, value string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, err := ds.keysAPI.Set(ctx, ds.parseKey(key), value, nil)
	return err
}

//GetAndDelete gets the value of an etcd key and returns it, and deletes the record
//afterwards
//part of KeyValueDataSource interface implementation
func (ds *EtcdDataSource) GetAndDelete(confCtx *cloudconfig.ConfigContext, key string) (string, error) {
	value, err := ds.Get(confCtx, key)
	if err != nil {
		return "", err
	}
	if err = ds.Delete(confCtx, key); err != nil {
		return "", err
	}
	return value, nil
}

//Delete erases the key from etcd
//part of KeyValueDataSource interface implementation
func (ds *EtcdDataSource) Delete(confCtx *cloudconfig.ConfigContext, key string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, err := ds.DataSource.Delete(ctx, ds.parseKey(key), nil)
	return err
}

type initialValues struct {
	CoreOSVersion string `yaml:"coreos-version"`
}

//NewEtcdDataSource gives blacksmith the ability to use an etcd endpoint as
//a MasterDataSource
func NewEtcdDataSource(kapi etcd.KeysAPI, client etcd.Client, leaseStart,
	leaseRange net.IP, etcdDir, workspacePath string) (MasterDataSource, error) {

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

	instance := &EtcdDataSource{
		KeysAPI:              dataSource,
		Client:               etcdClient,
		etcdDir:              etcdDir,
		workspacePath:        workspacePath,
		initialCoreOSVersion: iVals.CoreOSVersion,
	}

	_, err = instance.CoreOSVersion()
	if err != nil {
		etcdError, found := err.(etcd.Error)
		if found && etcdError.Code == etcd.ErrorCodeKeyNotFound {
			// Initializing
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			_, err = instance.keysAPI.Set(ctx, path.Join(etcdDir, "/coreos-version"), iVals.CoreOSVersion, nil)
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
