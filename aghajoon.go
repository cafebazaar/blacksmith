package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/danderson/pixiecore/tftp"
)

//go:generate go-bindata -o pxelinux_autogen.go -prefix=pxelinux -ignore=README.md pxelinux

var (
	// I'm sort of giving you the option to change these ports here,
	// but all of them except the HTTP port are hardcoded in the PXE
	// option ROM, so it's pretty pointless unless you'd playing
	// packet rewriting tricks or doing simulations with packet
	// generators.
	portDHCP = flag.Int("port-dhcp", 67, "Port to listen on for DHCP requests")
	portPXE  = flag.Int("port-pxe", 4011, "Port to listen on for PXE requests")
	portTFTP = flag.Int("port-tftp", 69, "Port to listen on for TFTP requests")
	portHTTP = flag.Int("port-http", 70, "Port to listen on for HTTP requests")

	workspace = flag.String("workspace", "", "Path to home of CoreOS images")

	debug = flag.Bool("debug", false, "Log more things that aren't directly related to booting a recognized client")

	help = flag.Bool("help", false, "Print this help.")
)

func initHTTPBooter(ldlinux []byte) (*httpServer, error) {
	if *workspace == "" {
		return nil, errors.New("must provide -workspace")
	}

	return HTTPBooter(*portHTTP, *workspace, ldlinux)
}

func main() {
	flag.Parse()

	if *help {
		flag.Usage()
		os.Exit(0)
	}

	pxelinux, err := Asset("lpxelinux.0")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	ldlinux, err := Asset("ldlinux.c32")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	httpBooter, err := initHTTPBooter(ldlinux)
	if err != nil {
		flag.Usage()
		fmt.Fprintf(os.Stderr, "\nERROR: %s\n", err)
		os.Exit(1)
	}

	go func() {
		log.Fatalln(ServeProxyDHCP(*portDHCP))
	}()
	go func() {
		log.Fatalln(ServePXE(*portPXE, *portHTTP))
	}()
	go func() {
		tftp.Log = func(msg string, args ...interface{}) { Log("TFTP", msg, args...) }
		tftp.Debug = func(msg string, args ...interface{}) { Debug("TFTP", msg, args...) }
		log.Fatalln(tftp.ListenAndServe("udp4", ":"+strconv.Itoa(*portTFTP), tftp.Blob(pxelinux)))
	}()
	go func() {
		log.Fatalln(ServeHTTP(httpBooter))
	}()
	RecordLogs(*debug)
}
