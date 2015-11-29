package web // import "github.com/cafebazaar/aghajoon/web"

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/cafebazaar/aghajoon/datasource"
	"github.com/cafebazaar/aghajoon/dhcp"
	"github.com/gorilla/mux"
)

type RestServer struct {
	pool          *dhcp.LeasePool
	uiPath        *string
	runtimeConfig *datasource.RuntimeConfiguration
}

type uploadedFile struct {
	Name                 string    `json:"name"`
	Size                 int64     `json:"size"`
	LastModificationDate time.Time `json:"lastModifiedDate"`
}

func (a *RestServer) deleteFile(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("name")

	if name != "" {
		err := os.Remove(filepath.Join(a.runtimeConfig.WorkspacePath, "files", name))

		if err != nil {
			http.Error(w, err.Error(), 404)

			return
		}
	} else {
		http.Error(w, "No file name specified.", 400)
	}

}

func (a *RestServer) files(w http.ResponseWriter, r *http.Request) {
	files, err := ioutil.ReadDir(filepath.Join(a.runtimeConfig.WorkspacePath, "files"))
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

func (a *RestServer) upload(w http.ResponseWriter, r *http.Request) {
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

	dst, err := os.Create(filepath.Join(a.runtimeConfig.WorkspacePath, "files", header.Filename))
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

func NewRest(leasePool *dhcp.LeasePool, uiPath *string, runtimeConfig *datasource.RuntimeConfiguration) *RestServer {
	return &RestServer{
		pool:          leasePool,
		uiPath:        uiPath,
		runtimeConfig: runtimeConfig,
	}
}

func (a *RestServer) Mux() *mux.Router {
	mux := mux.NewRouter()
	mux.HandleFunc("/api/nodes", a.nodesList)
	mux.HandleFunc("/api/etcd-endpoints", a.etcdEndpoints)

	mux.HandleFunc("/upload/", a.upload)
	mux.HandleFunc("/files", a.files).Methods("GET")
	mux.HandleFunc("/files", a.deleteFile).Methods("DELETE")
	mux.PathPrefix("/files/").Handler(http.StripPrefix("/files/", http.FileServer(http.Dir(filepath.Join(a.runtimeConfig.WorkspacePath, "files")))))
	if *a.uiPath != "" {
		mux.PathPrefix("/ui/").Handler(http.StripPrefix("/ui/", http.FileServer(http.Dir(*a.uiPath))))
	}

	//if *a.uiPath != "" {
	//	ui := http.FileServer(http.Dir(*a.uiPath))
	//	mux.Handle("/ui/", http.StripPrefix("/ui/", ui))
	//}

	return mux
}

func (a *RestServer) nodesList(w http.ResponseWriter, r *http.Request) {
	leases, err := a.pool.Leases()
	if err != nil {
		http.Error(w, "Error in fetching lease data", 500)
	}
	nodesJSON, err := json.Marshal(leases)
	if err != nil {
		io.WriteString(w, fmt.Sprintf("{'error': %s}", err))
		return
	}
	io.WriteString(w, string(nodesJSON))
}

func (a *RestServer) etcdEndpoints(w http.ResponseWriter, r *http.Request) {
	// a.runtimeConfig.
	endpointsJSON, err := json.Marshal(a.runtimeConfig.EtcdClient.Endpoints())
	if err != nil {
		io.WriteString(w, fmt.Sprintf("{'error': %s}", err))
		return
	}
	io.WriteString(w, string(endpointsJSON))
}

// ServeWeb serves api of Aghajoon and a ui connected to that api
func ServeWeb(rest *RestServer, listenAddr net.TCPAddr) error {
	s := &http.Server{
		Addr:           listenAddr.String(),
		Handler:        rest.Mux(),
		ReadTimeout:    100 * time.Second,
		WriteTimeout:   100 * time.Second,
		MaxHeaderBytes: 1 << 30,
	}

	return s.ListenAndServe()
}
