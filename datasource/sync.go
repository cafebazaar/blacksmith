package datasource

import (
	"github.com/coreos/etcd/cmd/vendor/golang.org/x/net/context"
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

		logging.Debug(debugTag, "watcher response: %s", resp)
		if resp.Action == "create" {
			file := &File{}
			json.Unmarshal([]byte(resp.Node.Value), file)
			if file.FromInstance == ds.serverIP.String() {continue}

			dst, err := os.Create(file.Location)
			if err != nil {
				logging.Debug(debugTag, "cloudn't create file due to: %s", err)
			}
			defer dst.Close()

			data, err := http.Get("http://" + file.FromInstance + ":8000" + file.Location)
			defer data.Body.Close()

			_, err = io.Copy(dst, data.Body)

		} else {
			logging.Debug(debugTag, "This Action has not been handled! %s", resp)
		}
	}
}
