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

	log "github.com/Sirupsen/logrus"

	"github.com/cafebazaar/blacksmith/datasource"
	"github.com/cafebazaar/blacksmith/templating"
	"github.com/cafebazaar/blacksmith/utils"
)

const (
	bootMessageTemplate = `
		Blacksmith ($VERSION) on $HOST
		+ MAC ADDR:	$MAC
`
)

type nodeContext struct {
	IP string
}

type HTTPBooter struct {
	listenAddr          net.TCPAddr
	ldlinux             []byte
	datasource          *datasource.EtcdDatasource
	bootParamsTemplates *template.Template
	webPort             int
	bootMessageTemplate string
}

func NewHTTPBooter(listenAddr net.TCPAddr, ldlinux []byte,
	ds *datasource.EtcdDatasource, webPort int) (*HTTPBooter, error) {
	bootMessageVersionedTemplate := strings.Replace(bootMessageTemplate,
		"$VERSION", ds.SelfInfo().Version, -1)
	bootMessageVersionedTemplate = strings.Replace(bootMessageTemplate,
		"$HOST", ds.SelfInfo().IP.String(), -1)
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
	utils.LogAccess(r).WithField("where", "pxe.ldlinuxHandler").Info()
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(b.ldlinux)
}

func (b *HTTPBooter) pxelinuxConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")

	macStr := filepath.Base(r.URL.Path)
	errStr := fmt.Sprintf("%s requested a pxelinux config from URL %q, which does not include a correct MAC address", r.RemoteAddr, r.URL)
	if !strings.HasPrefix(macStr, "01-") {
		utils.LogAccess(r).WithField("where", "pxe.pxelinuxConfig").Debug(errStr)
		http.Error(w, "Missing MAC address in request", http.StatusBadRequest)
		return
	}
	mac, err := net.ParseMAC(macStr[3:])
	if err != nil {
		utils.LogAccess(r).WithError(err).WithField("where", "pxe.pxelinuxConfig").Debug()
		http.Error(w, "Malformed MAC address in request", http.StatusBadRequest)
		return
	}

	machineInterface := b.datasource.GetMachine(mac)
	_, err = machineInterface.Machine(false, nil)
	if err != nil {
		utils.LogAccess(r).WithError(err).WithField("where", "pxe.pxelinuxConfig").Debug(
			"Machine not found")
		http.Error(w, "Machine not found", http.StatusNotFound)
		return
	}

	if _, _, err := net.SplitHostPort(r.Host); err != nil {
		r.Host = fmt.Sprintf("%s:%d", r.Host, b.listenAddr.Port)
	}

	coreOSVersion, err := machineInterface.GetVariable(datasource.SpecialKeyCoreosVersion)
	if err != nil {
		utils.LogAccess(r).WithError(err).WithField("where", "pxe.pxelinuxConfig").Warn(
			"error in getting coreOSVersion")
		http.Error(w, "error in getting coreOSVersion", 500)
		return
	}

	KernelURL := b.datasource.FileServer() + coreOSVersion + "/coreos_production_pxe.vmlinuz"
	InitrdURL := b.datasource.FileServer() + coreOSVersion + "/coreos_production_pxe_image.cpio.gz"

	host, _, err := net.SplitHostPort(r.Host)
	if err != nil {
		utils.LogAccess(r).WithError(err).WithField("where", "pxe.pxelinuxConfig").Debug(
			"error in parsing host and port")
		http.Error(w, "error in parsing host and port", 500)
		return
	}

	var ccuser string
	_, err = os.Stat(path.Join(b.datasource.WorkspacePath(), "repo", "config", "bootparams", "main"))
	if !os.IsNotExist(err) {
		ccuser, err = templating.ExecuteTemplateFolder(
			path.Join(b.datasource.WorkspacePath(), "repo", "config", "bootparams"), "main", b.datasource, machineInterface)
		if err != nil {
			http.Error(w, fmt.Sprintf(`Error while executing the template: %q`, err), 500)
		}
	}

	tmpl, err := templating.FSString(false, "/files/bootparams")
	if err != nil {
		http.Error(w, "Ebedded template not found: "+err.Error(), 500)
	}
	ccbase, err := templating.ExecuteTemplateFile(tmpl, b.datasource, machineInterface)
	if err != nil {
		http.Error(w, "Ebedded template can't be rendered: "+err.Error(), 500)
	}

	params := strings.Replace(ccbase+ccuser, "\n", " ", -1)

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

	utils.LogAccess(r).WithField("where", "pxe.pxelinuxConfig").Info()
}

// Get the contents of a blob mentioned in a previously issued
// BootSpec. Additionally returns a pretty name for the blob for
// logging purposes.
func (b *HTTPBooter) coreOS(version string, id string) (io.ReadCloser, error) {
	imagePath := b.datasource.FileServer()
	switch id {
	case "kernel":
		path := filepath.Join(imagePath, version, "coreos_production_pxe.vmlinuz")
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

	var (
		f   io.ReadCloser
		err error
	)

	f, err = b.coreOS(version, id)

	if err != nil {
		utils.LogAccess(r).WithError(err).WithField("where", "pxe.fileHandler").Warn(
			"error while getting CoreOS reader")
		http.Error(w, "Couldn't get byte stream", http.StatusInternalServerError)
		return
	}
	defer f.Close()

	w.Header().Set("Content-Type", "application/octet-stream")
	written, err := io.Copy(w, f)
	if err != nil {
		utils.LogAccess(r).WithError(err).WithField("where", "pxe.fileHandler").Debug(
			"error while copying from CoreOS reader")
		return
	}

	utils.LogAccess(r).WithField("where", "pxe.fileHandler").Infof("written=%d", written)
}

func HTTPBooterMux(listenAddr net.TCPAddr, ds *datasource.EtcdDatasource, webPort int) (*http.ServeMux, error) {
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

func ServeHTTPBooter(listenAddr net.TCPAddr, ds *datasource.EtcdDatasource, webPort int) error {
	mux, err := HTTPBooterMux(listenAddr, ds, webPort)
	if err != nil {
		return err
	}

	log.WithFields(log.Fields{
		"where":  "pxe.ServeHTTPBooter",
		"action": "announce",
	}).Infof("Listening on %s", listenAddr.String())

	return http.ListenAndServe(listenAddr.String(), mux)
}
