package web // import "github.com/cafebazaar/blacksmith/web"

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gorilla/mux"

	"github.com/cafebazaar/blacksmith/datasource"
	"github.com/cafebazaar/blacksmith/logging"
)

const (
	debugTag = "API"
)

type restServer struct {
	ds datasource.GeneralDataSource
}

// Version returns json encoded version details
func (rs *restServer) Version(w http.ResponseWriter, r *http.Request) {
	logging.LogHTTPRequest(debugTag, r)
	versionJSON, err := json.Marshal(rs.ds.Version())
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error": %q}`, err), 500)
		return
	}
	io.WriteString(w, string(versionJSON))
}

type nodeDetails struct {
	Name          string    `json:"name"`
	Nic           string    `json:"nic"`
	IP            net.IP    `json:"ip"`
	FirstAssigned time.Time `json:"firstAssigned"`
	LastAssigned  time.Time `json:"lastAssigned"`
}

func nodeToDetails(node datasource.Machine) (*nodeDetails, error) {
	name := node.Name()
	mac := node.Mac()
	ip, err := node.IP()
	if err != nil {
		return nil, errors.New("IP")
	}
	first, err := node.FirstSeen()
	if err != nil {
		return nil, errors.New("FIRST")
	}
	last, err := node.LastSeen()
	if err != nil {
		return nil, errors.New("LAST")
	}
	return &nodeDetails{name, mac.String(), ip, first, last}, nil
}

// NodesList creates a list of the currently known nodes based on the etcd
// entries
func (rs *restServer) NodesList(w http.ResponseWriter, r *http.Request) {
	logging.LogHTTPRequest(debugTag, r)

	machines, err := rs.ds.Machines()
	if err != nil || machines == nil {
		http.Error(w, fmt.Sprintf(`{"error": %q}`, err), 500)
		return
	}
	nodes := make([]*nodeDetails, 0, len(machines))
	for _, node := range machines {
		l, err := nodeToDetails(node)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error": %q}`, err), 500)
			return
		}
		if l != nil {
			nodes = append(nodes, l)
		}
	}

	nodesJSON, err := json.Marshal(nodes)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error": %q}`, err), 500)
		return
	}
	io.WriteString(w, string(nodesJSON))
}

type uploadedFile struct {
	Name                 string    `json:"name"`
	Size                 int64     `json:"size"`
	LastModificationDate time.Time `json:"lastModifiedDate"`
}

// Files allows utilization of the uploaded/shared files through http requests
func (rs *restServer) Files(w http.ResponseWriter, r *http.Request) {
	logging.LogHTTPRequest(debugTag, r)

	files, err := ioutil.ReadDir(filepath.Join(rs.ds.WorkspacePath(), "files"))
	if err != nil {
		http.Error(w, err.Error(), 500)
	}

	var filesList []uploadedFile
	for _, f := range files {
		if f.Name()[0] == '.' {
			continue
		}
		var uploadedFile uploadedFile
		uploadedFile.Size = f.Size()
		uploadedFile.LastModificationDate = f.ModTime()
		uploadedFile.Name = f.Name()
		filesList = append(filesList, uploadedFile)
	}

	jsoned, _ := json.Marshal(filesList)
	io.WriteString(w, string(jsoned))
}

// Upload does what it is supposed to do!
func (rs *restServer) Upload(w http.ResponseWriter, r *http.Request) {
	logging.LogHTTPRequest(debugTag, r)

	const MaxFileSize = 1 << 30
	// This feels like a bad hack...
	if r.ContentLength > MaxFileSize {
		http.Error(w, "Request too large", 400)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, MaxFileSize)

	err := r.ParseMultipartForm(1024)
	if err != nil {
		http.Error(w, "File too large", 400)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		panic(err)
	}

	dst, err := os.Create(filepath.Join(rs.ds.WorkspacePath(), "files", header.Filename))
	defer dst.Close()
	if err != nil {
		http.Error(w, err.Error(), 500)
	}

	written, err := io.Copy(dst, io.LimitReader(file, MaxFileSize))
	if err != nil {
		http.Error(w, err.Error(), 500)
	}

	if written == MaxFileSize {
		http.Error(w, "File too large", 400)
		return
	}
}

// DeleteFile allows the deletion of a file through http Request
func (rs *restServer) DeleteFile(w http.ResponseWriter, r *http.Request) {
	logging.LogHTTPRequest(debugTag, r)

	name := r.FormValue("name")

	if name != "" {
		err := os.Remove(filepath.Join(rs.ds.WorkspacePath(), "files", name))

		if err != nil {
			http.Error(w, err.Error(), 404)

			return
		}
	} else {
		http.Error(w, "No file name specified.", 400)
	}

}

// Handler uses a multiplexing router to route http requests
func (rs *restServer) Handler() http.Handler {
	mux := mux.NewRouter()

	mux.HandleFunc("/api/version", rs.Version)

	mux.HandleFunc("/api/nodes", rs.NodesList)

	mux.HandleFunc("/upload/", rs.Upload)
	mux.HandleFunc("/files", rs.Files).Methods("GET")
	mux.HandleFunc("/files", rs.DeleteFile).Methods("DELETE")
	mux.PathPrefix("/files/").Handler(http.StripPrefix("/files/",
		http.FileServer(http.Dir(filepath.Join(rs.ds.WorkspacePath(), "files")))))
	mux.PathPrefix("/ui/").Handler(http.FileServer(FS(false)))

	return mux
}

//ServeWeb serves api of Blacksmith and a ui connected to that api
func ServeWeb(ds datasource.GeneralDataSource, listenAddr net.TCPAddr) error {
	r := &restServer{ds: ds}
	s := &http.Server{
		Addr:    listenAddr.String(),
		Handler: r.Handler(),
	}

	return s.ListenAndServe()
}
