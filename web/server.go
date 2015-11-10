package web

import (
	"io"
	"net"
	"net/http"
)

func hello(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "Hello world!")
}

func ServeWeb(listenAddr net.TCPAddr) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api", hello)
	mux.Handle("/ui", http.FileServer(http.Dir("/vagrant/web/ui/")))
	http.ListenAndServe(listenAddr.String(), mux)
}
