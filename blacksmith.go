package main // import "github.com/cafebazaar/blacksmith"

import (
	"encoding/base64"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/pflag"

	log "github.com/Sirupsen/logrus"
	etcd "github.com/coreos/etcd/client"
	"github.com/spf13/viper"

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
				/config/cloudconfig/main
				/initial.yaml
`
)

var (
	// Build variables
	version   string
	commit    string
	buildTime string
)

func initConfig() {
	flagset := pflag.NewFlagSet(os.Args[0], pflag.ExitOnError)
	flagset.Bool("verbose", false, "Enable verbose mode")
	flagset.String("config", "", "Set config file")
	flagset.Bool("version", false, "Print version info and exit")
	flagset.Bool("debug", false, "Log more things that aren't directly related to booting a recognized client")
	flagset.String("if", "", "Interface name for DHCP and PXE to listen on")

	flagset.String("http-listen", "", "IP range to listen on for web UI requests")
	flagset.String("api-listen", "", "IP range to listen on for Swagger API requests")
	flagset.String("tls-cert", "", "API server TLS certificate")
	flagset.String("tls-key", "", "API server TLS key")
	flagset.String("tls-ca", "", "API server TLS certificate authority")
	flagset.String("agent-tls-cert", "", "API server TLS certificate")
	flagset.String("agent-tls-key", "", "API server TLS key")
	flagset.String("agent-tls-ca", "", "API server TLS ca")

	flagset.String("workspace", "/workspace", workspacePathHelp)
	flagset.String("workspace-repo", "", "Repository of workspace")
	flagset.String("workspace-repo-branch", "master", "Branch name for the repository of workspace")
	flagset.String("initial-config", "", "initial.yaml")
	flagset.String("file-server", "http://localhost/", "A HTTP server to serve needed files")
	flagset.String("insecure-registry", "localhost:5000", "Local HTTP docker registry")
	flagset.String("etcd", "", "Etcd endpoints")
	flagset.String("cluster-name", "blacksmith", "The name of this cluster. Will be used as etcd path prefixes.")
	flagset.String("dns", "8.8.8.8", "comma separated IPs which will be used as default nameservers for skydns.")
	flagset.String("lease-start", "", "Beginning of lease starting IP")
	flagset.String("private-key", "", "Base64 SSH private key used for cloning private workspace repositories.")
	flagset.Int("lease-range", 0, "Lease range")
	flagset.Parse(os.Args)

	viper.BindPFlags(flagset)
	viper.SetConfigName("config")             // name of config file (without extension)
	viper.AddConfigPath("$HOME/.blacksmith/") // adding home directory as first search path
	viper.AutomaticEnv()                      // read in environment variables that match

	configFilepath := flagset.Lookup("config").Value.String()
	if configFilepath != "" {
		viper.SetConfigFile(configFilepath)
	}

	if err := viper.ReadInConfig(); err != nil {
		if configFilepath != "" {
			log.WithFields(log.Fields{
				"err":         err,
				"config-flag": configFilepath,
			}).Fatal("could not load given config")
		}
	}

	if viper.GetBool("conf.verbose") {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
		for a, b := range viper.AllSettings() {
			log.WithFields(log.Fields{
				"name":  a,
				"value": b,
			}).Info("config")
		}
	}
}

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

func gracefulShutdown(etcdDataSource *datasource.EtcdDatasource) {
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

	initConfig()

	fmt.Printf("Blacksmith (%s)\n", version)
	fmt.Printf("  Commit:        %s\n", commit)
	fmt.Printf("  Build Time:    %s\n", buildTime)

	if viper.GetBool("conf.version") {
		os.Exit(0)
	}

	if viper.GetBool("conf.debug") {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}

	if viper.GetString("conf.etcd") == "" {
		fmt.Fprint(os.Stderr, "\nPlease specify the etcd blacksmith variable\n")
		os.Exit(1)
	}

	if viper.GetString("conf.cluster-name") == "" {
		fmt.Fprint(os.Stderr, "\nPlease specify the cluster-name blacksmith variable\n")
		os.Exit(1)
	}

	// finding interface by interface name
	var dhcpIF *net.Interface
	if viper.GetString("conf.if") != "" {
		dhcpIF, err = net.InterfaceByName(viper.GetString("conf.if"))
		if err != nil {
			fmt.Fprintf(os.Stderr, "\nError while trying to get the interface (%s): %s\n", viper.GetString("conf.if"), err)
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
	webAddr := parseTCPAddr(viper.GetString("conf.http-listen"))
	webAddrSwagger := parseTCPAddr(viper.GetString("conf.api-listen"))

	// other services are exposed just through the given interface, on hard coded ports
	var httpBooterAddr = net.TCPAddr{IP: serverIP, Port: 70}
	var tftpAddr = net.UDPAddr{IP: serverIP, Port: 69}
	var pxeAddr = net.UDPAddr{IP: serverIP, Port: 4011}
	var dnsUDPAddr = net.UDPAddr{IP: serverIP, Port: 53}
	var dnsTCPAddr = net.TCPAddr{IP: serverIP, Port: 53}
	// 67 -> dhcp

	// dhcp setting
	leaseStart := net.ParseIP(viper.GetString("conf.lease-start"))

	dnsIPStrings := strings.Split(viper.GetString("conf.dns"), ",")
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
	if viper.GetInt("conf.lease-range") <= 1 {
		fmt.Fprint(os.Stderr, "\nLease range should be greater that 1\n")
		os.Exit(1)
	}

	fmt.Printf("Interface IP:    %s\n", serverIP.String())
	fmt.Printf("Interface Name:  %s\n", dhcpIF.Name)

	// datasources
	etcdClient, err := etcd.New(etcd.Config{
		Endpoints:               strings.Split(viper.GetString("conf.etcd"), ","),
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
		ServiceStartTime: time.Now().UTC().Unix(),
	}
	etcdDataSource, err := datasource.NewEtcdDataSource(
		kapi,
		etcdClient,
		leaseStart,
		viper.GetInt("conf.lease-range"),
		viper.GetString("conf.cluster-name"),
		viper.GetString("conf.workspace"),
		viper.GetString("conf.workspace-repo"),
		viper.GetString("conf.workspace-repo-branch"),
		viper.GetString("conf.private-key"),
		viper.GetString("conf.initial-config"),
		viper.GetString("conf.file-server"),
		dnsIPStrings,
		selfInfo,
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\nCouldn't create runtime configuration: %s\n", err)
		os.Exit(1)
	}

	etcdDataSource.SetWebServer((&webAddr).String())
	etcdDataSource.SetBlacksmithVariable("blacksmith-image", viper.GetString("conf.blacksmith-image"))
	etcdDataSource.SetBlacksmithVariable("verbose", fmt.Sprintf("%v", viper.GetBool("conf.verbose")))
	etcdDataSource.SetBlacksmithVariable("config", viper.GetString("conf.config"))
	etcdDataSource.SetBlacksmithVariable("version", fmt.Sprintf("%v", viper.GetBool("conf.version")))
	etcdDataSource.SetBlacksmithVariable("debug", fmt.Sprintf("%v", viper.GetBool("conf.debug")))
	etcdDataSource.SetBlacksmithVariable("if", viper.GetString("conf.if"))
	etcdDataSource.SetBlacksmithVariable("http-listen", viper.GetString("conf.http-listen"))
	etcdDataSource.SetBlacksmithVariable("api-listen", viper.GetString("conf.api-listen"))
	etcdDataSource.SetBlacksmithVariable("tls-cert", viper.GetString("conf.tls-cert"))
	etcdDataSource.SetBlacksmithVariable("tls-key", viper.GetString("conf.tls-key"))
	etcdDataSource.SetBlacksmithVariable("tls-ca", viper.GetString("conf.tls-ca"))
	etcdDataSource.SetBlacksmithVariable("agent-tls-cert", viper.GetString("conf.agent-tls-cert"))
	etcdDataSource.SetBlacksmithVariable("agent-tls-key", viper.GetString("conf.agent-tls-key"))
	etcdDataSource.SetBlacksmithVariable("agent-tls-ca", viper.GetString("conf.agent-tls-ca"))
	etcdDataSource.SetBlacksmithVariable("workspace", viper.GetString("conf.workspace"))
	etcdDataSource.SetBlacksmithVariable("workspace-repo", viper.GetString("conf.workspace-repo"))
	etcdDataSource.SetBlacksmithVariable("workspace-repo-branch", viper.GetString("conf.workspace-repo-branch"))
	etcdDataSource.SetBlacksmithVariable("initial-config", viper.GetString("conf.initial-config"))
	etcdDataSource.SetBlacksmithVariable("file-server", viper.GetString("conf.file-server"))
	etcdDataSource.SetBlacksmithVariable("insecure-registry", viper.GetString("conf.insecure-registry"))
	etcdDataSource.SetBlacksmithVariable("etcd", viper.GetString("conf.etcd"))
	etcdDataSource.SetBlacksmithVariable("cluster-name", viper.GetString("conf.cluster-name"))
	etcdDataSource.SetBlacksmithVariable("dns", viper.GetString("conf.dns"))
	etcdDataSource.SetBlacksmithVariable("lease-start", viper.GetString("conf.lease-start"))
	etcdDataSource.SetBlacksmithVariable("lease-range", fmt.Sprintf("%v", viper.GetInt("conf.lease-range")))
	etcdDataSource.SetArrayVariable("ssh-keys", viper.GetStringSlice("ssh-keys"))

	etcdDataSource.SetBlacksmithVariable("agent-url", viper.GetString("conf.agent-url"))

	go func() {
		dns.ServeDNS(dnsTCPAddr, dnsUDPAddr, etcdDataSource)
	}()

	go func() {
		err := web.ServeWeb(etcdDataSource, webAddr)
		log.Fatalf("\nError while serving api: %s\n", err)
	}()

	go func() {
		tlsCert, err := base64.StdEncoding.DecodeString(viper.GetString("conf.tls-cert"))
		if err != nil {
			log.Fatalf("bad conf.tls-cert: %v", err)
		}
		tlsKey, err := base64.StdEncoding.DecodeString(viper.GetString("conf.tls-key"))
		if err != nil {
			log.Fatalf("bad conf.tls-key: %v", err)
		}
		tlsCa, err := base64.StdEncoding.DecodeString(viper.GetString("conf.tls-ca"))
		if err != nil {
			log.Fatalf("bad conf.tls-ca: %v", err)
		}
		if err := web.ServeSwaggerAPI(etcdDataSource, webAddrSwagger, string(tlsCert), string(tlsKey), string(tlsCa)); err != nil {
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

	go etcdDataSource.UpdateMyWorkspaceLoop()

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

	for {
		if err = etcdDataSource.WhileMaster(); err != nil {
			break
		}
		time.Sleep(datasource.ActiveMasterUpdateTime)
	}

	log.WithFields(log.Fields{
		"where":  "blacksmith.main",
		"action": "debug",
		"err":    err,
	}).Debug("Now we're NOT the master. Terminating. Hoping to be restarted by the service manager.")

	gracefulShutdown(etcdDataSource)
}
