package main // import "github.com/cafebazaar/blacksmith"

import (
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"path"
	"strconv"
	"strings"
	"syscall"
	"time"

	"golang.org/x/net/context"

	log "github.com/Sirupsen/logrus"
	etcd "github.com/coreos/etcd/client"

	"github.com/cafebazaar/blacksmith/datasource"
	"github.com/cafebazaar/blacksmith/dhcp"
	"github.com/cafebazaar/blacksmith/dns"
	"github.com/cafebazaar/blacksmith/pxe"
	"github.com/cafebazaar/blacksmith/web"
)

//go:generate esc -o pxe/pxelinux_autogen.go -prefix=pxe -pkg pxe -ignore=README.md pxe/pxelinux
//go:generate esc -o web/ui_autogen.go -prefix=web -ignore=bower_components -pkg web web/static
//go:generate esc -o templating/files_autogen.go -prefix=templating -pkg templating templating/files

const (
	workspacePathHelp = `Path to workspace which obey following structure
		/config/bootparams/main
		/config/cloudconfig/main
		/config/ignition/main
		/images/{core-os-version}/coreos_production_pxe_image.cpio.gz
		/images/{core-os-version}/coreos_production_pxe.vmlinuz
		/initial.yaml
`
)

var (
	versionFlag       = flag.Bool("version", false, "Print version info and exit")
	debugFlag         = flag.Bool("debug", false, "Log more things that aren't directly related to booting a recognized client")
	listenIFFlag      = flag.String("if", "", "Interface name for DHCP and PXE to listen on")
	httpListenFlag    = flag.String("http-listen", "", "IP range to listen on for web UI requests")
	apiListenFlag     = flag.String("api-listen", "", "IP range to listen on for Swagger API requests")
	workspacePathFlag = flag.String("workspace", "/workspace", workspacePathHelp)
	workspaceRepo     = flag.String("workspace-repo", "", "Repository of workspace")
	fileServer        = flag.String("file-server", "http://localhost/", "A HTTP server to serve needed files")
	etcdFlag          = flag.String("etcd", "", "Etcd endpoints")
	clusterNameFlag   = flag.String("cluster-name", "blacksmith", "The name of this cluster. Will be used as etcd path prefixes.")
	dnsAddressesFlag  = flag.String("dns", "8.8.8.8", "comma separated IPs which will be used as default nameservers for skydns.")

	leaseStartFlag = flag.String("lease-start", "", "Beginning of lease starting IP")
	leaseRangeFlag = flag.Int("lease-range", 0, "Lease range")

	version   string
	commit    string
	buildTime string
	debugMode string
)

func init() {
	// If version, commit, or build time are not set, make that clear.
	if version == "" {
		version = "unknown"
	}
	if commit == "" {
		commit = "unknown"
	}
	if buildTime == "" {
		buildTime = "unknown"
	}
	if debugMode == "" {
		debugMode = "false"
	}
}

func interfaceIP(iface *net.Interface) (net.IP, error) {
	addrs, err := iface.Addrs()
	if err != nil {
		return nil, err
	}
	fs := [](func(net.IP) bool){
		net.IP.IsGlobalUnicast,
		net.IP.IsLinkLocalUnicast,
		net.IP.IsLoopback,
	}
	for _, f := range fs {
		for _, a := range addrs {
			ipaddr, ok := a.(*net.IPNet)
			if !ok {
				continue
			}
			ip := ipaddr.IP.To4()
			if ip == nil {
				continue
			}
			if f(ip) {
				return ip, nil
			}
		}
	}
	return nil, fmt.Errorf("interface %s has no usable unicast addresses", iface.Name)
}

func gracefulShutdown(etcdDataSource datasource.DataSource) {
	err := etcdDataSource.Shutdown()
	if err != nil {
		fmt.Fprintf(os.Stderr, "\nError while removing the instance: %s\n", err)
	} else {
		fmt.Fprint(os.Stderr, "\nBlacksmith is gracefully shutdown\n")
	}
	os.Exit(0)
}

func parseTCPAddr(addr string) net.TCPAddr {
	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		fmt.Printf("Incorrect tcp address provided: %s\n", addr)
		os.Exit(1)
	}

	port, err := strconv.ParseInt(portStr, 10, 64)
	if err != nil {
		fmt.Printf("Incorrect tcp port provided: %s\n", portStr)
		os.Exit(1)
	}

	return net.TCPAddr{
		IP:   net.ParseIP(host),
		Port: int(port),
	}
}

