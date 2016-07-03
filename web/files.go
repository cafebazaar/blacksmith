package web

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"time"
	"github.com/cafebazaar/blacksmith/logging"
)

const DebugTag string = "WEB"

type uploadedFile struct {
	Name                 string    `json:"name"`
	Size                 int64     `json:"size"`
	LastModificationDate time.Time `json:"lastModifiedDate"`
}

// Files allows utilization of the uploaded/shared files through http requests
func (ws *webServer) Files(w http.ResponseWriter, r *http.Request) {
	files, err := ioutil.ReadDir(filepath.Join(ws.ds.WorkspacePath(), "files"))
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
func (ws *webServer) Upload(w http.ResponseWriter, r *http.Request) {
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

	dst, err := os.Create(filepath.Join(ws.ds.WorkspacePath(), "files", urandomString(6) + "-" + header.Filename))
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

	ws.ds.NewFile(header.Filename, dst.Name())
}

// DeleteFile allows the deletion of a file through http Request
func (ws *webServer) DeleteFile(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("name")

	if name != "" {
		err := os.Remove(filepath.Join(ws.ds.WorkspacePath(), "files", name))

		if err != nil {
			http.Error(w, err.Error(), 404)

			return
		}
	} else {
		http.Error(w, "No file name specified.", 400)
	}

}
