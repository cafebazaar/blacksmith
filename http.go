package main

import (
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

// pxelinux configuration that tells the PXE/UNDI stack to boot from
// local disk.
const bootFromDisk = `
DEFAULT local
LABEL local
LOCALBOOT 0
`

// A silly limerick displayed while pxelinux loads big OS
// images. Possibly the most important piece of this program.
const limerick = `
	        There once was a protocol called PXE,
	        Whose specification was overly tricksy.
	        A committee refined it,
	        Into a big Turing tarpit,
	        And now you're using it to boot your PC.
`

type httpServer struct {
	dataSource    *dataSource
	workspacePath string
	ldlinux       []byte
	key           [32]byte // to sign URLs
	port          int
}

type nodeContext struct {
	IP string
}

func (s *httpServer) Ldlinux(w http.ResponseWriter, r *http.Request) {
	Debug("HTTPBOOTER", "Starting send of ldlinux.c32 to %s (%d bytes)", r.RemoteAddr, len(s.ldlinux))
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(s.ldlinux)
	Log("HTTPBOOTER", "Sent ldlinux.c32 to %s (%d bytes)", r.RemoteAddr, len(s.ldlinux))
}

func (s *httpServer) PxelinuxConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")

	macStr := filepath.Base(r.URL.Path)
	errStr := fmt.Sprintf("%s requested a pxelinux config from URL %q, which does not include a MAC address", r.RemoteAddr, r.URL)
	if !strings.HasPrefix(macStr, "01-") {
		Debug("HTTPBOOTER", errStr)
		http.Error(w, "Missing MAC address in request", http.StatusBadRequest)
		return
	}
	mac, err := net.ParseMAC(macStr[3:])
	if err != nil {
		Debug("HTTPBOOTER", errStr)
		http.Error(w, "Malformed MAC address in request", http.StatusBadRequest)
		return
	}

	if _, _, err := net.SplitHostPort(r.Host); err != nil {
		r.Host = fmt.Sprintf("%s:%d", r.Host, s.port)
	}

	// TODO: Ask dataSource about the mac
	// coreOSVersion, err := b.dataSource.GetCoreOSVersion()

	// if ??? {
	// 	// We have a machine sitting in pxelinux, but the Booter says
	// 	// we shouldn't be netbooting. So, give it a config that tells
	// 	// pxelinux to shut down PXE booting and continue with the
	// 	// next local boot method.
	// 	Debug("HTTPBOOTER", "Telling pxelinux on %s (%s) to boot from disk because of API server verdict: %s", mac, r.RemoteAddr, err)
	// 	w.Write([]byte(bootFromDisk))
	// 	return
	// }

	coreOSVersion := "835.1.0"
	KernelURL := "http://" + r.Host + "/f/" + coreOSVersion + "/kernel"
	InitrdURL := "http://" + r.Host + "/f/" + coreOSVersion + "/initrd"
	Cmdline := fmt.Sprintf("cloud-config-url=http://%s/cloud-config.yml?mac=%s "+
		"coreos.config.url=http://%s/ignition-config.yml?mac=%s",
		r.Host, mac, r.Host, mac)

	cfg := fmt.Sprintf(`
SAY %s
DEFAULT linux
LABEL linux
LINUX %s
APPEND initrd=%s %s
`, strings.Replace(limerick, "\n", "\nSAY ", -1), KernelURL, InitrdURL, Cmdline)

	w.Write([]byte(cfg))
	Log("HTTPBOOTER", "Sent pxelinux config to %s (%s)", mac, r.RemoteAddr)
}

// Get the contents of a blob mentioned in a previously issued
// BootSpec. Additionally returns a pretty name for the blob for
// logging purposes.
func (s *httpServer) Read(version string, id string) (io.ReadCloser, error) {
	imagePath := filepath.Join(s.workspacePath, "images")
	switch id {
	case "kernel":
		path := filepath.Join(imagePath, version, "coreos_production_pxe.vmlinuz")
		Debug("HTTPBOOTER", "path=<%q>", path)
		return os.Open(path)
	case "initrd":
		return os.Open(filepath.Join(imagePath, version, "coreos_production_pxe_image.cpio.gz"))
	}
	return nil, fmt.Errorf("id=<%q> wasn't expected", id)
}

