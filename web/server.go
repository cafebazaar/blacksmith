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
	mux.HandleFunc("/api/", hello)
	mux.PathPrefix("/ui/").Handler(http.StripPrefix("/ui/", http.FileServer(http.Dir("/ui/"))))
	http.ListenAndServe(listenAddr.String(), mux)
}