func main() {
	var err error
	flag.Parse()

	fmt.Printf("Blacksmith (%s)\n", version)
	fmt.Printf("  Commit:        %s\n", commit)
	fmt.Printf("  Build Time:    %s\n", buildTime)

	if *versionFlag {
		os.Exit(0)
	}

	if *debugFlag {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}

	// etcd config
	if etcdFlag == nil || clusterNameFlag == nil {
		fmt.Fprint(os.Stderr, "\nPlease specify the etcd endpoints\n")
		os.Exit(1)
	}

	// finding interface by interface name
	var dhcpIF *net.Interface
	if *listenIFFlag != "" {
		dhcpIF, err = net.InterfaceByName(*listenIFFlag)
		if err != nil {
			fmt.Fprintf(os.Stderr, "\nError while trying to get the interface (%s): %s\n", *listenIFFlag, err)
			os.Exit(1)
		}
	} else {
		fmt.Fprint(os.Stderr, "\nPlease specify an interface\n")
		os.Exit(1)
	}

	serverIP, err := interfaceIP(dhcpIF)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\nError while trying to get the ip from the interface (%v)\n", dhcpIF)
		os.Exit(1)
	}

	// web api can be configured to listen on a custom address
	webAddr := parseTCPAddr(*httpListenFlag)
	webAddrSwagger := parseTCPAddr(*apiListenFlag)

	// other services are exposed just through the given interface, on hard coded ports
	var httpBooterAddr = net.TCPAddr{IP: serverIP, Port: 70}
	var tftpAddr = net.UDPAddr{IP: serverIP, Port: 69}
	var pxeAddr = net.UDPAddr{IP: serverIP, Port: 4011}
	var dnsUDPAddr = net.UDPAddr{IP: serverIP, Port: 53}
	var dnsTCPAddr = net.TCPAddr{IP: serverIP, Port: 53}
	// 67 -> dhcp

	// dhcp setting
	leaseStart := net.ParseIP(*leaseStartFlag)
	leaseRange := *leaseRangeFlag

	dnsIPStrings := strings.Split(*dnsAddressesFlag, ",")
	if len(dnsIPStrings) == 0 {
		fmt.Fprint(os.Stderr, "\nPlease specify an DNS server\n")
		os.Exit(1)
	}
	for _, ipString := range dnsIPStrings {
		ip := net.ParseIP(ipString)
		if ip == nil {
			fmt.Fprintf(os.Stderr, "\nInvalid dns ip: %s\n", ipString)
			os.Exit(1)
		}
	}

	if leaseStart == nil {
		fmt.Fprint(os.Stderr, "\nPlease specify the lease start ip\n")
		os.Exit(1)
	}
	if leaseRange <= 1 {
		fmt.Fprint(os.Stderr, "\nLease range should be greater that 1\n")
		os.Exit(1)
	}

	fmt.Printf("Interface IP:    %s\n", serverIP.String())
	fmt.Printf("Interface Name:  %s\n", dhcpIF.Name)

	// datasources
	etcdClient, err := etcd.New(etcd.Config{
		Endpoints:               strings.Split(*etcdFlag, ","),
		HeaderTimeoutPerRequest: 5 * time.Second,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "\nCouldn't create etcd connection: %s\n", err)
		os.Exit(1)
	}
	kapi := etcd.NewKeysAPI(etcdClient)

	selfInfo := datasource.InstanceInfo{
		IP:               serverIP,
		Nic:              dhcpIF.HardwareAddr,
		WebPort:          webAddr.Port,
		Version:          version,
		Commit:           commit,
		BuildTime:        buildTime,
		DebugMode:        debugMode,
		ServiceStartTime: time.Now().UTC().Unix(),
	}
	etcdDataSource, err := datasource.NewEtcdDataSource(kapi, etcdClient,
		leaseStart, leaseRange, *clusterNameFlag, *workspacePathFlag,
		*workspaceRepo, *fileServer, dnsIPStrings, selfInfo)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\nCouldn't create runtime configuration: %s\n", err)
		os.Exit(1)
	}

	webAddrPointer := &(webAddr)
	etcdDataSource.SetWebServer(webAddrPointer.String())

	go func() {
		dns.ServeDNS(dnsTCPAddr, dnsUDPAddr, etcdDataSource)
	}()

	go func() {
		err := web.ServeWeb(etcdDataSource, webAddr)
		log.Fatalf("\nError while serving api: %s\n", err)
	}()

	go func() {
		if err := web.ServeSwaggerAPI(etcdDataSource, webAddrSwagger); err != nil {
			log.Fatalf("\nError while serving swagger api: %s\n", err)
		}
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		for range c {
			gracefulShutdown(etcdDataSource)
		}
	}()

	go func() {
		for {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			watcher := kapi.Watcher(path.Join(etcdDataSource.ClusterName(), "workspace-update"), nil)
			defer cancel()
			_, err := watcher.Next(ctx)
			if err != nil {
				continue
			}
			etcdDataSource.UpdateWorkspace()
		}
	}()

	// waiting till we're officially the master instance
	for etcdDataSource.WhileMaster() != nil {
		log.WithFields(log.Fields{
			"where":  "blacksmith.main",
			"action": "debug",
		}).Debug("Not master, waiting to be promoted...")
		time.Sleep(datasource.StandbyMasterUpdateTime)
	}

	log.WithFields(log.Fields{
		"where":  "blacksmith.main",
		"action": "debug",
	}).Debug("Now we're the master instance. Starting the services...")

	// serving http booter
	go func() {
		err := pxe.ServeHTTPBooter(httpBooterAddr, etcdDataSource, webAddr.Port)
		log.Fatalf("\nError while serving http booter: %s\n", err)
	}()

	// serving tftp
	go func() {
		err := pxe.ServeTFTP(tftpAddr)
		log.Fatalf("\nError while serving tftp: %s\n", err)
	}()

	// pxe protocol
	go func() {
		err := pxe.ServePXE(pxeAddr, serverIP, httpBooterAddr)
		log.Fatalf("\nError while serving pxe: %s\n", err)
	}()

	// serving dhcp
	go func() {
		err := dhcp.StartDHCP(dhcpIF.Name, serverIP, etcdDataSource)
		log.Fatalf("\nError while serving dhcp: %s\n", err)
	}()

	for etcdDataSource.WhileMaster() == nil {
		time.Sleep(datasource.ActiveMasterUpdateTime)
	}

	log.WithFields(log.Fields{
		"where":  "blacksmith.main",
		"action": "debug",
	}).Debug("Now we're NOT the master. Terminating. Hoping to be restarted by the service manager.")

	gracefulShutdown(etcdDataSource)
}