func (s *httpServer) File(w http.ResponseWriter, r *http.Request) {
	Debug("HTTPBOOTER", "Got request for %s", r.URL.Path)

	splitPath := strings.SplitN(r.URL.Path, "/", 4)
	version := splitPath[2]
	id := splitPath[3]

	var (
		f   io.ReadCloser
		err error
	)

	f, err = s.Read(version, id)

	if err != nil {
		Log("HTTPBOOTER", "Couldn't get byte stream for %q from %s: %s", r.URL, r.RemoteAddr, err)
		http.Error(w, "Couldn't get byte stream", http.StatusInternalServerError)
		return
	}
	defer f.Close()

	w.Header().Set("Content-Type", "application/octet-stream")
	written, err := io.Copy(w, f)
	if err != nil {
		Log("HTTPBOOTER", "Error serving %s to %s: %s", id, r.RemoteAddr, err)
		return
	}
	Log("HTTPBOOTER", "Sent %s to %s (%d bytes)", id, r.RemoteAddr, written)
}

func (s *httpServer) CloudConfig(w http.ResponseWriter, r *http.Request) {
	Debug("HTTPBOOTER", "Starting send of cloud config to %s (%d bytes)", r.RemoteAddr, len(s.ldlinux))
	w.Header().Set("Content-Type", "application/x-yaml")
	tmpl, err := template.ParseFiles(filepath.Join(*workspace, "cloud-config.yml"))
	if err != nil {
		Log("HTTPBOOTER", "Error while trying to load cloud config template: %s", err)
		return
	}
	context := nodeContext{
		IP: r.RemoteAddr,
	}
	tmpl.Execute(w, context)
	Log("HTTPBOOTER", "Sent cloud config to %s", r.RemoteAddr)
}

func (s *httpServer) IgnitionConfig(w http.ResponseWriter, r *http.Request) {
	Debug("HTTPBOOTER", "Starting send of ignition config to %s (%d bytes)", r.RemoteAddr, len(s.ldlinux))
	w.Header().Set("Content-Type", "application/x-yaml")
	tmpl, err := template.ParseFiles(filepath.Join(*workspace, "ignition-config.yml"))
	if err != nil {
		Log("HTTPBOOTER", "Error while trying to load ignition config template: %s", err)
		return
	}
	context := nodeContext{
		IP: r.RemoteAddr,
	}
	tmpl.Execute(w, context)
	Log("HTTPBOOTER", "Sent ignition config to %s", r.RemoteAddr)
}

func HTTPBooter(port int, workspacePath string, ldlinux []byte) (*httpServer, error) {
	imagesPath := filepath.Join(*workspace, "images")
	files, err := ioutil.ReadDir(imagesPath)
	if err != nil {
		return nil, fmt.Errorf("error while reading images subdirecory: %s (path=%s)", err, imagesPath)
	} else if len(files) == 0 {
		return nil, errors.New("the images subdirecory of workspace should contains at least one version of CoreOS")
	}

	s := &httpServer{
		workspacePath: workspacePath,
		ldlinux:       ldlinux,
		port:          port,
	}

	return s, nil
}

func ServeHTTP(s *httpServer) error {
	if _, err := io.ReadFull(rand.Reader, s.key[:]); err != nil {
		return fmt.Errorf("cannot initialize ephemeral signing key: %s", err)
	}

	http.HandleFunc("/ldlinux.c32", s.Ldlinux)
	http.HandleFunc("/pxelinux.cfg/", s.PxelinuxConfig)
	http.HandleFunc("/f/", s.File)
	http.HandleFunc("/cloud-config.yml", s.CloudConfig)
	http.HandleFunc("/ignition-config.yml", s.IgnitionConfig)

	Log("HTTPBOOTER", "Listening on port %d", s.port)
	return http.ListenAndServe(fmt.Sprintf(":%d", s.port), nil)
}
