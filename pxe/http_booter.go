package pxe

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/cafebazaar/blacksmith/datasource"
	"github.com/cafebazaar/blacksmith/logging"
	"github.com/cafebazaar/blacksmith/templating"
)

const bootMessageTemplate = `
		Blacksmith ($VERSION)
		+ MAC ADDR:	$MAC
`

type nodeContext struct {
	IP string
}

type HTTPBooter struct {
	listenAddr          net.TCPAddr
	ldlinux             []byte
	datasource          datasource.DataSource
	bootParamsTemplates *template.Template
	webPort             int
	bootMessageTemplate string
}

func NewHTTPBooter(listenAddr net.TCPAddr, ldlinux []byte,
	ds datasource.DataSource, webPort int) (*HTTPBooter, error) {
	bootMessageVersionedTemplate := strings.Replace(bootMessageTemplate, "$VERSION", ds.Version().Version, -1)
	booter := &HTTPBooter{
		listenAddr:          listenAddr,
		ldlinux:             ldlinux,
		datasource:          ds,
		webPort:             webPort,
		bootMessageTemplate: bootMessageVersionedTemplate,
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
	errStr := fmt.Sprintf("%s requested a pxelinux config from URL %q, which does not include a correct MAC address", r.RemoteAddr, r.URL)
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

	machine, exist := b.datasource.GetMachine(mac)
	if !exist {
		logging.Debug("HTTPBOOTER", "Machine not found. mac=%s", mac)
		http.Error(w, "Machine not found", http.StatusNotFound)
		return
	}

	if _, _, err := net.SplitHostPort(r.Host); err != nil {
		r.Host = fmt.Sprintf("%s:%d", r.Host, b.listenAddr.Port)
	}

	coreOSVersion, _ := b.datasource.CoreOSVersion()

	KernelURL := "http://" + r.Host + "/f/" + coreOSVersion + "/kernel"
	InitrdURL := "http://" + r.Host + "/f/" + coreOSVersion + "/initrd"

	host, _, err := net.SplitHostPort(r.Host)
	if err != nil {
		logging.Log("HTTPBOOTER", "error in parsing host and port")
		http.Error(w, "error in parsing host and port", 500)
		return
	}

	params, err := templating.ExecuteTemplateFolder(
		path.Join(b.datasource.WorkspacePath(), "config", "bootparams"), machine)
	if err != nil {
		logging.Log("HTTPBOOTER", `Error while executing the template: %q`, err)
		http.Error(w, fmt.Sprintf(`Error while executing the template: %q`, err),
			http.StatusInternalServerError)
		return
	}

	if err != nil {
		logging.Log("HTTPBOOTER", "error in bootparam template - %s", err.Error())
		http.Error(w, "error in bootparam template", 500)
		return
	}
	params = strings.Replace(params, "\n", " ", -1)

	Cmdline := fmt.Sprintf(
		"cloud-config-url=http://%s:%d/t/cc/%s "+
			"coreos.config.url=http://%s:%d/t/ig/%s %s",
		host, b.webPort, mac.String(),
		host, b.webPort, mac.String(), params)
	bootMessage := strings.Replace(b.bootMessageTemplate, "$MAC", macStr, -1)
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

func HTTPBooterMux(listenAddr net.TCPAddr, ds datasource.DataSource, webPort int) (*http.ServeMux, error) {
	ldlinux, err := FSByte(false, "/pxelinux/ldlinux.c32")
	if err != nil {
		return nil, err
	}
	booter, err := NewHTTPBooter(listenAddr, ldlinux, ds, webPort)
	if err != nil {
		return nil, err
	}
	return booter.Mux(), nil
}

func ServeHTTPBooter(listenAddr net.TCPAddr, ds datasource.DataSource, webPort int) error {
	logging.Log("HTTPBOOTER", "Listening on %s", listenAddr.String())
	mux, err := HTTPBooterMux(listenAddr, ds, webPort)
	if err != nil {
		return err
	}
	return http.ListenAndServe(listenAddr.String(), mux)
}
