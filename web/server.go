package web // import "github.com/cafebazaar/aghajoon/web"

import (
	"encoding/json"
	"github.com/cafebazaar/aghajoon/dhcp"
	"github.com/gorilla/mux"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"time"
)

type RestServer struct {
	pool           *dhcp.LeasePool
	uiPath         *string
	filesDirectory string
}

type UploadedFile struct {
	Name                 string    `json:"name"`
	Size                 int64     `json:"size"`
	LastModificationDate time.Time `json:"lastModifiedDate"`
}

func (a *RestServer) deleteFile(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("name")

	if name != "" {
		err := os.Remove(a.filesDirectory + name)

		if err != nil {
			http.Error(w, err.Error(), 404)

			return
		}
	} else {
		http.Error(w, "No file name specified.", 400)
	}

}

func (a *RestServer) files(w http.ResponseWriter, r *http.Request) {
	files, err := ioutil.ReadDir(a.filesDirectory)
	if err != nil {
		http.Error(w, err.Error(), 500)
	}

	filesList := make([]UploadedFile, 0)
	for _, f := range files {
		var uploadedFile UploadedFile
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

	dst, err := os.Create(a.filesDirectory + header.Filename)
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

func NewRest(leasePool *dhcp.LeasePool, uiPath *string, workspacePathFlag *string) *RestServer {
	return &RestServer{
		pool:           leasePool,
		uiPath:         uiPath,
		filesDirectory: *workspacePathFlag + "/files/",
	}
}

func (a *RestServer) Mux() *mux.Router {
	mux := mux.NewRouter()
	mux.HandleFunc("/api/nodes", a.nodesList)

	mux.HandleFunc("/upload/", a.upload)
	mux.HandleFunc("/files", a.files).Methods("GET")
	mux.HandleFunc("/files", a.deleteFile).Methods("DELETE")
	mux.PathPrefix("/files/").Handler(http.StripPrefix("/files/", http.FileServer(http.Dir(a.filesDirectory))))
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
	json, _ := json.Marshal(leases)
	io.WriteString(w, string(json))
}

func ServeWeb(rest *RestServer, listenAddr net.TCPAddr) error {

	s := &http.Server{
		Addr:           ":8000",
		Handler:        rest.Mux(),
		ReadTimeout:    100 * time.Second,
		WriteTimeout:   100 * time.Second,
		MaxHeaderBytes: 1 << 30,
	}

	return s.ListenAndServe()

}
