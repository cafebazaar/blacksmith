package datasource

import (
	"golang.org/x/net/context"
	etcd "github.com/coreos/etcd/client"
	"github.com/cafebazaar/blacksmith/logging"
	"time"
	"encoding/json"
	"os"
	"net/http"
	"io"
)

const (
	filesEtcdDir = "files"
)


// Registers a new file meta data on etcd
func (ds *EtcdDataSource) NewFile(name string, fileHandler *os.File) {
	ctx, cancel := context.WithTimeout(context.Background(), etcdTimeout)
	defer cancel()
	options := etcd.CreateInOrderOptions{
		TTL: 0, // permanent
	}
	fileInfo, err := fileHandler.Stat()
	if err != nil {
		logging.Debug(debugTag, "couldn't get file stat due to: %s", err)
	}
	file := &File{
		Name: name,
		Location: fileHandler.Name(),
		Size: fileInfo.Size(),
		LastModificationDate: fileInfo.ModTime().Unix(),
		UploadedAt: time.Now().Unix(),
		FromInstance: ds.serverIP.String(),
	}
	jsoned, _ := json.Marshal(file)
	_, err = ds.keysAPI.CreateInOrder(ctx, ds.prefixify(filesEtcdDir), string(jsoned), &options)
	if err != nil {
		logging.Debug(debugTag, "Couldn't create new file node on etcd due to: %s", err)
	}
}

// Watches the files on etcd and update itself when something changes
func (ds *EtcdDataSource) WatchFileChanges()  {
	options := etcd.WatcherOptions{
		AfterIndex: 0, // current index
		Recursive: true,
	}
	watcher := ds.keysAPI.Watcher(ds.prefixify(filesEtcdDir), &options)

	for {
		ctx := context.Background()
		resp, err := watcher.Next(ctx)
		if err != nil {
			logging.Debug(debugTag, "couldn't retrieve data from etcd when watching files due to: %s", err)
		}

		if resp.Action == "create" {
			file := &File{}
			json.Unmarshal([]byte(resp.Node.Value), file)
			if file.FromInstance == ds.serverIP.String() {
				continue
			}
			dst, err := os.Create(file.Location)
			if err != nil {
				logging.Debug(debugTag, "cloudn't create file due to: %s", err)
			}
			defer dst.Close()

			data, err := http.Get("http://" + file.FromInstance + ":8000" + file.Location)
			defer data.Body.Close()

			_, err = io.Copy(dst, data.Body)

		} else if resp.Action == "delete" {
			file := &File{}
			json.Unmarshal([]byte(resp.PrevNode.Value), file)
			logging.Debug(debugTag, "file: %s", file)
			err := os.Remove(file.Location)
			if err != nil {
				logging.Debug(debugTag, "couldn't delete the file due to: %s", err)
			}
		} else {
			logging.Debug(debugTag, "This Action has not been handled! %s", resp)
		}
	}
}

// retrieve all the files meta data from etcd and retrun it as an array of file instances
func (ds *EtcdDataSource) GetAllFiles() []*File {
	resp, _ := ds.GetNodes(ds.prefixify(filesEtcdDir))

	var files []*File

	for _, node := range resp {
		if node.TTL > 0 {continue}
		file := &File{}
		json.Unmarshal([]byte(node.Value), file)
		file.Id = node.Key
		files = append(files, file)
	}
	return files
}

// retrieve file meta data from etcd and return it as a file instance
func (ds *EtcdDataSource) GetFile(key string) *File {
	file := &File{}
	val, _ := ds.GetAbsolute(key)
	file.Id = key
	json.Unmarshal([]byte(val), file)
	return file
}

// retrieve the file meta data on etcd and set a TTL on the key
func (ds *EtcdDataSource) DeleteFile(key string) *File {
	resp, err := ds.DeleteAbsolute(key)
	if err != nil {
		logging.Debug(debugTag, "couldn't delete file on etcd due to: %s", err)
	}
	file := &File{}
	file.Id = key
	json.Unmarshal([]byte(resp.Value), file)
	return file
}
