package pxe

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/cafebazaar/blacksmith/cloudconfig"
	"github.com/cafebazaar/blacksmith/datasource"
	"github.com/cafebazaar/blacksmith/logging"
)

// pxelinux configuration that tells the PXE/UNDI stack to boot from
// local disk.
const bootFromDisk = `
DEFAULT local
LABEL local
LOCALBOOT 0
`

const bootMessageTemplate = `
		Blacksmith 0.2
		+ MAC ADDR:	$MAC
`

type nodeContext struct {
	IP string
}

type HTTPBooter struct {
	listenAddr          net.TCPAddr
	ldlinux             []byte
	datasource          datasource.GeneralDataSource
	bootParamsTemplates *template.Template
}

func NewHTTPBooter(listenAddr net.TCPAddr, ldlinux []byte,
	ds datasource.GeneralDataSource, bootParamsTemplates *template.Template) (*HTTPBooter, error) {
	booter := &HTTPBooter{
		listenAddr:          listenAddr,
		ldlinux:             ldlinux,
		datasource:          ds,
		bootParamsTemplates: bootParamsTemplates,
	}
	return booter, nil
}

func (b *HTTPBooter) Mux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/ldlinux.c32", b.ldlinuxHandler)
	mux.HandleFunc("/pxelinux.cfg/", b.pxelinuxConfig)
	mux.HandleFunc("/f/", b.fileHandler)
	return mux
}

func (b *HTTPBooter) ldlinuxHandler(w http.ResponseWriter, r *http.Request) {
	logging.LogHTTPRequest("HTTPBOOTER", r)
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(b.ldlinux)
}

func (b *HTTPBooter) pxelinuxConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")

	macStr := filepath.Base(r.URL.Path)
	errStr := fmt.Sprintf("%s requested a pxelinux config from URL %q, which does not include a MAC address", r.RemoteAddr, r.URL)
	if !strings.HasPrefix(macStr, "01-") {
		logging.Debug("HTTPBOOTER", errStr)
		http.Error(w, "Missing MAC address in request", http.StatusBadRequest)
		return
	}
	mac, err := net.ParseMAC(macStr[3:])
	if err != nil {
		logging.Debug("HTTPBOOTER", errStr)
		http.Error(w, "Malformed MAC address in request", http.StatusBadRequest)
		return
	}

	if _, _, err := net.SplitHostPort(r.Host); err != nil {
		r.Host = fmt.Sprintf("%s:%d", r.Host, b.listenAddr.Port)
	}

	// TODO: Ask dataSource about the mac
	// We have a machine sitting in pxelinux, but the Booter says
	// we shouldn't be netbooting. So, give it a config that tells
	// pxelinux to shut down PXE booting and continue with the
	// next local boot method.

	coreOSVersion, _ := b.datasource.CoreOSVersion()

	KernelURL := "http://" + r.Host + "/f/" + coreOSVersion + "/kernel"
	InitrdURL := "http://" + r.Host + "/f/" + coreOSVersion + "/initrd"

	host, _, err := net.SplitHostPort(r.Host)
	if err != nil {
		logging.Log("HTTPBOOTER", "error in parsing host and port")
		http.Error(w, "error in parsing host and port", 500)
		return
	}

	// generate bootparams config
	params, err := cloudconfig.MacBootParamsGenerator(b.datasource, mac.String(),
		b.bootParamsTemplates)
	// params, err := b.bootParamsRepo.GenerateConfig(&cloudconfig.ConfigContext{
	// MacAddr: strings.Replace(mac.String(), ":", "", -1),
	// IP:      "",
	// })
	if err != nil {
		logging.Log("HTTPBOOTER", "error in bootparam template - %s", err.Error())
		http.Error(w, "error in bootparam template", 500)
		return
	}
	params = strings.Replace(params, "\n", " ", -1)

	// FIXME: 8001 is hardcoded
	Cmdline := fmt.Sprintf(
		"cloud-config-url=http://%s:8001/cloud/%s "+
			"coreos.config.url=http://%s:8001/ignition/%s %s",
		host, strings.Replace(mac.String(), ":", "", -1),
		host, strings.Replace(mac.String(), ":", "", -1), params)
	bootMessage := strings.Replace(bootMessageTemplate, "$MAC", macStr, -1)
	cfg := fmt.Sprintf(`
SAY %s
DEFAULT linux
LABEL linux
LINUX %s
APPEND initrd=%s %s
`, strings.Replace(bootMessage, "\n", "\nSAY ", -1), KernelURL, InitrdURL, Cmdline)
	w.Write([]byte(cfg))
	logging.Log("HTTPBOOTER", "Sent pxelinux config to %s (%s)", mac, r.RemoteAddr)
}

// Get the contents of a blob mentioned in a previously issued
// BootSpec. Additionally returns a pretty name for the blob for
// logging purposes.
func (b *HTTPBooter) coreOS(version string, id string) (io.ReadCloser, error) {
	imagePath := filepath.Join(b.datasource.WorkspacePath(), "images")
	switch id {
	case "kernel":
		path := filepath.Join(imagePath, version, "coreos_production_pxe.vmlinuz")
		logging.Debug("HTTPBOOTER", "path=<%q>", path)
		return os.Open(path)
	case "initrd":
		return os.Open(filepath.Join(imagePath, version, "coreos_production_pxe_image.cpio.gz"))
	}
	return nil, fmt.Errorf("id=<%q> wasn't expected", id)
}

func (b *HTTPBooter) fileHandler(w http.ResponseWriter, r *http.Request) {
	splitPath := strings.SplitN(r.URL.Path, "/", 4)
	version := splitPath[2]
	id := splitPath[3]

	logging.Debug("HTTPBOOTER", "Got request for %s", r.URL.Path)

	var (
		f   io.ReadCloser
		err error
	)

	f, err = b.coreOS(version, id)

	if err != nil {
		logging.Log("HTTPBOOTER", "Couldn't get byte stream for %q from %s: %s", r.URL, r.RemoteAddr, err)
		http.Error(w, "Couldn't get byte stream", http.StatusInternalServerError)
		return
	}
	defer f.Close()

	w.Header().Set("Content-Type", "application/octet-stream")
	written, err := io.Copy(w, f)
	if err != nil {
		logging.Log("HTTPBOOTER", "Error serving %s to %s: %s", id, r.RemoteAddr, err)
		return
	}
	logging.Log("HTTPBOOTER", "Sent %s to %s (%d bytes)", id, r.RemoteAddr, written)
}

func HTTPBooterMux(listenAddr net.TCPAddr, ds datasource.GeneralDataSource,
	bootParamsTemplates *template.Template) (*http.ServeMux, error) {
	ldlinux, err := FSByte(false, "/pxelinux/ldlinux.c32")
	if err != nil {
		return nil, err
	}
	booter, err := NewHTTPBooter(listenAddr, ldlinux, ds, bootParamsTemplates)
	if err != nil {
		return nil, err
	}
	return booter.Mux(), nil
}

func ServeHTTPBooter(listenAddr net.TCPAddr, ds datasource.GeneralDataSource,
	bootParamsTemplates *template.Template) error {
	logging.Log("HTTPBOOTER", "Listening on %s", listenAddr.String())
	mux, err := HTTPBooterMux(listenAddr, ds, bootParamsTemplates)
	if err != nil {
		return err
	}
	return http.ListenAndServe(listenAddr.String(), mux)
}
