package web

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"github.com/cafebazaar/blacksmith/logging"
)

const DebugTag string = "WEB"


// uploaded files metadata
func (ws *webServer) Files(w http.ResponseWriter, r *http.Request) {
	files := ws.ds.GetAllFiles()
	jsoned, _:= json.Marshal(files)
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

	ws.ds.NewFile(header.Filename, dst)
}

// DeleteFile allows the deletion of a file through http Request
func (ws *webServer) DeleteFile(w http.ResponseWriter, r *http.Request) {
	id := r.FormValue("id")

	if id != "" {
		ws.ds.GetAndDeleteFile(id)

	} else {
		http.Error(w, "No file name specified.", 400)
	}

}

			return
		}
	} else {
		http.Error(w, "No file name specified.", 400)
	}

}
